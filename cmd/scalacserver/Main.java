package scalacserver;

import java.io.IOException;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.concurrent.ScheduledThreadPoolExecutor;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class Main {
    private static final Logger logger = LoggerFactory.getLogger(Main.class);

    public static void main(String[] args) throws IOException, InterruptedException {
        Main main = new Main();

        TimeoutHandler timeoutHander = new TimeoutHandler(new ScheduledThreadPoolExecutor(1), main.idleTimeout());
        main.runServer(timeoutHander);
    }

    public void runServer(TimeoutHandler timeoutHandler) throws InterruptedException, IOException {
        GrpcServer gRPCServer = new GrpcServer(timeoutHandler);
        gRPCServer.start();
        gRPCServer.blockUntilShutdown();
    }

    // <=0 means don't timeout.
    private int idleTimeout() {
        return 0;
        // return line.hasOption("idle-timeout")
        // ? Integer.decode(line.getOptionValue("idle-timeout"))
        // : -1;
    }
}
