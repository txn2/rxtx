package rtq

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"errors"

	"github.com/coreos/bbolt"
	"github.com/satori/go.uuid"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Message to store and send
type Message struct {
	Seq      string                 `json:"seq"`
	Time     time.Time              `json:"time"`
	Uuid     string                 `json:"uuid"`
	Producer string                 `json:"producer"`
	Label    string                 `json:"label"`
	Key      string                 `json:"key"`
	Payload  map[string]interface{} `json:"payload"`
}

// MessageBatch Holds a batch of Messages for the server
type MessageBatch struct {
	Uuid     string    `json:"uuid"`
	Size     int       `json:"size"`
	Messages []Message `json:"messages"`
}

// Pmx
type Pmx struct {
	Processed         prometheus.Counter
	Queued            prometheus.Gauge
	TxBatches         prometheus.Counter
	TxFail            prometheus.Counter
	DbErr             prometheus.Counter
	MsgError          prometheus.Counter
	ResponseTime      prometheus.Summary
	ResponseTimeAsync prometheus.Summary
	ProcessingTime    prometheus.Summary
	ProcessingErrors  prometheus.Counter
}

// Config options for rxtx
type Config struct {
	Interval   time.Duration
	Batch      int
	MaxInQueue int
	Logger     *zap.Logger
	Receiver   string
	Path       string
	Pmx        Pmx
}

// rtQ private struct see NewQ
type rtQ struct {
	db          *bolt.DB                                  // the database
	cfg         Config                                    // configuration
	mq          chan Message                              // message channel
	remove      chan int                                  // number of records to remove
	txSeq       []byte                                    // max transmitted sequence
	mCount      int                                       // last reported message count
	status      func(msg string, fields ...zapcore.Field) // status output
	statusError func(msg string, fields ...zapcore.Field) // status error output
}

func (rt *rtQ) GetMessageCount() int {
	return rt.mCount
}

// NewQ returns a new rtQ
func NewQ(name string, cfg Config) (*rtQ, error) {
	db, err := bolt.Open(cfg.Path+name+".db", 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	}

	err = ensureMqBucket(db)
	if err != nil {
		return nil, err
	}

	// Prometheus Metrics
	cfg.Pmx.Processed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "rxtx_total_messages_received",
		Help: "Total number of messages received.",
	})

	cfg.Pmx.Queued = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "rxtx_messages_in_queue",
		Help: "Number os messages in the queue.",
	})

	cfg.Pmx.TxBatches = promauto.NewCounter(prometheus.CounterOpts{
		Name: "rxtx_tx_batches",
		Help: "Total number of batch transmissions.",
	})

	cfg.Pmx.TxFail = promauto.NewCounter(prometheus.CounterOpts{
		Name: "rxtx_tx_fails",
		Help: "Total number of transaction errors.",
	})

	cfg.Pmx.DbErr = promauto.NewCounter(prometheus.CounterOpts{
		Name: "rxtx_db_errors",
		Help: "Total number database errors.",
	})

	cfg.Pmx.MsgError = promauto.NewCounter(prometheus.CounterOpts{
		Name: "rxtx_msg_errors",
		Help: "Total number message errors.",
	})

	cfg.Pmx.ResponseTime = promauto.NewSummary(prometheus.SummaryOpts{
		Name: "rxtx_response_time",
		Help: "Time it took to respond to a post.",
	})

	cfg.Pmx.ResponseTimeAsync = promauto.NewSummary(prometheus.SummaryOpts{
		Name: "rxtx_response_time_async",
		Help: "Time it took to respond to a async post.",
	})

	cfg.Pmx.ProcessingTime = promauto.NewSummary(prometheus.SummaryOpts{
		Name: "rxtx_processing_time",
		Help: "Time it took to process a post.",
	})

	cfg.Pmx.ProcessingErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "rxtx_processing_errors",
		Help: "Total number of processing errors.",
	})

	mq := make(chan Message, 0)
	remove := make(chan int, 0)

	go messageHandler(db, mq, remove)

	rtq := &rtQ{
		db:          db,     // database
		cfg:         cfg,    // Config
		mq:          mq,     // Message channel
		remove:      remove, // Remove message channel
		status:      cfg.Logger.Info,
		statusError: cfg.Logger.Error,
	}

	// start transmitting
	go func() {
		// endless loop
		for {
			rtq.tx()
			rtq.status("TransmissionStatus", zapcore.Field{
				Key:     "WaitSecond",
				Type:    zapcore.Int32Type,
				Integer: int64(rtq.cfg.Interval / time.Second),
			})

			// wait for interval to pass
			<-time.After(rtq.cfg.Interval)
		}

	}()

	return rtq, nil
}

// getMessageBatch starts at the first record and
// builds a MessageBatch for each found key up to the
// batch size.
func (rt *rtQ) getMessageBatch() *MessageBatch {
	uuidV4, _ := uuid.NewV4()
	mb := &MessageBatch{
		Uuid: uuidV4.String(),
	}

	err := rt.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("mq"))

		// get bucket stats
		stats := bucket.Stats()

		// metric: messages_in_queue
		rt.cfg.Pmx.Queued.Set(float64(stats.KeyN))

		rt.status("QueueState", zapcore.Field{
			Key:     "TotalRecords",
			Type:    zapcore.Int32Type,
			Integer: int64(stats.KeyN),
		})

		ingressOver := 0
		if rt.mCount > 0 {
			ingressOver = stats.KeyN - rt.mCount
		}
		rt.status("QueueState", zapcore.Field{
			Key:     "NewRecords",
			Type:    zapcore.Int32Type,
			Integer: int64(ingressOver),
		})

		// store the new count
		rt.mCount = stats.KeyN

		c := bucket.Cursor()

		// get the first rt.cfg.Batch
		i := 1
		for k, v := c.First(); k != nil; k, v = c.Next() {
			msg := Message{}
			err := json.Unmarshal(v, &msg)
			if err != nil {
				rt.cfg.Logger.Warn("Can not unmarshal queued entry: " + err.Error())
				continue
			}
			mb.Messages = append(mb.Messages, msg)
			i++
			if i > rt.cfg.Batch {
				break
			}
		}

		return nil
	})

	if err != nil {
		// increment metric db_errors
		rt.cfg.Pmx.DbErr.Inc()

		rt.cfg.Logger.Error("bbolt db View error: " + err.Error())
	}

	mb.Size = len(mb.Messages)

	return mb
}

// transmit attempts to transmit a message batch
func (rt *rtQ) transmit(msgB *MessageBatch) error {

	jsonStr, err := json.Marshal(msgB)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", rt.cfg.Receiver, bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")

	var netTransport = &http.Transport{
		Dial: (&net.Dialer{
			Timeout: 10 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	client := &http.Client{
		Timeout:   time.Second * 60,
		Transport: netTransport,
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return errors.New(resp.Status)
	}

	rt.status("TransmissionStatus", zapcore.Field{
		Key:    "Reponse",
		Type:   zapcore.StringType,
		String: resp.Status,
	})

	return nil
}

// tx gets a batch of messages and transmits it to the server (receiver)
// TODO: some math to make sure we keep up (for now remove the batch if the count is above the threshold)
func (rt *rtQ) tx() {

	// get a message batch
	mb := rt.getMessageBatch()

	if mb.Size < 1 {
		// nothing to send
		rt.status("TransmissionSkipEmpty")
		return
	}

	rt.status("TransmissionAttempt", zapcore.Field{
		Key:     "Messages",
		Type:    zapcore.Int32Type,
		Integer: int64(mb.Size),
	})

	// try to send
	err := rt.transmit(mb)
	if err != nil {

		// increment metric tx_fails
		rt.cfg.Pmx.TxFail.Inc()

		// transmission failed
		rt.status("Transmission", zapcore.Field{
			Key:    "TransmissionError",
			Type:   zapcore.StringType,
			String: err.Error(),
		})

		// is the queue larger than allowed
		if rt.mCount > rt.cfg.MaxInQueue {
			rt.remove <- rt.mCount - rt.cfg.MaxInQueue
		}
		return
	}

	// increment metric tx_batches
	rt.cfg.Pmx.TxBatches.Inc()

	rt.status("TransmissionComplete", zapcore.Field{
		Key:     "RemovingMessages",
		Type:    zapcore.Int32Type,
		Integer: int64(mb.Size),
	})
	rt.remove <- mb.Size
}

// Write to the queue
func (rt *rtQ) QWrite(msg Message) error {

	// increment metric
	rt.cfg.Pmx.Processed.Inc()

	rt.mq <- msg

	return nil
}

// messageHandler listens to the mq and remove channels to add and
// remove messages
func messageHandler(db *bolt.DB, mq chan Message, remove chan int) {
	// begin kv writer
	for {
		select {
		case msg := <-mq:
			err := db.Update(func(tx *bolt.Tx) error {
				uuidV4, _ := uuid.NewV4()

				msg.Time = time.Now()
				msg.Uuid = uuidV4.String()

				b := tx.Bucket([]byte("mq"))
				id, _ := b.NextSequence()

				df := msg.Time.Format("20060102")

				msg.Seq = fmt.Sprintf("%s%011d", df, id)

				buf, err := json.Marshal(msg)
				if err != nil {
					return err
				}
				_ = b.Put([]byte(msg.Seq), buf)

				return nil
			})
			if err != nil {
				fmt.Println("Error updating database. FATAL")
				os.Exit(1)
			}
		case rmi := <-remove:
			err := db.Update(func(tx *bolt.Tx) error {
				bucket := tx.Bucket([]byte("mq"))

				c := bucket.Cursor()

				// get the first rt.cfg.Batch
				i := 1
				for k, _ := c.First(); k != nil; k, _ = c.Next() {
					_ = c.Delete()
					i++
					if i > rmi {
						break
					}
				}

				return nil
			})
			if err != nil {
				fmt.Println("Error updating database. FATAL")
				os.Exit(1)
			}

		}
	}
}

// ensureMqBucket makes a bucket for the message queue
func ensureMqBucket(db *bolt.DB) error {
	// make our message queue bucket
	err := db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("mq"))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		return nil
	})

	return err
}
