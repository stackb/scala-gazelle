import java.net.URI;

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

import com.google.gson.annotations.Expose;
import com.google.gson.Gson;
import com.google.gson.GsonBuilder;
import com.google.gson.JsonArray;
import com.google.gson.JsonElement;
import com.google.gson.JsonSerializationContext;
import com.google.gson.JsonSerializer;

public class JarIndexer {

    public Index index(String filename, String label) {
        final ScanResult scanResult = new ClassGraph()
                .verbose(false)
                .whitelistPackages()
                .overrideClasspath(filename)
                .enableInterClassDependencies()
                .enableExternalClasses()
                .enableAllInfo()
                .scan();

        return new Index(label, filename,
                scanResult.getAllClasses()
                        .stream()
                        .filter(cls -> !cls.isExternalClass())
                        .map(cls -> new File(cls))
                        .collect(Collectors.toList()));
    }

    private static class Index {
        @Expose()
        final String label;

        @Expose()
        final String filename;

        @Expose()
        final Set<String> classes;

        @Expose()
        final List<File> files;

        @Expose
        final Collection<String> packages = new TreeSet<>();

        Index(String label, String filename, List<File> files) {
            this.label = label;
            this.filename = filename;

            this.files = files;
            this.files.sort((File a, File b) -> a.name.compareTo(b.name));

            this.classes = this.files.stream().map(f -> f.name).collect(Collectors.toCollection(TreeSet::new));
            for (File f : this.files) {
                this.packages.add(f.info.getPackageName());
                f.filterInternalSymbols(classes);
            }
        }

        @Override
        public String toString() {
            return new GsonBuilder()
                    .registerTypeHierarchyAdapter(List.class, new ListAdapter())
                    .excludeFieldsWithoutExposeAnnotation()
                    .setPrettyPrinting()
                    .create()
                    .toJson(this);
        }
    }

    private static class File {
        private final ClassInfo info;

        @Expose
        private final String name;

        @Expose
        private final Set<String> symbols;

        File(ClassInfo info) {
            this.info = info;

            this.name = info.getName();
            this.symbols = collectSymbols(info);
        }

        void filterInternalSymbols(Set classes) {
            symbols.removeAll(classes);
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

        private static void addMethodParameterInfo(MethodParameterInfo mpi, Collection<String> symbols) {
            visitTypeSignature(mpi.getTypeDescriptor(), symbols);
        }

        private static void visitTypeParameter(TypeParameter tp, Collection<String> symbols) {
            visitTypeSignature(tp.getClassBound(), symbols);
            for (TypeSignature rts : tp.getInterfaceBounds()) {
                visitTypeSignature(rts, symbols);
            }
        }

        private static void visitMethodTypeSignature(MethodTypeSignature mts, Collection<String> symbols) {
            visitTypeSignature(mts.getResultType(), symbols);
            for (TypeSignature t : mts.getThrowsSignatures()) {
                visitTypeSignature(t, symbols);
            }
            for (TypeParameter t : mts.getTypeParameters()) {
                visitTypeParameter(t, symbols);
            }
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
    }

    public static class ListAdapter implements JsonSerializer<List<?>> {
        @Override
        public JsonElement serialize(List<?> src, Type typeOfSrc, JsonSerializationContext context) {
            if (src == null || src.isEmpty()) // exclusion is made here
                return null;

            JsonArray array = new JsonArray();

            for (Object child : src) {
                JsonElement element = context.serialize(child);
                array.add(element);
            }

            return array;
        }
    }

    public static void main(String[] args) throws FileNotFoundException {
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

        if (inputFiles.isEmpty()) {
            throw new IllegalArgumentException("malformed usage: no input files provided");
        }

        JarIndexer indexer = new JarIndexer();

        if (outputFile != null) {
            try (PrintWriter out = new PrintWriter(outputFile)) {
                for (String inputFile : inputFiles) {
                    // System.out.println("JarIndex " + inputFile);
                    Index index = indexer.index(inputFile, label);
                    out.println(index);
                }
            }
        } else {
            for (String inputFile : inputFiles) {
                Index index = indexer.index(inputFile, label);
                System.out.println(index.toString());
            }
        }
    }

}
