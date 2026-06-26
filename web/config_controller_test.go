package web

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func performConfigRequest(r http.Handler, method, path, body, contentType, remoteAddr string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, path, bytes.NewBufferString(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.RemoteAddr = remoteAddr
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// performJSONRequest issues a JSON request from a loopback client, the normal
// case for the config endpoints.
func performJSONRequest(r http.Handler, method, path, body string) *httptest.ResponseRecorder {
	return performConfigRequest(r, method, path, body, "application/json", "127.0.0.1:54321")
}

func findField(fields []configFieldStatus, key string) (configFieldStatus, bool) {
	for _, f := range fields {
		if f.Key == key {
			return f, true
		}
	}
	return configFieldStatus{}, false
}

func TestConfigController(t *testing.T) {
	srv, err := NewServer(true)
	if err != nil {
		t.Fatal(err)
	}

	// Ensure a clean environment for the configurable keys.
	for _, k := range configurableKeys {
		_ = os.Unsetenv(k)
	}

	t.Run("GET /api/config returns unconfigured fields", func(t *testing.T) {
		res := performJSONRequest(srv, http.MethodGet, "/api/config", "")
		assert.Equal(t, http.StatusOK, res.Result().StatusCode)

		body, _ := ioutil.ReadAll(res.Body)
		var parsed configStatusResponse
		assert.NoError(t, json.Unmarshal(body, &parsed))
		assert.True(t, parsed.Success)

		// All configurable keys are present.
		assert.Equal(t, len(configurableKeys), len(parsed.Fields))

		sid, ok := findField(parsed.Fields, "TWILIO_ACCOUNT_SID")
		assert.True(t, ok)
		assert.True(t, sid.Secret)
		assert.False(t, sid.Configured)
		assert.Empty(t, sid.Value)
	})

	t.Run("POST /api/config sets values into the process environment", func(t *testing.T) {
		defer func() {
			for _, k := range configurableKeys {
				_ = os.Unsetenv(k)
			}
		}()

		res := performJSONRequest(srv, http.MethodPost, "/api/config",
			`{"TWILIO_ACCOUNT_SID":"AC0123456789abcdef","TWILIO_AUTH_TOKEN":"tok_secret_value","BREACH_SCANNER_ENABLED":"true"}`)
		assert.Equal(t, http.StatusOK, res.Result().StatusCode)

		// Values are now live in the process environment (no restart).
		assert.Equal(t, "AC0123456789abcdef", os.Getenv("TWILIO_ACCOUNT_SID"))
		assert.Equal(t, "tok_secret_value", os.Getenv("TWILIO_AUTH_TOKEN"))
		assert.Equal(t, "true", os.Getenv("BREACH_SCANNER_ENABLED"))

		body, _ := ioutil.ReadAll(res.Body)
		var parsed configStatusResponse
		assert.NoError(t, json.Unmarshal(body, &parsed))

		// Secret fields are masked in the response (never the raw value).
		sid, _ := findField(parsed.Fields, "TWILIO_ACCOUNT_SID")
		assert.True(t, sid.Configured)
		assert.NotEqual(t, "AC0123456789abcdef", sid.Value)
		assert.Equal(t, "****cdef", sid.Value)

		// Non-secret toggles return their actual value.
		breach, _ := findField(parsed.Fields, "BREACH_SCANNER_ENABLED")
		assert.False(t, breach.Secret)
		assert.True(t, breach.Configured)
		assert.Equal(t, "true", breach.Value)
	})

	t.Run("POST /api/config rejects unknown keys", func(t *testing.T) {
		res := performJSONRequest(srv, http.MethodPost, "/api/config",
			`{"SOME_ARBITRARY_KEY":"value"}`)
		assert.Equal(t, http.StatusBadRequest, res.Result().StatusCode)

		// The arbitrary key must not have been set.
		assert.Empty(t, os.Getenv("SOME_ARBITRARY_KEY"))
	})

	t.Run("POST /api/config rejects malformed JSON", func(t *testing.T) {
		res := performJSONRequest(srv, http.MethodPost, "/api/config", `{not json`)
		assert.Equal(t, http.StatusBadRequest, res.Result().StatusCode)
	})

	t.Run("POST /api/config rejects non-JSON content type (CSRF defense)", func(t *testing.T) {
		res := performConfigRequest(srv, http.MethodPost, "/api/config",
			`{"BREACH_SCANNER_ENABLED":"true"}`, "text/plain", "127.0.0.1:54321")
		assert.Equal(t, http.StatusBadRequest, res.Result().StatusCode)
	})

	t.Run("config endpoints reject non-loopback clients", func(t *testing.T) {
		get := performConfigRequest(srv, http.MethodGet, "/api/config", "", "application/json", "203.0.113.5:40000")
		assert.Equal(t, http.StatusForbidden, get.Result().StatusCode)

		post := performConfigRequest(srv, http.MethodPost, "/api/config",
			`{"BREACH_SCANNER_ENABLED":"true"}`, "application/json", "203.0.113.5:40000")
		assert.Equal(t, http.StatusForbidden, post.Result().StatusCode)
	})
}
