package scala

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"

	"github.com/stackb/scala-gazelle/pkg/index"
)

// PlatformLabel represents a label that does not need to be included in deps.
// Example: 'java.lang.Boolean'.
var PlatformLabel = label.New("platform", "", "do_not_import")

func newScalaClassIndexResolver(depsRecorder DependencyRecorder) *scalaClassIndexResolver {
	return &scalaClassIndexResolver{
		byLabel:      make(map[string][]label.Label),
		preferred:    make(map[label.Label]bool),
		symbols:      NewSymbolTable(),
		depsRecorder: depsRecorder,
	}
}

// scalaClassIndexResolver provides a cross-resolver for symbols extracted from
// jar files.  If -scala_class_index_file is configured, the internal cache will
// be bootstrapped with the contents of that file (typically the `java_index`
// rule is used to generate it).  JarSpec entries that have an empty label are
// assigned a special label 'PlatformLabel' which means ("you don't need to add
// anything to deps for this import, it's implied by the platform").  Typically
// platform implied jars are specified in the `java_index.platform_jars`
// attribute. At runtime, the cache is used to resolve scala import symbols
// during the gazelle dependency resolution phase. If a query for
// 'com.google.gson.Gson' yields '@maven//:com_google_code_gson_gson', that
// value should be added to deps.  If a query for 'java.lang.Boolean' yields the
// PlatformLabel, it can be skipped.
type scalaClassIndexResolver struct {
	// depsRecorder is used to write dependencies that are discovered when the
	// JarSpecIndex is read.
	depsRecorder DependencyRecorder
	// indexIn is the filesystem path to the index.
	indexIn string
	// byLabel is a mapping from an import string to the label that provides it.
	// It is possible more than one label provides a class.
	byLabel map[string][]label.Label
	// the full list of symbols
	symbols *SymbolTable
	// preferred is a mapping of preferred labels
	preferred map[label.Label]bool
}

// RegisterFlags implements part of the ConfigurableCrossResolver interface.
func (r *scalaClassIndexResolver) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {
	fs.StringVar(&r.indexIn, "scala_class_index_file", "", "name of the scala class index file to read/write")
}

// CheckFlags implements part of the ConfigurableCrossResolver interface.
func (r *scalaClassIndexResolver) CheckFlags(fs *flag.FlagSet, c *config.Config) error {
	if r.indexIn == "" {
		return nil
	}
	return r.readIndex()
}

func (r *scalaClassIndexResolver) readIndex() error {
	// perform indexing here
	index, err := index.ReadIndexSpec(r.indexIn)
	if err != nil {
		return fmt.Errorf("error while reading index specification file %s: %v", r.indexIn, err)
	}

	isPredefined := make(map[label.Label]bool)
	for _, v := range index.Predefined {
		lbl, err := label.Parse(v)
		if err != nil {
			return fmt.Errorf("bad predefined label %q: %v", v, err)
		}
		isPredefined[lbl] = true
	}

	for _, v := range index.Preferred {
		lbl, err := label.Parse(v)
		if err != nil {
			return fmt.Errorf("bad preferred label %q: %v", v, err)
		}
		r.preferred[lbl] = true
	}

	for _, jarSpec := range index.JarSpecs {
		jarLabel, err := label.Parse(jarSpec.Label)
		if err != nil {
			if jarSpec.Label == "" {
				jarLabel = PlatformLabel
			} else {
				log.Fatalf("bad label while loading jar spec %s: %v", jarSpec.Filename, err)
				continue
			}
		}
		if isPredefined[jarLabel] {
			jarLabel = PlatformLabel
		}
		for _, pkg := range jarSpec.Packages {
			r.byLabel[pkg] = append(r.byLabel[pkg], jarLabel)
		}

		for _, class := range jarSpec.Classes {
			r.byLabel[class] = append(r.byLabel[class], jarLabel)
		}

		for _, file := range jarSpec.Files {
			r.byLabel[file.Name] = append(r.byLabel[file.Name], jarLabel)

			// transform "org.json4s.package$MappingException" ->
			// "org.json4s.MappingException" so that
			// "org.json4s.MappingException" is resolveable.
			pkgIndex := strings.LastIndex(file.Name, ".package$")
			if pkgIndex != -1 && !strings.HasSuffix(file.Name, ".package$") {
				name := file.Name[0:pkgIndex] + "." + file.Name[pkgIndex+len(".package$"):]
				r.byLabel[name] = append(r.byLabel[name], jarLabel)
			}

			for _, idx := range file.Classes {
				dst := jarSpec.Symbols[idx]
				r.addDependency(file.Name, dst)
			}
			for _, symbol := range file.Symbols {
				r.addDependency(file.Name, symbol)
			}
		}
	}

	return nil
}

func (r *scalaClassIndexResolver) addDependency(src, dst string) {
	r.depsRecorder(src, dst)
	// record a dependency like akka.grpc.GrpcClientSettings$ -> io.grpc.netty.shaded.io.grpc.netty.NettyChannelBuilder as well.
	if strings.HasSuffix(src, "$") {
		r.depsRecorder(src[:len(src)-1], dst)
	}
}

// OnResolvePhase implements GazellePhaseTransitionListener.
func (r *scalaClassIndexResolver) OnResolvePhase() error {
	return nil
}

// Provided implements the protoc.ImportProvider interface.
func (r *scalaClassIndexResolver) Provided(lang, impLang string) map[label.Label][]string {
	if lang != "scala" && impLang != "scala" {
		return nil
	}

	result := make(map[label.Label][]string)
	for imp, ll := range r.byLabel {
		for _, l := range ll {
			result[l] = append(result[l], imp)
		}
	}

	return result
}

// CrossResolve implements the CrossResolver interface.
func (r *scalaClassIndexResolver) CrossResolve(c *config.Config, ix *resolve.RuleIndex, imp resolve.ImportSpec, lang string) (result []resolve.FindResult) {
	defer func() {
		log.Println("(scala class resolver) CrossResolved", len(result), "for", lang, imp.Lang, imp.Imp)
	}()

	if !(lang == ScalaLangName || imp.Lang == ScalaLangName) {
		return
	}

	sym := strings.TrimSuffix(imp.Imp, "._")

	resolved := r.byLabel[sym]
	if len(resolved) == 0 {
		return
	}

	result = make([]resolve.FindResult, len(resolved))
	for i, v := range resolved {
		if r.preferred[v] {
			return []resolve.FindResult{{Label: v}}
		}
		result[i] = resolve.FindResult{Label: v}
	}

	return
}
