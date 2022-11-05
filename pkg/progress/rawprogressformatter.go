package progress

import "fmt"

const streamNewline = "\r\n"

type rawProgressFormatter struct{}

func (sf *rawProgressFormatter) formatStatus(id, format string, a ...interface{}) []byte {
	return []byte(fmt.Sprintf(format, a...) + streamNewline)
}

func (sf *rawProgressFormatter) formatProgress(id, action string, progress *JSONProgress, aux interface{}) []byte {
	if progress == nil {
		progress = &JSONProgress{}
	}
	endl := "\r"
	if progress.String() == "" {
		endl += "\n"
	}
	return []byte(action + " " + progress.String() + endl)
}
