package log

import (
	l "log"
	"os"
)

var (
	flag     = l.Lshortfile | l.Ldate | l.Ltime
	fatalLog = l.New(os.Stderr, "FATAL\t", flag)
	errLog   = l.New(os.Stderr, "ERROR\t", flag)
	warnLog  = l.New(os.Stderr, "WARN \t", flag)
	infoLog  = l.New(os.Stderr, "INFO \t", flag)
	debugLog = l.New(os.Stderr, "DEBUG\t", flag)
)

// Fatal Fatal
func Fatal(msg ...interface{}) {
	fatalLog.Fatal(msg...)
}

// Error Error
func Error(msg ...interface{}) {
	errLog.Println(msg...)
}

// Warn Warn
func Warn(msg ...interface{}) {
	warnLog.Println(msg...)
}

// Info  Info
func Info(msg ...interface{}) {
	infoLog.Println(msg...)
}

// Debug Debug
func Debug(msg ...interface{}) {
	debugLog.Println(msg...)
}
