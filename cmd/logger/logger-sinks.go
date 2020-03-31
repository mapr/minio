package logger

import (
	"io"
	"os"
)

// Sink interface for logger outputs.
type loggerSink interface {
	io.Writer
	io.Closer
	Reopen() error
}

// Use stdSink to redirect logs to stout or stderr
type stdSink struct {
	out io.Writer
}

func (s *stdSink) Write(p []byte) (n int, err error) {
	return s.out.Write(p)
}

func (s *stdSink) Close() error {
	return nil
}

func (s *stdSink) Reopen() error {
	return nil
}

// Fabric for stdout sink
func getStdoutSink() (loggerSink, error) {
	sink := &stdSink{os.Stdout}
	return sink, nil
}

// Use file sink to redirect logs to files
type fileSynk struct {
	name string
	out  *os.File
}

func (s *fileSynk) Write(p []byte) (n int, err error) {
	return s.out.Write(p)
}

func (s *fileSynk) Close() error {
	return s.out.Close()
}

func (s *fileSynk) Reopen() error {
	err := s.out.Close()

	if err != nil {
		return err
	}

	f, err := os.OpenFile(s.name, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	s.out = f

	return err
}

// Fabric for file sink
func getFileSink(name string) (loggerSink, error) {
	f, err := os.OpenFile(name, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)

	if err != nil {
		return nil, err
	}

	sink := &fileSynk{name, f}
	return sink, nil
}
