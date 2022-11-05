package progress

import (
	"fmt"
)

type Progress struct {
	ID string

	// Progress contains a Message or...
	Message string

	// ...progress of an action
	Action  string
	Current int64
	Total   int64

	// If true, don't show xB/yB
	HideCounts bool
	// If not empty, use units instead of bytes for counts
	Units string

	// Aux contains extra information not presented to the user, such as
	// digests for push signing.
	Aux interface{}

	LastUpdate bool
}

// Update is a convenience function to write a progress update to the channel.
func Update(out Output, id, action string) {
	out.WriteProgress(Progress{ID: id, Action: action})
}

// Updatef is a convenience function to write a printf-formatted progress update
// to the channel.
func Updatef(out Output, id, format string, a ...interface{}) {
	Update(out, id, fmt.Sprintf(format, a...))
}

// Message is a convenience function to write a progress message to the channel.
func Message(out Output, id, message string) {
	out.WriteProgress(Progress{ID: id, Message: message})
}

// Messagef is a convenience function to write a printf-formatted progress
// message to the channel.
func Messagef(out Output, id, format string, a ...interface{}) {
	Message(out, id, fmt.Sprintf(format, a...))
}
