package main

import (
	"os"

	"encoding/json"

	"fmt"

	"github.com/bhoriuchi/go-bunyan/bunyan"
	"github.com/cjimti/rxtx/rtq"
	"github.com/gin-gonic/gin"
)

func main() {
	port := "8080"

	logConfig := bunyan.Config{
		Name:   "rxtx",
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
		Interval: 10,   // send every 10 seconds
		Batch:    1000, // batch size
	}) // send to server ever 10 seconds
	if err != nil {
		panic(err)
	}

	// server
	gin.SetMode(gin.ReleaseMode)
	gin.DisableConsoleColor()

	// todo: send gin logs to bunyan
	r := gin.Default()

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

	// block on server run
	r.Run(":" + port)
}
