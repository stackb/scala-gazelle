package scala

import (
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/emicklei/dot"

	"github.com/stackb/scala-gazelle/pkg/collections"
	"github.com/stackb/scala-gazelle/pkg/index"
)

//go:embed *
var gen embed.FS

type graphvizServer struct {
	packages       map[string]*scalaPackage
	registry       *importRegistry
	host           string
	port           string
	blockOnResolve bool
	env            *Env
	mux            *http.ServeMux
}

func newGraphvizServer(packages map[string]*scalaPackage, registry *importRegistry) *graphvizServer {
	return &graphvizServer{
		packages: packages,
		registry: registry,
	}
}

// RegisterFlags implements part of the Configurer interface.
func (v *graphvizServer) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {
	fs.StringVar(&v.host, "graphviz_host", "localhost", "bind host name for the graphviz server")
	fs.StringVar(&v.port, "graphviz_port", "", "port number for the graphviz server")
	fs.BoolVar(&v.blockOnResolve, "graphviz_block_on_resolve", false, "if true, block the process at the beginning of resolve phase (Ctrl-C to continue)")
	log.Println("all flags registered")
}

// CheckFlags implements part of the Configurer interface.
func (v *graphvizServer) CheckFlags(fs *flag.FlagSet, c *config.Config) error {
	// configuring an empty port disables the server
	if v.port == "" {
		// return fmt.Errorf("no port configured")
		return nil
	}

	// Initialise our app-wide environment with the services/info we need.
	v.env = &Env{
		Registry: v.registry,
		Packages: v.packages,
		Host:     getenv("HOST", v.host),
		Port:     getenv("PORT", v.port),
	}

	v.mux = newServeMux(v.env)

	// Logs the error if ListenAndServe fails.
	go func() {
		hostPort := fmt.Sprintf("%s:%s", v.env.Host, v.env.Port)
		log.Println("starting graphviz server:", hostPort)
		log.Fatal(http.ListenAndServe(hostPort, v.mux))
	}()

	return nil
}

func (v *graphvizServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	v.mux.ServeHTTP(w, r)
}

func newServeMux(env *Env) *http.ServeMux {
	mux := http.NewServeMux()

	mux.Handle("/api/imports", Handler{env, applicationJSON(apiImports)})

	mux.Handle("/ui/symbols", Handler{env, textHTML(uiSymbols)})
	mux.Handle("/ui/imp/", Handler{env, textHTML(uiImport)})
	mux.Handle("/ui/pkg/", Handler{env, textHTML(uiPkg)})
	mux.Handle("/ui/packages", Handler{env, textHTML(uiPackages)})
	mux.Handle("/ui/rules", Handler{env, textHTML(uiRules)})
	mux.Handle("/ui/rule/", Handler{env, textHTML(uiRule)})
	mux.Handle("/ui/file/", Handler{env, textHTML(uiFile)})
	mux.Handle("/ui/home", Handler{env, textHTML(uiHome)})
	mux.Handle("/", Handler{env, textHTML(uiHome)})

	return mux
}

// OnResolve implements part of GazellePhaseTransitionListener.
func (v *graphvizServer) OnResolve() error {
	// if v.blockOnResolve {
	// 	if v.blockOnResolve {
	// 		v.waitForInterrupt()
	// 	}
	// }
	return nil
}

// OnEnd implements part of GazellePhaseTransitionListener.
func (v *graphvizServer) OnEnd() error {
	if v.blockOnResolve {
		v.waitForInterrupt()
	}
	return nil
}

// waitForInterrupt blocks until the user presses ctrl-c
func (v *graphvizServer) waitForInterrupt() {
	log.Printf("graphviz server waiting for requests on http://%s:%s", v.env.Host, v.env.Port)
	log.Println("(press Enter to continue)")
	fmt.Scanln()
}

// ***************** CONTENT-TYPES *****************

func textGraphviz(callback func(e *Env, r *http.Request) (*dot.Graph, error)) func(e *Env, w http.ResponseWriter, r *http.Request) error {
	return func(e *Env, w http.ResponseWriter, r *http.Request) error {
		g, err := callback(e, r)
		if err != nil {
			return err
		}
		w.Header().Add("Content-Type", "text/vnd.graphviz")
		w.Write([]byte(g.String()))
		return nil
	}
}

func applicationJSON(callback func(e *Env, r *http.Request) (interface{}, error)) func(e *Env, w http.ResponseWriter, r *http.Request) error {
	return func(e *Env, w http.ResponseWriter, r *http.Request) error {
		entity, err := callback(e, r)
		if err != nil {
			return err
		}
		data, err := json.Marshal(entity)
		if err != nil {
			return StatusError{500, err}
		}
		w.Header().Add("Content-Type", "application/json")
		w.Write(data)
		return nil
	}
}

func textHTML(callback func(e *Env, r *http.Request) (interface{}, *template.Template, error)) func(e *Env, w http.ResponseWriter, r *http.Request) error {
	return func(e *Env, w http.ResponseWriter, r *http.Request) error {
		data, tmpl, err := callback(e, r)
		if err != nil {
			return err
		}
		w.Header().Add("Content-Type", "text/html")
		if err := tmpl.ExecuteTemplate(w, "index.gohtml", data); err != nil {
			log.Printf("html rendering error: %v", err)
		}
		return nil
	}
}

// ***************** API ENTRYPOINTS *****************

func apiImports(e *Env, r *http.Request) (interface{}, error) {
	return e.Registry.symbols.symbols, nil
}

// ***************** UI ENTRYPOINTS *****************

func uiHome(e *Env, r *http.Request) (interface{}, *template.Template, error) {
	data := newPageData(e)

	tmpl := newFSTemplate("home.gohtml")
	return data, tmpl, nil
}

func uiPackages(e *Env, r *http.Request) (interface{}, *template.Template, error) {
	data := newPageData(e)

	for name := range e.Packages {
		data.Packages = append(data.Packages, name)
	}
	sort.Strings(data.Packages)

	g, err := transitiveDeps(e.Registry, "ws/default")
	if err != nil {
		return nil, nil, StatusError{500, err}
	}
	data.Graph = g.String()

	tmpl := newFSTemplate("packages.gohtml")
	return data, tmpl, nil
}

func uiPkg(e *Env, r *http.Request) (interface{}, *template.Template, error) {
	pkg := strings.TrimPrefix(r.URL.Path, "/ui/pkg/")

	data := newPageData(e)
	data.Package = e.Packages[pkg]
	data.LabelMappings = makeReverseLabelMappings(data.Package, e.Packages)
	tmpl := newFSTemplate("pkg.gohtml")

	g, err := transitiveDeps(e.Registry, "pkg/"+pkg)
	if err != nil {
		return nil, nil, StatusError{500, err}
	}
	data.Graph = g.String()

	return data, tmpl, nil
}

// makeReverseLabelMappings create a slice of LabelMapping. Each mapping .From
// represents the "actual" bazel label, whereas .To represents the name of the
// rule found in the BUILD file.
func makeReverseLabelMappings(pkg *scalaPackage, packages map[string]*scalaPackage) []*LabelMapping {
	mappings := make([]*LabelMapping, 0)
	for pkg, p := range packages {
		for _, r := range p.rules {
			from := label.New("", pkg, r.Name())
			if mapping, ok := p.cfg.mapKindImportNames[r.Kind()]; ok {
				mappings = append(mappings, &LabelMapping{
					From: mapping.Rename(from),
					To:   from,
				})
			}
		}
	}
	return mappings
}

func uiSymbols(e *Env, r *http.Request) (interface{}, *template.Template, error) {
	data := newPageData(e)

	tmpl := newFSTemplate("symbols.gohtml")
	return data, tmpl, nil
}

func uiRules(e *Env, r *http.Request) (interface{}, *template.Template, error) {
	data := newPageData(e)

	tmpl := newFSTemplate("rules.gohtml")
	return data, tmpl, nil
}

func uiImport(e *Env, r *http.Request) (interface{}, *template.Template, error) {
	imp := path.Join("imp", path.Base(r.URL.Path))
	if _, ok := e.Registry.symbols.Get(imp); !ok {
		return nil, nil, StatusError{404, fmt.Errorf(imp + " not found")}
	}

	directs, _ := e.Registry.DirectImports(imp)
	sort.Strings(directs)

	transitives, _ := e.Registry.TransitiveImports([]string{imp}, -1)
	sort.Strings(transitives)

	data := newPageData(e)
	data.Import = ImportData{
		Name:        imp,
		Directs:     directs,
		Transitives: transitives,
	}

	g, err := transitiveDeps(e.Registry, imp)
	if err != nil {
		return nil, nil, StatusError{500, err}
	}
	data.Graph = g.String()

	tmpl := newFSTemplate("import.gohtml")
	return data, tmpl, nil
}

func uiRule(e *Env, r *http.Request) (interface{}, *template.Template, error) {
	rest := strings.TrimPrefix(r.URL.Path, "/ui/rule/")

	var repo, pkg, name string

	if strings.HasPrefix(rest, "@") {
		slash := strings.IndexRune(rest, '/')
		repo = rest[1:slash]
		rest = rest[slash+1:]
	}

	pkg = path.Dir(rest)
	if pkg == "." {
		pkg = ""
	}
	name = path.Base(rest)
	colon := strings.LastIndex(name, ":")
	if colon != -1 {
		pkg = path.Join(pkg, name[:colon])
		name = name[colon+1:]
	} else {
		pkg = path.Join(pkg, name)
	}
	from := label.New(repo, pkg, name)

	data := newPageData(e)

	scalaPkg := e.Packages[pkg]
	if scalaPkg != nil {
		rule, _ := scalaPkg.getRule(name)
		data.Rule = rule
	} else {
		log.Println("warning: no scala package known for:", pkg)
	}

	var tmpl *template.Template

	if sourceRule := e.Registry.sourceRuleRegistry.GetScalaRules()[from]; sourceRule != nil {
		data.RuleSpec = sourceRule
		tmpl = newFSTemplate("rule.gohtml")
	} else if jar, ok := e.Registry.classFileRegistry.LookupJar(from); ok {
		data.Jar = jar
		tmpl = newFSTemplate("jar.gohtml")
	} else {
		return nil, nil, StatusError{404, fmt.Errorf("%v not found", from)}
	}

	g, err := transitiveDeps(e.Registry, "rule/"+from.String())
	if err != nil {
		return nil, nil, StatusError{500, err}
	}
	data.Graph = g.String()

	return data, tmpl, nil
}

func uiFile(e *Env, r *http.Request) (interface{}, *template.Template, error) {
	filename := strings.TrimPrefix(r.URL.Path, "/ui/file/")

	file := e.Registry.sourceRuleRegistry.GetScalaFile(filename)
	if file == nil {
		return nil, nil, StatusError{404, fmt.Errorf(filename + " not found")}
	}

	data := newPageData(e)
	data.File = file

	g, err := transitiveDeps(e.Registry, path.Join("file", filename))
	if err != nil {
		return nil, nil, StatusError{500, err}
	}
	data.Graph = g.String()

	tmpl := newFSTemplate("file.gohtml")
	return data, tmpl, nil
}

// ***************** SUPPORT FUNCTIONS *****************

func transitiveDeps(registry *importRegistry, dep string) (*dot.Graph, error) {
	g := newGraph()
	seen := make(map[uint32]struct{})

	// log.Println("resolving transitive imports of", dep)
	id, ok := registry.symbols.Get(dep)
	if !ok {
		return nil, StatusError{404, fmt.Errorf("symbol not found: %s", dep)}
	}

	if got := registry.symbols.resolveAt(id); got != dep {
		log.Panicf("dep %q (id=%d) was not idempotent to add (got %q instead)", dep, id, got)
	}

	dst := g.Node(dep)
	in := registry.Previous(dep)
	it := in.Iterator()
	for it.HasNext() {
		next := it.Next()
		that := registry.symbols.resolveAt(next)
		src := g.Node(that)
		edge := g.Edge(src, dst)
		edge.Label(registry.EdgeKind(next, id))
	}

	stack := make(collections.UInt32Stack, 0)
	stack.Push(id)

	for !stack.IsEmpty() {
		current, _ := stack.Pop()
		if _, ok := seen[current]; ok {
			continue
		}
		seen[current] = struct{}{}

		out, ok := registry.dependencies[current]
		if !ok {
			continue
		}

		this := registry.symbols.resolveAt(current)
		src := g.Node(this)

		it := out.Iterator()
		for it.HasNext() {
			next := it.Next()
			that := registry.symbols.resolveAt(next)
			dst := g.Node(that)
			edge := g.Edge(src, dst)
			edge.Label(registry.EdgeKind(current, next))
			if len(stack) < 4 {
				stack.Push(next)
			}
		}
	}

	return g, nil
}

func newGraph() *dot.Graph {
	g := dot.NewGraph(dot.Directed)
	g.Attr("rankdir", "LR")
	g.EdgeInitializer(func(e dot.Edge) {
		e.Attr("color", "gray85")
		e.Attr("arrowsize", "0.7")
	})
	g.NodeInitializer(func(n dot.Node) {
		id := n.Value("label").(string)
		fields := strings.SplitN(id, "/", 2)
		kind := fields[0]
		label := fields[1]
		n.Label(label)
		n.Attr("URL", fmt.Sprintf("/ui/%v", id))
		n.Attr("shape", "record")
		n.Attr("style", "filled")

		switch kind {
		case "imp":
			n.Attr("fillcolor", "gray95")
		case "file":
			n.Attr("fontcolor", "white")
			n.Attr("color", "darkred")
			n.Attr("fillcolor", "red")
		case "jar":
			n.Attr("fontcolor", "white")
			n.Attr("fillcolor", "red")
		case "rule":
			n.Attr("fontcolor", "white")
			n.Attr("fillcolor", "green")
		case "pkg":
			n.Attr("fontcolor", "white")
			n.Attr("fillcolor", "blue")
		}
	})
	return g
}

func newPageData(e *Env) *PageData {
	symbols := e.Registry.symbols.symbols

	byLabel := e.Registry.sourceRuleRegistry.GetScalaRules()
	byKey := make(map[string]*index.ScalaRuleSpec)
	keys := make([]string, 0, len(byLabel))

	for from, rule := range byLabel {
		key := from.String()
		keys = append(keys, key)
		byKey[key] = rule
	}

	sort.Strings(keys)

	rules := make([]*index.ScalaRuleSpec, len(keys))
	for i, key := range keys {
		rules[i] = byKey[key]
	}

	return &PageData{
		Symbols: symbols,
		Rules:   rules,
	}
}

func newFSTemplate(files ...string) *template.Template {
	return template.Must(template.New("index").Funcs(template.FuncMap{
		"printRule": func(values ...interface{}) (string, error) {
			if len(values) != 1 {
				return "", errors.New("invalid printRule call")
			}
			r, ok := values[0].(*rule.Rule)
			if !ok {
				return "", errors.New("printRule arg must have type *rule.Rule")
			}
			file := rule.EmptyFile("", "")
			r.Insert(file)
			return string(file.Format()), nil
		},
		"privateAttrString": func(values ...interface{}) (string, error) {
			if len(values) != 2 {
				return "", errors.New("invalid privateAttrString call (want RULE ATTR_NAME)")
			}
			r, ok := values[0].(*rule.Rule)
			if !ok {
				return "", errors.New("privateAttrString arg.0 must have type *rule.Rule")
			}
			attrName, ok := values[1].(string)
			if !ok {
				return "", errors.New("privateAttrString arg.1 must have type string")
			}
			value, ok := r.PrivateAttr(attrName).(string)
			if !ok {
				return "", nil
			}
			return value, nil
		},
		"dict": func(values ...interface{}) (map[string]interface{}, error) {
			if len(values)%2 != 0 {
				return nil, errors.New("invalid dict call")
			}
			dict := make(map[string]interface{}, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, errors.New("dict keys must be strings")
				}
				dict[key] = values[i+1]
			}
			return dict, nil
		},
	}).ParseFS(gen, append([]string{"index.gohtml", "header.gohtml"}, files...)...))
}

// func (h *graphvizHandler) transitiveDeps2(e *Env, typeName string) (*dot.Graph, error) {
// 	transitive, unresolved := e.Registry.TransitiveImports([]string{typeName})

// 	g := h.newGraph()

// 	src := g.Node(typeName)

// 	for _, name := range transitive {
// 		dst := g.Node(name)
// 		g.Edge(src, dst)
// 	}

// 	if len(unresolved) > 0 {
// 		src = g.Node("unresolved")
// 		for _, name := range unresolved {
// 			dst := g.Node(name)
// 			g.Edge(src, dst)
// 		}
// 	}

// 	return g, nil
// }

// A (simple) example of our application-wide configuration.

// ***************** SUPPORT TYPES *****************

type Env struct {
	Registry *importRegistry
	Packages map[string]*scalaPackage
	Port     string
	Host     string
}

// The Handler struct that takes a configured Env and a function matching
// our useful signature.
type Handler struct {
	*Env
	H func(e *Env, w http.ResponseWriter, r *http.Request) error
}

// ServeHTTP allows our Handler type to satisfy http.Handler.
func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := h.H(h.Env, w, r)
	if err != nil {
		switch e := err.(type) {
		case Error:
			// We can retrieve the status here and write out a specific
			// HTTP status code.
			log.Printf("HTTP %d - %s", e.Status(), e)
			http.Error(w, e.Error(), e.Status())
		default:
			// Any error types we don't specifically look out for default
			// to serving a HTTP 500
			http.Error(w, http.StatusText(http.StatusInternalServerError),
				http.StatusInternalServerError)
		}
	}
}

// Error represents a handler error. It provides methods for a HTTP status
// code and embeds the built-in error interface.
type Error interface {
	error
	Status() int
}

// StatusError represents an error with an associated HTTP status code.
type StatusError struct {
	Code int
	Err  error
}

// Allows StatusError to satisfy the error interface.
func (se StatusError) Error() string {
	return se.Err.Error()
}

// Returns our HTTP status code.
func (se StatusError) Status() int {
	return se.Code
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

type PageData struct {
	Symbols       []string
	Rules         []*index.ScalaRuleSpec
	Packages      []string
	Package       *scalaPackage
	Import        ImportData
	RuleSpec      *index.ScalaRuleSpec
	Rule          *rule.Rule
	LabelMappings []*LabelMapping
	File          *index.ScalaFileSpec
	Jar           *index.JarSpec
	Graph         string
}

type ImportData struct {
	Name        string
	Directs     []string
	Transitives []string
}

type LabelMapping struct {
	From label.Label
	To   label.Label
}
