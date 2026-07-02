package server

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// TestRegisterLookupRoutes asserts the server constructs without panicking on the static
// `/lookups/latest` vs param `/lookups/:id` sibling routes, and that all five lookup routes
// are registered.
func TestRegisterLookupRoutes(t *testing.T) {
	var s *Server
	require.NotPanics(t, func() { s = NewServer() })

	registered := make(map[string]bool)
	for _, r := range s.Routes() {
		registered[r.Method+" "+r.Path] = true
	}

	expected := []string{
		"POST /v2/lookups",
		"POST /v2/lookups/:id/close",
		"GET /v2/lookups",
		"GET /v2/lookups/latest",
		"GET /v2/lookups/:id",
	}
	for _, route := range expected {
		assert.Truef(t, registered[route], "route %q should be registered", route)
	}

	// Backward compatibility: the pre-existing routes must remain.
	for _, route := range []string{
		"POST /v2/numbers",
		"POST /v2/scanners/:scanner/dryrun",
		"POST /v2/scanners/:scanner/run",
		"GET /v2/scanners",
	} {
		assert.Truef(t, registered[route], "existing route %q must remain", route)
	}
}
