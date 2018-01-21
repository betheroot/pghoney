package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
)

type PostgresServer struct {
	listener net.Listener

	waitGroup *sync.WaitGroup

	hpfeedsChan    chan []byte
	hpfeedsEnabled bool

	addr string
	port string

	pgUsers    map[string]bool
	cleartext  bool
	tcpTimeout time.Duration
}

func NewPostgresServer(port string, addr string, users []string, cleartext bool, tcpTimeout time.Duration, hpfeedsChan chan []byte, hpfeedsEnabled bool) *PostgresServer {
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
		tcpTimeout:     tcpTimeout,
		pgUsers:        pgUsers,
	}
}

func (p *PostgresServer) Close() {
	p.waitGroup.Wait()
	p.listener.Close()
}

func (p *PostgresServer) Listen() {
	log.Infof("Starting to listening on %s:%s...", p.addr, p.port)
	for {
		conn, err := p.listener.Accept()
		if err != nil {
			log.Warn("Error accepting: %s", err)
			continue
		}

		pgConn := NewPostgresConnection(conn, p.tcpTimeout)

		p.waitGroup.Add(1)
		go p.handleRequest(pgConn)
	}
}

func (p *PostgresServer) sendToHpFeeds(pgConn *PostgresConnection) error {
	sourceAddr := pgConn.connection.RemoteAddr().String()
	event := HpFeedsEvent{
		Packet:     pgConn.buffer,
		SourceIP:   strings.Split(sourceAddr, ":")[0],
		SourcePort: strings.Split(sourceAddr, ":")[1],
		DestIP:     p.addr,
		DestPort:   p.port,
	}

	eventJson, err := json.Marshal(event)
	if err != nil {
		log.Errorf("Error sending event to hpfeeds. Err: %s", err)
		return err
	}

	select {
	case p.hpfeedsChan <- eventJson:
		log.Debug("Sent event to hpfeeds")
	default:
		log.Warn("Channel full, discarding message - check HPFeeds configuration")
		log.Infof("Discarded buffer: %s", pgConn.buffer)
	}

	return err
}

func (p *PostgresServer) handleRequest(pgConn *PostgresConnection) {
	defer p.waitGroup.Done()
	defer pgConn.Close()

	for {
		err := pgConn.readOffConnection()
		if err != nil {
			handleConnReadError(err)
			break
		}

		// Send to hpfeeds if turned on
		if p.hpfeedsEnabled {
			p.sendToHpFeeds(pgConn)
		}

		pgConn.logger.Debugf("Packet contents: %v", pgConn.buffer)

		//FIXME: Remove conditional complexity
		if pgConn.isSSLRequest() {
			pgConn.handleSSLRequest()
			continue
		}

		if !pgConn.hasSentStartup {
			ok := p.handleStartup(pgConn)
			if !ok {
				break
			}
			pgConn.hasSentStartup = true
			continue
		}

		pktType := pgConn.postgresPacket.string()
		if pktType == "p" {
			p.handlePassword(pgConn)
			break
		} else {
			// TODO
			pgConn.logger.Info("TODO")
		}
	}
}

func (p *PostgresServer) handleStartup(pgConn *PostgresConnection) bool {
	pgConn.logger.Debug("Handling startup message...")
	buf := postgresRequest(pgConn.buffer)
	// Actual length finds the last byte and then adds two, because there is two null terminators at the end of the packet.
	actualLength := indexOfLastFilledByte(buf) + 2
	claimedLength := buf.int32()

	if (actualLength == 0) || (claimedLength != actualLength) {
		pgConn.logger.Debugf("Invalid handshake request received from %s, ", pgConn.connection.RemoteAddr())
		pgConn.logger.Debugf("claimed length: %d, actual length: %d", claimedLength, actualLength)
		pgConn.connection.Write(handshakeErrorResponse())
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
		// TODO: Support more auth types
		// Looking for requesting cleartext passwords would be a good way to finger print
		// pghoney. We should have md5 be the default since it is the postgres default.
		if p.cleartext {
			pgConn.connection.Write(cleartextAuthResponse())
		} else {
			pgConn.connection.Write(md5AuthResponse())
		}
		return true
	}

	pgConn.connection.Write(userDoesntExistResponse(startupMap["user"]))
	return false
}

func (p *PostgresServer) handlePassword(pgConn *PostgresConnection) {
	pgConn.logger.Debug("Handling password")

	buf := postgresRequest(pgConn.buffer)
	// Skip the packet type
	_ = buf.string()
	if p.cleartext {
		_ = buf.next(3) // null terminators and the length
		pgConn.logger.WithFields(log.Fields{
			"cleartext_password": fmt.Sprintf("%v", buf.string()),
		}).Info("Got cleartext password")
	} else {
		// skip the length and `md5` bit
		_ = buf.next(6)
		pgConn.logger.WithFields(log.Fields{
			"md5_hashed_password": fmt.Sprintf("%v", buf.string()),
		}).Info("Got md5 hashed password")
	}
	pgConn.connection.Write(authFailedResponse())
}
