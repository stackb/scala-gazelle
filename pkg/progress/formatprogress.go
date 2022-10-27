package progress

type formatProgress interface {
	formatStatus(id, format string, a ...interface{}) []byte
	formatProgress(id, action string, progress *JSONProgress, aux interface{}) []byte
}
