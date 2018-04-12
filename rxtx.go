package main

import (
	"os"

	"encoding/json"

	"fmt"

	"io/ioutil"

	"flag"

	"time"

	"github.com/bhoriuchi/go-bunyan/bunyan"
	"github.com/cjimti/gin-bunyan"
	"github.com/cjimti/rxtx/rtq"
	"github.com/gin-gonic/gin"
)

func main() {
	var port = flag.String("port", "8080", "Server port.")
	var name = flag.String("name", "rxtx", "Service name.")
	var interval = flag.Int("interval", 30, "Seconds between intervals.")
	var batch = flag.Int("batch", 1000, "Batch size.")
	var ingest = flag.String("ingest", "http://localhost:8081/ingest", "Ingest server.")

	flag.Parse()

	logConfig := bunyan.Config{
		Name:   *name,
		Stream: os.Stdout,
		Level:  bunyan.LogLevelDebug,
	}

	blog, err := bunyan.CreateLogger(logConfig)
	if err != nil {
		panic(err)
	}

	blog.Info("Starting rxtx...")

	// database
	q, err := rtq.NewQ("rxtx", rtq.Config{
		Interval: time.Duration(*interval) * time.Second, // send every 10 seconds
		Batch:    *batch,                                 // batch size
		Logger:   &blog,
		Receiver: *ingest, // can receive a POST with JSON txMessageBatch
	})
	if err != nil {
		panic(err)
	}

	// gin config
	gin.SetMode(gin.ReleaseMode)
	gin.DisableConsoleColor()

	// discard default logger
	gin.DefaultWriter = ioutil.Discard

	//get a router
	r := gin.Default()

	// use bunyan logger
	r.Use(ginbunyan.Ginbunyan(&blog))

	r.POST("/rx/:producer/:label/*key", func(c *gin.Context) {
		//var json map[string]interface{}
		producer := c.Param("producer")
		label := c.Param("label")
		key := c.Param("key")

		rawData, _ := c.GetRawData()

		// all data is json
		payload := make(map[string]interface{})
		err := json.Unmarshal(rawData, &payload)
		if err != nil {
			c.JSON(500, gin.H{
				"status":  "FAIL",
				"message": fmt.Sprintf("could not unmarshal json: %s", rawData),
			})
			return
		}

		// build the message
		msg := rtq.Message{
			Producer: producer,
			Label:    label,
			Key:      key,
			Payload:  payload,
		}

		// write the message
		err = q.QWrite(msg)
		if err != nil {
			c.JSON(500, gin.H{
				"status":  "FAIL",
				"message": fmt.Sprintf("failed to write message: %s", err.Error()),
			})
			return
		}

		c.JSON(200, gin.H{
			"status": "OK",
		})

	})

	blog.Info("Listening on port %s", *port)
	// block on server run
	r.Run(":" + *port)
}
