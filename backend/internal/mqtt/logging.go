package mqtt

import (
	"context"
	"fmt"
	"log/slog"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// pahoLogger adapts paho's logger interface (Println/Printf) onto slog so the
// library's internal connection diagnostics (auth failures, network errors,
// reconnect attempts) surface in our structured logs.
type pahoLogger struct {
	log   *slog.Logger
	level slog.Level
}

func (p pahoLogger) Println(v ...any) {
	p.log.Log(context.Background(), p.level, "paho: "+fmt.Sprint(v...))
}

func (p pahoLogger) Printf(format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	if n := len(msg); n > 0 && msg[n-1] == '\n' {
		msg = msg[:n-1]
	}
	p.log.Log(context.Background(), p.level, "paho: "+msg)
}

// enablePahoLogging routes paho's package-level loggers to slog. ERROR/CRITICAL
// are always captured (these report connection/auth failures); DEBUG is only
// enabled when the logger is at debug level to avoid noise.
func enablePahoLogging(log *slog.Logger) {
	mqtt.ERROR = pahoLogger{log: log, level: slog.LevelError}
	mqtt.CRITICAL = pahoLogger{log: log, level: slog.LevelError}
	mqtt.WARN = pahoLogger{log: log, level: slog.LevelWarn}
	if log.Enabled(context.Background(), slog.LevelDebug) {
		mqtt.DEBUG = pahoLogger{log: log, level: slog.LevelDebug}
	}
}
