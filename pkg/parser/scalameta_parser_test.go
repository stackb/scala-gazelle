package parser

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/testtools"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
	"github.com/stackb/scala-gazelle/pkg/bazel"
	"github.com/stackb/scala-gazelle/pkg/collections"
)

var update = flag.Bool("update", false, "update golden files")

func TestServerParse(t *testing.T) {
	server := NewScalametaParser()
	if err := server.Start(); err != nil {
		t.Fatal("server start:", err)
	}
	defer server.Stop()

	for name, tc := range map[string]*struct {
		files []testtools.FileSpec
		in    sppb.ParseRequest
		want  sppb.ParseResponse
	}{
		"degenerate": {
			want: sppb.ParseResponse{
				Error: `bad request: expected '{ "filenames": [LIST OF FILES TO PARSE] }', but filenames list was not present`,
			},
		},
		"single file": {
			files: []testtools.FileSpec{
				{
					Path: "A.scala",
					Content: `package a
import java.util.HashMap

class Foo extends HashMap {
}
`,
				},
			},
			want: sppb.ParseResponse{
				Files: []*sppb.File{
					{
						Filename: "A.scala",
						Packages: []string{"a"},
						Classes:  []string{"a.Foo"},
						Imports:  []string{"java.util.HashMap"},
						Extends: map[string]*sppb.ClassList{
							"class a.Foo": {
								Classes: []string{"java.util.HashMap"},
							},
						},
						Names: []string{"Foo", "HashMap"},
					},
				},
			},
		},
		"nested import": {
			files: []testtools.FileSpec{
				{
					Path: "Example.scala",
					Content: `
package example

import com.typesafe.scalalogging.LazyLogging
import corp.common.core.vm.utils.ArgProcessor

object Main extends LazyLogging {
	def main(args: Array[String]): Unit = {
	import corp.common.core.reports.DotFormatReport
	ArgProcessor.process(args)
	logger.info(DotFormatReport(new BlendTestService).dotForm())
	}
}
`,
				},
			},
			want: sppb.ParseResponse{
				Files: []*sppb.File{
					{
						Filename: "Example.scala",
						Packages: []string{"example"},
						Objects:  []string{"example.Main"},
						Imports: []string{
							"com.typesafe.scalalogging.LazyLogging",
							"corp.common.core.reports.DotFormatReport",
							"corp.common.core.vm.utils.ArgProcessor",
						},
						Extends: map[string]*sppb.ClassList{
							"object example.Main": {
								Classes: []string{"com.typesafe.scalalogging.LazyLogging"},
							},
						},
						Names: []string{
							"ArgProcessor",
							"ArgProcessor.process",
							"Array",
							"BlendTestService",
							"DotFormatReport",
							"LazyLogging",
							"Main",
							"String",
							"Unit",
						},
					},
				},
			},
		},
		"extends with": {
			files: []testtools.FileSpec{
				{
					Path: "FooTest.scala",
					Content: `
package foo.test

import org.scalatest.{FlatSpec, Matchers}
import java.time.{LocalDate, LocalTime}

class FooTest extends FlatSpec with Matchers {
}
`,
				},
			},
			want: sppb.ParseResponse{
				Files: []*sppb.File{
					{
						Filename: "FooTest.scala",
						Packages: []string{"foo.test"},
						Classes:  []string{"foo.test.FooTest"},
						Imports: []string{
							"java.time.LocalDate",
							"java.time.LocalTime",
							"org.scalatest.FlatSpec",
							"org.scalatest.Matchers",
						},
						Extends: map[string]*sppb.ClassList{
							"class foo.test.FooTest": {
								Classes: []string{
									"org.scalatest.FlatSpec",
									"org.scalatest.Matchers",
								},
							},
						},
						Names: []string{
							"FlatSpec",
							"FooTest",
							"Matchers",
						},
					},
				},
			},
		},
		"nested import rename": {
			files: []testtools.FileSpec{
				{
					Path: "Palette.scala",
					Content: `
package color

import java.awt.Color

object Palette {
  val random100: MandelPalette = {
    import scala.util.Random.{nextInt => rint}
    Palette(100, Seq.tabulate[Color](100)(_ => new Color(rint(255), rint(255), rint(255))).toArray)
  }
}
`,
				},
			},
			want: sppb.ParseResponse{
				Files: []*sppb.File{
					{
						Filename: "Palette.scala",
						Packages: []string{"color"},
						Objects:  []string{"color.Palette"},
						Imports: []string{
							"java.awt.Color",
							"scala.util.Random.nextInt",
						},
						Names: []string{
							"Color",
							"MandelPalette",
							"Palette",
							"Seq",
							"Seq.tabulate",
						},
					},
				},
			},
		},
		"nested import same package": {
			files: []testtools.FileSpec{
				{
					Path: "Main.scala",
					Content: `
package example

import akka.actor.ActorSystem

object MainContext {
	implicit var asys: ActorSystem = _
}

object Main {
	private def makeRequest(params: Map[String, String]): Unit = {
		import MainContext._
	}	
}
`,
				},
			},
			want: sppb.ParseResponse{
				Files: []*sppb.File{
					{
						Filename: "Main.scala",
						Packages: []string{"example"},
						Objects: []string{
							"example.Main",
							"example.MainContext",
						},
						Imports: []string{
							"MainContext._",
							"akka.actor.ActorSystem",
						},
						Names: []string{
							"ActorSystem",
							"Main",
							"MainContext",
							"Map",
							"String",
							"Unit",
						},
					},
				},
			},
		},
		"skips parameter names": {
			files: []testtools.FileSpec{
				{
					Path: "Main.scala",
					Content: `
package example

class Main {
	override def sayHello(request: HelloHopRequest, meta: Metadata): Future[HelloReply] = {
	actions.instrumented(meta, request, "sayHello") { implicit tracing =>
		val name = request.name
		// Use the invoke method on ClientTracingInvoker that will propagate extracted headers from parent Ingress request
		// Send one async request this way.
		tracing.invoke(client.sayHello(), HelloRequest(name))
	}
	}
}
`,
				},
			},
			want: sppb.ParseResponse{
				Files: []*sppb.File{
					{
						Filename: "Main.scala",
						Packages: []string{"example"},
						Classes: []string{
							"example.Main",
						},
						Names: []string{
							"Future",
							"HelloHopRequest",
							"HelloReply",
							"HelloRequest",
							"Main",
							"Metadata",
						},
					},
				},
			},
		},
		"peerlink": {
			files: []testtools.FileSpec{
				{
					Path: "Main.scala",
					Content: `
package corp.core.linkage

import corp.core.linkage.Transport.Response
import corp.core.vm.HostComms
import corp.core.vm.HostComms.PeerPrefixPath
import corp.core.vm.utils.{DaemonThread, Startable}
import corp.core.{HostSignal, Location, Path}
import java.io.{ObjectInputStream, ObjectOutputStream}
import java.net.{ConnectException, InetSocketAddress, ServerSocket, Socket, SocketTimeoutException}
import java.util.concurrent.ConcurrentHashMap
import java.util.{Timer, TimerTask}

import com.typesafe.scalalogging.LazyLogging

/** The PeerLink module looks after peer connections and communications
	*/
object PeerLink extends LazyLogging {
	case class ConnectionRequest(peerTag: String, peerHost: String, peerPort: Int)
	case class ConnectionResponse(hostTag: String, host: String, hostPort: Int)
	case class PeerAlreadyKnown(peerTag: String)

	sealed trait Peer
	case object NoPeer extends Peer
	case object PendingOutPeer extends Peer
	case object PendingInPeer extends Peer

	case class PeerConnection(
		peerTag: String,
		peerHost: String,
		peerPort: Int,
		location: Location,
		in: ObjectInputStream,
		out: ObjectOutputStream)
		extends Peer

	/** The PeerLink acts as a coordinator and repository for external peer connections.<p>
	*
	* The PeerLink through its ConnectionHandlers (one per connection) forwards inbound messages
	*
	* @param port      The port to use for the ServerSocket
	* @param transport The Related Transport which connects the internal comms to the PeerLink entities
	* @param continue  An external predicate to allow full shutdown to be decided for the PeerLink entities
	* @param deliver   An external callback for messages
	*/
	case class PeerLink(port: Int, transport: SimpleTransport, continue: Unit => Boolean, deliver: AnyRef => Unit) {
	// TODO: Implement
	def send(origin: Location, dest: Location, msg: AnyRef): Response = ???
	// TODO: Implement
	def sendHostSignal(origin: Location, signal: HostSignal): Response = ???

	// scalastyle:off
	val peers: ConcurrentHashMap[(String, Int), Peer] = new ConcurrentHashMap()
	val timer: Timer = new Timer("PeerLinkTimer", true)
	// scalastyle:on

	private def listPeers(): List[String] = {
		import scala.jdk.CollectionConverters._
		peers
		.values()
		.asScala
		.collect { case p: PeerConnection => p.peerTag + "@" + p.peerHost + ":" + p.peerPort }
		.toList
	}

}
}
					
					`,
				},
			},
			want: sppb.ParseResponse{
				Files: []*sppb.File{
					{
						Filename: "Main.scala",
						Imports: []string{
							"com.typesafe.scalalogging.LazyLogging",
							"corp.core.HostSignal",
							"corp.core.Location",
							"corp.core.Path",
							"corp.core.linkage.Transport.Response",
							"corp.core.vm.HostComms",
							"corp.core.vm.HostComms.PeerPrefixPath",
							"corp.core.vm.utils.DaemonThread",
							"corp.core.vm.utils.Startable",
							"java.io.ObjectInputStream",
							"java.io.ObjectOutputStream",
							"java.net.ConnectException",
							"java.net.InetSocketAddress",
							"java.net.ServerSocket",
							"java.net.Socket",
							"java.net.SocketTimeoutException",
							"java.util.Timer",
							"java.util.TimerTask",
							"java.util.concurrent.ConcurrentHashMap",
							"scala.jdk.CollectionConverters._",
						},
						Objects:  []string{"corp.core.linkage.PeerLink"},
						Packages: []string{"corp.core.linkage"},
						Extends: map[string]*parse.ClassList{
							"object corp.core.linkage.PeerLink": {
								Classes: []string{
									"com.typesafe.scalalogging.LazyLogging",
								},
							},
						},
						Names: []string{
							"AnyRef",
							"Boolean",
							"ConcurrentHashMap",
							"ConnectionRequest",
							"ConnectionResponse",
							"HostSignal",
							"Int",
							"LazyLogging",
							"List",
							"Location",
							"NoPeer",
							"ObjectInputStream",
							"ObjectOutputStream",
							"Peer",
							"PeerAlreadyKnown",
							"PeerConnection",
							"PeerLink",
							"PendingInPeer",
							"PendingOutPeer",
							"Response",
							"SimpleTransport",
							"String",
							"Timer",
							"Unit",
						},
					},
				},
			},
		},
		"not-scalameta": {
			files: []testtools.FileSpec{
				{
					Path: "Main.scala",
					Content: `
package trumid.common.truscale.core

object MetaData {
	val NoMetaData: MetaData = MetaData(Map())
	val NoMD: MetaData = NoMetaData

	val AckResponse = "AckResponse"
	val UnexpectedResponse = "UnexpectedResponse"

	// Standard keys
	val FailureReason = "FailureReason"
}

case class MetaData(meta: Map[String, Any] = Map()) {
	def keys: Iterable[String] = meta.keys
	def this(k: String, v: Any) = this(Map(k -> v))
	def update(key: String, value: Any): MetaData = MetaData(meta.updated(key, value))
	def apply(key: String): Any = meta(key)
}					
					`,
				},
			},
			want: sppb.ParseResponse{
				Files: []*sppb.File{
					{
						Filename: "Main.scala",
						Classes:  []string{"trumid.common.truscale.core.MetaData"},
						Objects:  []string{"trumid.common.truscale.core.MetaData"},
						Packages: []string{"trumid.common.truscale.core"},
						Names: []string{
							"AckResponse",
							"Any",
							"FailureReason",
							"Iterable",
							"Map",
							"MetaData",
							"NoMD",
							"NoMetaData",
							"String",
							"UnexpectedResponse",
						},
					},
				},
			},
		},
		"parameter-type-names": {
			files: []testtools.FileSpec{
				{
					Path: "Main.scala",
					Content: `
package gum.entity

case class CounterpartyUser(user: UserRecord[ActorId, EntityId])
	extends UserComposition[ActorId, EntityId]
	with HasIsOpsAdmin[ActorId, EntityId] {

	def toProto(userNameProvider: Int => Option[String], counterparty: Counterparty): UserProto = {
	}
}
					`,
				},
			},
			want: sppb.ParseResponse{
				Files: []*sppb.File{
					{
						Filename: "Main.scala",
						Packages: []string{"gum.entity"},
						Classes:  []string{"gum.entity.CounterpartyUser"},
						Names: []string{
							"ActorId",
							"Counterparty",
							"CounterpartyUser",
							"EntityId",
							"HasIsOpsAdmin",
							"Int",
							"Option",
							"String",
							"UserComposition",
							"UserProto",
							"UserRecord",
						},
						Extends: map[string]*parse.ClassList{
							"class gum.entity.CounterpartyUser": {
								Classes: []string{
									"UserComposition",
									"HasIsOpsAdmin",
								},
							},
						},
					},
				},
			},
		},
		"option-parameter-type-name": {
			files: []testtools.FileSpec{
				{
					Path: "Main.scala",
					Content: `
case class TradingAccountUser(
	preferences: Option[TradingAccountUserPreferences] = None,
)			
					`,
				},
			},
			want: sppb.ParseResponse{
				Files: []*sppb.File{
					{
						Filename: "Main.scala",
						Classes:  []string{"TradingAccountUser"},
						Names: []string{
							"None",
							"Option",
							"TradingAccountUser",
							"TradingAccountUserPreferences",
						},
					},
				},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			tmpDir, err := bazel.NewTmpDir("")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			files := mustWriteTestFiles(t, tmpDir, tc.files)
			tc.in.Filenames = files

			got, err := server.Parse(context.Background(), &tc.in)
			if err != nil {
				t.Fatal(err)
			}
			got.ElapsedMillis = 0

			// remove tmpdir prefix and zero the time delta for diff comparison
			for i := range got.Files {
				if strings.HasPrefix(got.Files[i].Filename, tmpDir) {
					got.Files[i].Filename = got.Files[i].Filename[len(tmpDir)+1:]
				}
			}

			if diff := cmp.Diff(&tc.want, got, cmpopts.IgnoreUnexported(
				sppb.ParseResponse{},
				sppb.File{},
				sppb.ClassList{},
			)); diff != "" {
				t.Errorf(".Parse (-want +got):\n%s", diff)
			}
		})
	}
}

func TestScalaParseTree(t *testing.T) {
	rel := "pkg/parser"
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	if bwd, ok := os.LookupEnv("BUILD_WORKSPACE_DIRECTORY"); ok {
		dir = filepath.Join(bwd, rel)
	}
	t.Log("dir:", dir)

	srcs, err := collections.CollectFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("srcs:", srcs)

	server := NewScalametaParser()
	if err := server.Start(); err != nil {
		t.Fatal("server start:", err)
	}
	defer server.Stop()

	for _, src := range srcs {
		if filepath.Ext(src) != ".scala" {
			continue
		}
		t.Run(src, func(t *testing.T) {
			goldenFile := filepath.Join(dir, src+".golden.json")
			response, err := server.Parse(context.Background(), &sppb.ParseRequest{
				Filenames:     []string{filepath.Join(dir, src)},
				WantParseTree: true,
			})
			if err != nil {
				t.Fatal(err)
			}
			got := response.Files[0].Tree

			if *update {
				if err := os.WriteFile(goldenFile, []byte(got), os.ModePerm); err != nil {
					t.Fatal(err)
				}
				log.Println("Wrote golden file:", goldenFile)
				return
			}

			var want string
			if data, err := os.ReadFile(goldenFile); err != nil {
				t.Fatal(err)
			} else {
				want = string(data)
			}

			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetFreePort(t *testing.T) {
	got, err := getFreePort()
	if err != nil {
		t.Fatal(err)
	}
	if got == 0 {
		t.Error("expected non-zero port number")
	}
}

func TestNewHttpScalaParseRequest(t *testing.T) {
	for name, tc := range map[string]struct {
		url      string
		in       *sppb.ParseRequest
		want     *http.Request
		wantBody string
	}{
		"prototypical": {
			url: "http://localhost:3000",
			in: &sppb.ParseRequest{
				Filenames: []string{"A.scala", "B.scala"},
			},
			want: &http.Request{
				Method:        "POST",
				URL:           mustParseURL(t, "http://localhost:3000"),
				Proto:         "HTTP/1.1",
				ProtoMajor:    1,
				ProtoMinor:    1,
				Header:        http.Header{"Content-Type": {"application/json"}},
				ContentLength: 36, // or 35, see below!
				Host:          "localhost:3000",
			},
			wantBody: `{"filenames":["A.scala","B.scala"]}`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := newHttpParseRequest(tc.url, tc.in)
			if err != nil {
				t.Fatal(err)
			}
			body, err := io.ReadAll(got.Body)
			if err != nil {
				t.Fatal(err)
			}
			// remove all whitespace (and ignore content length) for the test:
			// seeing CI failures between macos (M1) and linux.  Very strange!
			gotBody := strings.ReplaceAll(string(body), " ", "")
			if diff := cmp.Diff(tc.want, got,
				cmpopts.IgnoreUnexported(http.Request{}),
				cmpopts.IgnoreFields(http.Request{}, "GetBody", "Body", "ContentLength"),
			); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantBody, gotBody); diff != "" {
				t.Errorf("body (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewHttpParseRequestError(t *testing.T) {
	for name, tc := range map[string]struct {
		url  string
		in   *sppb.ParseRequest
		want error
	}{
		"missing-url": {
			want: fmt.Errorf("rpc error: code = InvalidArgument desc = request URL is required"),
		},
		"missing-request": {
			url:  "http://localhost:3000",
			want: fmt.Errorf("rpc error: code = InvalidArgument desc = ParseRequest is required"),
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, got := newHttpParseRequest(tc.url, tc.in)
			if got == nil {
				t.Fatalf("error was expected: %v", tc.want)
			}
			if diff := cmp.Diff(tc.want.Error(), got.Error()); diff != "" {
				t.Errorf("newHttpScalaParseRequest error (-want +got):\n%s", diff)
			}
		})
	}
}

func mustParseURL(t *testing.T, raw string) *url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("url parse error: %v", err)
	}
	return u
}

func mustWriteTestFiles(t *testing.T, tmpDir string, files []testtools.FileSpec) []string {
	var filenames []string
	for _, file := range files {
		abs := filepath.Join(tmpDir, file.Path)
		dir := filepath.Dir(abs)
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			t.Fatal(err)
		}
		if !file.NotExist {
			if err := os.WriteFile(abs, []byte(file.Content), os.ModePerm); err != nil {
				t.Fatal(err)
			}
		}
		filenames = append(filenames, abs)
	}
	return filenames
}
