package log

import (
	"github.com/rs/zerolog/log"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"os"
)

func InitZeroLog(path string, maxSize, maxBackups int) func() {
	return func() {
		var logFD io.Writer
		if path == "" {
			logFD = os.Stdout
		} else {
			logFD = &lumberjack.Logger{
				Filename:   path,
				MaxSize:    maxSize,
				MaxAge:     0,
				MaxBackups: maxBackups,
				LocalTime:  false,
				Compress:   false,
			}
		}

		log.Logger = log.With().CallerWithSkipFrameCount(2).Logger().Output(logFD)
	}

}
