package resolver

import "strings"

// SymbolMap is a map of symbols that are in a scope.  For the import
// 'com.foo.Bar', the map key is 'Bar' and the map value is the known import
// for it.
type SymbolMap map[string]*Symbol

// Add tries to put the given import into the scope.  If the import does not
// have a valid base name, returns false.
func (s SymbolMap) Add(known *Symbol) bool {
	if basename, ok := symbolBasename(known.Name); ok {
		s[basename] = known
		return true
	}
	return false
}

// Get returns a symbol in scope.
func (s SymbolMap) Get(basename string) (*Symbol, bool) {
	known, ok := s[basename]
	return known, ok
}

func symbolBasename(imp string) (string, bool) {
	index := strings.LastIndex(imp, ".")
	if index <= 0 {
		return "", false
	}
	return imp[index+1:], true
}
