package main

import (
	"net"

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
	postgresPacket readBuf
}

func NewPostgresConnection(conn net.Conn) *PostgresConnection {
	return &PostgresConnection{
		buffer:     make([]byte, maxBufSize),
		connection: conn,
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
	pgConn.postgresPacket = readBuf(pgConn.buffer)
	return nil
}

func (pgConn *PostgresConnection) resetBuffer() {
	pgConn.buffer = make([]byte, maxBufSize)
}

func (pgConn *PostgresConnection) isSSLRequest() bool {
	return isSSLRequest(pgConn.buffer)
}

func (pgConn *PostgresConnection) handleSSLRequest() {
	log.Debug("Got ssl request...")
	pgConn.connection.Write([]byte("N"))
}
