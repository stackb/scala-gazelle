package resolver

import "strings"

// KnownImportScope is a map of symbols that are in a scope.  For the import
// 'com.foo.Bar', the map key is 'Bar' and the map value is the known import
// for it.
type KnownImportScope map[string]*KnownImport

// Add tries to put the given import into the scope.  If the import does not
// have a valid base name, returns false.
func (s KnownImportScope) Add(known *KnownImport) bool {
	if basename, ok := importBasename(known.Import); ok {
		s[basename] = known
		return true
	}
	return false
}

// Get returns a symbol in scope.
func (s KnownImportScope) Get(basename string) (*KnownImport, bool) {
	known, ok := s[basename]
	return known, ok
}

func importBasename(imp string) (string, bool) {
	index := strings.LastIndex(imp, ".")
	if index <= 0 {
		return "", false
	}
	return imp[index+1:], true
}
