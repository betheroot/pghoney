package main

import (
	"bytes"
	"io"
	"net"
	"os"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
)

var (
	// Make configurable
	tcpTimeout = 10 * time.Second
	// TODO: Make configurable + match wire protocol
	maxBufSize = 512
)

var USERS_THAT_EXIST = map[string]bool{"postgres": true}

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

func (p *PostgresServer) handleRequest(conn net.Conn) {
	defer p.waitGroup.Done()
	defer conn.Close()

	sentStartup := false

	buf := make([]byte, maxBufSize)
	for {
		readLen, err := conn.Read(buf)
		if err != nil {
			if err != io.EOF {
				operr, ok := err.(*net.OpError)
				if ok && operr.Timeout() {
					log.Info("Timed out when reading buffer. Err: %s", err)
					break
				}

				log.Warn("Error reading buffer. Err: %s", err)
			}
			break
		}

		buf := buf[:readLen]

		if isSSLRequest(buf) {
			log.Debug("Got ssl request, responding with: 'N'")
			conn.Write([]byte("N"))
			continue
		}

		if !sentStartup {
			handleStartup(buf, conn)
			sentStartup = true
			continue
		}

		log.Info(string(buf))
		log.Fatal("Asd")
	}
}

// Initial requests:
// 	SSL Request - 00 00 00 08 04 d2 16 2f
func isSSLRequest(payload []byte) bool {
	if bytes.Compare(payload[:8], []byte{0, 0, 0, 8, 4, 210, 22, 47}) == 0 {
		return true
	}
	return false
}

func handleStartup(payload []byte, conn net.Conn) {
	buf := readBuf(payload)

	log.Debug("Buffer str: " + string(buf))
	length := buf.int32()
	_ = buf.int32()
	log.Debugf("Length: %d", length)

	startupMap := map[string]string{}
	for len(buf) > 1 {
		k := buf.string()
		v := buf.string()
		startupMap[k] = v
	}
	log.Debugf("Startup Map: %s", startupMap)

	if userExists(startupMap["user"]) {
		// TODO: Support multiple auth types
		conn.Write(authResponse())
	}

	// Write response
}

// Currently only supports cleartext auth
func authResponse() []byte {
	buf := &writeBuf{
		buf: []byte{'R', 0, 0, 0, 0}, //
		pos: 1,
	}
	// cleartext
	buf.int32(3)

	log.Debug("Response:")
	log.Debug(buf.wrap())

	return buf.wrap()
}

func userExists(user string) bool {
	return USERS_THAT_EXIST[user]
}
