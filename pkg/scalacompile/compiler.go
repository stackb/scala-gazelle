package scalacompile

import (
	"bytes"
	"encoding/xml"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
)

const debugCompiler = true

// NOT_FOUND is the diagnostic message prefix we expect from the scala compiler
// when it can't resolve types or values.
const NOT_FOUND = "not found: "

var notPackageMemberRe = regexp.MustCompile(`^object ([^ ]+) is not a member of package (.*)$`)

func NewCompiler() *Compiler {
	return &Compiler{}
}

// Compiler implements a scala compiler frontend that communicates with a
// backend process over HTTP.
type Compiler struct {
	backendHost        string
	backendPort        int
	backendUrl         string
	backendClient      *http.Client
	backendDialTimeout time.Duration

	// repoRoot is typically the config.Config.RepoRoot
	repoRoot string
	// cacheDir is the location where we can write cache files
	cacheDir string
	// scalacserverJarPath is the unresolved runfile
	scalacserverJarPath string
	// javaBinPath is the path to the java interpreter
	javaBinPath string

	// the process
	cmd *exec.Cmd
}

// RegisterFlags implements part of the Configurer interface.
func (p *Compiler) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {
	fs.StringVar(&p.scalacserverJarPath, "scala_compiler_jar_path", "", "filesystem path to the scala compiler tool jar")
	fs.StringVar(&p.javaBinPath, "scala_compiler_java_bin_path", "", "filesystem path to the java tool $(location @local_jdk//:bin/java)")
	fs.StringVar(&p.backendHost, "scala_compiler_backend_host", "localhost", "bind host for the backend server")
	fs.IntVar(&p.backendPort, "scala_compiler_backend_port", 0, "bind port for the backend server")
	fs.StringVar(&p.cacheDir, "scala_compiler_cache_dir", "/tmp/scala_compiler", "Cache directory for scala compiler.  If unset, diables the cache")
	fs.DurationVar(&p.backendDialTimeout, "scala_compiler_backend_dial_timeout", time.Second*3, "compiler backend dial timeout")
}

// CheckFlags implements part of the Configurer interface.
func (p *Compiler) CheckFlags(fs *flag.FlagSet, c *config.Config) error {
	p.repoRoot = c.RepoRoot
	p.javaBinPath = os.ExpandEnv(p.javaBinPath)
	p.scalacserverJarPath = os.ExpandEnv(p.scalacserverJarPath)

	// start is disabled if the backendUrl is set (via a test) or the
	// backendPort is not set (typical case).
	if p.backendPort == 0 {
		if err := p.start(); err != nil {
			return err
		}
	}

	return p.startHttpClient()
}

func (s *Compiler) start() error {
	t1 := time.Now()

	//
	// ensure we have a port
	//
	if s.backendPort == 0 {
		port, err := getFreePort()
		if err != nil {
			return status.Errorf(codes.FailedPrecondition, "getting http port: %v", err)
		}
		s.backendPort = port
	}

	//
	// Start the backend process
	//
	cmd := exec.Command(s.javaBinPath,
		fmt.Sprintf("-Dscalac.server.port=%d", s.backendPort),
		"-jar", s.scalacserverJarPath,
	)
	// cmd.Dir = s.repoRoot
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	s.cmd = cmd

	// if debugCompiler {
	// 	listFiles(cmd.Dir)
	// }

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting process %s: %w", s.scalacserverJarPath, err)
	}
	go func() {
		// does it make sense to wait for the process?  We kill it forcefully
		// at the end anyway...
		if err := cmd.Wait(); err != nil {
			if err.Error() != "signal: killed" {
				log.Printf("command wait err: %v", err)
			}
		}
	}()

	t2 := time.Since(t1).Round(1 * time.Millisecond)
	log.Printf("compiler started (%v)", t2)

	return nil
}

func (s *Compiler) startHttpClient() error {
	s.backendUrl = fmt.Sprintf("http://%s:%d", s.backendHost, s.backendPort)

	if !waitForConnectionAvailable(s.backendHost, s.backendPort, s.backendDialTimeout) {
		return fmt.Errorf("timed out waiting to connect to scalacserver http://%s:%d within %s", s.backendHost, s.backendPort, s.backendDialTimeout)
	}

	s.backendClient = &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout: 5 * time.Second,
			}).Dial,
			TLSHandshakeTimeout: 5 * time.Second,
		},
	}

	return nil
}

func (s *Compiler) Stop() error {
	if s.cmd != nil {
		if err := s.cmd.Process.Kill(); err != nil {
			return err
		}
		s.cmd = nil
	}
	return nil
}

// Compile a Scala file and returns the index. An error is raised if
// communicating with the long-lived Scala compiler over stdin and stdout fails.
func (p *Compiler) CompileScala(from label.Label, kind, dir string, filenames ...string) (*sppb.Rule, error) {
	t1 := time.Now()

	// if false {
	// 	filename := filenames[0]

	// 	// log.Printf("--- COMPILE <%s> ---", filename)
	// 	specFile := filepath.Join(p.cacheDir, filename+".json")

	// 	if false && p.cacheDir != "" {
	// 		if _, err := os.Stat(specFile); errors.Is(err, os.ErrNotExist) {
	// 			log.Printf("Compile cache miss: <%s>", filename)
	// 		} else {
	// 			if spec, err := ReadScalaCompileSpec(specFile); err != nil {
	// 				log.Printf("Compile cache error: <%s>: %v", filename, err)
	// 			} else {
	// 				// log.Printf("Compile cache hit: <%s>", filename)
	// 				return spec, nil
	// 			}
	// 		}
	// 	}
	// }

	compileRequest := &CompileRequest{Dir: dir, Files: filenames}

	out, err := xml.Marshal(compileRequest)
	if err != nil {
		return nil, fmt.Errorf("request error %w", err)
	}

	resp, err := p.backendClient.Post(p.backendUrl, "text/xml", bytes.NewReader(out))
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

	// if false {
	// 	if p.cacheDir != "" {
	// 		outfile := filepath.Join(p.cacheDir, filename+".xml")
	// 		outdir := filepath.Dir(outfile)
	// 		if err := os.MkdirAll(outdir, os.ModePerm); err != nil {
	// 			return nil, err
	// 		}
	// 		if err := ioutil.WriteFile(outfile, data, os.ModePerm); err != nil {
	// 			return nil, err
	// 		}
	// 	}
	// }

	var compileResponse CompileResponse
	if err := xml.Unmarshal(data, &compileResponse); err != nil {
		return nil, fmt.Errorf("failed to compile %v: %w", filenames, err)
	}

	fileMap := make(map[string]*sppb.File)
	seen := make(map[string]bool)

	for _, d := range compileResponse.Diagnostics {
		if d.Source == "" {
			if false {
				log.Printf("skipping diagnostic: %v (no file)", d)
			}
			continue
		}
		// log.Printf("diagnostic: %+v", d)

		file, ok := fileMap[d.Source]
		if !ok {
			file = &sppb.File{Filename: d.Source}
			fileMap[d.Source] = file
		}
		processDiagnostic(&d, file, seen)
	}

	// TODO: dedup the ScalaCompileSpec?

	// if p.cacheDir != "" {
	// 	outdir := filepath.Dir(specFile)
	// 	if err := os.MkdirAll(outdir, os.ModePerm); err != nil {
	// 		return nil, err
	// 	}
	// 	if err := WriteJSONFile(specFile, &spec); err != nil {
	// 		return nil, err
	// 	}
	// 	log.Printf("Compile cache put: <%s>", filename)
	// }

	keys := make([]string, 0, len(fileMap))
	for k := range fileMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	files := make([]*sppb.File, len(keys))
	for i, k := range keys {
		files[i] = fileMap[k]
		filename := strings.TrimPrefix(files[i].Filename, dir)
		filename = strings.TrimPrefix(filename, "/")
		files[i].Filename = filename
	}

	t2 := time.Since(t1).Round(1 * time.Millisecond)
	log.Printf("Compiled %s (%d files, %v)", from, len(files), t2)

	return &sppb.Rule{
		Label: from.String(),
		Kind:  kind,
		Files: files,
	}, nil
}

func processDiagnostic(d *Diagnostic, file *sppb.File, seen map[string]bool) {
	switch d.Severity {
	case "ERROR":
		processErrorDiagnostic(d, file, seen)
	default:
		return
	}
}

func processErrorDiagnostic(d *Diagnostic, file *sppb.File, seen map[string]bool) {
	if strings.HasPrefix(d.Message, NOT_FOUND) {
		processNotFoundErrorDiagnostic(d.Message[len(NOT_FOUND):], file, seen)
	} else if match := notPackageMemberRe.FindStringSubmatch(d.Message); match != nil {
		processNotPackageMemberErrorDiagnostic(match[1], match[2], file, seen)
	}
}

func processNotFoundErrorDiagnostic(msg string, file *sppb.File, seen map[string]bool) {
	if seen[msg] {
		return
	}
	seen[msg] = true
	fields := strings.Fields(msg)
	if len(fields) < 2 {
		return
	}
	file.Symbols = append(file.Symbols, &sppb.Symbol{
		Type: parseSymbolType(fields[0]),
		Name: fields[1],
	})
}

func processNotPackageMemberErrorDiagnostic(obj, pkg string, file *sppb.File, seen map[string]bool) {
	msg := fmt.Sprintf("not-member:%s:%s", obj, pkg)
	if seen[msg] {
		return
	}
	seen[msg] = true

	file.Symbols = append(file.Symbols, &sppb.Symbol{
		Type: sppb.SymbolType_SYMBOL_PACKAGE,
		Name: pkg + "?" + obj,
	})
	// spec.NotMember = append(spec.NotMember, &NotMemberSymbol{Kind: "object", Name: obj, Package: pkg})
}

func parseSymbolType(val string) sppb.SymbolType {
	switch val {
	case "object":
		return sppb.SymbolType_SYMBOL_OBJECT
	case "type":
		return sppb.SymbolType_SYMBOL_TYPE
	case "value":
		return sppb.SymbolType_SYMBOL_VALUE
	default:
		log.Panicf("unknown symbol type: %q", val)
		return sppb.SymbolType_SYMBOL_TYPE_UNKNOWN
	}
}

// getFreePort asks the kernel for a free open port that is ready to use.
func getFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// waitForConnectionAvailable pings a tcp connection every 250 milliseconds
// until it connects and returns true.  If it fails to connect by the timeout
// deadline, returns false.
func waitForConnectionAvailable(host string, port int, timeout time.Duration) bool {
	target := fmt.Sprintf("%s:%d", host, port)
	var wg sync.WaitGroup
	wg.Add(1)
	then := time.Now()

	success := make(chan bool, 1)

	go func() {
		go func() {
			defer wg.Done()
			for {
				_, err := net.Dial("tcp", target)
				if err == nil {
					if debugCompiler {
						log.Printf("%s is available after %s", target, time.Since(then))
					}
					break
				}
				time.Sleep(250 * time.Millisecond)
			}
		}()
		wg.Wait()
		success <- true
	}()

	select {
	case <-success:
		return true
	case <-time.After(timeout):
		return false
	}
}

// listFiles is a convenience debugging function to log the files under a given dir.
func listFiles(dir string) error {
	log.Println("Listing files under " + dir)
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("%v\n", err)
			return err
		}
		log.Println(path)
		return nil
	})
}
