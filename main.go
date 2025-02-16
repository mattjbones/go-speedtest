package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/showwin/speedtest-go/speedtest"
	"github.com/showwin/speedtest-go/speedtest/transport"

	"github.com/influxdata/influxdb/client/v2"
)

type Result struct {
	Latency    time.Duration
	Download   speedtest.ByteRate
	Upload     speedtest.ByteRate
	Jitter     time.Duration
	PacketLoss transport.PLoss
}

func runSpeedtest(conf *Config) {
	log.Println("running speedtest")
	var userConfig = speedtest.UserConfig{
		LocationFlag: conf.speedtestLocation,
	}
	var speedtestClient = speedtest.New(speedtest.WithUserConfig(&userConfig))
	serverList, _ := speedtestClient.FetchServers()
	targets, _ := serverList.FindServer([]int{})
	log.Printf("using %v, server(s)", len(targets))

	results := make([]Result, 1)
	for _, s := range targets {
		s.PingTest(nil)
		s.DownloadTest()
		s.UploadTest()

		// Note: The unit of s.DLSpeed, s.ULSpeed is bytes per second, this is a float64.
		log.Printf("latency: %s, download: %s, upload: %s, jitter: %s", s.Latency, s.DLSpeed, s.ULSpeed, s.Jitter)
		results = append(results, Result{Latency: s.Latency, Download: s.DLSpeed, Upload: s.ULSpeed, Jitter: s.Jitter, PacketLoss: s.PacketLoss})
		s.Context.Reset() // reset counter
	}

	writeToDb(conf, results)
}

func writeToDb(conf *Config, results []Result) {

	log.Println("writing results")
	// Create a new HTTPClient
	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     conf.dbUrl,
		Username: conf.dbUsername,
		Password: conf.dbPassword,
	})
	if err != nil {
		log.Fatalf("unable to start influxdb: %v", err)
	}
	defer c.Close()

	// Create a new point batch
	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  conf.dbDatabase,
		Precision: "s",
	})
	if err != nil {
		log.Fatalf("cannot create batch: %v", err)
	}

	// Create a point and add to batch
	for _, res := range results {
		tags := map[string]string{"speedtest": "results"}
		fields := map[string]interface{}{
			"download":   res.Download.Mbps(),
			"upload":     res.Upload.Mbps(),
			"latency":    res.Latency.Milliseconds(),
			"jitter":     res.Jitter.Milliseconds(),
			"packetLoss": res.PacketLoss.Loss(),
		}

		pt, err := client.NewPoint("speedtest", tags, fields, time.Now())
		if err != nil {
			log.Fatal(err)
		}
		bp.AddPoint(pt)
	}

	// Write the batch
	if err := c.Write(bp); err != nil {
		log.Fatalf("unable to write results: %v", err)
	}

}

func startScheduler(conf *Config) {
	log.Printf("polling every %v", conf.pollingInterval)
	for {
		time.Sleep(conf.pollingInterval)
		go runSpeedtest(conf)
	}
}

type Config struct {
	dbUrl             string        `env:"DB_URL"`
	dbUsername        string        `env:"DB_USERNAME"`
	dbPassword        string        `env:"DB_PASSWORD"`
	dbDatabase        string        `env:"DB_DATABASE"`
	speedtestLocation string        `env:"SPEEDTEST_LOCATION"`
	pollingInterval   time.Duration `env:"POLLING_INTERVAL"`
}

func main() {
	log.Println("starting go-speedtest")

	err := godotenv.Load()
	if err != nil {
		log.Println("no .env file reading environment")
	}

	pollingInterval, err := time.ParseDuration(os.Getenv("POLLING_INTERVAL"))
	if err != nil {
		log.Fatal("Error parsing POLLING_INTERVAL")
	}

	config := Config{
		dbUrl:             os.Getenv("DB_URL"),
		dbUsername:        os.Getenv("DB_USERNAME"),
		dbPassword:        os.Getenv("DB_PASSWORD"),
		dbDatabase:        os.Getenv("DB_DATABASE"),
		speedtestLocation: os.Getenv("SPEEDTEST_LOCATION"),
		pollingInterval:   pollingInterval,
	}

	go runSpeedtest(&config)
	go startScheduler(&config)

	r := gin.Default()

	r.GET("/health", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.PUT("/speedtest", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"status": "starting"})
		go runSpeedtest(&config)
	})

	log.Println("listening on 8080")
	r.Run()
}
