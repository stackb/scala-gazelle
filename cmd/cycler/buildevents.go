package main

import (
	"encoding/json"
	"io"
	"os"
)

// BuildEvent represents a single build event in a bep.json file, or at least
// the minimal structure required for our purposes.
type BuildEvent struct {
	Action   *Action   `json:"action,omitempty"`
	Progress *Progress `json:"progress,omitempty"`
}

func readBuildEvents(filename string) ([]*BuildEvent, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return readBuildEventsIn(f)
}

func readBuildEventsIn(in io.Reader) ([]*BuildEvent, error) {
	var events []*BuildEvent

	decoder := json.NewDecoder(in)
	for {
		var evt BuildEvent

		err := decoder.Decode(&evt)
		if err == io.EOF {
			// all done
			break
		}
		if err != nil {
			return nil, err
		}

		events = append(events, &evt)
		// log.Printf("event: %+v", evt)
	}

	return events, nil
}
