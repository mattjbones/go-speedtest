# Go Speedtest

Runs a speedtest via golang library and posts resutls to InfluxDB (v1) 

## Quickstart 

Create a `.env` file with the following:

```
DB_URL=[influxdb URL]
DB_DATABASE=speedtest
SPEEDTEST_LOCATION=london
POLLING_INTERVAL=1h
```

```
go run main.go
```

or Dockerfile for fun an profit


