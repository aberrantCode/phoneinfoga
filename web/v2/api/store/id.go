package store

import (
	"crypto/rand"
	"fmt"
)

// NewUUID returns a random RFC 4122 version-4 UUID in canonical string form. It
// uses crypto/rand (no external dependency); an error is returned only if the
// system's secure random source cannot be read.
func NewUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("store: failed to read random bytes for UUID: %w", err)
	}

	// Set the version (4) and variant (10xx) bits per RFC 4122 §4.4.
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
