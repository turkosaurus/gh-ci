package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// newFileLogger sets up a "text" or "json" logger.
func newFileLogger(format string) (*slog.Logger, error) {
	rotated := false
	maxAge := 1 * time.Second // FIXME: be realistic
	logFile := os.Getenv("GH_CI_LOG_FILE")
	if logFile == "" {
		home, err := os.UserHomeDir()
		logFile = filepath.Join(home, ".config", "gh-ci", "ci.log") // TODO: integrate with config
		if err != nil {
			return nil, fmt.Errorf("determine home directory: %w", err)
		}
		f, err := os.Stat(logFile)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("check log file: %w", err)
			}
		}
		if time.Since(f.ModTime()) > maxAge {
			err = os.Remove(logFile)
			if err != nil {
				return nil, fmt.Errorf("rotate log file: %w", err)
			}
			rotated = true
		}
		err = os.MkdirAll(filepath.Dir(logFile), 0755)
		if err != nil {
			return nil, fmt.Errorf("create log directory: %w", err)
		}
	}

	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}

	level := slog.LevelInfo
	if os.Getenv("DEBUG") != "" {
		level = slog.LevelDebug
	} else if os.Getenv("VERBOSE") != "" {
		level = slog.LevelInfo
	} else {
		level = slog.LevelWarn
	}

	var handler slog.Handler
	switch format {
	case "json":
		handler = slog.NewJSONHandler(f, &slog.HandlerOptions{Level: level})
	case "text":
		handler = slog.NewTextHandler(f, &slog.HandlerOptions{Level: level})
	default:
		return nil, fmt.Errorf("invalid log format: %s", format)
	}

	logger := slog.New(handler)
	if rotated {
		logger.Debug("logs rotated due to age")
	}
	logger.Debug("initialized text file logger",
		"path", logFile,
		"level", level.String(),
	)
	return logger, nil
}
