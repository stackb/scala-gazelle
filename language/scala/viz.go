package scala

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/emicklei/dot"

	"github.com/stackb/scala-gazelle/pkg/collections"
)

// go:embed ..
var gen embed.FS

type graphvizServer struct {
	registry       *importRegistry
	host           string
	port           string
	blockOnResolve bool
	env            *Env
}

func newGraphvizServer(registry *importRegistry) *graphvizServer {
	return &graphvizServer{registry: registry}
}

// RegisterFlags implements part of the Configurer interface.
func (v *graphvizServer) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {
	fs.StringVar(&v.host, "graphviz_host", "localhost", "bind host name for the graphviz server")
	fs.StringVar(&v.port, "graphviz_port", "", "port number for the graphviz server")
	fs.BoolVar(&v.blockOnResolve, "graphviz_block_on_resolve", false, "if true, block the process at the beginning of resolve phase (Ctrl-C to continue)")
}

// CheckFlags implements part of the Configurer interface.
func (v *graphvizServer) CheckFlags(fs *flag.FlagSet, c *config.Config) error {
	// configuring an empty port disables the server
	if v.port == "" {
		return nil
	}

	// Initialise our app-wide environment with the services/info we need.
	v.env = &Env{
		Registry: v.registry,
		Host:     getenv("HOST", v.host),
		Port:     getenv("PORT", v.port),
	}

	// Logs the error if ListenAndServe fails.
	go func() {
		hostPort := fmt.Sprintf("%s:%s", v.env.Host, v.env.Port)
		log.Println("starting graphviz server:", hostPort)
		log.Fatal(http.ListenAndServe(hostPort, newServeMux(v.env)))
	}()

	return nil
}

func newServeMux(env *Env) *http.ServeMux {
	mux := http.NewServeMux()

	// Note that we're using http.Handle, not http.HandleFunc. The
	// latter only accepts the http.HandlerFunc type, which is not
	// what we have here.
	mux.Handle("/api/imports", Handler{env, applicationJSON(apiImports)})
	mux.Handle("/dot/transitive/", Handler{env, textGraphviz(transitiveImports)})
	mux.Handle("/ui/imports", Handler{env, textHTML(uiImports)})

	return mux
}

// OnResolvePhase implements part of GazellePhaseTransitionListener.
func (v *graphvizServer) OnResolvePhase() error {
	if v.blockOnResolve {
		log.Printf("graphviz server waiting for requests on http://%s:%s, use ctrl-c to continue", v.env.Host, v.env.Port)
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		for sig := range c {
			// sig is a ^C, handle it
			log.Println("graphviz server: caught", sig)
			break
		}
	}
	return nil
}

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
		if err := tmpl.Execute(w, data); err != nil {
			log.Println("html rendering error: %v", err)
		}
		return nil
	}
}

func transitiveImports(e *Env, r *http.Request) (*dot.Graph, error) {
	typeName := path.Base(r.URL.Path)
	deps := []string{typeName}

	g := newGraph()
	unresolved := []string{}
	seen := make(map[uint32]struct{})

	for _, dep := range deps {
		// log.Println("resolving transitive imports of", dep)
		id, ok := e.Registry.symbols.Get(dep)
		if !ok {
			unresolved = append(unresolved, dep)
		}

		stack := make(collections.UInt32Stack, 0)
		stack.Push(id)
		src := g.Node(dep)

		for !stack.IsEmpty() {
			current, _ := stack.Pop()

			if _, ok := seen[current]; ok {
				continue
			}
			seen[current] = struct{}{}

			deps, ok := e.Registry.dependencies[current]
			if !ok {
				continue
			}

			it := deps.Iterator()
			for it.HasNext() {
				next := it.Next()
				other := e.Registry.symbols.Resolve(next)
				dst := g.Node(other)
				g.Edge(src, dst)
				stack.Push(next)
			}

		}
	}

	if len(unresolved) > 0 {
		src := g.Node("unresolved")
		for _, name := range unresolved {
			dst := g.Node(name)
			g.Edge(src, dst)
		}
	}

	return g, nil
}

func apiImports(e *Env, r *http.Request) (interface{}, error) {
	return e.Registry.symbols.symbols, nil
}

func uiImports(e *Env, r *http.Request) (interface{}, *template.Template, error) {
	tmpl := template.Must(template.ParseFS(gen, "imports.tmpl"))
	return e.Registry.symbols.symbols, tmpl, nil
}

func newGraph() *dot.Graph {
	g := dot.NewGraph(dot.Directed)
	g.Attr("rankdir", "LR")
	g.EdgeInitializer(func(e dot.Edge) {
		e.Attr("color", "gray50")
	})
	g.NodeInitializer(func(n dot.Node) {
		n.Attr("shape", "record")
		n.Attr("style", "filled")
		n.Attr("fillcolor", "gray95")
	})
	return g
}

// func (h *graphvizHandler) transitiveImports2(e *Env, typeName string) (*dot.Graph, error) {
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
type Env struct {
	Registry *importRegistry
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
