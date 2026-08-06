// Package logger builds the application logger.
package logger

import (
	"log/slog"
	"os"
)

// New returns a structured logger writing to stderr, so it never mixes into the
// agent's stdout. debug wins over level. An unparseable level falls back to info
// and says so — a bad log level should not stop a session.
func New(level string, debug bool) *slog.Logger {
	lvl := slog.LevelInfo
	var levelErr error
	if debug {
		lvl = slog.LevelDebug
	} else if level != "" {
		levelErr = lvl.UnmarshalText([]byte(level))
	}

	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: lvl,
		// Timestamps are noise in a foreground CLI session.
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	}))
	if levelErr != nil {
		log.Warn("unknown log level, using info", "level", level)
	}
	return log
}
