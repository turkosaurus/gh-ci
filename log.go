package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

func newFileLogger() (*slog.Logger, error) {
	logFile := os.Getenv("GH_CI_LOG_FILE")
	if logFile == "" {
		home, err := os.UserHomeDir()
		logFile = filepath.Join(home, ".config", "gh-ci", "ci.log") // TODO: integrate with config
		if err != nil {
			return nil, fmt.Errorf("determine home directory: %w", err)
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
	}
	slog.Debug("initialized logger", "logFile", logFile, "level", level.String())

	handler := slog.NewTextHandler(f, &slog.HandlerOptions{Level: level})
	logger := slog.New(handler)
	return logger, nil
}
