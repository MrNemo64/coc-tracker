package testutil

import (
	"log/slog"
	"testing"
)

type TestLogBuffer struct {
	lines  []string
	Logger *slog.Logger
}

func (buff *TestLogBuffer) Write(b []byte) (int, error) {
	buff.lines = append(buff.lines, string(b))
	return len(b), nil
}

func (buff *TestLogBuffer) AssertLogIsEqual(t *testing.T, expected []string) {
	if len(expected) != len(buff.lines) {
		t.Errorf("Invalid number of lines, expected %d but had %d", len(expected), len(buff.lines))
	}

	for i := 0; i < len(expected); i++ {
		if expected[i] != buff.lines[i] {
			t.Errorf("Line %d does not match, expected '%s' but had '%s'", i, expected[i], buff.lines[i])
		}
	}
}

func MakeTestLogger() *TestLogBuffer {
	buff := &TestLogBuffer{
		lines: make([]string, 0),
	}

	logger := slog.New(slog.NewJSONHandler(buff, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
	}))

	buff.Logger = logger
	return buff
}
