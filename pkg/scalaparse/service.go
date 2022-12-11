package scalaparse

// Service is implemented by implementations that can read cached rule state and
// possibly manage a subprocess.
type Service interface {
	// Start begins the parser.
	Start() error
	Stop()
}
