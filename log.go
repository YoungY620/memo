package main

import (
	"log"
	"strings"
)

// Log levels: error=0, notice=1, info=2, debug=3
var logLevel = 2 // default: info

func SetLogLevel(level string) {
	switch strings.ToLower(level) {
	case "error":
		logLevel = 0
	case "notice":
		logLevel = 1
	case "info":
		logLevel = 2
	case "debug":
		logLevel = 3
	default:
		logLevel = 2
	}
}

func logError(format string, v ...any) {
	if logLevel >= 0 {
		log.Printf("[ERROR] "+format, v...)
	}
}

func logNotice(format string, v ...any) {
	if logLevel >= 1 {
		log.Printf("[NOTICE] "+format, v...)
	}
}

func logInfo(format string, v ...any) {
	if logLevel >= 2 {
		log.Printf("[INFO] "+format, v...)
	}
}

func logDebug(format string, v ...any) {
	if logLevel >= 3 {
		log.Printf("[DEBUG] "+format, v...)
	}
}
