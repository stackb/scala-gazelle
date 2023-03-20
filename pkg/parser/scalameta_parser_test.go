package parser

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/testtools"
	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
)

func TestServerParse(t *testing.T) {
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
						Names: []string{"Foo", "HashMap", "a"},
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
							"ArgProcessor.process",
							"LazyLogging",
							"Main",
							"Unit",
							"example",
							"logger.info",
							"main",
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
						Names: []string{"FlatSpec", "FooTest", "Matchers", "foo.test"},
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
							"MandelPalette",
							"Palette",
							"color",
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
							"akka.actor.ActorSystem",
						},
						Names: []string{"ActorSystem", "Main", "MainContext", "Unit", "example", "makeRequest"},
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
							"Main",
							"actions.instrumented",
							"example",
							"sayHello",
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
package trumid.common.truscale.core.linkage

import trumid.common.truscale.core.linkage.Transport.Response
import trumid.common.truscale.core.vm.HostComms
import trumid.common.truscale.core.vm.HostComms.PeerPrefixPath
import trumid.common.truscale.core.vm.utils.{DaemonThread, Startable}
import trumid.common.truscale.core.{HostSignal, Location, Path}
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

	// ---------- Outbound connection requests ----------

	private def newScanTask(startPort: Int, endPort: Int): ScanTask = ScanTask(startPort, endPort)

	case class ScanTask(startPort: Int, endPort: Int) extends TimerTask with LazyLogging {
		override def run(): Unit = {
		logger.info(listPeers.mkString("Active peers: ", ", ", ""))
		for {
			peerPort <- startPort to endPort
			if peerPort != HostComms.hostPort
		} {
			peerConnectionRequest("localhost", peerPort)
		}
		if (continue(())) timer.schedule(newScanTask(startPort, endPort), 15000)
		}
	}

	/** Scan for peers within the port range (startPort - endPort)<p>
		*
		* Ignore any already connected peers.
		*
		* @param startPort The first port in the peer port range (typically 50000)
		* @param endPort   The last port in the peer port range (typically 50010)
		*/
	def scanForPeers(startPort: Int, endPort: Int): Unit = {
		timer.schedule(newScanTask(startPort, endPort), 5000)
	}

	// TODO: Add protection for timeouts

	def peerConnectionRequest(peerHost: String, peerPort: Int): Unit = {
		DaemonThread(autoStart = true)(ConnectionRequestor(peerHost, peerPort, continue, this))
	}

	// TODO: Add protection and Peer removal on failure as appropriate

	/** Contacts a detected peer and sends a connection request, recording details in the peers Map.
		*
		* @param peerHost  The Peer's host
		* @param peerPort  The Peer's port
		* @param continue  A high level external predicate to shut down all PeerLink activity
		* @param peerLink  The parent PeerLink, required for Peer Locations
		*/
	case class ConnectionRequestor(peerHost: String, peerPort: Int, continue: Unit => Boolean, peerLink: PeerLink)
		extends Startable
		with LazyLogging {
		def start(): Unit = {
		peers.computeIfAbsent(
			(peerHost, peerPort),
			_ => PendingOutPeer
		) // Sets a mark to indicate it is working on establishing a Peer
		peers.compute((peerHost, peerPort), (_, peer) => if (peer == PendingOutPeer) connect else peer)
		peers.entrySet().removeIf(e => e.getValue == NoPeer)
		}

		private def connect: Peer = try {
		val soc = new Socket()
		val isa = new InetSocketAddress(peerHost, peerPort)
		soc.connect(isa, 5000)

		val (in, out) = (new ObjectInputStream(soc.getInputStream), new ObjectOutputStream(soc.getOutputStream))
		val conRequest = ConnectionRequest(HostComms.hostIdTag, HostComms.hostName, HostComms.hostPort)
		logger.info("Sending connection request: " + conRequest + " to " + peerHost + ":" + peerPort)
		out.writeObject(conRequest)
		in.readObject match {
			case cr: ConnectionResponse =>
			logger.info("Peer acknowledged: " + cr)
			val peerLocation = Location(Path(PeerPrefixPath, cr.hostTag))
			transport.registerLocation(peerLocation, peerLink)
			PeerConnection(cr.hostTag, cr.host, cr.hostPort, peerLocation, in, out)
			case PeerAlreadyKnown(peerTag) =>
			logger.info(s"PeerLink.ConnectionRequestor: Peer[$peerTag] already known")
			NoPeer
			case x: AnyRef =>
			logger.error(s"PeerLink.ConnecttionRequestor: received an unexpected message = $x")
			NoPeer
			// Comms outbound will access the out stream in the Peer instance
			// Comms inbound will be redirected through the Peer instance from the ConnectionHandler
		}
		} catch {
		case e: ConnectException =>
			NoPeer
		case _: SocketTimeoutException =>
			NoPeer
		// expected, no peer found
		case e: Exception =>
			logger.warn(s"ConnectionRequestor [$peerHost, $peerPort] error: ${e}")
			NoPeer
		}
		peers.entrySet().removeIf(e => e.getValue == NoPeer)
	}

	// ------------------------------ Inbound connection responses ------------------------------

	def startConnectionHandler(): Unit = DaemonThread(autoStart = true) { () =>
		val ss: ServerSocket = new ServerSocket(port)
		while (continue(())) {
		handleConnection(ss.accept)
		}
	}

	def handleConnection(socket: Socket): Unit = {
		logger.info(s"Received peer connection attempt from ${socket.getRemoteSocketAddress}")
		DaemonThread(autoStart = true) { ConnectionHandler(socket, transport, continue, this) }
	}

	// TODO: Add protection and Peer removal on failure as appropriate

	/** Handles ConnectionRequests and subsequently any messages sent from connected peers
		*
		* @param soc         The Socket from the ServerSocket.accept
		* @param transport   The Transport instance to register Peer Locations with
		* @param continue    A high level external predicate to shut down all PeerLink activity
		* @param peerLink    The parent PeerLink so it can be referenced in the Peer Location
		*/
	case class ConnectionHandler(soc: Socket, transport: SimpleTransport, continue: Unit => Boolean, peerLink: PeerLink)
		extends Startable
		with LazyLogging {
		def start(): Unit = {
		val out = new ObjectOutputStream(soc.getOutputStream)
		val in = new ObjectInputStream(soc.getInputStream)
		while (true) {
			in.readObject() match {
			case req: ConnectionRequest =>
				logger.info(s"Peer requested connection from ${req.peerTag} - ${req.peerHost}:${req.peerPort}")
				peers.computeIfAbsent((req.peerHost, req.peerPort), _ => PendingInPeer)
				peers.compute(
				(req.peerHost, req.peerPort),
				(_, peer) =>
					if (peer == PendingInPeer) {
					logger.info(s"Accepted connection ${req.peerTag} ${req.peerHost}, ${req.peerPort}")
					out.writeObject(ConnectionResponse(HostComms.hostIdTag, HostComms.hostName, HostComms.hostPort))
					val peerLocation = Location(Path(PeerPrefixPath, req.peerTag))
					transport.registerLocation(peerLocation, peerLink)
					PeerConnection(req.peerTag, req.peerHost, req.peerPort, peerLocation, in, out)
					} else {
					logger.info(s"Telling the requestor that already had a peer: $peer")
					out.writeObject(PeerAlreadyKnown(req.peerTag))
					peer
					}
				)

			case msg: AnyRef =>
			// TODO: Add forwarding of other messages from connected peers
			}
		}
		logger.warn(s"Connection handler is closing")
		}
	}

	startConnectionHandler()
	logger.info("Connection handler started ...")
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
							"trumid.common.truscale.core.HostSignal",
							"trumid.common.truscale.core.Location",
							"trumid.common.truscale.core.Path",
							"trumid.common.truscale.core.linkage.Transport.Response",
							"trumid.common.truscale.core.vm.HostComms",
							"trumid.common.truscale.core.vm.HostComms.PeerPrefixPath",
							"trumid.common.truscale.core.vm.utils.DaemonThread",
							"trumid.common.truscale.core.vm.utils.Startable",
						},
						Objects:  []string{"trumid.common.truscale.core.linkage.PeerLink"},
						Packages: []string{"trumid.common.truscale.core.linkage"},
						Extends: map[string]*parse.ClassList{
							"object trumid.common.truscale.core.linkage.PeerLink": {
								Classes: []string{
									"com.typesafe.scalalogging.LazyLogging",
								},
							},
						},
						Names: []string{
							"AnyRef",
							"ConcurrentHashMap",
							"ConnectException",
							"ConnectionHandler",
							"ConnectionRequest",
							"ConnectionRequestor",
							"ConnectionResponse",
							"DaemonThread",
							"Exception",
							"InetSocketAddress",
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
							"ScanTask",
							"Socket",
							"SocketTimeoutException",
							"Startable",
							"Timer",
							"TimerTask",
							"Unit",
							"connect",
							"continue",
							"handleConnection",
							"in.readObject",
							"listPeers",
							"logger.error",
							"logger.info",
							"logger.warn",
							"newScanTask",
							"out.writeObject",
							"peerConnectionRequest",
							"peerHost",
							"peerPort",
							"peers.compute",
							"peers.computeIfAbsent",
							"run",
							"scanForPeers",
							"send",
							"sendHostSignal",
							"soc.connect",
							"soc.getInputStream",
							"soc.getOutputStream",
							"start",
							"startConnectionHandler",
							"timer.schedule",
							"transport.registerLocation",
							"trumid.common.truscale.core.linkage",
						},
					},
				},
			},
		},
	} {
		if name != "peerlink" {
			continue
		}
		t.Run(name,
			func(t *testing.T) {
				tmpDir, err := bazel.NewTmpDir("")
				if err != nil {
					t.Fatal(err)
				}
				defer os.RemoveAll(tmpDir)

				files := mustWriteTestFiles(t, tmpDir, tc.files)
				tc.in.Filenames = files

				server := NewScalametaParser()
				if err := server.Start(); err != nil {
					t.Fatal("server start:", err)
				}
				defer server.Stop()

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

				// if diff := cmp.Diff(&tc.want.Files[0].Imports, got.Files[0].Imports, cmpopts.IgnoreUnexported(
				// 	sppb.ParseResponse{},
				// 	sppb.File{},
				// 	sppb.ClassList{},
				// )); diff != "" {
				// 	t.Errorf(".Imports (-want +got):\n%s", diff)
				// }

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
			body, err := ioutil.ReadAll(got.Body)
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
			if err := ioutil.WriteFile(abs, []byte(file.Content), os.ModePerm); err != nil {
				t.Fatal(err)
			}
		}
		filenames = append(filenames, abs)
	}
	return filenames
}
