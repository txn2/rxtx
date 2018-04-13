package rtq

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"encoding/json"

	"github.com/bhoriuchi/go-bunyan/bunyan"
	"github.com/coreos/bbolt"
	"github.com/satori/go.uuid"
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

// Config options for rxtx
type Config struct {
	Interval time.Duration
	Batch    int
	Logger   *bunyan.Logger
	Receiver string
}

// rtQ private struct see NewQ
type rtQ struct {
	db          *bolt.DB                  // the database
	cfg         Config                    // configuration
	mq          chan Message              // message channel
	txSeq       []byte                    // max transmitted sequence
	status      func(args ...interface{}) // status output
	statusError func(args ...interface{}) // status error output
}

// NewQ returns a new rtQ
func NewQ(name string, cfg Config) (*rtQ, error) {
	db, err := bolt.Open(name+".db", 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	}

	// make our message queue bucket
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("mq"))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	mq := make(chan Message, 0)

	go func() {
		// begin kv writer
		for {
			select {
			case msg := <-mq:
				db.Update(func(tx *bolt.Tx) error {
					uuidV4 := uuid.NewV4()

					msg.Time = time.Now()
					msg.Uuid = uuidV4.String()

					b := tx.Bucket([]byte("mq"))
					id, _ := b.NextSequence()

					msg.Seq = fmt.Sprintf("%d%d%d%012d", msg.Time.Year(), msg.Time.Month(), msg.Time.Day(), id)

					buf, err := json.Marshal(msg)
					if err != nil {
						return err
					}
					b.Put([]byte(msg.Seq), buf)

					return nil
				})
			}
		}
	}()

	rtq := &rtQ{
		db:          db,  // database
		cfg:         cfg, // Config
		mq:          mq,  // Message channel
		status:      cfg.Logger.Info,
		statusError: cfg.Logger.Error,
	}

	go rtq.tx() // start transmitting

	return rtq, nil
}

// getMessageBatch starts at the first record and
// builds a MessageBatch for each found key up to the
// batch size.
func (rt *rtQ) getMessageBatch() MessageBatch {
	uuidV4 := uuid.NewV4()
	mb := MessageBatch{
		Uuid: uuidV4.String(),
	}

	rt.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte("mq")).Cursor()

		// get the first rt.cfg.Batch
		i := 1
		for k, v := c.First(); k != nil; k, v = c.Next() {
			msg := Message{}
			json.Unmarshal(v, &msg)
			mb.Messages = append(mb.Messages, msg)
			i++
			if i > rt.cfg.Batch {
				break
			}
		}

		return nil
	})

	mb.Size = len(mb.Messages)

	return mb
}

// transmit attempts to transmit a message batch
func (rt *rtQ) transmit(msgB MessageBatch) error {

	jsonStr, err := json.Marshal(msgB)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", rt.cfg.Receiver, bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	rt.status("Tx Status: %s", resp.Status)

	return nil
}

// tx gets a batch of messages and transmits it to the server (receiver)
// TODO: some math to make sure we keep up
func (rt *rtQ) tx() {

	// get a message batch
	mb := rt.getMessageBatch()

	rt.status("Txing %d Messages.", mb.Size)

	// try to send
	err := rt.transmit(mb)
	if err != nil {
		rt.statusError("Tx Error: %s", err.Error())
		rt.waitTx()
		return
	}

	rt.waitTx()
}

// waitTx sleeps for rt.cfg.Interval * time.Second then performs a tx.
func (rt *rtQ) waitTx() {
	rt.status("Tx Waiting %d seconds.", rt.cfg.Interval/time.Second)
	time.Sleep(rt.cfg.Interval)
	rt.tx() // recursion
}

// Write to the queue
func (rt *rtQ) QWrite(msg Message) error {

	rt.status("Send to channel: %s", msg)
	rt.mq <- msg

	return nil
}
