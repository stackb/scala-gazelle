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

    public static void main(String[] args) throws IOException, InterruptedException, ParserConfigurationException {
        final DocumentBuilderFactory factory = DocumentBuilderFactory.newInstance();
        final DocumentBuilder builder = factory.newDocumentBuilder();

        Integer port = Integer.getInteger(PORT_NAME, DEFAULT_PORT);

        HttpServer server = HttpServer.create(new InetSocketAddress(port), 0);
        server.createContext("/", new GlobalHandler(builder));
        server.setExecutor(null);
        server.start();

        System.out.format("Scalac server listening on port %d: hit any key to exit...\n", port);
        // Thread.currentThread().join();

        System.in.read();
        server.stop(0);
        System.out.format("Scalac server stopped.\n");
    }

    static class GlobalHandler implements HttpHandler {
        final DocumentBuilder builder;

        GlobalHandler(DocumentBuilder builder) {
            this.builder = builder;
        }

        @Override
        public void handle(HttpExchange t) throws IOException {
            int responseCode = 500;
            String responseBody = "unknown error";

            try {
                List<String> args = new ArrayList<>();
                args.add("-usejavacp");
                args.add("-Ystop-before:refcheck");
                // args.add("-Ystop-before:jvm");
                fromCompileRequest(t.getRequestBody(), args);
                String[] result = new String[args.size()];

                List<XmlReporter.Diagnostic> diagnostics = compile(args.toArray(result));
                Document response = toCompileResponse(diagnostics);

                responseBody = toXml(response);

                responseCode = 200;

            } catch (DOMException | SAXException | IOException domex) {
                responseCode = 400;
                responseBody = domex.getMessage();
            } catch (Exception ex) {
                responseBody = ex.getMessage();
            }

            System.out.println(responseBody);
            t.sendResponseHeaders(responseCode, responseBody.length());
            try (OutputStream os = t.getResponseBody()) {
                os.write(responseBody.getBytes());
            } catch (IOException ioex) {
                ioex.printStackTrace();
            }
        }

        private List<XmlReporter.Diagnostic> compile(String[] args) {
            XmlReportableMainClass main = new XmlReportableMainClass();
            boolean ok = main.process(args);
            return main.reporter.getDiagnostics();
        }

        private void fromCompileRequest(InputStream in, List<String> args)
                throws DOMException, SAXException, IOException {
            // Expecting <compileRequest><file>Foo.scala</file></compileRequest>
            Document doc = builder.parse(in);
            Element root = doc.getDocumentElement();
            if (!root.getTagName().equals("compileRequest")) {
                throw new DOMException(DOMException.HIERARCHY_REQUEST_ERR, "expected root element <compileRequest>");
            }

            NodeList nodeList = root.getChildNodes();
            for (int i = 0; i < nodeList.getLength(); i++) {
                Node node = nodeList.item(i);
                if (node.getNodeType() != Node.ELEMENT_NODE) {
                    continue;
                }
                Element elem = (Element) node;
                switch (elem.getTagName()) {
                    case "file":
                        String name = elem.getTextContent();
                        args.add(name);
                        break;
                    default:
                        throw new DOMException(DOMException.HIERARCHY_REQUEST_ERR, "unexpected element:" + elem);
                }
            }
        }

        private Document toCompileResponse(List<XmlReporter.Diagnostic> diagnostics) {
            Document doc = builder.newDocument();
            Element compileResponse = doc.createElement("compileResponse");
            doc.appendChild(compileResponse);
            for (XmlReporter.Diagnostic diagnostic : diagnostics) {
                Element d = doc.createElement("diagnostic");
                d.setAttribute("sev", diagnostic.sev.toString());
                if (!diagnostic.pos.source().path().equals("<no file>")) {
                    d.setAttribute("source", diagnostic.pos.source().path());
                }
                if (diagnostic.pos.safeLine() != 0) {
                    d.setAttribute("line", Integer.toString(diagnostic.pos.safeLine()));
                }
                d.setTextContent(diagnostic.msg);
                compileResponse.appendChild(d);
            }
            return doc;
        }

        // write doc to output stream
        private static String toXml(Document doc) throws TransformerException {
            TransformerFactory transformerFactory = TransformerFactory.newInstance();
            Transformer transformer = transformerFactory.newTransformer();
            StringWriter writer = new StringWriter();
            transformer.transform(new DOMSource(doc), new StreamResult(writer));
            return writer.getBuffer().toString();
        }
    }
}