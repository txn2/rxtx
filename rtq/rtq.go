package rtq

import (
	"fmt"
	"time"

	"encoding/json"

	"strconv"

	"github.com/coreos/bbolt"
	"github.com/satori/go.uuid"
)

// Message to store and send
type Message struct {
	Seq      int                    `json:"seq"`
	Time     time.Time              `json:"time"`
	Uuid     string                 `json:"uuid"`
	Producer string                 `json:"producer"`
	Label    string                 `json:"label"`
	Key      string                 `json:"key"`
	Payload  map[string]interface{} `json:"payload"`
}

// Config options for rxtx
type Config struct {
	Interval time.Duration
	Batch    int
}

// rtQ private struct see NewQ
type rtQ struct {
	db    *bolt.DB     // the database
	cfg   Config       // configuration
	mq    chan Message // message channel
	txSeq int          // max transmitted sequence
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
					msg.Time = time.Now()
					msg.Uuid = uuid.NewV4().String()

					b := tx.Bucket([]byte("mq"))
					id, _ := b.NextSequence()

					msg.Seq = int(id)
					buf, err := json.Marshal(msg)
					if err != nil {
						return err
					}

					err = b.Put([]byte(strconv.Itoa(msg.Seq)), buf)

					return err
				})
			}
		}
	}()

	rtq := &rtQ{
		db:    db,  // database
		cfg:   cfg, // Config
		mq:    mq,  // Message channel
		txSeq: 0,   // highest sequence sent
	}

	go rtq.tx() // start transmitting

	return rtq, nil
}

// transmit batches of 1000 at a time to the server
// TODO: some math to make sure we keep up
func (rt *rtQ) tx() {
	fmt.Println("GOT TX:")
	rt.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte("mq")).Cursor()

		// Our time range spans the 90's decade.
		//min := []byte(strconv.Itoa(rt.txSeq))
		//max := []byte(strconv.Itoa(rt.txSeq + rt.cfg.Batch))

		// gest the first rt.cfg.Batch
		i := 0
		for k, v := c.First(); k != nil; k, v = c.Next() {
			fmt.Printf("key=%s, value=%s\n", k, v)
			i++
			if i > 10 {
				break
			}
		}

		return nil
	})

	time.Sleep(rt.cfg.Interval)
	rt.tx() // recursion
}

// Write to the queue
func (rt *rtQ) QWrite(msg Message) error {

	fmt.Printf("Send to channel: %s", msg)
	rt.mq <- msg

	return nil
}
