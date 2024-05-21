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

func CreateLogger(prodLogger *bool) zerolog.Logger {
	infoLevels := []zerolog.Level{
		zerolog.DebugLevel, zerolog.InfoLevel, zerolog.WarnLevel,
	}
	errLevels := []zerolog.Level{
		zerolog.ErrorLevel, zerolog.FatalLevel, zerolog.PanicLevel,
	}

	localStdoutWriter := SpecificLevelWriter{
		Writer: zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339},
		Levels: infoLevels,
	}
	localStderrWriter := SpecificLevelWriter{
		Writer: zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339},
		Levels: errLevels,
	}
	prodStdoutWriter := SpecificLevelWriter{
		Writer: os.Stdout,
		Levels: infoLevels,
	}
	prodStderrWriter := SpecificLevelWriter{
		Writer: os.Stderr,
		Levels: errLevels,
	}

	writer := zerolog.MultiLevelWriter(localStdoutWriter, localStderrWriter)

	if *prodLogger {
		// todo: safe non blocking writer? https://github.com/rs/zerolog?tab=readme-ov-file#thread-safe-lock-free-non-blocking-writer
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
		writer = zerolog.MultiLevelWriter(prodStdoutWriter, prodStderrWriter)
	}

	return zerolog.New(writer).With().Timestamp().Logger()
}
