package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/txn2/rxtx/rtq"
	"go.uber.org/zap"
)

func main() {

	var port = flag.String("port", "8080", "Server port.")
	var path = flag.String("path", "./", "Directory to store database.")
	var interval = flag.Int("interval", 60, "Seconds between intervals.")
	var batch = flag.Int("batch", 100, "Batch size.")
	var maxq = flag.Int("maxq", 100000, "Max number of message in queue.")
	var ingest = flag.String("ingest", "http://localhost:8081/in", "Ingest server.")
	var verbose = flag.Bool("verbose", false, "Verbose")

	// Instrumentation
	var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
	var memprofile = flag.String("memprofile", "", "write memory profile to `file`")

	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}

		err = pprof.StartCPUProfile(f)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
	}

	// Instrumentation and Signal handling
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			if sig.String() == "interrupt" {
				fmt.Printf("Exiting on interrupt...")
				if *cpuprofile != "" {
					pprof.StopCPUProfile()
				}

				if *memprofile != "" {
					f, err := os.Create(*memprofile)
					if err != nil {
						fmt.Println("could not create memory profile: " + err.Error())
					}
					runtime.GC() // get up-to-date statistics
					if err := pprof.WriteHeapProfile(f); err != nil {
						fmt.Println("could not write memory profile: " + err.Error())
					}
					err = f.Close()
					if err != nil {
						fmt.Printf("Can not close memory profile: %s\n", err.Error())
						return
					}
				}

				os.Exit(0)
			}
		}
	}()

	zapCfg := zap.NewProductionConfig()
	zapCfg.DisableCaller = true
	zapCfg.DisableStacktrace = true

	logger, err := zapCfg.Build()
	if err != nil {
		fmt.Printf("Can not build logger: %s\n", err.Error())
		return
	}

	err = logger.Sync()
	if err != nil {
		fmt.Println("WARNING: Logger sync error: " + err.Error())
	}

	logger.Info("Starting rxtx...")

	// Prometheus Metrics
	processed := promauto.NewCounter(prometheus.CounterOpts{
		Name: "rxtx_total_messages_received",
		Help: "Total number of messages received.",
	})

	queued := promauto.NewGauge(prometheus.GaugeOpts{
		Name: "rxtx_messages_in_queue",
		Help: "Number os messages in the queue.",
	})

	txBatches := promauto.NewCounter(prometheus.CounterOpts{
		Name: "rxtx_tx_batches",
		Help: "Total number of batch transmissions.",
	})

	txFail := promauto.NewCounter(prometheus.CounterOpts{
		Name: "rxtx_tx_fails",
		Help: "Total number of transaction errors.",
	})

	dbErr := promauto.NewCounter(prometheus.CounterOpts{
		Name: "rxtx_db_errors",
		Help: "Total number database errors.",
	})

	msgErr := promauto.NewCounter(prometheus.CounterOpts{
		Name: "rxtx_msg_errors",
		Help: "Total number message errors.",
	})

	// database
	q, err := rtq.NewQ("rxtx", rtq.Config{
		Interval:   time.Duration(*interval) * time.Second,
		Batch:      *batch,
		MaxInQueue: *maxq,
		Logger:     logger,
		Receiver:   *ingest,
		Path:       *path,
		Processed:  processed,
		Queued:     queued,
		TxBatches:  txBatches,
		TxFail:     txFail,
		DbErr:      dbErr,
		MsgError:   msgErr,
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

	// use zap logger on http
	if *verbose == true {
		r.Use(ginzap.Ginzap(logger, time.RFC3339, true))
	}

	rxRoute := "/rx/:producer/:key/*label"
	r.POST(rxRoute, rtq.RxRouteHandler)
	r.OPTIONS(rxRoute, preflight)

	// Prometheus Metrics
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

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
