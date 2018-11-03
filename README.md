![rxtx data transmission](mast.jpg)
[![irsync Release](https://img.shields.io/github/release/txn2/rxtx.svg)](https://github.com/txn2/rxtx/releases)
[![Build Status](https://travis-ci.org/txn2/rxtx.svg?branch=master)](https://travis-ci.org/txn2/rxtx)
[![Go Report Card](https://goreportcard.com/badge/github.com/txn2/rxtx)](https://goreportcard.com/report/github.com/txn2/rxtx)
[![Maintainability](https://api.codeclimate.com/v1/badges/c4cbc94c46027f0e3161/maintainability)](https://codeclimate.com/github/txn2/rxtx/maintainability)
[![GoDoc](https://godoc.org/github.com/txn2/irsync/rxtx?status.svg)](https://godoc.org/github.com/txn2/rxtx/rtq)
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Ftxn2%2Frxtx.svg?type=shield)](https://app.fossa.io/projects/git%2Bgithub.com%2Ftxn2%2Frxtx?ref=badge_shield)

[![Docker Container Image Size](https://shields.beevelop.com/docker/image/image-size/txn2/rxtx/latest.svg)](https://hub.docker.com/r/txn2/irsync/)
[![Docker Container Layers](https://shields.beevelop.com/docker/image/layers/txn2/rxtx/latest.svg)](https://hub.docker.com/r/txn2/irsync/)
[![Docker Container Pulls](https://img.shields.io/docker/pulls/txn2/rxtx.svg)](https://hub.docker.com/r/txn2/rxtx/)

# rxtx
**rxtx** is a queue based data collector > data transmitter. Useful for online/offline data collection, back pressure buffering or general queuing. **rxtx** uses [bbolt](https://github.com/coreos/bbolt) maintained by CoreOs, a single file database for storing messages before they can be sent.

[rtbeat](https://github.com/txn2/rtbeat) was developed to consume **rxtx** POST data and publish as events into [elasticsearch], [logstash], [kafka], [redis] or directly to log files.

## Test on MacOs

### Install with [brew]
```bash
brew tap txn2/homebrew-tap
brew install rxtx
```

### Help
```bash
rxtx -h
```

## Test [Docker] Container

### Help
```bash
docker run --rm -it txn2/rxtx -h
```
on arm 6/7 based device:
```bash
docker run --rm -it txn2/rxtx:arm32v6-1.2.0 -h
```


## Test Source

#### Help
```bash
go run ./rxtx.go -h

Usage of rxtx:
  -batch int
        Batch size. (default 5000)
  -ingest string
        Ingest server. (default "http://localhost:8081/in")
  -interval int
        Seconds between intervals. (default 30)
  -maxq int
        Max number of message in queue. (default 2000000)
  -name string
        Service name. (default "rxtx")
  -path string
        Directory to store database. (default "./")
  -port string
        Server port. (default "8080")

```

#### Start server on 8080
```bash
go run ./rxtx.go 
```

#### Add message to queue

The **rxtx** services accepts HTTP **POST** data to an API endpoint in the following form /rx/**PRODUCER**/**KEY**/**LABEL/...**/. One label is required, however as many labels as necessary may be added, separated by a forward slash.

```bash
curl -w "\n" -d "{\"generic\": \"$RANDOM\"}" -X POST http://localhost:8080/rx/me/generic_data/generic/test/data
```

#### Add message to queue every second
```bash
 while true; do curl -w "\n" -d "{\"generic\": \"$RANDOM\"}" -X POST http://localhost:8080/rx/me/generic_data/generic/test/data; sleep 1; done
 ```

#### Add 1000 messages to the queue.
```bash
 time for i in {1..1000}; do curl -w "\n" -d "{\"generic\": \"$RANDOM\"}" -X POST http://localhost:8080/rx/me/generic_data/generic/test/data; done
```

### Profile

```bash
go build ./rxtx.go && time ./rxtx --path=./data/ --cpuprofile=rxtxcpu.prof --memprofile=rxtxmem.prof
```

Browser-based profile viewer:
```bash
go tool pprof -http=:8081 rxtxcpu.prof
```

### Building and Releasing

**rxtx** uses [GORELEASER] to build binaries and [Docker] containers.

#### Test Release Steps

Install [GORELEASER] with [brew] (MacOS):
```bash
brew install goreleaser/tap/goreleaser
```

Build without releasing:
```bash
goreleaser --skip-publish --rm-dist --skip-validate
```

#### Release Steps

- Commit latest changes
- [Tag] a version `git tag -a v2.0 -m "Version 2.0"`
- Push tag `git push origin v2.0`
- Run: `GITHUB_TOKEN=$GITHUB_TOKEN goreleaser --rm-dist`

## Resources

- [elasticsearch]
- [logstash]
- [kafka]
- [redis]
- [GORELEASER]
- [Docker]
- [homebrew]

## License
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Ftxn2%2Frxtx.svg?type=large)](https://app.fossa.io/projects/git%2Bgithub.com%2Ftxn2%2Frxtx?ref=badge_large)

[homebrew]: https://brew.sh/
[brew]: https://brew.sh/
[GORELEASER]: https://goreleaser.com/
[Docker]: https://www.docker.com/
[Tag]: https://git-scm.com/book/en/v2/Git-Basics-Tagging
[elasticsearch]: https://www.elastic.co/
[logstash]: https://www.elastic.co/products/logstash
[kafka]: https://kafka.apache.org/
[redis]: https://redis.io/
