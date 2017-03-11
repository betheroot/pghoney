package main

import (
	"net"
	"os"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
)

var (
	// Make configurable
	tcpTimeout = 60 * time.Second
	// Make configurable
	maxBufSize = 4096 * 20
)

type PostgresServer struct {
	listener       net.Listener
	waitGroup      *sync.WaitGroup
	hpfeedsChan    chan []byte
	hpfeedsEnabled bool
}

func NewPostgresServer(hpfeedsChan chan []byte, hpfeedsEnabled bool) *PostgresServer {
	host := "127.0.0.1"
	port := "5433"
	l, err := net.Listen("tcp", host+":"+port)
	if err != nil {
		log.Errorf("Error listening: %s", err)
		os.Exit(1)
	}

	return &PostgresServer{
		listener:       l,
		waitGroup:      new(sync.WaitGroup),
		hpfeedsChan:    hpfeedsChan,
		hpfeedsEnabled: hpfeedsEnabled,
	}
}

func (p *PostgresServer) Close() {
	p.listener.Close()
}

func (p *PostgresServer) Listen() {
	defer p.waitGroup.Done()
	log.Debug("Starting to listen...")
	for {
		conn, err := p.listener.Accept()
		if err != nil {
			log.Warn("Error accepting: %s", err)
			continue
		}

		p.waitGroup.Add(1)
		conn.SetDeadline(time.Now().Add(tcpTimeout))

		go p.handleRequest(conn)
	}
}

func (p *PostgresServer) handleRequest(conn net.Conn) error {
	defer p.waitGroup.Done()
	buf := make([]byte, maxBufSize)
	_, err := conn.Read(buf)
	if err != nil {
		log.Warn("Error reading buffer: %s", err)
	}

	conn.Write([]byte("Response yo"))
	conn.Close()

	return nil
}
