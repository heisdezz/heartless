package id

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// New generates a unique identifier with an optional prefix.
func New(prefix string) string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		ts := time.Now().UTC().UnixNano()
		if prefix == "" {
			return fmt.Sprintf("%x", ts)
		}
		return fmt.Sprintf("%s_%x", prefix, ts)
	}
	s := hex.EncodeToString(b[:])
	if prefix == "" {
		return s
	}
	return prefix + "_" + s
}
