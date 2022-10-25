package scalaparse

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/amenzhinsky/go-memexec"
	"github.com/bazelbuild/rules_go/go/tools/bazel"

	sppb "github.com/stackb/scala-gazelle/api/scalaparse"
)

type ScalaParseClient struct {
}

type ScalaParseServer struct {
	sppb.UnimplementedScalaParserServer

	process    *memexec.Exec
	processDir string
	grpcServer *grpc.Server

	httpClient *http.Client
	httpUrl    string

	HttpPort int
}

func (s *ScalaParseServer) Stop() {
	if s.process != nil {
		s.process.Close()
		s.process = nil
	}
	if s.processDir != "" {
		os.RemoveAll(s.processDir)
		s.processDir = ""
	}
	if s.httpClient != nil {
		s.httpClient.CloseIdleConnections()
		s.httpClient = nil
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

	scriptPath := filepath.Join(processDir, "sourceindexer.js")
	parserPath := filepath.Join(processDir, "node_modules", "scalameta-parsers", "index.js")

	if err := os.MkdirAll(filepath.Dir(parserPath), os.ModePerm); err != nil {
		return err
	}
	if err := ioutil.WriteFile(scriptPath, []byte(sourceindexerJs), os.ModePerm); err != nil {
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
	cmd := exe.Command("./sourceindexer.js")
	cmd.Dir = processDir
	cmd.Env = []string{
		"NODE_PATH=" + processDir,
		fmt.Sprintf("PORT=%d", s.HttpPort),
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return err
	}

	//
	// Setup the http client
	//
	s.httpClient = &http.Client{
		Timeout: 60 * time.Second,
	}
	s.httpUrl = fmt.Sprintf("http://localhost:%d", s.HttpPort)

	return nil
}

func (s *ScalaParseServer) Parse(in *sppb.ScalaParseRequest) (*sppb.ScalaParseResponse, error) {
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

	data, err := ioutil.ReadAll(w.Body)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "response data error: %v", err)
	}

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
