package scalaparse

// Service is implemented by implementations that possibly manage a subprocess.
type Service interface {
	Start() error
	Stop()
}
