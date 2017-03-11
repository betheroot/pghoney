package main

import (
	"time"

	log "github.com/Sirupsen/logrus"
	hpfeeds "github.com/fw42/go-hpfeeds"
)

func init() {
	log.SetLevel(log.DebugLevel)
}

var ()

type HpFeedsConfig struct {
	Host    string `json:"host"`
	Port    int    `json:"port"`
	Channel string `json:"channel"`
	Ident   string `json:"ident"`
	Secret  string `json:"secret"`
	Enabled bool   `json:"enabled"`
}

func hpfeedsConnect(hpfeedsConfig *HpFeedsConfig, hpfeedsChannel chan []byte) {
	backoff := 1
	hp := hpfeeds.NewHpfeeds(hpfeedsConfig.Host, hpfeedsConfig.Port, hpfeedsConfig.Ident, hpfeedsConfig.Secret)
	hp.Log = true

	log.Infof("Connecting to hpfeed at %s:%d", hpfeedsConfig.Host, hpfeedsConfig.Port)
	for {
		err := hp.Connect()
		if err == nil {
			log.Infof("Connected to Hpfeeds server.")
			hp.Publish(hpfeedsConfig.Channel, hpfeedsChannel)

			<-hp.Disconnected

			log.Info("Lost connection to hpfeed.")
		}

		log.Infof("Reconnecting to hpfeed at %s:%d in %d seconds", hpfeedsConfig.Host, hpfeedsConfig.Port, backoff)
		time.Sleep(time.Duration(backoff) * time.Second)
		if backoff <= 20 {
			backoff++
		}
	}
}

func main() {
	hpFeedsConfig := &HpFeedsConfig{
		Enabled: false,
	}
	hpfeedsChannel := make(chan []byte)
	if hpFeedsConfig.Enabled {
		go hpfeedsConnect(hpFeedsConfig, hpfeedsChannel)
	}

	postgresServer := NewPostgresServer(hpfeedsChannel, hpFeedsConfig.Enabled)

	defer postgresServer.Close()
	postgresServer.Listen()
}
