package rtq

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func (rt *rtQ) processMessage(msg Message, rawData []byte) error {
	start := time.Now()
	defer rt.cfg.Pmx.ProcessingTime.Observe(float64(time.Since(start).Seconds()))

	// all data is json
	payload := make(map[string]interface{})
	err := json.Unmarshal(rawData, &payload)
	if err != nil {
		// increment metric msg_errors
		rt.cfg.Pmx.MsgError.Inc()
		return errors.New(fmt.Sprintf("could not unmarshal json: %s", rawData))
	}

	msg.Payload = payload

	// write the message
	err = rt.QWrite(msg)
	if err != nil {

		// increment metric msg_errors
		rt.cfg.Pmx.MsgError.Inc()
		return errors.New(fmt.Sprintf("failed to write message: %s", err.Error()))
	}

	return nil
}

// RxRouteHandler handles the http route for inbound data
func (rt *rtQ) RxRouteHandler(c *gin.Context) {
	start := time.Now()
	defer rt.cfg.Pmx.ResponseTime.Observe(float64(time.Since(start).Seconds()))

	rawData, err := c.GetRawData()

	if err != nil {
		rt.cfg.Logger.Error("Payload error", zap.Error(err))
		c.JSON(500, gin.H{
			"status":  "FAIL",
			"message": err.Error(),
		})
		return
	}

	err = rt.processMessage(Message{
		Producer: c.Param("producer"),
		Key:      c.Param("key"),
		Label:    c.Param("label"),
	}, rawData)

	if err != nil {
		rt.cfg.Logger.Error("Message processing er, or", zap.Error(err))
		c.JSON(500, gin.H{
			"status":  "FAIL",
			"message": err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"status": "OK",
	})
}

func (rt *rtQ) RxRouteHandlerAsync(c *gin.Context) {
	start := time.Now()

	rawData, err := c.GetRawData()

	if err != nil {
		rt.cfg.Logger.Error("Payload error", zap.Error(err))
		c.JSON(500, gin.H{
			"status":  "FAIL",
			"message": err.Error(),
		})
	}

	msg := Message{
		Producer: c.Param("producer"),
		Key:      c.Param("key"),
		Label:    c.Param("label"),
	}

	go func(msg Message, rawData []byte) {
		err := rt.processMessage(msg, rawData)
		if err != nil {
			rt.cfg.Logger.Error("Message processing error", zap.Error(err))
			c.JSON(500, gin.H{
				"status":  "FAIL",
				"message": err.Error(),
			})
		}
	}(msg, rawData)

	rt.cfg.Pmx.ResponseTimeAsync.Observe(float64(time.Since(start).Seconds()))
	c.JSON(200, gin.H{
		"status": "OK",
	})
}
