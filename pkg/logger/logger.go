package logger

import (
	"io"
)

// Logger represents loggin object
type Logger struct {
	info io.Writer
	err  io.Writer
}

// New creates and returns new Logger object
func New(i io.Writer, e io.Writer) *Logger {
	var l Logger
	l.info = i
	l.err = e
	return &l
}

// Info writes informational string message to a log
func (l Logger) Info(message string) {
	l.info.Write([]byte(message + "\n"))
}

// Error writes error string message to a log
func (l Logger) Error(message string) {
	l.err.Write([]byte(message + "\n"))
}
