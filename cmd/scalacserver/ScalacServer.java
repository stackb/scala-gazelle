package scalacserver;

import com.sun.net.httpserver.HttpExchange;
import com.sun.net.httpserver.HttpHandler;
import com.sun.net.httpserver.HttpServer;

import java.io.InputStream;
import java.io.OutputStream;
import java.io.StringWriter;
import java.io.IOException;
import java.net.InetSocketAddress;
import java.util.List;
import java.util.ArrayList;

import javax.xml.parsers.DocumentBuilder;
import javax.xml.parsers.DocumentBuilderFactory;
import javax.xml.parsers.ParserConfigurationException;
import javax.xml.transform.Transformer;
import javax.xml.transform.TransformerException;
import javax.xml.transform.TransformerFactory;
import javax.xml.transform.dom.DOMSource;
import javax.xml.transform.stream.StreamResult;

import org.w3c.dom.DOMException;
import org.w3c.dom.Document;
import org.w3c.dom.Element;
import org.w3c.dom.Node;
import org.w3c.dom.NodeList;
import org.xml.sax.SAXException;

import scala.tools.nsc.Global;
import scala.tools.nsc.Global.Run;
import scala.tools.nsc.MainClass;
import scala.tools.nsc.reporters.Reporter;
import scala.tools.nsc.Settings;

/**
 * ScalacServer implements an xml-over-http interface to the scala compiler. Why
 * http? Because using stdin/stdout turned out to be hard (compiler did not seem
 * to respect only using stdout vs stderr, perhaps it was just me). Why XML?
 * Because java does not have a JSON parser in the stdlib and did not want an
 * additional dependency. Why this old-school com.sun.net.httpserver.HttpServer?
 * Same reason, to avoid an external dependency.
 *
 * Use -Dscalac.server.port=12345 to override the http port.
 */
public class ScalacServer {
    static final String PORT_NAME = "scalac.server.port";
    static final int DEFAULT_PORT = 8040;
    static final boolean DEBUG = false;

    public static void main(String[] args) throws IOException, InterruptedException, ParserConfigurationException {
        final DocumentBuilderFactory factory = DocumentBuilderFactory.newInstance();
        final DocumentBuilder builder = factory.newDocumentBuilder();

        Integer port = Integer.getInteger(PORT_NAME, DEFAULT_PORT);

        HttpServer server = HttpServer.create(new InetSocketAddress(port), 0);
        server.createContext("/", new ScalacServerHandler(builder));
        server.setExecutor(null);
        server.start();

        if (DEBUG) {
            System.out.format("Scalac server listening on port %d: hit any key to exit...\n", port);
        }
        // Thread.currentThread().join();

        System.in.read();
        server.stop(0);

        if (DEBUG) {
            System.out.format("Scalac server stopped.\n");
        }
    }
}