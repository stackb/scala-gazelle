package collections

import "strings"

type StringSlice []string

func (i *StringSlice) String() string {
	return strings.Join(*i, ",")
}

// Set implements the flag.Value interface.
func (i *StringSlice) Set(value string) error {
	*i = append(*i, value)
	return nil
}
