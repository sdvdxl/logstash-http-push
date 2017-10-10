package log

import (
	"fmt"
	l "log"
	"os"
)

var (
	flag     = l.Lshortfile | l.Ldate | l.Ltime
	fatalLog = l.New(os.Stderr, "FATAL\t", flag)
	errLog   = l.New(os.Stderr, "ERROR\t", flag)
	warnLog  = l.New(os.Stdout, "WARN \t", flag)
	infoLog  = l.New(os.Stdout, "INFO \t", flag)
	debugLog = l.New(os.Stdout, "DEBUG\t", flag)
)

// Fatal Fatal
func Fatal(msg ...interface{}) {
	fatalLog.Output(2, fmt.Sprint(msg))
	fatalLog.Fatalln("application exit now")
}

// Error Error
func Error(msg ...interface{}) {
	errLog.Output(2, fmt.Sprint(msg))
}

// Warn Warn
func Warn(msg ...interface{}) {
	warnLog.Output(2, fmt.Sprint(msg))
}

// Info  Info
func Info(msg ...interface{}) {
	infoLog.Output(2, fmt.Sprint(msg))
}

// Debug Debug
func Debug(msg ...interface{}) {
	debugLog.Output(2, fmt.Sprint(msg))
}
