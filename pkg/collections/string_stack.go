// stack of strings
package collections

type StringStack []string

// IsEmpty checks if the stack is empty
func (s *StringStack) IsEmpty() bool {
	return len(*s) == 0
}

// Push a new string onto the stack
func (s *StringStack) Push(x string) {
	*s = append(*s, x)
}

// Pop: remove and return top element of stack, return false if stack is empty
func (s *StringStack) Pop() (string, bool) {
	if s.IsEmpty() {
		return "", false
	}

	i := len(*s) - 1
	x := (*s)[i]
	*s = (*s)[:i]

	return x, true
}

// Peek: return top element of stack, return false if stack is empty
func (s *StringStack) Peek() (string, bool) {
	if s.IsEmpty() {
		return "", false
	}

	i := len(*s) - 1
	x := (*s)[i]

	return x, true
}
