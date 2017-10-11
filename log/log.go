package log

import (
	"fmt"
	"github.com/sdvdxl/logstash-http-push/config"
	l "log"
	"os"
	"strings"
)

var (
	flag     = l.Lshortfile | l.Ldate | l.Ltime
	fatalLog = l.New(os.Stderr, "FATAL\t", flag)
	errLog   = l.New(os.Stderr, "ERROR\t", flag)
	warnLog  = l.New(os.Stdout, "WARN \t", flag)
	infoLog  = l.New(os.Stdout, "INFO \t", flag)
	debugLog = l.New(os.Stdout, "DEBUG\t", flag)
	logLevel = inf
)

type level int

const (
	deb level = iota
	inf
	warn
	err
	fatal
)

func Init(cfg *config.Config) {
	switch strings.ToUpper(cfg.LogLevel) {
	case "DEBUG":
		logLevel = deb
	case "INFO":
		logLevel = inf
	case "WARN":
		logLevel = warn
	case "ERROR":
		logLevel = err
	case "FATAL":
		logLevel = fatal
	}
}

// Fatal Fatal
func Fatal(msg ...interface{}) {
	fatalLog.Output(2, fmt.Sprint(msg))
	fatalLog.Fatalln("application exit now")
}

// Error Error
func Error(msg ...interface{}) {
	if logLevel > err {
		return
	}
	errLog.Output(2, fmt.Sprint(msg))
}

// Warn Warn
func Warn(msg ...interface{}) {
	if logLevel > warn {
		return
	}
	warnLog.Output(2, fmt.Sprint(msg))
}

// Info  Info
func Info(msg ...interface{}) {
	if logLevel > inf {
		return
	}
	infoLog.Output(2, fmt.Sprint(msg))
}

// Debug Debug
func Debug(msg ...interface{}) {
	if logLevel > deb {
		return
	}

	debugLog.Output(2, fmt.Sprint(msg))
}
