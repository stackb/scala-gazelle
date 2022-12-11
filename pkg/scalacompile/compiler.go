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
	"strings"
	"sync"
	"time"

	"github.com/bazelbuild/bazel-gazelle/config"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const debugCompiler = false

// NOT_FOUND is the diagnostic message prefix we expect from the scala compiler
// when it can't resolve types or values.
const NOT_FOUND = "not found: "

var notPackageMemberRe = regexp.MustCompile(`^object ([^ ]+) is not a member of package (.*)$`)

func NewCompiler() *Compiler {
	return &Compiler{}
}

// Compiler implements a compiler frontend for scala files that extracts
// the index information.  The compiler backend runs as a separate process.
type Compiler struct {
	backendRawURL string
	repoRoot      string
	// cacheDir is the location where we can write cache files
	cacheDir string
	// scalacserverJarPath is the unresolved runfile
	scalacserverJarPath string
	// javaBinPath is the path to the java interpreter
	javaBinPath string
	// if we should start a subprocess for the compiler
	startSubprocess bool
	cmd             *exec.Cmd

	httpClient *http.Client
	httpUrl    string

	HttpPort int
}

// RegisterFlags implements part of the Configurer interface.
func (p *Compiler) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {
	fs.StringVar(&p.scalacserverJarPath, "scala_compiler_jar_path", "", "filesystem path to the scala compiler tool jar")
	fs.StringVar(&p.javaBinPath, "scala_compiler_java_bin_path", "", "filesystem path to the java tool $(location @local_jdk//:bin/java)")
	fs.StringVar(&p.backendRawURL, "scala_compiler_url", "http://127.0.0.1:8040", "bind address for the server")
	fs.StringVar(&p.cacheDir, "scala_compiler_cache_dir", "/tmp/scala_compiler", "Cache directory for scala compiler.  If unset, diables the cache")
	fs.BoolVar(&p.startSubprocess, "scala_compiler_start_subprocess", true, "whether to start the compiler subprocess")
	fs.DurationVar(&p.maxCompileDialSeconds, "scala_compiler_dial_timeout", time.Second*5, "compiler dial timeout")
}

// CheckFlags implements part of the Configurer interface.
func (p *Compiler) CheckFlags(fs *flag.FlagSet, c *config.Config) error {
	p.repoRoot = c.RepoRoot
	p.javaBinPath = os.ExpandEnv(p.javaBinPath)
	p.scalacserverJarPath = os.ExpandEnv(p.scalacserverJarPath)

	if debugCompiler {
		for _, e := range os.Environ() {
			log.Println(e)
		}
	}
	if !p.startSubprocess {
		return nil
	}
	if err := p.Start(); err != nil {
		return err
	}
	return nil
}

func (s *Compiler) Start() error {
	t1 := time.Now()

	//
	// ensure we have a port
	//
	if s.HttpPort == 0 {
		port, err := getFreePort()
		if err != nil {
			return status.Errorf(codes.FailedPrecondition, "getting http port: %v", err)
		}
		s.HttpPort = port
	}
	s.httpUrl = fmt.Sprintf("http://127.0.0.1:%d", s.HttpPort)

	//
	// Start the bun process
	//
	cmd := exec.Command(s.javaBinPath,
		fmt.Sprintf("-Dscalac.server.port=%d", s.HttpPort),
		"-jar",
		s.scalacserverJarPath,
	)
	// cmd.Dir = s.repoRoot
	cmd.Env = []string{
		fmt.Sprintf("PORT=%d", s.HttpPort),
	}
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

	host := "localhost"
	port := s.HttpPort
	timeout := 3 * time.Second
	if !waitForConnectionAvailable(host, port, timeout) {
		return fmt.Errorf("waiting to connect to scalacserver %s:%d within %s", host, port, timeout)
	}

	//
	// Setup the http client
	//
	s.httpClient = &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout: 5 * time.Second,
			}).Dial,
			TLSHandshakeTimeout: 5 * time.Second,
		},
	}

	t2 := time.Since(t1).Round(1 * time.Millisecond)
	log.Printf("compiler started (%v)", t2)

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
func (p *Compiler) CompileScala(dir string, filenames []string) (*ScalaCompileSpec, error) {
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

	files := make([]string, len(filenames))
	for i, filename := range filenames {
		files[i] = filepath.Join(p.repoRoot, filename)
	}
	compileRequest := &CompileRequest{Files: files}

	out, err := xml.Marshal(compileRequest)
	if err != nil {
		return nil, fmt.Errorf("request error %w", err)
	}

	url := fmt.Sprintf("http://127.0.0.1:%d", p.HttpPort)

	resp, err := p.httpClient.Post(url, "text/xml", bytes.NewReader(out))
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

	var spec ScalaCompileSpec
	for _, d := range compileResponse.Diagnostics {
		processDiagnostic(&d, &spec)
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

	t2 := time.Since(t1).Round(1 * time.Millisecond)
	log.Printf("compiled %v (%v)", filenames, t2)

	return &spec, nil
}

func processDiagnostic(d *Diagnostic, spec *ScalaCompileSpec) {
	switch d.Severity {
	case "ERROR":
		processErrorDiagnostic(d, spec)
	default:
		return
	}
}

func processErrorDiagnostic(d *Diagnostic, spec *ScalaCompileSpec) {
	if strings.HasPrefix(d.Message, NOT_FOUND) {
		processNotFoundErrorDiagnostic(d.Message[len(NOT_FOUND):], spec)
	} else if match := notPackageMemberRe.FindStringSubmatch(d.Message); match != nil {
		processNotPackageMemberErrorDiagnostic(match[1], match[2], spec)
	}
}

func processNotFoundErrorDiagnostic(msg string, spec *ScalaCompileSpec) {
	fields := strings.Fields(msg)
	if len(fields) < 2 {
		return
	}
	for _, sym := range spec.NotFound {
		if sym.Kind == fields[0] && sym.Name == fields[1] {
			return
		}
	}
	spec.NotFound = append(spec.NotFound, &NotFoundSymbol{Kind: fields[0], Name: fields[1]})
}

func processNotPackageMemberErrorDiagnostic(obj, pkg string, spec *ScalaCompileSpec) {
	spec.NotMember = append(spec.NotMember, &NotMemberSymbol{Kind: "object", Name: obj, Package: pkg})
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
