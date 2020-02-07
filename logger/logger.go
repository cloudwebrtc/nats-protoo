package logger

import (
	"os"

	"github.com/rs/zerolog"
)

var logger zerolog.Logger

const (
	timeFormat = "2006-01-02 15:04:05.999"
)

func Init(level string) {
	l := zerolog.GlobalLevel()
	switch level {
	case "debug":
		l = zerolog.DebugLevel
	case "info":
		l = zerolog.InfoLevel
	case "warn":
		l = zerolog.WarnLevel
	case "error":
		l = zerolog.ErrorLevel
	}
	zerolog.TimeFieldFormat = timeFormat
	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: timeFormat}
	logger = zerolog.New(output).Level(l).With().Timestamp().Logger()
}

func Infof(format string, v ...interface{}) {
	logger.Info().Msgf(format, v...)
}

func Debugf(format string, v ...interface{}) {
	logger.Debug().Msgf(format, v...)
}

func Warnf(format string, v ...interface{}) {
	logger.Warn().Msgf(format, v...)
}

func Errorf(format string, v ...interface{}) {
	logger.Error().Msgf(format, v...)
}

func Panicf(format string, v ...interface{}) {
	logger.Panic().Msgf(format, v...)
}
