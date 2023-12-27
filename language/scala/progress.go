package scala

import (
	"sync"

	"github.com/pcj/mobyprogress"
)

var mutex sync.Mutex

func writeParseProgress(output mobyprogress.Output, message string) {
	output.WriteProgress(mobyprogress.Progress{
		ID:      "parse",
		Action:  "parse",
		Message: message,
	})
}

func writeGenerateProgress(output mobyprogress.Output, current, total int) {
	output.WriteProgress(mobyprogress.Progress{
		ID:      "walk",
		Action:  "generating rules",
		Current: int64(current),
		Total:   int64(total),
		Units:   "packages",
	})
}

func writeResolveProgress(output mobyprogress.Output, current, total int) {
	mutex.Lock()
	output.WriteProgress(mobyprogress.Progress{
		ID:         "resolve",
		Action:     "resolving dependencies",
		Current:    int64(total - current),
		Total:      int64(total),
		Units:      "packages",
		LastUpdate: current == 0,
	})
	mutex.Unlock()
}
