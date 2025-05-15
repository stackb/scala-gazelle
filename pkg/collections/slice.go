package collections

func SliceRemoveIndex[T any](slice []T, i int) []T {
	// Create a new slice with capacity one less than original
	result := make([]T, 0, len(slice)-1)

	// Add all elements except the one at index i
	result = append(result, slice[:i]...)
	result = append(result, slice[i+1:]...)

	return result
}

func SliceInsertAt[T any](slice []T, i int, value T) []T {
	// Create a new slice with capacity one more than original
	result := make([]T, 0, len(slice)+1)

	// Add elements before index i
	result = append(result, slice[:i]...)

	// Add the new value
	result = append(result, value)

	// Add elements after index i
	result = append(result, slice[i:]...)

	return result
}

// SliceDeduplicate removes duplicate entries
func SliceDeduplicate[T any](in []T) (out []T) {
	if len(in) == 0 {
		return in
	}
	seen := make(map[any]bool)
	for _, v := range in {
		if seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return
}
