package progress

type Output interface {
	WriteProgress(Progress) error
}
