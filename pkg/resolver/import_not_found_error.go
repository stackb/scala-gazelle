package resolver

import "fmt"

// ErrImportNotFound is an error value assigned to an Import when the import
// could not be resolved.
var ErrImportNotFound = fmt.Errorf("import not found")
