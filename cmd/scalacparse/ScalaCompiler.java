package scalaparse;

import java.io.IOException;

public class ScalaCompiler {
    public static void main(String[] args) throws IOException {
        System.out.println("Starting scala compiler.");
        JSONReportableMainClass main = new JSONReportableMainClass();
        boolean ok = main.process(args);

        System.out.println("Hit any key to exit...");
        System.in.read();

        if (main.server != null) {
            main.server.stop(0);
        }
    }
}