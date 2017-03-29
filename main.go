package main

import (
	"flag"
	"os"
	"os/signal"
	"strings"
	"syscall"

	log "github.com/Sirupsen/logrus"
)

func init() {
	log.SetLevel(log.InfoLevel)
}

func main() {
	port := flag.String("port", "5432", "port to run pghoney on")
	addr := flag.String("addr", "127.0.0.1", "addr to run pghoney on")
	pgUsers := flag.String("pg-users", "postgres", "comma seperated list of users to say exist in your fake postgres server")
	debug := flag.Bool("debug", false, "debug logging")
	flag.Parse()

	if *debug {
		log.SetLevel(log.DebugLevel)
	}

	hpFeedsConfig := &HpFeedsConfig{
		Enabled: false,
	}
	hpfeedsChannel := make(chan []byte)
	if hpFeedsConfig.Enabled {
		go hpfeedsConnect(hpFeedsConfig, hpfeedsChannel)
	}

	postgresServer := NewPostgresServer(
		*port,
		*addr,
		strings.Split(*pgUsers, ","),
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
