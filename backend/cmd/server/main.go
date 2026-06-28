// Command server runs the Nimly admin panel: it connects to the MQTT broker,
// serves the JSON API and the embedded Svelte SPA, and authenticates users via
// PocketID (OIDC).
package main

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Donkie/nimly-panel/backend/internal/api"
	"github.com/Donkie/nimly-panel/backend/internal/audit"
	"github.com/Donkie/nimly-panel/backend/internal/auth"
	"github.com/Donkie/nimly-panel/backend/internal/config"
	"github.com/Donkie/nimly-panel/backend/internal/lock"
	"github.com/Donkie/nimly-panel/backend/internal/mqtt"
)

func main() {
	level := slog.LevelInfo
	if strings.EqualFold(os.Getenv("LOG_LEVEL"), "debug") {
		level = slog.LevelDebug
	}
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))

	if err := run(log); err != nil {
		log.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	log.Info("starting nimly-panel",
		"app_base_url", cfg.AppBaseURL,
		"mqtt_broker", cfg.MQTTBrokerURL,
		"mqtt_username", cfg.MQTTUsername,
		"lock_topic", cfg.LockTopic,
		"oidc_issuer", cfg.OIDCIssuer,
		"log_level", cfg.LogLevel,
	)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Audit logger.
	au, err := audit.New(log, cfg.AuditLogPath)
	if err != nil {
		return err
	}
	defer au.Close()

	// Persistent PIN store (panel is the source of truth for PIN slots).
	pinStore, err := lock.NewPinStore(cfg.PinStorePath, cfg.PinStoreKey)
	if err != nil {
		return err
	}
	if !pinStore.Persistent() {
		log.Warn("PIN_STORE_PATH not set; PIN list is in-memory only and will reset on restart")
	} else if !pinStore.Encrypted() {
		log.Warn("PIN_STORE_KEY not set; PIN store is written as plaintext (set a key to encrypt at rest)")
	} else {
		log.Info("PIN store encrypted at rest")
	}

	// Lock state cache + MQTT client + service.
	store := lock.NewStore()
	store.SeedPins(pinStore.List()) // hydrate the live cache from disk
	mqttClient := mqtt.New(mqtt.Options{
		BrokerURL: cfg.MQTTBrokerURL,
		Username:  cfg.MQTTUsername,
		Password:  cfg.MQTTPassword,
		ClientID:  cfg.MQTTClientID,
		LockTopic: cfg.LockTopic,
	}, store, log)

	svc := lock.NewService(mqttClient, store, pinStore, cfg.LockTopic)
	mqttClient.Bind(svc)

	// Connect in the background: with connect-retry enabled the connect token
	// only completes once the broker is reachable, so blocking here would stop
	// the HTTP server from ever starting when MQTT is down.
	go func() {
		if err := mqttClient.Connect(); err != nil {
			log.Warn("mqtt connect failed; retrying in background", "err", err)
		}
	}()
	defer mqttClient.Disconnect()

	// Authentication.
	authenticator, err := auth.New(ctx, cfg)
	if err != nil {
		return err
	}

	// Static SPA assets.
	static, err := staticHandler(log)
	if err != nil {
		return err
	}

	srv := api.NewServer(svc, authenticator, au, log, static)

	httpServer := &http.Server{
		Addr:              cfg.Addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      0, // 0 = no write timeout (SSE stream is long-lived)
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		log.Info("http server listening", "addr", cfg.Addr, "base_url", cfg.AppBaseURL)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("http server error", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return httpServer.Shutdown(shutdownCtx)
}

// staticHandler returns the SPA handler backed by the embedded frontend build.
func staticHandler(log *slog.Logger) (http.Handler, error) {
	sub, err := fs.Sub(frontendFS, "frontend_dist")
	if err != nil {
		return nil, err
	}
	// If the build is missing (e.g. backend-only dev), serve a helpful message.
	if _, err := fs.Stat(sub, "index.html"); err != nil {
		log.Warn("embedded frontend build not found; serving placeholder")
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "frontend not built; run the frontend build or use the dev server", http.StatusServiceUnavailable)
		}), nil
	}
	return api.SPAHandler(sub), nil
}
