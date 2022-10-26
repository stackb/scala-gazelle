package scalaparse

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/amenzhinsky/go-memexec"
	"github.com/bazelbuild/rules_go/go/tools/bazel"

	sppb "github.com/stackb/scala-gazelle/api/scalaparse"
)

const contentTypeJSON = "application/json"

type ScalaParseClient struct {
}

func NewScalaParseServer() *ScalaParseServer {
	return &ScalaParseServer{}
}

type ScalaParseServer struct {
	sppb.UnimplementedScalaParserServer

	process    *memexec.Exec
	processDir string
	cmd        *exec.Cmd

	grpcServer *grpc.Server

	httpClient *http.Client
	httpUrl    string

	HttpPort int
}

func (s *ScalaParseServer) Stop() {
	log.Println("stopping server")
	if s.httpClient != nil {
		log.Println("closing idle http connections")
		s.httpClient.CloseIdleConnections()
		s.httpClient = nil
	}
	if s.cmd != nil {
		s.cmd.Process.Kill()
		s.cmd = nil
	}

	if s.process != nil {
		log.Println("stopping server process")
		s.process.Close()
		s.process = nil
	}
	if s.processDir != "" {
		log.Println("cleaning temp processDir", s.processDir)
		os.RemoveAll(s.processDir)
		s.processDir = ""
	}
}

func (s *ScalaParseServer) Start() error {
	//
	// Setup temp process directory and write js files
	//
	processDir, err := bazel.NewTmpDir("")
	if err != nil {
		return err
	}

	scriptPath := filepath.Join(processDir, "sourceindexer.mjs")
	parserPath := filepath.Join(processDir, "node_modules", "scalameta-parsers", "index.js")

	if err := os.MkdirAll(filepath.Dir(parserPath), os.ModePerm); err != nil {
		return err
	}
	if err := ioutil.WriteFile(scriptPath, []byte(sourceindexerMjs), os.ModePerm); err != nil {
		return err
	}
	if err := ioutil.WriteFile(parserPath, []byte(scalametaParsersIndexJs), os.ModePerm); err != nil {
		return err
	}

	if debugParse {
		listFiles(".")
	}

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
	// Setup the bun process
	//
	exe, err := memexec.New(bunExe)
	if err != nil {
		return err
	}
	s.process = exe

	//
	// Start the bun process
	//
	cmd := exe.Command("./sourceindexer.mjs")
	cmd.Dir = processDir
	cmd.Env = []string{
		"NODE_PATH=" + processDir,
		fmt.Sprintf("PORT=%d", s.HttpPort),
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	s.cmd = cmd

	log.Println("starting process")
	if err := cmd.Start(); err != nil {
		return err
	}

	go func() {
		log.Println("waiting on process")
		if err := cmd.Wait(); err != nil {
			log.Printf("command wait err: %v", err)
		}
	}()

	if true {
		//
		// Wait for connection to become available
		//
		var wg sync.WaitGroup
		wg.Add(1)
		then := time.Now()

		go func() {
			defer wg.Done()
			for {
				log.Printf("checking connection available for %s...", s.httpUrl)

				_, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", s.HttpPort))
				if err == nil {
					log.Printf("%s is available after %s", s.httpUrl, time.Since(then))
					break
				}
				time.Sleep(time.Second)
			}
		}()

		wg.Wait()
	} else {
		time.Sleep(time.Second)
	}

	//
	// Setup the http client
	//
	s.httpClient = &http.Client{
		Timeout: 10 * time.Second,
		// Transport: &http.Transport{
		// 	Dial: (&net.Dialer{
		// 		Timeout: 5 * time.Second,
		// 	}).Dial,
		// 	TLSHandshakeTimeout: 5 * time.Second,
		// },
	}

	// time.Sleep(1 * time.Second)
	return nil
}

func (s *ScalaParseServer) Parse(ctx context.Context, in *sppb.ScalaParseRequest) (*sppb.ScalaParseResponse, error) {
	if false {
		return nil, status.Error(codes.Unimplemented, "not implemented")
	}

	req, err := newHttpScalaParseRequest(s.httpUrl, in)
	if err != nil {
		return nil, err
	}
	w, err := s.httpClient.Do(req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "response error: %v", err)
	}

	respDump, err := httputil.DumpResponse(w, true)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("RESPONSE:\n%s", string(respDump))

	contentType := w.Header.Get("Content-Type")
	if contentType != contentTypeJSON {
		return nil, status.Errorf(codes.Internal, "response content-type error, want %q, got: %q", contentTypeJSON, contentType)
	}

	data, err := ioutil.ReadAll(w.Body)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "response data error: %v", err)
	}

	log.Printf("response body: %q", string(data))
	var response sppb.ScalaParseResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, status.Errorf(codes.Internal, "response body error: %v", err)
	}

	return &response, nil
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

func newHttpScalaParseRequest(url string, in *sppb.ScalaParseRequest) (*http.Request, error) {
	if url == "" {
		return nil, status.Errorf(codes.InvalidArgument, "request URL is required")
	}
	if in == nil {
		return nil, status.Errorf(codes.InvalidArgument, "ScalaParseRequest is required")
	}
	values := map[string]interface{}{"label": in.Label, "files": in.Filename}
	jsonValue, err := json.Marshal(values)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "marshaling request: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonValue))
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	return req, nil
}
