package main

// Buffer type (taken directly from https://github.com/lib/pq/blob/master/buf.go)

import (
	"bytes"
	"encoding/binary"

	log "github.com/Sirupsen/logrus"
	"github.com/lib/pq/oid"
)

type postgresRequest []byte

func (b *postgresRequest) int32() (n int) {
	n = int(int32(binary.BigEndian.Uint32(*b)))
	*b = (*b)[4:]
	return
}

func (b *postgresRequest) oid() (n oid.Oid) {
	n = oid.Oid(binary.BigEndian.Uint32(*b))
	*b = (*b)[4:]
	return
}

// N.B: this is actually an unsigned 16-bit integer, unlike int32
func (b *postgresRequest) int16() (n int) {
	n = int(binary.BigEndian.Uint16(*b))
	*b = (*b)[2:]
	return
}

func (b *postgresRequest) string() string {
	i := bytes.IndexByte(*b, 0)
	if i < 0 {
		log.Error("invalid message format; expected string terminator")
	}
	s := (*b)[:i]
	*b = (*b)[i+1:]
	return string(s)
}

func (b *postgresRequest) next(n int) (v []byte) {
	v = (*b)[:n]
	*b = (*b)[n:]
	return
}

func (b *postgresRequest) byte() byte {
	return b.next(1)[0]
}

type postgresResponse struct {
	buf []byte
	pos int
}

func (b *postgresResponse) int32(n int) {
	x := make([]byte, 4)
	binary.BigEndian.PutUint32(x, uint32(n))
	b.buf = append(b.buf, x...)
}

func (b *postgresResponse) int16(n int) {
	x := make([]byte, 2)
	binary.BigEndian.PutUint16(x, uint16(n))
	b.buf = append(b.buf, x...)
}

func (b *postgresResponse) string(s string) {
	b.buf = append(b.buf, (s + "\000")...)
}

func (b *postgresResponse) byte(c byte) {
	b.buf = append(b.buf, c)
}

func (b *postgresResponse) bytes(v []byte) {
	b.buf = append(b.buf, v...)
}

func (b *postgresResponse) wrap() []byte {
	p := b.buf[b.pos:]
	binary.BigEndian.PutUint32(p, uint32(len(p)))
	return b.buf
}

func (b *postgresResponse) next(c byte) {
	p := b.buf[b.pos:]
	binary.BigEndian.PutUint32(p, uint32(len(p)))
	b.pos = len(b.buf) + 1
	b.buf = append(b.buf, c, 0, 0, 0, 0)
}
