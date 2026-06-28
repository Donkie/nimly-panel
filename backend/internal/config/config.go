// Package config loads and validates runtime configuration from environment
// variables. All secrets (MQTT credentials, OIDC client secret, session key)
// are supplied via the environment and never leave the server.
package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Config holds all runtime configuration.
type Config struct {
	// HTTP
	Addr       string // listen address, e.g. ":8080"
	AppBaseURL string // public base URL, e.g. https://lock.example.com
	DevMode    bool   // relaxes cookie Secure flag and proxies the frontend
	LogLevel   string // debug | info (default info)

	// MQTT
	MQTTBrokerURL string // e.g. tcp://broker.example.lan:1883 or tls://broker.example.lan:8883
	MQTTUsername  string
	MQTTPassword  string
	MQTTClientID  string
	LockTopic     string // Z2M base topic, e.g. "zigbee2mqtt/Front Door Lock"

	// OIDC (PocketID)
	OIDCIssuer       string
	OIDCClientID     string
	OIDCClientSecret string
	OIDCRedirectURL  string   // {AppBaseURL}/api/auth/callback
	OIDCScopes       []string // defaults to openid,profile,email,groups
	AllowedGroups    []string // optional allowlist by group claim
	AllowedSubjects  []string // optional allowlist by subject

	// Session
	SessionSecret   string
	SessionLifetime time.Duration
	SessionIdle     time.Duration

	// Audit
	AuditLogPath string // optional JSONL append file; empty = stdout only

	// PIN persistence (panel is source of truth — the lock can't report its
	// stored PINs). Path to a JSON file, ideally on a mounted volume.
	PinStorePath string
	// Encryption key for the PIN store (any string; a 32-byte AES key is
	// derived from it). If empty, the store is written as plaintext JSON.
	PinStoreKey string
}

// Load reads configuration from the environment, applies defaults and validates
// required fields.
func Load() (*Config, error) {
	c := &Config{
		Addr:            getenv("ADDR", ":8080"),
		AppBaseURL:      strings.TrimRight(os.Getenv("APP_BASE_URL"), "/"),
		DevMode:         boolenv("DEV_MODE", false),
		LogLevel:        getenv("LOG_LEVEL", "info"),
		MQTTBrokerURL:   os.Getenv("MQTT_BROKER_URL"),
		MQTTUsername:    os.Getenv("MQTT_USERNAME"),
		MQTTPassword:    os.Getenv("MQTT_PASSWORD"),
		MQTTClientID:    getenv("MQTT_CLIENT_ID", "nimlypanel"),
		LockTopic:       strings.TrimRight(os.Getenv("MQTT_LOCK_TOPIC"), "/"),
		OIDCIssuer:      strings.TrimRight(os.Getenv("OIDC_ISSUER"), "/"),
		OIDCClientID:    os.Getenv("OIDC_CLIENT_ID"),
		OIDCClientSecret: os.Getenv("OIDC_CLIENT_SECRET"),
		OIDCRedirectURL: os.Getenv("OIDC_REDIRECT_URL"),
		OIDCScopes:      splitlist(getenv("OIDC_SCOPES", "openid,profile,email,groups")),
		AllowedGroups:   splitlist(os.Getenv("ALLOWED_GROUPS")),
		AllowedSubjects: splitlist(os.Getenv("ALLOWED_SUBS")),
		SessionSecret:   os.Getenv("SESSION_SECRET"),
		SessionLifetime: durenv("SESSION_LIFETIME", 12*time.Hour),
		SessionIdle:     durenv("SESSION_IDLE_TIMEOUT", 1*time.Hour),
		AuditLogPath:    os.Getenv("AUDIT_LOG_PATH"),
		PinStorePath:    os.Getenv("PIN_STORE_PATH"),
		PinStoreKey:     os.Getenv("PIN_STORE_KEY"),
	}

	if c.OIDCRedirectURL == "" && c.AppBaseURL != "" {
		c.OIDCRedirectURL = c.AppBaseURL + "/api/auth/callback"
	}

	var missing []string
	required := map[string]string{
		"APP_BASE_URL":       c.AppBaseURL,
		"MQTT_BROKER_URL":    c.MQTTBrokerURL,
		"MQTT_LOCK_TOPIC":    c.LockTopic,
		"OIDC_ISSUER":        c.OIDCIssuer,
		"OIDC_CLIENT_ID":     c.OIDCClientID,
		"OIDC_CLIENT_SECRET": c.OIDCClientSecret,
		"SESSION_SECRET":     c.SessionSecret,
	}
	for k, v := range required {
		if v == "" {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}
	if len(c.SessionSecret) < 32 {
		return nil, fmt.Errorf("SESSION_SECRET must be at least 32 bytes")
	}

	return c, nil
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func boolenv(key string, def bool) bool {
	switch strings.ToLower(os.Getenv(key)) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return def
	}
}

func durenv(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

func splitlist(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
