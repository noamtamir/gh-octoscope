package main

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

type SpecificLevelWriter struct {
	io.Writer
	Levels []zerolog.Level
}

func (w SpecificLevelWriter) WriteLevel(level zerolog.Level, p []byte) (int, error) {
	for _, l := range w.Levels {
		if l == level {
			return w.Write(p)
		}
	}
	return len(p), nil
}

func CreateLogger() zerolog.Logger {
	// todo: switch ConsoleWrite with Stdout/Stderr for production
	writer := zerolog.MultiLevelWriter(
		SpecificLevelWriter{
			Writer: zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339},
			Levels: []zerolog.Level{
				zerolog.DebugLevel, zerolog.InfoLevel, zerolog.WarnLevel,
			},
		},
		SpecificLevelWriter{
			Writer: zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339},
			Levels: []zerolog.Level{
				zerolog.ErrorLevel, zerolog.FatalLevel, zerolog.PanicLevel,
			},
		},
	)

	return zerolog.New(writer).With().Timestamp().Logger()
}
