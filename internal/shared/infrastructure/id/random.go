package id

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

type RandomIDGenerator struct{}

func NewRandomIDGenerator() *RandomIDGenerator {
	return &RandomIDGenerator{}
}

func (g *RandomIDGenerator) NewID() string {
	now := time.Now().UTC().Format("20060102T150405.000000000")
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return now
	}
	return now + "-" + hex.EncodeToString(b[:])
}
