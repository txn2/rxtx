# rxtx
[wip] Data collector / Data transmitter

## Test

Start server on 8080:
```bash
go run ./rxtx.go 
```

Add message to queue:
```bash
curl -w "\n" -d '{"generic": "data"}' -X POST http://localhost:8080/rx/producer/data/test
```