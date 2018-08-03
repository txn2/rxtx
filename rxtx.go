package main

import (
	"flag"
	"io/ioutil"
	"net/http"
	"time"

	"fmt"

	"github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/txn2/rxtx/rtq"
	"go.uber.org/zap"
)

func main() {
	var port = flag.String("port", "8080", "Server port.")
	var path = flag.String("path", "./", "Directory to store database.")
	//var name = flag.String("name", "rxtx", "Service name.")
	var interval = flag.Int("interval", 60, "Seconds between intervals.")
	var batch = flag.Int("batch", 100, "Batch size.")
	var maxq = flag.Int("maxq", 100000, "Max number of message in queue.")
	var ingest = flag.String("ingest", "http://localhost:8081/in", "Ingest server.")

	flag.Parse()

	zapCfg := zap.NewProductionConfig()
	zapCfg.DisableCaller = true
	zapCfg.DisableStacktrace = true

	logger, err := zapCfg.Build()
	if err != nil {
		fmt.Printf("Can not build logger: %s\n", err.Error())
		return
	}

	logger.Sync()

	logger.Info("Starting rxtx...")

	// database
	q, err := rtq.NewQ("rxtx", rtq.Config{
		Interval:   time.Duration(*interval) * time.Second,
		Batch:      *batch,
		MaxInQueue: *maxq,
		Logger:     logger,
		Receiver:   *ingest,
		Path:       *path,
	})
	if err != nil {
		panic(err)
	}

	// gin config
	gin.SetMode(gin.ReleaseMode)
	gin.DisableConsoleColor()

	// discard default logger
	gin.DefaultWriter = ioutil.Discard

	// gin router
	r := gin.New()

	// add queue to the context
	r.Use(func(c *gin.Context) {
		c.Set("Q", q)
		c.Next()
	})

	// use zap logger
	r.Use(ginzap.Ginzap(logger, time.RFC3339, true))

	rxRoute := "/rx/:producer/:key/*label"
	r.POST(rxRoute, rtq.RxRouteHandler)
	r.OPTIONS(rxRoute, preflight)

	logger.Info("Listening on port: " + *port)
	// block on server run
	r.Run(":" + *port)
}

// preflight permissive CORS headers
func preflight(c *gin.Context) {
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Headers", "access-control-allow-origin, access-control-allow-headers, content-type")
	c.JSON(http.StatusOK, struct{}{})
}
