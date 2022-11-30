package crossresolve

import (
	"flag"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
)

// PlatformLabel represents a label that does not need to be included in deps.
// Example: 'java.lang.Boolean'.
var PlatformLabel = label.New("platform", "", "do_not_import")

type ScalaJarResolver interface {
	resolve.CrossResolver
}

func NewJarIndexCrossResolver(lang string, depsRecorder DependencyRecorder) *JarIndexCrossResolver {
	return &JarIndexCrossResolver{
		lang:         lang,
		byLabel:      make(map[string][]label.Label),
		preferred:    make(map[label.Label]bool),
		symbols:      NewSymbolTable(),
		depsRecorder: depsRecorder,
	}
}

// JarIndexCrossResolver provides a cross-resolver for symbols extracted from
// jar files.  If -jar_index_file is configured, the internal cache will
// be bootstrapped with the contents of that file (typically the `classfile_index`
// rule is used to generate it).  JarSpec entries that have an empty label are
// assigned a special label 'PlatformLabel' which means ("you don't need to add
// anything to deps for this import, it's implied by the platform").  Typically
// platform implied jars are specified in the `classfile_index.platform_jars`
// attribute. At runtime, the cache is used to resolve scala import symbols
// during the gazelle dependency resolution phase. If a query for
// 'com.google.gson.Gson' yields '@maven//:com_google_code_gson_gson', that
// value should be added to deps.  If a query for 'java.lang.Boolean' yields the
// PlatformLabel, it can be skipped.
type JarIndexCrossResolver struct {
	lang string
	// depsRecorder is used to write dependencies that are discovered when the
	// JarSpecIndex is read.
	depsRecorder DependencyRecorder
	// jarIndexProtoFiles is a comma-separated list of filesystem paths.
	jarIndexProtoFiles string
	// byLabel is a mapping from an import string to the label that provides it.
	// It is possible more than one label provides a class.
	byLabel map[string][]label.Label
	// the full list of symbols
	symbols *SymbolTable
	// preferred is a mapping of preferred labels
	preferred map[label.Label]bool
}

// RegisterFlags implements part of the ConfigurableCrossResolver interface.
func (r *JarIndexCrossResolver) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {
	fs.StringVar(&r.jarIndexProtoFiles, "jarindex_proto_files", "", "comma-separated list of jarindex proto files")
}

// CheckFlags implements part of the ConfigurableCrossResolver interface.
func (r *JarIndexCrossResolver) CheckFlags(fs *flag.FlagSet, c *config.Config) error {
	if r.jarIndexProtoFiles == "" {
		return nil
	}
	for _, filename := range strings.Split(r.jarIndexProtoFiles, ",") {
		if !filepath.IsAbs(filename) {
			filename = filepath.Join(c.WorkDir, filename)
		}
		if err := r.readIndex(filename); err != nil {
			return err
		}
	}
	return nil
}

func (r *JarIndexCrossResolver) readIndex(filename string) error {
	return fmt.Errorf("no longer implemented")
	// // perform indexing here
	// index, err := index.ReadIndexSpec(filename)
	// if err != nil {
	// 	return fmt.Errorf("error while reading index specification file %s: %v", r.jarIndexProtoFiles, err)
	// }

	// isPredefined := make(map[label.Label]bool)
	// for _, v := range index.Predefined {
	// 	lbl, err := label.Parse(v)
	// 	if err != nil {
	// 		return fmt.Errorf("bad predefined label %q: %v", v, err)
	// 	}
	// 	isPredefined[lbl] = true
	// }

	// for _, v := range index.Preferred {
	// 	lbl, err := label.Parse(v)
	// 	if err != nil {
	// 		return fmt.Errorf("bad preferred label %q: %v", v, err)
	// 	}
	// 	r.preferred[lbl] = true
	// }

	// for _, jarSpec := range index.JarSpecs {
	// 	jarLabel, err := label.Parse(jarSpec.Label)
	// 	if err != nil {
	// 		if jarSpec.Label == "" {
	// 			jarLabel = PlatformLabel
	// 		} else {
	// 			log.Fatalf("bad label while loading jar spec %s: %v", jarSpec.Filename, err)
	// 			continue
	// 		}
	// 	}

	// 	if jarSpec.Filename == "" {
	// 		log.Panicf("unnamed jar? %+v", jarSpec)
	// 	}

	// 	if isPredefined[jarLabel] {
	// 		jarLabel = PlatformLabel
	// 	}
	// 	for _, pkg := range jarSpec.Packages {
	// 		r.byLabel[pkg] = append(r.byLabel[pkg], jarLabel)
	// 	}

	// 	for _, class := range jarSpec.Classes {
	// 		r.byLabel[class] = append(r.byLabel[class], jarLabel)
	// 	}

	// 	ruleNodeID := "rule/" + jarSpec.Label

	// 	for _, file := range jarSpec.Files {
	// 		r.byLabel[file.Name] = append(r.byLabel[file.Name], jarLabel)
	// 		// transform "org.json4s.package$MappingException" ->
	// 		// "org.json4s.MappingException" so that
	// 		// "org.json4s.MappingException" is resolveable.
	// 		pkgIndex := strings.LastIndex(file.Name, ".package$")
	// 		if pkgIndex != -1 && !strings.HasSuffix(file.Name, ".package$") {
	// 			name := file.Name[0:pkgIndex] + "." + file.Name[pkgIndex+len(".package$"):]
	// 			r.byLabel[name] = append(r.byLabel[name], jarLabel)
	// 		}

	// 		fileNodeID := path.Join("imp", file.Name)
	// 		r.addDependency(fileNodeID, ruleNodeID, "rule")

	// 		// for _, idx := range file.Classes {
	// 		// 	dst := path.Join("imp", jarSpec.Symbols[idx])
	// 		// 	r.addDependency(src, dst, "requires-class")
	// 		// }
	// 		for _, symbol := range file.Symbols {
	// 			impNodeID := path.Join("imp", symbol)
	// 			r.addDependency(fileNodeID, impNodeID, "import")
	// 		}
	// 	}
	// }
	// return nil
}

func (r *JarIndexCrossResolver) addDependency(src, dst, kind string) {
	r.depsRecorder(src, dst, kind)
	// record a dependency like akka.grpc.GrpcClientSettings$ -> io.grpc.netty.shaded.io.grpc.netty.NettyChannelBuilder as well.
	if strings.HasSuffix(src, "$") {
		r.depsRecorder(src[:len(src)-1], dst, kind)
	}
}

// Provided implements the protoc.ImportProvider interface.
func (r *JarIndexCrossResolver) Provided(lang, impLang string) map[label.Label][]string {
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
func (r *JarIndexCrossResolver) CrossResolve(c *config.Config, ix *resolve.RuleIndex, imp resolve.ImportSpec, lang string) []resolve.FindResult {
	if !crossResolverNameMatches(r.lang, lang, imp) {
		return nil
	}

	sym := strings.TrimSuffix(imp.Imp, "._")

	resolved := r.byLabel[sym]
	if len(resolved) == 0 {
		return nil
	}

	result := make([]resolve.FindResult, len(resolved))
	for i, v := range resolved {
		if r.preferred[v] {
			return []resolve.FindResult{{Label: v}}
		}
		result[i] = resolve.FindResult{Label: v}
	}

	return result
}
