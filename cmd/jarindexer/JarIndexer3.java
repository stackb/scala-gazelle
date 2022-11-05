import build.stack.scala.gazelle.api.jarindex.Index;
import java.io.FileNotFoundException;
import java.io.FileOutputStream;
import java.io.IOException;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.List;

public class JarIndexer3 {

    public static void main(String[] args) throws FileNotFoundException, IOException {
        if (args.length == 0) {
            System.err.println("USAGE: $0 [file.jar]+\n");
            System.exit(1);
        }

        String label = null;
        String outputFile = null;
        List<String> inputFiles = new ArrayList<>();
        int maxArg = args.length - 1;
        for (int i = 0; i < args.length; i++) {
            String arg = args[i];
            if ("--output_file".equals(arg)) {
                if (i + 1 > maxArg) {
                    throw new IllegalArgumentException("malformed --output_file: no argument provided");
                }
                outputFile = args[i + 1];
                i++;
                continue;
            }
            if ("--label".equals(arg)) {
                if (i + 1 > maxArg) {
                    throw new IllegalArgumentException("malformed --label: no argument provided");
                }
                label = args[i + 1];
                i++;
                continue;
            }
            inputFiles.add(arg);
        }
        if (label == null || label.isEmpty()) {
            throw new IllegalArgumentException("malformed usage: no label provided");
        }
        if (outputFile == null) {
            throw new IllegalArgumentException("malformed usage: no output file specified");
        }
        if (inputFiles.isEmpty()) {
            throw new IllegalArgumentException("malformed usage: no input files provided");
        }

        Index.Builder index = Index.newBuilder();
        Indexer indexer = new Indexer(index);

        for (final String input : inputFiles) {
            indexer.index(label, Path.of(input));
        }
        try (FileOutputStream fos = new FileOutputStream(outputFile)) {
            index.build().writeTo(fos);
        }
    }

}
