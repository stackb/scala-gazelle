package progress

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/morikuni/aec"
)

// RFC3339NanoFixed is time.RFC3339Nano with nanoseconds padded using zeros to
// ensure the formatted time isalways the same number of characters.
const RFC3339NanoFixed = "2006-01-02T15:04:05.000000000Z07:00"

// JSONMessage defines a message struct. It describes
// the created time, where it from, status, ID of the
// message. It's used for docker events.
type JSONMessage struct {
	Stream          string        `json:"stream,omitempty"`
	Status          string        `json:"status,omitempty"`
	Progress        *JSONProgress `json:"progressDetail,omitempty"`
	ProgressMessage string        `json:"progress,omitempty"` // deprecated
	ID              string        `json:"id,omitempty"`
	From            string        `json:"from,omitempty"`
	Time            int64         `json:"time,omitempty"`
	TimeNano        int64         `json:"timeNano,omitempty"`
	Error           *JSONError    `json:"errorDetail,omitempty"`
	ErrorMessage    string        `json:"error,omitempty"` // deprecated
	// Aux contains out-of-band data, such as digests for push signing and image id after building.
	Aux *json.RawMessage `json:"aux,omitempty"`
}

func clearLine(out io.Writer) {
	eraseMode := aec.EraseModes.All
	cl := aec.EraseLine(eraseMode)
	fmt.Fprint(out, cl)
}

func cursorUp(out io.Writer, l uint) {
	fmt.Fprint(out, aec.Up(l))
}

func cursorDown(out io.Writer, l uint) {
	fmt.Fprint(out, aec.Down(l))
}

// Display displays the JSONMessage to `out`. If `isTerminal` is true, it will erase the
// entire current line when displaying the progressbar.
func (jm *JSONMessage) Display(out io.Writer, isTerminal bool) error {
	if jm.Error != nil {
		if jm.Error.Code == 401 {
			return fmt.Errorf("authentication is required")
		}
		return jm.Error
	}
	var endl string
	if isTerminal && jm.Stream == "" && jm.Progress != nil {
		clearLine(out)
		endl = "\r"
		fmt.Fprint(out, endl)
	} else if jm.Progress != nil && jm.Progress.String() != "" { // disable progressbar in non-terminal
		return nil
	}
	if jm.TimeNano != 0 {
		fmt.Fprintf(out, "%s ", time.Unix(0, jm.TimeNano).Format(RFC3339NanoFixed))
	} else if jm.Time != 0 {
		fmt.Fprintf(out, "%s ", time.Unix(jm.Time, 0).Format(RFC3339NanoFixed))
	}
	if jm.ID != "" {
		fmt.Fprintf(out, "%s: ", jm.ID)
	}
	if jm.From != "" {
		fmt.Fprintf(out, "(from %s) ", jm.From)
	}
	if jm.Progress != nil && isTerminal {
		fmt.Fprintf(out, "%s %s%s", jm.Status, jm.Progress.String(), endl)
	} else if jm.ProgressMessage != "" { // deprecated
		fmt.Fprintf(out, "%s %s%s", jm.Status, jm.ProgressMessage, endl)
	} else if jm.Stream != "" {
		fmt.Fprintf(out, "%s%s", jm.Stream, endl)
	} else {
		fmt.Fprintf(out, "%s%s\n", jm.Status, endl)
	}
	return nil
}

// DisplayJSONMessagesStream displays a json message stream from `in` to `out`, `isTerminal`
// describes if `out` is a terminal. If this is the case, it will print `\n` at the end of
// each line and move the cursor while displaying.
func DisplayJSONMessagesStream(in io.Reader, out io.Writer, terminalFd uintptr, isTerminal bool, auxCallback func(JSONMessage)) error {
	var (
		dec = json.NewDecoder(in)
		ids = make(map[string]uint)
	)

	for {
		var diff uint
		var jm JSONMessage
		if err := dec.Decode(&jm); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		if jm.Aux != nil {
			if auxCallback != nil {
				auxCallback(jm)
			}
			continue
		}

		if jm.Progress != nil {
			jm.Progress.terminalFd = terminalFd
		}
		if jm.ID != "" && (jm.Progress != nil || jm.ProgressMessage != "") {
			line, ok := ids[jm.ID]
			if !ok {
				// NOTE: This approach of using len(id) to
				// figure out the number of lines of history
				// only works as long as we clear the history
				// when we output something that's not
				// accounted for in the map, such as a line
				// with no ID.
				line = uint(len(ids))
				ids[jm.ID] = line
				if isTerminal {
					fmt.Fprintf(out, "\n")
				}
			}
			diff = uint(len(ids)) - line
			if isTerminal {
				cursorUp(out, diff)
			}
		} else {
			// When outputting something that isn't progress
			// output, clear the history of previous lines. We
			// don't want progress entries from some previous
			// operation to be updated (for example, pull -a
			// with multiple tags).
			ids = make(map[string]uint)
		}
		err := jm.Display(out, isTerminal)
		if jm.ID != "" && isTerminal {
			cursorDown(out, diff)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

type stream interface {
	io.Writer
	FD() uintptr
	IsTerminal() bool
}

// DisplayJSONMessagesToStream prints json messages to the output stream
func DisplayJSONMessagesToStream(in io.Reader, stream stream, auxCallback func(JSONMessage)) error {
	return DisplayJSONMessagesStream(in, stream, stream.FD(), stream.IsTerminal(), auxCallback)
}
