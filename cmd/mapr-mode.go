package cmd

import "strings"

type MapRMode string

const (
	FS      MapRMode = "FS"
	S3      MapRMode = "S3"
	UNKNOWN MapRMode = "UNKNOWN"
)

func (mode MapRMode) toString() string {
	return string(mode)
}

func StringToMode(mode string) MapRMode {
	switch strings.ToUpper(mode) {
	case FS.toString():
		return FS
	case S3.toString():
		return S3
	default:
		return UNKNOWN
	}
}
