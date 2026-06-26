package web

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/sundowndev/phoneinfoga/v2/web/errors"
)

// configurableKeys is the allowlist of environment variables the runtime
// config endpoint is permitted to read and set. Restricting writes to this
// set prevents the API from mutating arbitrary process environment variables.
var configurableKeys = []string{
	"TWILIO_ACCOUNT_SID",
	"TWILIO_AUTH_TOKEN",
	"TWILIO_LOOKUP_FIELDS",
	"BREACH_SCANNER_ENABLED",
	"DEHASHED_EMAIL",
	"DEHASHED_API_KEY",
	"BREACH_INCLUDE_FIELDS",
	"DEHASHED_API_URL",
}

// secretKeys holds the configurable keys whose values must never be returned
// in full by the read path. They are reported as a masked hint instead.
var secretKeys = map[string]bool{
	"TWILIO_ACCOUNT_SID": true,
	"TWILIO_AUTH_TOKEN":  true,
	"DEHASHED_EMAIL":     true,
	"DEHASHED_API_KEY":   true,
}

type configFieldStatus struct {
	Key        string `json:"key"`
	Secret     bool   `json:"secret"`
	Configured bool   `json:"configured"`
	// Value is the actual value for non-secret keys, or a masked hint
	// (last 4 characters) for secret keys. Empty when unconfigured.
	Value string `json:"value,omitempty"`
}

type configStatusResponse struct {
	JSONResponse
	Fields []configFieldStatus `json:"fields"`
}

func isConfigurableKey(key string) bool {
	for _, k := range configurableKeys {
		if k == key {
			return true
		}
	}
	return false
}

// maskSecret returns a non-reversible hint for a secret value: the last four
// characters prefixed with a fixed mask. Short values are fully masked.
func maskSecret(v string) string {
	if v == "" {
		return ""
	}
	if len(v) <= 4 {
		return "****"
	}
	return "****" + v[len(v)-4:]
}

func currentConfigStatus() configStatusResponse {
	fields := make([]configFieldStatus, 0, len(configurableKeys))
	for _, key := range configurableKeys {
		raw := os.Getenv(key)
		secret := secretKeys[key]

		value := raw
		if secret {
			value = maskSecret(raw)
		}

		fields = append(fields, configFieldStatus{
			Key:        key,
			Secret:     secret,
			Configured: raw != "",
			Value:      value,
		})
	}

	return configStatusResponse{
		JSONResponse: JSONResponse{Success: true},
		Fields:       fields,
	}
}

// @ID getConfig
// @Tags Config
// @Summary Report the runtime configuration status of scanner credentials.
// @Description Returns whether each configurable scanner credential is set.
// Secret values are masked and never returned in full.
// @Produce json
// @Success 200 {object} configStatusResponse
// @Router /config [get]
func getConfig(c *gin.Context) {
	c.JSON(http.StatusOK, currentConfigStatus())
}

// @ID updateConfig
// @Tags Config
// @Summary Set scanner credentials in the running process (no restart).
// @Description Accepts a JSON object of allowlisted environment variable
// names to values and applies them to the running process. Unknown keys are
// rejected.
// @Accept json
// @Produce json
// @Success 200 {object} configStatusResponse
// @Success 400 {object} JSONResponse
// @Router /config [post]
func updateConfig(c *gin.Context) {
	var req map[string]string
	if err := c.ShouldBindJSON(&req); err != nil {
		handleError(c, errors.NewBadRequest(err))
		return
	}

	// Validate every key before mutating any environment variable, so a
	// single unknown key rejects the whole request atomically.
	for key := range req {
		if !isConfigurableKey(key) {
			handleError(c, errors.NewBadRequest(fmt.Errorf("unknown config key: %s", key)))
			return
		}
	}

	for key, value := range req {
		if err := os.Setenv(key, value); err != nil {
			handleError(c, errors.NewInternalError(err))
			return
		}
	}

	c.JSON(http.StatusOK, currentConfigStatus())
}
