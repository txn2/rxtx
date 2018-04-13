![rxtx data transmission](mast.jpg)
[![Build Status](https://travis-ci.org/cjimti/rxtx.svg?branch=master)](https://travis-ci.org/cjimti/rxtx)
[![Go Report Card](https://goreportcard.com/badge/github.com/cjimti/rxtx)](https://goreportcard.com/report/github.com/cjimti/rxtx)
[![Maintainability](https://api.codeclimate.com/v1/badges/c4cbc94c46027f0e3161/maintainability)](https://codeclimate.com/github/cjimti/rxtx/maintainability)
[![GoDoc](https://godoc.org/github.com/cjimti/irsync/rxtx?status.svg)](https://godoc.org/github.com/cjimti/rxtx/rtq)
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fcjimti%2Frxtx.svg?type=shield)](https://app.fossa.io/projects/git%2Bgithub.com%2Fcjimti%2Frxtx?ref=badge_shield)

# rxtx
[wip] Data collector / Data transmitter

## Test

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

The **rxtx** services accepts http **POST** data to an API endpoint in the following form /rx/**PRODUCER**/**KEY**/**LABEL/...**/. One label is required, however as many labels as nessary may be added, separated by a forward slash.

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

#### Release Steps

- Commit latest changes
- [Tag] a version `git tag -a v2.0 -m "Version 2.0"`
- Push tag `git push origin v2.0`
- Run: `GITHUB_TOKEN=$GITHUB_TOKEN goreleaser --rm-dist`

## Resources

- [GORELEASER]
- [Docker]
- [homebrew]


## License
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fcjimti%2Frxtx.svg?type=large)](https://app.fossa.io/projects/git%2Bgithub.com%2Fcjimti%2Frxtx?ref=badge_large)



[homebrew]: https://brew.sh/
[brew]: https://brew.sh/
[GORELEASER]: https://goreleaser.com/
[Docker]: https://www.docker.com/
[Tag]: https://git-scm.com/book/en/v2/Git-Basics-Tagging
