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
	listener net.Listener

	waitGroup *sync.WaitGroup

	hpfeedsChan    chan []byte
	hpfeedsEnabled bool

	port string
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
		port:           port,
	}
}

func (p *PostgresServer) Close() {
	p.listener.Close()
}

func (p *PostgresServer) Listen() {
	defer p.waitGroup.Done()
	log.Infof("Starting to listening on port %s...", p.port)
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
		_, err := conn.Read(buf)
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

		// Send to hpfeeds if turned on
		if p.hpfeedsEnabled {
			p.hpfeedsChan <- buf
		}

		if isSSLRequest(buf) {
			log.Debug("Got ssl request...")
			conn.Write([]byte("N"))
			continue
		}

		if !sentStartup {
			ok := handleStartup(buf, conn)
			if !ok {
				break
			}
			sentStartup = true
			continue
		}

		buffer := readBuf(buf)
		pktType := buffer.string()

		if pktType == "p" {
			log.Debug("Handling password...")
			handlePassword(buffer, conn)
			break
		} else {
			// TODO
			log.Info("TODO")
		}
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

func handleStartup(buff readBuf, conn net.Conn) bool {
	buf := readBuf(buff)
	_ = buf.int32()
	_ = buf.int32()

	startupMap := map[string]string{}
	for len(buf) > 1 {
		k := buf.string()
		v := buf.string()
		startupMap[k] = v
	}

	if userExists(startupMap["user"]) {
		// TODO: Support multiple auth types
		conn.Write(authResponse())
		return true
	}

	conn.Write(userDoesntExistResponse(startupMap["user"]))
	return false
}

// Currently only supports cleartext auth
func authResponse() []byte {
	buf := &writeBuf{
		buf: []byte{'R', 0, 0, 0, 0}, //
		pos: 1,
	}
	// cleartext
	buf.int32(3)
	return buf.wrap()
}

func userExists(user string) bool {
	return USERS_THAT_EXIST[user]
}

func handlePassword(buf readBuf, conn net.Conn) {
	// TODO: Save somewhere
	conn.Write(authFailedResponse())
}

// Taken from network capture and https://www.postgresql.org/docs/9.3/static/protocol-error-fields.html
func authErrorResponse(message string) []byte {
	buf := &writeBuf{
		buf: []byte{'E', 0, 0, 0, 0},
		pos: 1,
	}
	// Severity
	buf.string("SERROR")
	// Code & Position
	buf.string("C08P01")
	// Message
	buf.string("M" + message + "\000")
	return buf.wrap()
}

func authFailedResponse() []byte {
	return authErrorResponse("Auth failed")
}

func userDoesntExistResponse(user string) []byte {
	return authErrorResponse("No such user: " + user)
}
