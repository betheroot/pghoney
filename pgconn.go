package main

import (
	"net"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
)

var (
	// TODO: Make configurable + match wire protocol
	maxBufSize = 512
)

type PostgresConnection struct {
	buffer         []byte
	connection     net.Conn
	hasSentStartup bool
	postgresPacket postgresRequest
	logger         *log.Entry
}

func NewPostgresConnection(conn net.Conn, tcpTimeout time.Duration) *PostgresConnection {
	conn.SetDeadline(time.Now().Add(tcpTimeout))

	remote_addr := strings.Split(conn.RemoteAddr().String(), ":")
	source_ip := remote_addr[0]
	source_port := remote_addr[1]
	connectionLogger := log.WithFields(log.Fields{
		"source_ip":   source_ip,
		"source_port": source_port,
	})
	return &PostgresConnection{
		buffer:     make([]byte, maxBufSize),
		connection: conn,
		logger:     connectionLogger,
	}
}

func (pgConn *PostgresConnection) Close() error {
	return pgConn.connection.Close()
}

func (pgConn *PostgresConnection) readOffConnection() error {
	pgConn.resetBuffer()
	_, err := pgConn.connection.Read(pgConn.buffer)
	if err != nil {
		return err
	}
	pgConn.postgresPacket = postgresRequest(pgConn.buffer)
	return nil
}

func (pgConn *PostgresConnection) resetBuffer() {
	pgConn.buffer = make([]byte, maxBufSize)
}

func (pgConn *PostgresConnection) isSSLRequest() bool {
	return isSSLRequest(pgConn.buffer)
}

func (pgConn *PostgresConnection) handleSSLRequest() {
	pgConn.logger.Debug("Got ssl request...")
	pgConn.connection.Write([]byte("N"))
}
