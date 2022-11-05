
import static org.junit.Assert.assertEquals;

import build.stack.scala.gazelle.api.jarindex.Index;
import build.stack.scala.gazelle.api.jarindex.JarFile;

import com.google.protobuf.util.JsonFormat;

import java.util.Collection;
import java.util.Map;
import java.util.Optional;
import java.util.Set;
import java.util.logging.Level;
import java.util.logging.Logger;
import java.util.Arrays;
import java.util.stream.Stream;

import javax.print.DocFlavor.STRING;

import java.util.stream.Collectors;
import java.io.File;
import java.io.IOException;
import java.nio.file.FileSystems;
import java.nio.file.Files;
import java.nio.file.Paths;
import java.nio.file.StandardCopyOption;
import java.nio.file.Path;
import java.nio.file.PathMatcher;

import org.junit.Test;
import org.junit.Rule;
import org.junit.rules.TemporaryFolder;

public class IndexerTest {

    static final Logger LOGGER = Logger.getLogger(IndexerTest.class.getName());
    static final boolean wantUpdate = System.getenv("BUILD_WORKING_DIRECTORY") != null;

    @Test
    public void testGoldens() throws IOException {
        listEnv();

        final Path srcdir = mustGetSelfLocationDir();
        LOGGER.log(Level.INFO, "srcdir: " + srcdir);
        // listDir(srcdir);

        final Set<Path> dirs = listTestDataDirectories(srcdir);

        if (dirs.isEmpty()) {
            throw new IllegalStateException(
                    "expected subdirectories of testdata/, but none found: " + srcdir.toString());
        }

        for (final Path dir : dirs) {
            this.testGoldenDir(srcdir, dir);
        }

        if (false) {
            throw new RuntimeException("WIP");
        }
    }

    private void testGoldenDir(final Path srcdir, final Path dir) throws IOException {
        LOGGER.log(Level.INFO, "golden test dir: " + dir);
        listDir(dir);
        final Set<Path> files = listFiles(dir);
        LOGGER.log(Level.INFO, "testing files: " + files.size());

        final Optional<Path> goldenFile = files.stream()
                .filter(path -> path.getFileName().endsWith("golden.json"))
                .findFirst();
        if (!goldenFile.isPresent()) {
            throw new IllegalStateException(
                    String.format("testdata subdirectory %s must contain a file named 'golden.json'", dir));
        }
        LOGGER.log(Level.INFO, "golden file: " + goldenFile);

        final Index want = mustParseIndexJson(goldenFile.get());
        final Path tmpDir = mustGetTestTmpDir();

        Indexer indexer = new Indexer(tmpDir);
        for (Path srcFile : files) {
            LOGGER.log(Level.INFO, "possible file: " + srcFile);
            if (srcFile.toString().endsWith(".jar")) {
                Path rel = srcdir.relativize(srcFile);
                LOGGER.log(Level.INFO, "indexing jar: " + rel);
                final Path tmpFile = tmpDir.resolve(rel.toString());

                LOGGER.log(Level.INFO, "copying jar: " + tmpFile);
                Files.createDirectories(tmpFile.getParent());
                Files.copy(srcFile, tmpFile, StandardCopyOption.REPLACE_EXISTING);

                indexer.index("//fake:label", tmpFile);
            }
        }

        final Index got = indexer.build();

        if (wantUpdate) {
            Path sourceFile = getSourceFile(goldenFile.get());
            LOGGER.log(Level.INFO, "updating source file: " + sourceFile);
            mustWriteIndexJson(sourceFile, got);
        } else {
            assertEquals(want, got);
        }
    }

    private static Set<Path> listTestDataDirectories(Path srcdir) throws IOException {
        final Path path = srcdir.resolve("testdata");
        try (Stream<Path> stream = Files.list(path)) {
            return stream
                    .filter(file -> Files.isDirectory(file))
                    // .map(Path::toAbsolutePath)
                    .collect(Collectors.toSet());
        }
    }

    private static Path mustGetSelfLocationDir() {
        final String dir = System.getenv("SELF_LOCATION");
        if (dir == null || dir.isEmpty()) {
            throw new IllegalStateException("SELF_LOCATION not set!");
        }
        return Path.of(dir).getParent();
    }

    private static Path mustGetBuildWorkingDirectory() {
        final String dir = System.getenv("BUILD_WORKING_DIRECTORY");
        if (dir == null || dir.isEmpty()) {
            throw new IllegalStateException(
                    "BUILD_WORKING_DIRECTORY not set!  Are you trying up update golden files without using 'bazel run'?");
        }
        return Path.of(dir);
    }

    private static Path mustGetWorkspaceDirectory() {
        Path bwd = mustGetBuildWorkingDirectory();
        Path current = Path.of(bwd.toString());
        while (current != null) {
            final Path workspaceFile = current.resolve("WORKSPACE");
            if (Files.exists(workspaceFile)) {
                return current;
            }
            current = current.getParent();
        }
        throw new IllegalStateException("Could not find WORKSPACE dir from: " + bwd);
    }

    private static Path mustGetRunfilesDirectory() {
        final String dir = System.getenv("RUNFILES_DIR");
        if (dir == null || dir.isEmpty()) {
            throw new IllegalStateException("RUNFILES_DIR not set!");
        }
        return Path.of(dir);
    }

    private static Path mustGetTestTmpDir() {
        final String dir = System.getenv("TEST_TMPDIR");
        if (dir == null || dir.isEmpty()) {
            throw new IllegalStateException("TEST_TMPDIR not set!");
        }
        return Path.of(dir);
    }

    private static String mustGetWorkspaceName() {
        final String name = System.getenv("TEST_WORKSPACE");
        if (name == null || name.isEmpty()) {
            throw new IllegalStateException("TEST_WORKSPACE not set!");
        }
        return name;
    }

    private static void listEnv() {
        Map<String, String> env = System.getenv();
        env.forEach((k, v) -> LOGGER.info(k + ":" + v));
    }

    private static void listDir(final Path path) throws IOException {
        try (Stream<Path> stream = Files.walk(path)) {
            stream.filter(Files::isRegularFile)
                    .forEach(System.out::println);
        }
    }

    private static Set<Path> listFiles(final Path path) throws IOException {
        try (Stream<Path> stream = Files.walk(path)) {
            return stream.filter(Files::isRegularFile)
                    .collect(Collectors.toSet());
        }
    }

    private static Path getSourceFile(Path path) {
        Path runfilesDir = mustGetRunfilesDirectory();
        String workspaceName = mustGetWorkspaceName();
        Path parent = runfilesDir.resolve(workspaceName);
        String rel = path.toString().substring(parent.toString().length() + 1);
        LOGGER.info("source file relative filename: " + rel);
        return mustGetWorkspaceDirectory().resolve(rel);
    }

    private static Index mustParseIndexJson(Path path) throws IOException {
        JsonFormat.Parser parser = JsonFormat.parser();
        Index.Builder index = Index.newBuilder();
        String content = String.join("", Files.readAllLines(path));
        LOGGER.info(content);
        parser.merge(content, index);
        return index.build();
    }

    private static void mustWriteIndexJson(Path path, Index index) throws IOException {
        JsonFormat.Printer printer = JsonFormat.printer();
        Files.write(path, printer.print(index).getBytes());
    }

}
