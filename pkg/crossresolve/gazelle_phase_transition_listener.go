package crossresolve

// GazellePhaseTransitionListener is an optional interface for a cross-resolver
// that wants phase transition notification.
type GazellePhaseTransitionListener interface {
	OnResolve()
	OnEnd()
}
