package scala

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/stackb/scala-gazelle/pkg/procutil"
	"github.com/stackb/scala-gazelle/pkg/scalaconfig"
)

const (
	// SCALA_GAZELLE_UNMANAGED_DEPS_FILE is an environment variable that, if
	// defined, activates the saving of uncorrelated deps for a given set of
	// rules that have recorded it.
	SCALA_GAZELLE_UNMANAGED_DEPS_FILE = procutil.EnvVar("SCALA_GAZELLE_UNMANAGED_DEPS_FILE")
)

// cleanup is the top-level function for various cleanup related features.
func (sl *scalaLang) cleanup() {
	if err := sl.cleanupUnmanagedDeps(); err != nil {
		log.Println("warning: cleanup uncorrelated deps failed: %v", err)
	}
}

func (sl *scalaLang) cleanupUnmanagedDeps() error {
	if filename, ok := procutil.LookupEnv(SCALA_GAZELLE_UNMANAGED_DEPS_FILE); ok {
		return sl.saveUncorrelatedDepsFile(filename)
	} else {
		sl.logger.Debug().Msg("SCALA_GAZELLE_UNMANAGED_DEPS_FILE not set")
	}
	return nil
}

func (sl *scalaLang) saveUncorrelatedDepsFile(filename string) error {
	deps := sl.makeUncorrelatedDeps()
	if len(deps) == 0 {
		sl.logger.Debug().Msg("SCALA_GAZELLE_UNMANAGED_DEPS_FILE not written (no uncorrelated deps to write)")
		return nil
	}

	var out bytes.Buffer
	for _, d := range deps {
		out.WriteString(d.String())
		out.WriteRune('\n')
	}

	if err := os.WriteFile(filename, out.Bytes(), os.ModePerm); err != nil {
		return err
	}

	sl.logger.Debug().Msg("Wrote uncorrelated deps to " + filename)

	return nil
}

func (sl *scalaLang) makeUncorrelatedDeps() []UncorrelatedDeps {
	nonDirect := make(map[label.Label]UncorrelatedDeps)

	for from, rule := range sl.knownRules {
		if deps, ok := rule.PrivateAttr(scalaconfig.UnmanagedDepsPrivateAttrName).([]string); ok {
			nonDirect[from] = UncorrelatedDeps{from: from, deps: deps}
			sl.logger.Debug().Str("from", from.String()).Msgf("uncorrelated deps: %v", deps)
		}
	}

	deps := make([]UncorrelatedDeps, 0, len(nonDirect))
	for _, d := range nonDirect {
		deps = append(deps, d)
	}

	sort.Slice(deps, func(i, j int) bool {
		a := deps[i]
		b := deps[j]
		return a.from.String() < b.from.String()
	})

	return deps
}

type UncorrelatedDeps struct {
	from label.Label
	deps []string
}

func (td *UncorrelatedDeps) String() string {
	return fmt.Sprintf("%v %v", td.from.String(), strings.Join(td.deps, " "))
}
