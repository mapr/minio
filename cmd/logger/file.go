package logger

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"strings"
)

// sink: logger output interface
var (
	sink loggerSink = nil
)

var (
	fileLogEnabled = false
)

// Reopen : This routine will reopen output sink if applicable. For use with external programs for log rotation.
func Reopen() error {
	return sink.Reopen()
}

// Set log file for as output
// Empty string or '-' symbol will redirect output to stdout
func SetOutput(filename string) {
	var newSink loggerSink
	if filename == "" || filename == "-" {
		newSink, _ = getStdoutSink()
	} else {
		var err error
		newSink, err = getFileSink(filename)
		if err != nil {
			logrus.Fatalf("Can not open new log file %s", filename)
		}
		fileLogEnabled = true
	}

	logrus.SetOutput(newSink)

	sink.Close()
	sink = newSink
}

// Set verbosity level
// 0 - Panic, 1 - Fatal, 2 - Error, 3 - Warning, 4 - Info, 5 - Debug
// Or use embedded logrus constants from logrus.PanicLevel to logrus.DebugLevel.
func SetLevel(level logrus.Level) {
	logrus.SetLevel(level)
}

// enableFileJSON - outputs logs in json format.
func enableFileJSON() {
	logrus.SetFormatter(new(logrus.JSONFormatter))
}

func initFile() {
	if sink == nil {
		sink, _ = getStdoutSink()
		logrus.SetOutput(sink)
	}

}

func prepareEntry(ctx context.Context) *logrus.Entry {
	req := GetReqInfo(ctx)

	if req == nil {
		req = &ReqInfo{API: "SYSTEM"}
	}

	API := "SYSTEM"
	if req.API != "" {
		API = req.API
	}

	tags := make(map[string]interface{})
	for _, entry := range req.GetTags() {
		tags[entry.Key] = entry.Val
	}

	// Get full stack trace
	trace := getTrace(2)

	stdLog := logrus.StandardLogger()
	entry := stdLog.WithField("api", API)

	if req.RemoteHost != "" {
		entry = entry.WithField("remotehost", req.RemoteHost)
	}

	if req.RequestID != "" {
		entry = entry.WithField("requestID", req.RequestID)
	}

	if req.UserAgent != "" {
		entry = entry.WithField("userAgent", req.UserAgent)
	}

	if req.BucketName != "" {
		entry = entry.WithField("bucket", req.BucketName)
	}

	if req.ObjectName != "" {
		entry = entry.WithField("object", req.ObjectName)
	}

	// Add trace log only for debug level
	if stdLog.Level == logrus.DebugLevel {
		if len(trace) > 0 {
			entry = entry.WithField("trace", strings.Join(trace, "\n"))
		}
	}

	if len(tags) > 0 {
		entry = entry.WithField("tags", fmt.Sprint(tags))
	}

	return entry
}

// LogIf :
func logIfFile(ctx context.Context, err error) {
	if err == nil {
		return
	}

	entry := prepareEntry(ctx)
	entry.Error(err.Error())
}

// CriticalIf :
// Like LogIf with exit
// It'll be called for fatal error conditions during run-time
func criticalIfFile(ctx context.Context, err error) {
	if err != nil {
		entry := prepareEntry(ctx)
		entry.Fatal(err.Error())
	}
}

// FatalIf :
// Just fatal error message, no stack trace
// It'll be called for input validation failures
func fatalIfFile(err error, msg string, data ...interface{}) {
	if msg != "" {
		logrus.Fatalf(msg, data...)
	} else {
		logrus.Fatal(err.Error())
	}
}

// Info :
func infoFile(msg string, data ...interface{}) {
	logrus.Infof(msg, data...)
}
