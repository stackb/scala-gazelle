// stack of integers
package collections

type UInt32Stack []uint32

// IsEmpty checks if the stack is empty
func (s *UInt32Stack) IsEmpty() bool {
	return len(*s) == 0
}

// Push a new integer onto the stack
func (s *UInt32Stack) Push(x uint32) {
	*s = append(*s, x)
}

// Pop: remove and return top element of stack, return false if stack is empty
func (s *UInt32Stack) Pop() (uint32, bool) {
	if s.IsEmpty() {
		return 0, false
	}

	i := len(*s) - 1
	x := (*s)[i]
	*s = (*s)[:i]

	return x, true
}

// Peek: return top element of stack, return false if stack is empty
func (s *UInt32Stack) Peek() (uint32, bool) {
	if s.IsEmpty() {
		return 0, false
	}

	i := len(*s) - 1
	x := (*s)[i]

	return x, true
}
