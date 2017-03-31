package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	log "github.com/Sirupsen/logrus"
)

type Configuration struct {
	Port          int      `json:"port"`
	Address       string   `json:"address"`
	PgUsers       []string `json:"pgUsers"`
	Debug         bool     `json:"debug"`
	Cleartext     bool     `json:"cleartext"`
	HpFeedsConfig `json:"hpfeedsConfig"`
}

func init() {
	log.SetLevel(log.InfoLevel)
}

func configurationFrom(configFile string) Configuration {
	var config Configuration
	jsonConfig, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Fatalf("Couldn't %s", err)
	}

	err = json.Unmarshal(jsonConfig, &config)
	if err != nil {
		log.Fatalf("Couldn't parse JSON in %s: %s", configFile, err)
	}

	return config
}

func main() {
	configFile := flag.String("config", "pghoney.conf", "JSON configuration file")
	flag.Parse()
	config := configurationFrom(*configFile)

	port := strconv.Itoa(config.Port)
	addr := config.Address
	pgUsers := config.PgUsers
	debug := config.Debug
	cleartext := config.Cleartext
	hpFeedsConfig := config.HpFeedsConfig

	if debug {
		log.SetLevel(log.DebugLevel)
	}

	hpfeedsChannel := make(chan []byte, 1024)
	if hpFeedsConfig.Enabled {
		go hpfeedsConnect(&hpFeedsConfig, hpfeedsChannel)
	}

	postgresServer := NewPostgresServer(
		port,
		addr,
		pgUsers,
		cleartext,
		hpfeedsChannel,
		hpFeedsConfig.Enabled,
	)

	// Capture 'shutdown' signals and shutdown gracefully.
	shutdownSignal := make(chan os.Signal)
	signal.Notify(shutdownSignal, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		sig := <-shutdownSignal
		log.Infof("Process got signal: %s", sig)
		log.Infof("Shutting down...")

		postgresServer.Close()

		os.Exit(0)
	}()

	postgresServer.Listen()
}
