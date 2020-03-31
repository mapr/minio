package logger

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"strings"
)

// sink: logger output interface
var (
	sink loggerSink
)

// Reopen : This routine will reopen output sink if applicable. For use with external programs for log rotation.
func Reopen() error {
	return sink.Reopen()
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
