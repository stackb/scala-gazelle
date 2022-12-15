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
                .build();
    }

    /** Start serving requests. */
    public void start() throws IOException {
        server.start();

        logger.debug("Server started, listening on {}", server.getPort());
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
            List<Diagnostic> diagnostics = compileArgs(dir, args.toArray(result));

            return CompileResponse.newBuilder()
                    .addAllDiagnostics(diagnostics)
                    .build();
        }

        private List<Diagnostic> compileArgs(String dir, String[] args) {
            DiagnosticReportableMainClass main = new DiagnosticReportableMainClass(dir);
            boolean ok = main.process(args);
            return main.reporter.getDiagnostics();
        }
    }
}