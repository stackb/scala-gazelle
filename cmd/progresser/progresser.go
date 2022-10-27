package main

import (
	"fmt"
	"os"
	"time"

	"github.com/stackb/scala-gazelle/pkg/progress"
)

func main() {

	out := progress.NewOut(os.Stdout)

	prog := progress.NewProgressOutput(out)
	if err := prog.WriteProgress(progress.Progress{
		ID:     "1",
		Action: "initializing",
		// Message: "hello, boulder!",
		Current: 33,
		Total:   100,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "progress error: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	if err := prog.WriteProgress(progress.Progress{
		ID:     "1",
		Action: "setup",
		// Message: "hello, colorado!",
		Current: 66,
		Total:   100,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "progress error: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	if err := prog.WriteProgress(progress.Progress{
		ID:     "1",
		Action: "verifying",
		// Message: "hello, usa!",
		Current: 99,
		Total:   100,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "progress error: %v", err)
	}

}
