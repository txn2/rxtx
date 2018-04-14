package rtq

import (
	"encoding/json"
	"fmt"

	"github.com/gin-gonic/gin"
)

// RxRouteHandler handles the http route for inbound data
func RxRouteHandler(c *gin.Context) {
	//var json map[string]interface{}
	producer := c.Param("producer")
	key := c.Param("key")
	label := c.Param("label")

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
	msg := Message{
		Producer: producer,
		Label:    label,
		Key:      key,
		Payload:  payload,
	}

	// write the message
	q := c.MustGet("Q").(*rtQ)
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

}
