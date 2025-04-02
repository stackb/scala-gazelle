package testutil

import (
	"testing"
)

// TestLogger is a wrapper around testing.T that implements a subset
// of the log.Logger interface by forwarding log messages to t.Log
type TestLogger struct {
	t *testing.T
}

// New creates a new TestLogger that writes to the provided testing.T
func NewTestLogger(t *testing.T) *TestLogger {
	return &TestLogger{t: t}
}

// Printf formats the log message and writes it to the test log
func (l *TestLogger) Printf(format string, v ...any) {
	l.t.Logf(format, v...)
}

// Print writes the log message to the test log
func (l *TestLogger) Print(v ...any) {
	l.t.Log(v...)
}

// Println writes the log message to the test log with a newline
func (l *TestLogger) Println(v ...any) {
	l.t.Log(v...)
}

// Fatal logs the message and fails the test
func (l *TestLogger) Fatal(v ...any) {
	l.t.Fatal(v...)
}

// Fatalf logs the formatted message and fails the test
func (l *TestLogger) Fatalf(format string, v ...any) {
	l.t.Fatalf(format, v...)
}

// Fatalln logs the message and fails the test
func (l *TestLogger) Fatalln(v ...any) {
	l.t.Fatal(v...)
}

// Panic logs the message and fails the test
func (l *TestLogger) Panic(v ...any) {
	l.t.Fatal(v...)
}

// Panicf logs the formatted message and fails the test
func (l *TestLogger) Panicf(format string, v ...any) {
	l.t.Fatalf(format, v...)
}

// Panicln logs the message and fails the test
func (l *TestLogger) Panicln(v ...any) {
	l.t.Fatal(v...)
}
