package main

import (
	"bytes"
	"io"
	"net"

	log "github.com/Sirupsen/logrus"
)

// Initial requests:
// 	SSL Request - 00 00 00 08 04 d2 16 2f
func isSSLRequest(payload []byte) bool {
	sslRequestMagicNumber := []byte{0, 0, 0, 8, 4, 210, 22, 47}
	if bytes.Compare(payload[:8], sslRequestMagicNumber) == 0 {
		return true
	}
	return false
}

// -1 means everything is null
func indexOfLastFilledByte(buf readBuf) int {
	for i := 0; i < len(buf); i += 4 {
		word := buf[i : i+4]
		if isNullWord(word) {
			return i - numberOfTrailingNulls(buf[i-4:i])
		}
	}
	return len(buf) - 1
}

// Takes a word like: %v[108, 0, 0, 0] and returns 3, the number of trailing nulls.
func numberOfTrailingNulls(word []byte) int {
	counter := 0
	for i := len(word) - 1; i >= 0; i-- {
		if word[i] == 0 {
			counter++
		} else {
			return counter
		}
	}
	return counter
}

func isNullWord(word []byte) bool {
	for _, v := range word {
		if v != 0 {
			return false
		}
	}
	return true
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
	// FIXME: Don't hardcode the salt
	buf.bytes([]byte{51, 111, 191, 210})
	return buf.wrap()
}

func authResponsePrefix() *writeBuf {
	return &writeBuf{
		buf: []byte{'R', 0, 0, 0, 0},
		pos: 1,
	}
}

func authFailedResponse() []byte {
	return authErrorResponse("Auth failed")
}

func userDoesntExistResponse(user string) []byte {
	return authErrorResponse("No such user: " + user)
}

// Taken from network capture and https://www.postgresql.org/docs/9.3/static/protocol-error-fields.html
func authErrorResponse(message string) []byte {
	buf := errorResponsePrefix()
	// Severity
	buf.string("SERROR")
	// Code & Position
	buf.string("C08P01")
	// Message
	buf.string("M" + message + "\000")
	return buf.wrap()
}

// Taken from a tcpdump of an nmap scan error
func handshakeErrorResponse() []byte {
	buf := errorResponsePrefix()
	// Severity
	buf.string("SERROR")
	// Code
	buf.string("C0A000")
	// Message - TODO: make more dynamic
	buf.string("Munsupported frontend protocol 65363.19778: server supports 1.0 to 3.0")
	// File
	buf.string("Fpostmaster.c")
	// Line
	buf.string("L2005")
	// Routine
	buf.string("RProcessStartupPacket" + "\000")
	return buf.wrap()
}

func errorResponsePrefix() *writeBuf {
	return &writeBuf{
		buf: []byte{'E', 0, 0, 0, 0},
		pos: 1,
	}
}

func handleConnReadError(err error) {
	if err != io.EOF {
		operr, ok := err.(*net.OpError)
		if ok && operr.Timeout() {
			log.Info("Timed out when reading buffer. Err: %s", err)
			return
		}

		log.Warn("Error reading buffer. Err: %s", err)
	}
}
