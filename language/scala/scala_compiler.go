package scala

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	// "github.com/stackb/rules_proto/pkg/protoc"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/stackb/scala-gazelle/pkg/index"
)

// NOT_FOUND is the diagnostic message prefix we expect from the scala compiler
// when it can't resolve types or values.
const NOT_FOUND = "not found: "

var notPackageMemberRe = regexp.MustCompile(`^object ([^ ]+) is not a member of package (.*)$`)

// ScalaCompiler knowns how to compile scala files.  This is not really meant to
// offer "real" compilation of files, but we can use the mostly standard scala
// compiler without a classpath such that a few of the initial passes are run,
// get a bunch of errors back, and use those diagnostics to glean info about the
// type system.
type ScalaCompiler interface {
	// Compile compiles the file and returns a compilespec.
	Compile(dir, filename string) (*index.ScalaCompileSpec, error)
}

func newScalaCompiler() *scalaCompiler {
	return &scalaCompiler{}
}

// scalaCompiler implements a compiler frontend for scala files that extracts
// the index information.  The compiler backend runs as a separate process.
type scalaCompiler struct {
	backendRawURL string
	// backendURL is the bind address for the compiler server
	backendURL *url.URL
	// cacheDir is the location where we can write cache files
	cacheDir string
	// jarPath is the unresolved runfile
	jarPath string
	// toolPath is the resolved path to the tool
	toolPath string
	// if we should start a subprocess for the compiler
	startSubprocess bool
	// maxCompileDialSeconds sets the timeout for the transport
	maxCompileDialSeconds time.Duration
	// maxCompileRequestSeconds sets the timeout for the transport
	maxCompileRequestSeconds time.Duration
	// process cancellation function
	cancel func()
	// http client
	client http.Client
}

// RegisterFlags implements part of the Configurer interface.
func (p *scalaCompiler) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {
	fs.StringVar(&p.jarPath, "scala_compiler_jar_path", "scala_compiler.jar", "filesystem path to the scala compiler tool jar")
	fs.StringVar(&p.backendRawURL, "scala_compiler_url", "http://127.0.0.1:8040", "bind address for the server")
	fs.StringVar(&p.cacheDir, "scala_cache_dir", "/tmp/scala_compiler", "Cache directory for scala compiler.  If unset, diables the cache")
	fs.BoolVar(&p.startSubprocess, "scala_compiler_subprocess", false, "whether to start the compiler subprocess")
	fs.DurationVar(&p.maxCompileDialSeconds, "scala_compiler_dial_timeout", time.Second*5, "compiler dial timeout")
	fs.DurationVar(&p.maxCompileRequestSeconds, "scala_compiler_request_timeout", time.Second*60, "compiler request timeout")
}

// CheckFlags implements part of the Configurer interface.
func (p *scalaCompiler) CheckFlags(fs *flag.FlagSet, c *config.Config) error {
	tool, err := bazel.Runfile(p.jarPath)
	if err != nil {
		log.Printf("failed to initialize compiler: %v\n", err)
		index.ListFiles(".")
		return err
	}
	p.toolPath = tool

	if err := p.initHTTPClient(); err != nil {
		return err
	}

	return nil
}

func (p *scalaCompiler) initHTTPClient() error {
	uri, err := url.Parse(p.backendRawURL)
	if err != nil {
		return fmt.Errorf("bad -scala_compiler_url: %w", err)
		return err
	}
	p.backendURL = uri

	if p.startSubprocess {
		if err := p.start(); err != nil {
			return err
		}
	}

	transport := &http.Transport{
		Dial: (&net.Dialer{
			Timeout: p.maxCompileDialSeconds,
		}).Dial,
		TLSHandshakeTimeout: p.maxCompileDialSeconds,
	}

	p.client = http.Client{Transport: transport, Timeout: p.maxCompileRequestSeconds}

	// log.Println("Created compiler http client:", p.backendURL)

	timeout := 1 * time.Second
	conn, err := net.DialTimeout("tcp", p.backendURL.Hostname()+":"+p.backendURL.Port(), timeout)
	if err != nil {
		log.Fatalln("Compiler unreachable, error:", err)
	}
	defer conn.Close()

	return nil
}

func (p *scalaCompiler) start() error {
	if _, err := os.Stat(p.toolPath); errors.Is(err, os.ErrNotExist) {
		return err
	}

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*1)
	cmd := exec.CommandContext(ctx, "java", "-Dscalac.server.port="+p.backendURL.Port(), "-jar", p.toolPath)

	p.cancel = cancel

	log.Println("Starting compiler:", p.toolPath, p.backendURL)

	if err := cmd.Start(); err != nil {
		log.Printf("failed to run compiler: %v\n", err)
		return err
	}

	return nil
}

func (p *scalaCompiler) stop() {
	if p.cancel != nil {
		p.cancel()
	}
}

// OnResolve implements GazellePhaseTransitionListener.
func (p *scalaCompiler) OnResolve() {
}

// OnEnd implements GazellePhaseTransitionListener.
func (p *scalaCompiler) OnEnd() {
}

// Compile a Scala file and returns the index. An error is raised if
// communicating with the long-lived Scala compiler over stdin and stdout fails.
func (p *scalaCompiler) Compile(dir, filename string) (*index.ScalaCompileSpec, error) {
	// log.Printf("--- COMPILE <%s> ---", filename)
	specFile := filepath.Join(p.cacheDir, filename+".json")

	if p.cacheDir != "" {
		if _, err := os.Stat(specFile); errors.Is(err, os.ErrNotExist) {
			log.Printf("Compile cache miss: <%s>", filename)
		} else {
			if spec, err := index.ReadScalaCompileSpec(specFile); err != nil {
				log.Printf("Compile cache error: <%s>: %v", filename, err)
			} else {
				// log.Printf("Compile cache hit: <%s>", filename)
				return spec, nil
			}
		}
	}

	compileRequest := &CompileRequest{Files: []string{filename}}

	out, err := xml.Marshal(compileRequest)
	if err != nil {
		return nil, fmt.Errorf("request error %w", err)
	}

	resp, err := p.client.Post(p.backendURL.String(), "text/xml", bytes.NewReader(out))
	if err != nil {
		return nil, fmt.Errorf("response error: %w", err)
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("compiler backend error: %v: %w", resp.Status, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("compiler backend error: %v: %s", resp.Status, string(data))
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("compiler backend error: %v, but empty response", resp.Status)
	}
	// fmt.Printf("Response Body : %s", data)

	if p.cacheDir != "" {
		outfile := filepath.Join(p.cacheDir, filename+".xml")
		outdir := filepath.Dir(outfile)
		if err := os.MkdirAll(outdir, os.ModePerm); err != nil {
			return nil, err
		}
		if err := ioutil.WriteFile(outfile, data, os.ModePerm); err != nil {
			return nil, err
		}
	}

	var compileResponse CompileResponse
	if err := xml.Unmarshal(data, &compileResponse); err != nil {
		return nil, fmt.Errorf("failed to compile %s: %w", filename, err)
	}

	var spec index.ScalaCompileSpec
	for _, d := range compileResponse.Diagnostics {
		processDiagnostic(&d, &spec)
	}

	// TODO: dedup the ScalaCompileSpec?

	if p.cacheDir != "" {
		outdir := filepath.Dir(specFile)
		if err := os.MkdirAll(outdir, os.ModePerm); err != nil {
			return nil, err
		}
		if err := index.WriteJSONFile(specFile, &spec); err != nil {
			return nil, err
		}
		log.Printf("Compile cache put: <%s>", filename)
	}

	return &spec, nil
}

func processDiagnostic(d *Diagnostic, spec *index.ScalaCompileSpec) {
	switch d.Severity {
	case "ERROR":
		processErrorDiagnostic(d, spec)
	default:
		return
	}
}

func processErrorDiagnostic(d *Diagnostic, spec *index.ScalaCompileSpec) {
	if strings.HasPrefix(d.Message, NOT_FOUND) {
		processNotFoundErrorDiagnostic(d.Message[len(NOT_FOUND):], spec)
	} else if match := notPackageMemberRe.FindStringSubmatch(d.Message); match != nil {
		processNotPackageMemberErrorDiagnostic(match[1], match[2], spec)
	}
}

func processNotFoundErrorDiagnostic(msg string, spec *index.ScalaCompileSpec) {
	fields := strings.Fields(msg)
	if len(fields) < 2 {
		return
	}
	for _, sym := range spec.NotFound {
		if sym.Kind == fields[0] && sym.Name == fields[1] {
			return
		}
	}
	spec.NotFound = append(spec.NotFound, &index.NotFoundSymbol{Kind: fields[0], Name: fields[1]})
}

func processNotPackageMemberErrorDiagnostic(obj, pkg string, spec *index.ScalaCompileSpec) {
	spec.NotMember = append(spec.NotMember, &index.NotMemberSymbol{Kind: "object", Name: obj, Package: pkg})
}

type CompileRequest struct {
	XMLName xml.Name `xml:"compileRequest"`
	Files   []string `xml:"file"`
}

type CompileResponse struct {
	XMLName     xml.Name     `xml:"compileResponse"`
	Diagnostics []Diagnostic `xml:"diagnostic"`
}

type Diagnostic struct {
	XMLName  xml.Name `xml:"diagnostic"`
	Source   string   `xml:"source,attr"`
	Line     int      `xml:"line,attr"`
	Severity string   `xml:"sev,attr"`
	Message  string   `xml:",chardata"`
}
