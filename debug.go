package syn

import "log"

var DebugLogger *log.Logger = nil

func debugf(format string, args ...interface{}) {
	if DebugLogger == nil {
		return
	}

	DebugLogger.Printf(format, args...)
}
