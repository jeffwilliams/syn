package syn

import "log"

// DebugLogger, when set, is used to log debug messages during lexing.
var DebugLogger *log.Logger = nil

func debugf(format string, args ...interface{}) {
	if DebugLogger == nil {
		return
	}

	DebugLogger.Printf(format, args...)
}
