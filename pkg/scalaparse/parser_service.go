package scalaparse

// ParserService is a mashup of interfaces.
type ParserService interface {
	Service
	Reader
	Parser
}
