package scala

import (
	"github.com/pcj/mobyprogress"
)

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

func writeResolveProgress(output mobyprogress.Output, current, total int, lastUpdate bool) {
	output.WriteProgress(mobyprogress.Progress{
		ID:         "resolve",
		Action:     "resolving dependencies",
		Current:    int64(current),
		Total:      int64(total),
		Units:      "packages",
		LastUpdate: lastUpdate,
	})
}
