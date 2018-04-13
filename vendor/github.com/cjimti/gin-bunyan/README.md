# Gin Bunyan

[Bunyan (JSON logging)]() middleware for [Gin Gonic](https://gin-gonic.github.io/gin/).

## Use

Go get:
```golang
go get github.com/cjimti/gin-bunyan
```

Import:
```golang
import (
	"io/ioutil"

	"github.com/bhoriuchi/go-bunyan/bunyan"
	"github.com/cjimti/gin-bunyan"
	"github.com/gin-gonic/gin"
)
```

Implementation:
```golang
logConfig := bunyan.Config{
    Name:   "myApplication",
    Stream: os.Stdout,
    Level:  bunyan.LogLevelDebug,
}

blog, err := bunyan.CreateLogger(logConfig)
if err != nil {
    panic(err)
}

// gin config
gin.SetMode(gin.ReleaseMode)

// discard default logger
gin.DefaultWriter = ioutil.Discard

// get a router
r := gin.Default()

// use bunyan logger
r.Use(ginbunyan.Ginbunyan(&blog))
```