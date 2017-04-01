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
	// TODO: Make configurable
	tcpTimeout = 10 * time.Second
	// TODO: Make configurable + match wire protocol
	maxBufSize = 512
)

type PostgresServer struct {
	listener net.Listener

	waitGroup *sync.WaitGroup

	hpfeedsChan    chan []byte
	hpfeedsEnabled bool

	addr string
	port string

	pgUsers   map[string]bool
	cleartext bool
}

func NewPostgresServer(port string, addr string, users []string, cleartext bool, hpfeedsChan chan []byte, hpfeedsEnabled bool) *PostgresServer {
	listener, err := net.Listen("tcp", addr+":"+port)
	if err != nil {
		log.Errorf("Error listening: %s", err)
		os.Exit(1)
	}

	pgUsers := map[string]bool{}
	for _, u := range users {
		pgUsers[u] = true
	}

	return &PostgresServer{
		listener:       listener,
		waitGroup:      new(sync.WaitGroup),
		hpfeedsChan:    hpfeedsChan,
		hpfeedsEnabled: hpfeedsEnabled,
		addr:           addr,
		port:           port,
		cleartext:      cleartext,
		pgUsers:        pgUsers,
	}
}

func (p *PostgresServer) Close() {
	p.listener.Close()
	p.waitGroup.Done()
}

func (p *PostgresServer) Listen() {
	defer p.waitGroup.Done()
	log.Infof("Starting to listening on %s:%s...", p.addr, p.port)
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
			select {
			case p.hpfeedsChan <- buf:
			default:
				log.Warn("Channel full, discarding message - check HPFeeds configuration")
				log.Infof("Discarded buffer: %s", buf)
			}
		}

		if isSSLRequest(buf) {
			log.Debug("Got ssl request...")
			conn.Write([]byte("N"))
			continue
		}

		if !sentStartup {
			log.Debug("Handling startup message...")
			ok := p.handleStartup(buf, conn)
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

// -1 means everything is null
func indexOfLastFilledByte(buf readBuf) int {
	for i := 0; i < len(buf); i += 4 {
		word := buf[i : i+4]
		if isNullWord(word) {
			return i - 1
		}
	}
	return len(buf) - 1
}

func isNullWord(word []byte) bool {
	for _, v := range word {
		if v != 0 {
			return false
		}
	}
	return true
}

func (p *PostgresServer) handleStartup(buff readBuf, conn net.Conn) bool {
	buf := readBuf(buff)
	// Read out the initial two numbers so we are just left with the k/v pairs.
	actualLength := indexOfLastFilledByte(buf) + 1
	claimedLength := buf.int32()

	if (actualLength == 0) || (claimedLength != actualLength) {
		log.Debugf("Invalid handshake request received from %s, ", conn.RemoteAddr())
		log.Debugf("claimed length: %d, actual length: %d", claimedLength, actualLength)
		conn.Write(handshakeErrorResponse())
		return true
	}
	_ = buf.int32()

	startupMap := map[string]string{}
	for len(buf) > 1 {
		k := buf.string()
		v := buf.string()
		startupMap[k] = v
	}

	if p.pgUsers[startupMap["user"]] {
		// TODO: Support multiple auth types
		// Looking for requesting cleartext passwords would be a good way to finger print
		// pghoney. We should have md5 be the default since it is the postgres default.
		if p.cleartext {
			conn.Write(cleartextAuthResponse())
		} else {
			conn.Write(md5AuthResponse())
		}
		return true
	}

	conn.Write(userDoesntExistResponse(startupMap["user"]))
	return false
}

func cleartextAuthResponse() []byte {
	buf := authResponsePrefix()
	// cleartext
	buf.int32(3)
	return buf.wrap()
}

func md5AuthResponse() []byte {
	buf := authResponsePrefix()
	// md5
	buf.int32(5)
	// Byte4 - "The salt to use when encrypting the password."
	// TODO:
	// Should this be hardcoded to 33 6f b7 d2 ? Feels like a good way to fingerprint pghoney.
	buf.bytes([]byte{51, 111, 191, 210})
	return buf.wrap()
}

func authResponsePrefix() *writeBuf {
	return &writeBuf{
		buf: []byte{'R', 0, 0, 0, 0},
		pos: 1,
	}
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

// Taken from a tcpdump of an nmap scan error
func handshakeErrorResponse() []byte {
	return []byte{69, 0, 0, 0, 132, 83, 70, 65, 84, 65, 76, 0, 67, 48, 65, 48, 48, 48,
		0, 77, 117, 110, 115, 117, 112, 112, 111, 114, 116, 101, 100, 32, 102, 114,
		111, 110, 116, 101, 110, 100, 32, 112, 114, 111, 116, 111, 99, 111, 108,
		32, 54, 53, 51, 54, 51, 46, 49, 57, 55, 55, 56, 58, 32, 115, 101, 114, 118,
		101, 114, 32, 115, 117, 112, 112, 111, 114, 116, 115, 32, 49, 46, 48, 32,
		116, 111, 32, 51, 46, 48, 0, 70, 112, 111, 115, 116, 109, 97, 115, 116,
		101, 114, 46, 99, 0, 76, 50, 48, 48, 53, 0, 82, 80, 114, 111, 99, 101,
		115, 115, 83, 116, 97, 114, 116, 117, 112, 80, 97, 99, 107, 101, 116, 0, 0}
}
