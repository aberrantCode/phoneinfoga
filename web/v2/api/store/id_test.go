package store

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

var uuidV4Pattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestNewUUIDFormat(t *testing.T) {
	id, err := NewUUID()
	assert.NoError(t, err)
	assert.Regexp(t, uuidV4Pattern, id, "id %q must be a canonical UUIDv4", id)
}

func TestNewUUIDUniqueness(t *testing.T) {
	const n = 1000
	seen := make(map[string]struct{}, n)
	for i := 0; i < n; i++ {
		id, err := NewUUID()
		assert.NoError(t, err)
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate UUID generated: %s", id)
		}
		seen[id] = struct{}{}
	}
	assert.Len(t, seen, n)
}
