package main

import (
	"flag"
	"io/ioutil"
	"os"
	"time"

	"net/http"

	"github.com/bhoriuchi/go-bunyan/bunyan"
	"github.com/cjimti/gin-bunyan"
	"github.com/cjimti/rxtx/rtq"
	"github.com/gin-gonic/gin"
)

func main() {
	var port = flag.String("port", "8080", "Server port.")
	var path = flag.String("path", "./", "Directory to store database.")
	var name = flag.String("name", "rxtx", "Service name.")
	var interval = flag.Int("interval", 30, "Seconds between intervals.")
	var batch = flag.Int("batch", 5000, "Batch size.")
	var maxq = flag.Int("maxq", 2000000, "Max number of message in queue.")
	var ingest = flag.String("ingest", "http://localhost:8081/in", "Ingest server.")

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
		Interval:   time.Duration(*interval) * time.Second, // send every 10 seconds
		Batch:      *batch,                                 // batch size
		Logger:     &blog,
		Receiver:   *ingest, // can receive a POST with JSON txMessageBatch
		MaxInQueue: *maxq,
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

	// get a router
	r := gin.Default()

	// add queue to the context
	r.Use(func(c *gin.Context) {
		c.Set("Q", q)
		c.Next()
	})

	// use bunyan logger
	r.Use(ginbunyan.Ginbunyan(&blog))

	rxRoute := "/rx/:producer/:key/*label"
	r.POST(rxRoute, rtq.RxRouteHandler)
	r.OPTIONS(rxRoute, preflight)

	blog.Info("Listening on port %s", *port)
	// block on server run
	r.Run(":" + *port)
}

// CORS
func preflight(c *gin.Context) {
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Headers", "access-control-allow-origin, access-control-allow-headers, content-type")
	c.JSON(http.StatusOK, struct{}{})
}
