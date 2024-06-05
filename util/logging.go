package util

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

var logger *slog.Logger

func SetupLog() {
	var logLevel slog.Level
	switch os.Getenv("LOG_LEVEL") {
	default:
		logLevel = slog.LevelInfo
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	}

	var wrtiter io.Writer
	if os.Getenv("LOG_OUT") == "stdout" {
		wrtiter = os.Stdout
	} else if strings.HasPrefix(os.Getenv("LOG_OUT"), "file:") {
		file, _ := strings.CutPrefix(os.Getenv("LOG_OUT"), "file:")
		w, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
		if err != nil {
			panic(err)
		}
		wrtiter = w
	} else if os.Getenv("LOG_OUT") == "off" {
		wrtiter = io.Discard
	} else {
		panic("Could not setup logs")
	}

	var options = &slog.HandlerOptions{
		Level:     logLevel,
		AddSource: true,
	}
	var handler slog.Handler
	switch os.Getenv("LOG_HANDLE") {
	default:
		handler = slog.NewJSONHandler(wrtiter, options)
	case "json":
		handler = slog.NewJSONHandler(wrtiter, options)
	case "text":
		handler = slog.NewTextHandler(wrtiter, options)
	}

	logger = slog.New(handler)
}

func GetLogger(name string) *slog.Logger {
	if logger == nil {
		SetupLog()
	}
	return logger.With(slog.String("name", name))
}
