[![Build Status](https://travis-ci.org/cjimti/rxtx.svg?branch=master)](https://travis-ci.org/cjimti/rxtx)

[![Go Report Card](https://goreportcard.com/badge/github.com/cjimti/rxtx)](https://goreportcard.com/report/github.com/cjimti/rxtx)
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fcjimti%2Frxtx.svg?type=shield)](https://app.fossa.io/projects/git%2Bgithub.com%2Fcjimti%2Frxtx?ref=badge_shield)

[![Maintainability](https://api.codeclimate.com/v1/badges/c4cbc94c46027f0e3161/maintainability)](https://codeclimate.com/github/cjimti/rxtx/maintainability)

[![GoDoc](https://godoc.org/github.com/cjimti/irsync/rxtx?status.svg)](https://godoc.org/github.com/cjimti/rxtx/rtq)

# rxtx
[wip] Data collector / Data transmitter

## Test

#### Help
```bash
go run ./rxtx.go -h

Usage of rxtx:
  -batch int
        Batch size. (default 1000)
  -ingest string
        Ingest server. (default "http://localhost:8081/ingest")
  -interval int
        Seconds between intervals. (default 30)
  -name string
        Service name. (default "rxtx")
  -port string
        Server port. (default "8080")

```

#### Start server on 8080
```bash
go run ./rxtx.go 
```

#### Add message to queue

The **rxtx** services accepts http **POST** data to an API endpoint in the following form /rx/**PRODUCER**/**KEY**/**LABEL/...**/. One label is required, however as many labels as nessary may be added, separated by a forward slash.

```bash
curl -w "\n" -d "{\"generic\": \"$RANDOM\"}" -X POST http://localhost:8080/rx/me/generic_data/generic/test/data
```

#### Add message to queue every second
```bash
 while true; do curl -w "\n" -d "{\"generic\": \"$RANDOM\"}" -X POST http://localhost:8080/rx/me/generic_data/generic/test/data; sleep 1; done
 ```

## License
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fcjimti%2Frxtx.svg?type=large)](https://app.fossa.io/projects/git%2Bgithub.com%2Fcjimti%2Frxtx?ref=badge_large)