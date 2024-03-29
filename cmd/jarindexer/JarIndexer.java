import java.net.URI;
import java.nio.file.Path;
import java.io.FileOutputStream;
import java.io.IOException;
import java.io.FileNotFoundException;
import java.io.PrintWriter;
import java.lang.reflect.Type;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.Collection;
import java.util.Collections;
import java.util.TreeSet;
import java.util.HashSet;
import java.util.List;
import java.util.Map;
import java.util.Optional;
import java.util.Set;
import java.util.stream.Collectors;
import java.util.logging.Level;
import java.util.logging.Logger;

import io.github.classgraph.ArrayTypeSignature;
import io.github.classgraph.ClassGraph;
import io.github.classgraph.ClassInfo;
import io.github.classgraph.ClassInfoList;
import io.github.classgraph.ClassRefTypeSignature;
import io.github.classgraph.FieldInfo;
import io.github.classgraph.MethodInfo;
import io.github.classgraph.MethodInfo;
import io.github.classgraph.MethodParameterInfo;
import io.github.classgraph.MethodTypeSignature;
import io.github.classgraph.ScanResult;
import io.github.classgraph.TypeParameter;
import io.github.classgraph.TypeSignature;
import io.github.classgraph.TypeVariableSignature;

import build.stack.gazelle.scala.jarindex.JarIndex;
import build.stack.gazelle.scala.jarindex.JarFile;
import build.stack.gazelle.scala.jarindex.ClassFile;

public class JarIndexer extends Object {

    static Logger logger = Logger.getLogger(JarIndexer.class.getName());

    private final Path baseDir;
    private final JarIndex.Builder index = JarIndex.newBuilder();

    public JarIndexer(final Path baseDir) {
        this.baseDir = baseDir;
    }

    public JarIndex build() {
        return index.build();
    }

    public void index(String label, String kind, Path path) {
        logger.log(Level.FINE, "indexing file {0}", path);

        index.addJarFile(this.makeJarFile(label, kind, path));
    }

    public JarFile makeJarFile(String label, String kind, Path path) {
        JarFile.Builder jarFile = JarFile.newBuilder();
        jarFile.setFilename(this.baseDir.relativize(path).toString());
        jarFile.setLabel(label);
        jarFile.setKind(kind);

        final ScanResult scanResult = new ClassGraph()
                .verbose(false)
                .whitelistPackages()
                .overrideClasspath(path.toString())
                .enableAllInfo()
                .scan();

        Set<String> packages = new TreeSet<>();

        for (ClassInfo classInfo : scanResult.getAllClasses()) {
            logger.log(Level.FINE, "processing classInfo {0}", classInfo.getName());
            if (!classInfo.getPackageName().isEmpty()) {
                packages.add(classInfo.getPackageName());
            }
            jarFile.addClassFile(handleClassInfo(classInfo));
        }

        jarFile.addAllPackageName(packages);
        return jarFile.build();
    }

    private static ClassFile handleClassInfo(ClassInfo classInfo) {
        ClassFile.Builder classFile = ClassFile.newBuilder();
        classFile.setName(classInfo.getName());
        classFile.addAllSymbols(collectSymbols(classInfo));
        classFile.setIsInterface(classInfo.isInterface());

        for (ClassInfo cls : classInfo.getSuperclasses()) {
            classFile.addSuperclasses(cls.getName());
        }
        for (ClassInfo ifc : classInfo.getInterfaces()) {
            classFile.addInterfaces(ifc.getName());
        }

        return classFile.build();
    }

    private static Set<String> collectSymbols(ClassInfo info) {
        Set<String> symbols = new TreeSet<>();

        for (ClassInfo cls : info.getSuperclasses()) {
            symbols.add(cls.getName());
        }
        for (ClassInfo ifc : info.getInterfaces()) {
            symbols.add(ifc.getName());
        }
        for (MethodInfo m : info.getMethodInfo()) {
            visitMethodTypeSignature(m.getTypeDescriptor(), symbols);

            MethodParameterInfo[] params = m.getParameterInfo();
            for (int i = 0; i < params.length; i++) {
                addMethodParameterInfo(params[i], symbols);
            }
        }

        return symbols;
    }

    private static void visitMethodTypeSignature(MethodTypeSignature mts,
            Collection<String> symbols) {
        visitTypeSignature(mts.getResultType(), symbols);
        for (TypeSignature t : mts.getThrowsSignatures()) {
            visitTypeSignature(t, symbols);
        }
        for (TypeParameter t : mts.getTypeParameters()) {
            visitTypeParameter(t, symbols);
        }
    }

    private static void visitTypeParameter(TypeParameter tp, Collection<String> symbols) {
        visitTypeSignature(tp.getClassBound(), symbols);
        for (TypeSignature rts : tp.getInterfaceBounds()) {
            visitTypeSignature(rts, symbols);
        }
    }

    private static void addMethodParameterInfo(MethodParameterInfo mpi,
            Collection<String> symbols) {
        visitTypeSignature(mpi.getTypeDescriptor(), symbols);
    }

    private static void visitTypeSignature(TypeSignature ts, Collection<String> symbols) {
        if (ts instanceof ClassRefTypeSignature) {
            String fqn = ((ClassRefTypeSignature) ts).getFullyQualifiedClassName();
            symbols.add(fqn);
            return;
        }
        if (ts instanceof ArrayTypeSignature) {
            ArrayTypeSignature ats = (ArrayTypeSignature) ts;
            visitTypeSignature(ats.getElementTypeSignature(), symbols);
            return;
        }
        if (ts instanceof TypeVariableSignature) {
            // TODO
            return;
        }
    }

    public static void main(String[] args) throws FileNotFoundException, IOException {
        if (args.length == 0) {
            System.err.println("USAGE: $0 --label LABEL --output_file FILE [file.jar]+\n");
            System.exit(1);
        }

        String label = null;
        String kind = null;
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
            if ("--kind".equals(arg)) {
                if (i + 1 > maxArg) {
                    throw new IllegalArgumentException("malformed --kind: no argument provided");
                }
                kind = args[i + 1];
                i++;
                continue;
            }
            inputFiles.add(arg);
        }
        if (label == null || label.isEmpty()) {
            throw new IllegalArgumentException("malformed usage: no label provided");
        }
        if (kind == null || kind.isEmpty()) {
            throw new IllegalArgumentException("malformed usage: no kind provided");
        }
        if (outputFile == null) {
            throw new IllegalArgumentException("malformed usage: no output file specified");
        }
        if (inputFiles.isEmpty()) {
            throw new IllegalArgumentException("malformed usage: no input files provided");
        }

        JarIndexer indexer = new JarIndexer(Path.of("."));

        for (final String inputFile : inputFiles) {
            indexer.index(label, kind, Path.of(inputFile));
        }

        final JarIndex index = indexer.build();
        try (FileOutputStream fos = new FileOutputStream(outputFile)) {
            index.writeTo(fos);
        }
    }

}
