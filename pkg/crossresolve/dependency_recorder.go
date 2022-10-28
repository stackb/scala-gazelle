package crossresolve

// DependencyRecorder is a function that records a dependency between src and
// dst.  For example, class java.util.ArrayList (src) has a dependency on
// java.util.List (dst).
type DependencyRecorder func(src, dst, kind string)
