package mylog

import (
	"path"
	"runtime"
	"strconv"

	"github.com/Hongssd/cgolatencytest/config"

	"github.com/sirupsen/logrus"
)

var Log *logrus.Logger

func init() {
	Log = logrus.New()
	logLevel := config.GetConfig("logLevel")
	Log.SetLevel(GetLogLevelFromString(logLevel))
	Log.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05.000",
		FullTimestamp:   true,
		ForceColors:     true,
		CallerPrettyfier: func(frame *runtime.Frame) (function string, file string) {
			fileName := "[" + path.Base(frame.File) + ":" + strconv.Itoa(frame.Line) + "]"
			return "", fileName
		},
	})
	Log.SetReportCaller(true)
}

func GetLogLevelFromString(logLevel string) logrus.Level {
	switch logLevel {
	case "Panic":
		return logrus.PanicLevel
	case "Fatal":
		return logrus.FatalLevel
	case "Error":
		return logrus.ErrorLevel
	case "Warn":
		return logrus.WarnLevel
	case "Info":
		return logrus.InfoLevel
	case "Debug":
		return logrus.DebugLevel
	case "Trace":
		return logrus.TraceLevel
	default:
		return logrus.DebugLevel
	}
}
