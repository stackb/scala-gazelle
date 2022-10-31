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

import build.stack.scala.gazelle.api.jarindex.Index;
import build.stack.scala.gazelle.api.jarindex.JarFile;
import build.stack.scala.gazelle.api.jarindex.ClassFile;

public class Indexer {

    static Logger logger = Logger.getLogger(Indexer.class.getName());

    private Index.Builder index = Index.newBuilder();

    public Indexer() {
        // logger.setLevel(Level.WARNING);
    }

    public Index build() {
        return index.build();
    }

    public void index(String label, Path path) {
        logger.log(Level.INFO, "indexing file {0}", path);

        index.addJarFile(this.makeJarFile(label, path));
    }

    public JarFile makeJarFile(String label, Path path) {
        JarFile.Builder jarFile = JarFile.newBuilder();
        jarFile.setFilename(path.toString());
        jarFile.setLabel(label);

        final ScanResult scanResult = new ClassGraph()
                .verbose(true)
                .whitelistPackages()
                .overrideClasspath(path.toString())
                .enableInterClassDependencies()
                .enableExternalClasses()
                .enableAllInfo()
                .scan();

        for (ClassInfo classInfo : scanResult.getAllClasses()) {
            logger.log(Level.INFO, "processing classInfo {0}", classInfo.getName());

            jarFile.addClassFile(handleClassInfo(classInfo));
            jarFile.addPackageName(classInfo.getPackageName());
            jarFile.addClassName(classInfo.getName());
        }

        return jarFile.build();
    }

    private static ClassFile handleClassInfo(ClassInfo classInfo) {
        ClassFile.Builder classFile = ClassFile.newBuilder();
        classFile.setName(classInfo.getName());
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
        // for (MethodInfo m : info.getMethodInfo()) {
        // visitMethodTypeSignature(m.getTypeDescriptor(), symbols);

        // MethodParameterInfo[] params = m.getParameterInfo();
        // for (int i = 0; i < params.length; i++) {
        // addMethodParameterInfo(params[i], symbols);
        // }
        // }

        return symbols;

    }

    // Index(String label, String filename, List<File> files) {
    // this.label = label;
    // this.filename = filename;

    // this.files = files;
    // this.files.sort((File a, File b) -> a.name.compareTo(b.name));

    // this.classes = this.files.stream().map(f ->
    // f.name).collect(Collectors.toCollection(TreeSet::new));
    // for (File f : this.files) {
    // this.packages.add(f.info.getPackageName());
    // f.filterInternalSymbols(classes);
    // }
    // }

    // @Override
    // public String toString() {
    // return new GsonBuilder()
    // .registerTypeHierarchyAdapter(List.class, new ListAdapter())
    // .excludeFieldsWithoutExposeAnnotation()
    // .setPrettyPrinting()
    // .create()
    // .toJson(this);
    // }
    // }

    // private static class File {
    // private final ClassInfo info;

    // @Expose
    // private final String name;

    // @Expose
    // private final Set<String> symbols;

    // File(ClassInfo info) {
    // this.info = info;

    // this.name = info.getName();
    // this.symbols = collectSymbols(info);
    // }

    // void filterInternalSymbols(Set classes) {
    // symbols.removeAll(classes);
    // }

    // private static Set<String> collectSymbols(ClassInfo info) {
    // Set<String> symbols = new TreeSet<>();

    // for (ClassInfo cls : info.getSuperclasses()) {
    // symbols.add(cls.getName());
    // }
    // for (ClassInfo ifc : info.getInterfaces()) {
    // symbols.add(ifc.getName());
    // }
    // for (MethodInfo m : info.getMethodInfo()) {
    // visitMethodTypeSignature(m.getTypeDescriptor(), symbols);

    // MethodParameterInfo[] params = m.getParameterInfo();
    // for (int i = 0; i < params.length; i++) {
    // addMethodParameterInfo(params[i], symbols);
    // }
    // }

    // return symbols;

    // }

    // private static void addMethodParameterInfo(MethodParameterInfo mpi,
    // Collection<String> symbols) {
    // visitTypeSignature(mpi.getTypeDescriptor(), symbols);
    // }

    // private static void visitTypeParameter(TypeParameter tp, Collection<String>
    // symbols) {
    // visitTypeSignature(tp.getClassBound(), symbols);
    // for (TypeSignature rts : tp.getInterfaceBounds()) {
    // visitTypeSignature(rts, symbols);
    // }
    // }

    // private static void visitMethodTypeSignature(MethodTypeSignature mts,
    // Collection<String> symbols) {
    // visitTypeSignature(mts.getResultType(), symbols);
    // for (TypeSignature t : mts.getThrowsSignatures()) {
    // visitTypeSignature(t, symbols);
    // }
    // for (TypeParameter t : mts.getTypeParameters()) {
    // visitTypeParameter(t, symbols);
    // }
    // }

    // private static void visitTypeSignature(TypeSignature ts, Collection<String>
    // symbols) {
    // if (ts instanceof ClassRefTypeSignature) {
    // String fqn = ((ClassRefTypeSignature) ts).getFullyQualifiedClassName();
    // symbols.add(fqn);
    // return;
    // }
    // if (ts instanceof ArrayTypeSignature) {
    // ArrayTypeSignature ats = (ArrayTypeSignature) ts;
    // visitTypeSignature(ats.getElementTypeSignature(), symbols);
    // return;
    // }
    // if (ts instanceof TypeVariableSignature) {
    // // TODO
    // return;
    // }
    // }
    // }

    // public static class ListAdapter implements JsonSerializer<List<?>> {
    // @Override
    // public JsonElement serialize(List<?> src, Type typeOfSrc,
    // JsonSerializationContext context) {
    // if (src == null || src.isEmpty()) // exclusion is made here
    // return null;

    // JsonArray array = new JsonArray();

    // for (Object child : src) {
    // JsonElement element = context.serialize(child);
    // array.add(element);
    // }

    // return array;
    // }
    // }

    public static void main(String[] args) throws FileNotFoundException, IOException {
        if (args.length == 0) {
            System.err.println("USAGE: $0 --label LABEL --output_file FILE [file.jar]+\n");
            System.exit(1);
        }

        String label = null;
        String outputFile = null;
        List<String> inputFiles = new ArrayList<>();
        int maxArg = args.length - 1;
        for (int i = 0; i < args.length; i++) {
            String arg = args[i];
            System.out.println("processing arg:" + arg);
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
        Indexer indexer = new Indexer();

        for (final String input : inputFiles) {
            indexer.index(label, Path.of(input));
        }

        try (FileOutputStream fos = new FileOutputStream(outputFile)) {
            index.build().writeTo(fos);
        }
    }

}
