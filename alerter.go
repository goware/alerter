package alerter

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	LevelDebug Level = iota
	LevelInfo
	// pings the alert role
	LevelError
)

type Level int

type Alerter interface {
	Alert(ctx context.Context, level Level, format string, v ...interface{})
}

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "debug"
	case LevelInfo:
		return "info"
	case LevelError:
		return "error"
	default:
		return "unknown"
	}
}

func (l Level) Color() int {
	switch l {
	case LevelDebug:
		return 0xffd300
	case LevelInfo:
		return 0x3cb043
	case LevelError:
		return 0xd30000
	default:
		return 0x000000
	}
}

func (l Level) ZeroLogEvent() *zerolog.Event {
	switch l {
	case LevelDebug:
		return log.Debug()
	case LevelInfo:
		return log.Info()
	case LevelError:
		return log.Error()
	default:
		return log.Info()
	}
}
