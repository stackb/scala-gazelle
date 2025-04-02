package collections

import (
	"sort"
	"strings"
)

type StringSlice []string

func (i *StringSlice) String() string {
	return strings.Join(*i, ",")
}

// Set implements the flag.Value interface.
func (i *StringSlice) Set(value string) error {
	*i = append(*i, value)
	return nil
}

// DeduplicateAndSort removes duplicate entries and sorts the list
func DeduplicateAndSort(in []string) (out []string) {
	if len(in) == 0 {
		return in
	}
	seen := make(map[string]bool)
	for _, v := range in {
		if seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	sort.Strings(out)
	return
}
