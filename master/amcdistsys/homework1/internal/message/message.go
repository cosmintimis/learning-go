package message

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"math/rand/v2"
)

const (
	MessageSize = 1024
	payloadEnd  = 1004 // bytes 0-1003: node index + random (1004 bytes)
	sha1Size    = 20   // bytes 1004-1023: SHA-1 checksum
)

// Message is a fixed-size 1024-byte UDP payload.
type Message struct {
	raw [MessageSize]byte
}

// BuildMessage constructs a new Message for the given sender index.
// Byte 0 = senderIndex, bytes 1-1003 = random, bytes 1004-1023 = SHA-1(bytes 0-1003).
func BuildMessage(senderIndex uint8) *Message {
	m := &Message{}
	m.raw[0] = senderIndex
	for i := 1; i < payloadEnd; i++ {
		m.raw[i] = byte(rand.IntN(256))
	}
	sum := sha1.Sum(m.raw[:payloadEnd])
	copy(m.raw[payloadEnd:], sum[:])
	return m
}

// ParseMessage wraps a raw 1024-byte buffer into a Message.
func ParseMessage(buf []byte) (*Message, error) {
	if len(buf) != MessageSize {
		return nil, fmt.Errorf("ParseMessage: expected %d bytes, got %d", MessageSize, len(buf))
	}
	m := &Message{}
	copy(m.raw[:], buf)
	return m, nil
}

// Bytes returns the raw byte slice for sending over UDP.
func (m *Message) Bytes() []byte {
	return m.raw[:]
}

// SenderIndex returns the sender node index from byte 0.
func (m *Message) SenderIndex() uint8 {
	return m.raw[0]
}

// Verify computes SHA-1 of bytes 0-1003 and compares with stored bytes 1004-1023.
// Returns (sentHex, calculatedHex, ok).
func (m *Message) Verify() (sentHex, calcHex string, ok bool) {
	calc := sha1.Sum(m.raw[:payloadEnd])
	sentHex = hex.EncodeToString(m.raw[payloadEnd:])
	calcHex = hex.EncodeToString(calc[:])
	ok = sentHex == calcHex
	return
}
