package scalacserver;

import static java.nio.file.StandardCopyOption.ATOMIC_MOVE;

import build.stack.gazelle.scala.parse.CompileRequest;
import build.stack.gazelle.scala.parse.CompileResponse;
import build.stack.gazelle.scala.parse.Diagnostic;
import build.stack.gazelle.scala.parse.CompilerGrpc;

import com.google.common.collect.Iterables;

import io.grpc.Server;
import io.grpc.ServerBuilder;
import io.grpc.Status;
import io.grpc.StatusRuntimeException;
// import io.grpc.protobuf.services.ProtoReflectionService;
import io.grpc.stub.StreamObserver;
import java.io.IOException;
import java.io.File;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.SortedSet;
import java.util.concurrent.TimeUnit;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class GrpcServer {
    private static final Logger logger = LoggerFactory.getLogger(GrpcServer.class);
    static final String PORT_NAME = "scalac.server.port";
    static final int DEFAULT_PORT = 8040;

    private final TimeoutHandler timeoutHandler;
    private final Server server;

    /**
     * Create a BuildFileGenerator server using serverBuilder as a base and features
     * as data.
     */
    public GrpcServer(TimeoutHandler timeoutHandler) {
        this.timeoutHandler = timeoutHandler;
        Integer port = Integer.getInteger(PORT_NAME, DEFAULT_PORT);

        ServerBuilder serverBuilder = ServerBuilder.forPort(port);
        this.server = serverBuilder
                .addService(new GrpcService(timeoutHandler))
                // .addService(new LifecycleService())
                // .addService(ProtoReflectionService.newInstance())
                .build();
    }

    /** Start serving requests. */
    public void start() throws IOException {
        server.start();

        logger.info("Server started, listening on {}", server.getPort());
        Runtime.getRuntime()
                .addShutdownHook(
                        new Thread() {
                            @Override
                            public void run() {
                                timeoutHandler.cancelOutstanding();
                                try {
                                    GrpcServer.this.stop();
                                } catch (InterruptedException e) {
                                    e.printStackTrace(System.err);
                                }
                            }
                        });
    }

    /** Stop serving requests and shutdown resources. */
    public void stop() throws InterruptedException {
        server.shutdownNow().awaitTermination(5, TimeUnit.SECONDS);
    }

    /**
     * Await termination on the main thread since the grpc library uses daemon
     * threads.
     */
    public void blockUntilShutdown() throws InterruptedException {
        server.awaitTermination();
    }

    private static class GrpcService extends CompilerGrpc.CompilerImplBase {
        private final TimeoutHandler timeoutHandler;

        GrpcService(TimeoutHandler timeoutHandler) {
            this.timeoutHandler = timeoutHandler;
        }

        @Override
        public void compile(CompileRequest request, StreamObserver<CompileResponse> responseObserver) {
            timeoutHandler.startedRequest();

            try {
                responseObserver.onNext(compileInternal(request.getDir(), request.getFilenamesList()));
                responseObserver.onCompleted();
            } catch (Exception ex) {
                logger.error(
                        "Got Exception compiling {}: {}", request, ex.getMessage());
                responseObserver.onError(ex);
                responseObserver.onCompleted();
            } finally {
                timeoutHandler.finishedRequest();
            }
        }

        private CompileResponse compileInternal(String dir, List<String> files) {
            List<String> args = new ArrayList<>();
            args.add("-usejavacp");
            args.add("-Ystop-before:refcheck");

            for (String file : files) {
                if (dir.length() > 0) {
                    file = dir + File.separatorChar + file;
                }
                args.add(file);
            }

            String[] result = new String[args.size()];
            List<Diagnostic> diagnostics = compileArgs(args.toArray(result));

            return CompileResponse.newBuilder()
                    .addAllDiagnostics(diagnostics)
                    .build();
        }

        private List<Diagnostic> compileArgs(String[] args) {
            boolean debug = false;
            DiagnosticReportableMainClass main = new DiagnosticReportableMainClass(debug);
            boolean ok = main.process(args);
            return main.reporter.getDiagnostics();
        }

        // private Diagnostic createDiagnostic(CompileResponse.Builder builder,
        // List<XmlReporter.Diagnostic> diagnostics) {
        // for (XmlReporter.Diagnostic diagnostic : diagnostics) {

        // d.setAttribute("sev", diagnostic.sev.toString());
        // if (!diagnostic.pos.source().path().equals("<no file>")) {
        // d.setAttribute("source", diagnostic.pos.source().path());
        // }
        // if (diagnostic.pos.safeLine() != 0) {
        // d.setAttribute("line", Integer.toString(diagnostic.pos.safeLine()));
        // }
        // d.setTextContent(diagnostic.msg);
        // compileResponse.appendChild(d);
        // }

        // }

        // private void writeDiagnostics(CompileResponse.Builder builder,
        // List<XmlReporter.Diagnostic> diagnostics) {
        // for (XmlReporter.Diagnostic diagnostic : diagnostics) {

        // d.setAttribute("sev", diagnostic.sev.toString());
        // if (!diagnostic.pos.source().path().equals("<no file>")) {
        // d.setAttribute("source", diagnostic.pos.source().path());
        // }
        // if (diagnostic.pos.safeLine() != 0) {
        // d.setAttribute("line", Integer.toString(diagnostic.pos.safeLine()));
        // }
        // d.setTextContent(diagnostic.msg);
        // compileResponse.appendChild(d);
        // }

        // }

    }

    // private final Path workspace;

    // @Override
    // public void parsePackage(
    // ParsePackageRequest request, StreamObserver<Package> responseObserver) {
    // timeoutHandler.startedRequest();

    // try {
    // responseObserver.onNext(getImports(request));
    // responseObserver.onCompleted();
    // } catch (Exception ex) {
    // logger.error(
    // "Got Exception parsing package {}: {}", Paths.get(request.getRel()),
    // ex.getMessage());
    // responseObserver.onError(ex);
    // responseObserver.onCompleted();
    // } finally {
    // timeoutHandler.finishedRequest();
    // }
    // }

    // private Package getImports(ParsePackageRequest request) {
    // List<String> files = new ArrayList<>();
    // for (int i = 0; i < request.getFilesCount(); i++) {
    // files.add(request.getFiles(i));
    // }
    // logger.debug("Working relative directory: {}", request.getRel());
    // logger.debug("processing files: {}", files);

    // ClasspathParser parser = new ClasspathParser();
    // Path directory = workspace.resolve(request.getRel());

    // try {
    // parser.parseClasses(directory, files);
    // } catch (IOException exception) {
    // // If we fail to process a directory, which can happen with the module level
    // // processing
    // // or can't parse any of the files, just return an empty response.
    // return Package.newBuilder().setName("").build();
    // }
    // Set<String> packages = parser.getPackages();
    // if (packages.size() > 1) {
    // logger.error(
    // "Set of classes in {} should have only one package, instead is: {}",
    // request.getRel(),
    // packages);
    // throw new StatusRuntimeException(Status.INVALID_ARGUMENT);
    // } else if (packages.isEmpty()) {
    // logger.info(
    // "Set of classes in {} has no package",
    // Paths.get(request.getRel()).toAbsolutePath());
    // packages.add("");
    // }
    // logger.debug("Got package: {}", Iterables.getOnlyElement(packages));
    // logger.debug("Got used types: {}", parser.getUsedTypes());

    // Builder packageBuilder = Package.newBuilder()
    // .setName(Iterables.getOnlyElement(packages))
    // .addAllImports(parser.getUsedTypes())
    // .addAllMains(parser.getMainClasses());
    // for (Map.Entry<String, SortedSet<String>> annotations :
    // parser.getAnnotatedClasses().entrySet()) {
    // packageBuilder.putPerClassMetadata(
    // annotations.getKey(),
    // PerClassMetadata.newBuilder()
    // .addAllAnnotationClassNames(annotations.getValue())
    // .build());
    // }
    // return packageBuilder.build();
    // }
    // }
}