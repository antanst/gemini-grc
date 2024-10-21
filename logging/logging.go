package logging

import (
	"fmt"

	zlog "github.com/rs/zerolog/log"
)

func LogDebug(format string, args ...interface{}) {
	zlog.Debug().Msg(fmt.Sprintf(format, args...))
}

func LogInfo(format string, args ...interface{}) {
	zlog.Info().Msg(fmt.Sprintf(format, args...))
}

func LogWarn(format string, args ...interface{}) {
	zlog.Warn().Msg(fmt.Sprintf(format, args...))
}

func LogError(format string, args ...interface{}) {
	zlog.Error().Err(fmt.Errorf(format, args...)).Msg("")
}
