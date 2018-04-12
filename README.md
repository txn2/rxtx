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
```bash
curl -w "\n" -d '{"generic": "data"}' -X POST http://localhost:8080/rx/producer/data/test
```

#### Add message to queue every second
```bash
 while true; do curl -w "\n" -d "{\"generic\": \"$RANDOM\"}" -X POST http://localhost:8080/rx/producer/data/test; sleep 1; done
 ```