import java.net.URI;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.Collection;
import java.util.HashSet;
import java.util.Map;
import java.util.Optional;
import java.util.List;
import java.util.Set;
import java.util.Collections;
import java.util.HashSet;
import java.util.stream.Collectors;

import javax.lang.model.element.TypeParameterElement;

import java.util.ArrayList;
import java.io.FileNotFoundException;
import java.io.PrintWriter;
import java.lang.reflect.Type;

import io.github.classgraph.ClassGraph;
import io.github.classgraph.ClassInfo;
import io.github.classgraph.FieldInfo;
import io.github.classgraph.MethodInfo;
import io.github.classgraph.TypeSignature;
import io.github.classgraph.TypeParameter;
import io.github.classgraph.MethodInfo;
import io.github.classgraph.MethodTypeSignature;
import io.github.classgraph.MethodParameterInfo;
import io.github.classgraph.ClassRefTypeSignature;
import io.github.classgraph.TypeVariableSignature;
import io.github.classgraph.ArrayTypeSignature;

import io.github.classgraph.ClassInfoList;
import io.github.classgraph.ScanResult;

import com.google.gson.Gson;
import com.google.gson.GsonBuilder;
import com.google.gson.JsonSerializer;
import com.google.gson.JsonSerializationContext;
import com.google.gson.JsonElement;
import com.google.gson.JsonArray;
import com.google.gson.annotations.Expose;

public class JarIndexer {

    public JarIndexer() {
    }

    public Index index(String filename) {
        ScanResult scanResult = new ClassGraph()
                .verbose(false)
                .whitelistPackages()
                .overrideClasspath(filename)
                .enableInterClassDependencies()
                .enableExternalClasses()
                .enableAllInfo()
                .scan();

        List<File> files = new ArrayList<>();

        for (ClassInfo cls : scanResult.getAllClasses()) {
            final File file = new File(cls);
            files.add(file);
        }

        return new Index(files);
    }

    private static class Index {
        @Expose(serialize = true)
        final List<File> files;

        Index(List<File> files) {
            this.files = files;
        }

        @Override
        public String toString() {
            Gson gson = new GsonBuilder()
                    .registerTypeHierarchyAdapter(List.class, new ListAdapter())
                    .excludeFieldsWithoutExposeAnnotation()
                    .setPrettyPrinting()
                    .create();

            return gson.toJson(this);
        }
    }

    private static class File {
        private final ClassInfo info;

        @Expose
        private final String name;
        // @Expose
        // private final List<String> superclasses;
        // @Expose
        // private final List<String> interfaces;
        // @Expose
        // private final List<Method> methods;
        // @Expose
        // private final List<Field> fields;
        @Expose
        private final List<String> symbols;

        File(ClassInfo info) {
            this.info = info;
            this.name = info.getName();
            // this.superclasses = info.getSuperclasses().stream().map(c -> c.getName())
            // .collect(Collectors.toUnmodifiableList());
            // this.interfaces = info.getInterfaces().stream().map(c -> c.getName())
            // .collect(Collectors.toUnmodifiableList());
            // this.methods = info.getMethodInfo().stream().map(m -> new Method(m))
            // .collect(Collectors.toUnmodifiableList());
            // this.fields = info.getFieldInfo().stream().map(f -> new Field(f))
            // .collect(Collectors.toUnmodifiableList());

            this.symbols = collectSymbols(info);
        }

        private List<String> collectSymbols(ClassInfo info) {
            Set<String> symbols = new HashSet<>();

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

            List<String> out = new ArrayList(symbols);
            Collections.sort(out);
            return out;
        }

        private static void addMethodParameterInfo(MethodParameterInfo mpi, Collection<String> symbols) {
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

    private static class Field {
        private final FieldInfo info;

        @Expose
        private final String name;

        @Expose
        private final TypeRef type;

        Field(FieldInfo info) {
            this.info = info;
            this.name = info.getName();
            this.type = new TypeRef(info.getTypeDescriptor());
        }
    }

    private static class Method {
        private final MethodInfo info;

        @Expose
        private final String name;

        @Expose
        private final TypeRef returns;

        @Expose
        private final List<MethodParameter> params;

        @Expose
        private final List<TypeParam> types;

        @Expose
        private final List<TypeRef> throwz;

        Method(MethodInfo info) {
            this.info = info;
            this.name = info.getName();
            if (info.getParameterInfo() != null) {
                this.params = Arrays.stream(info.getParameterInfo())
                        .map(t -> new MethodParameter(t))
                        .collect(Collectors.toList());
            } else {
                params = List.of();
            }

            this.types = info.getTypeDescriptor()
                    .getTypeParameters().stream().map(t -> new TypeParam(t))
                    .collect(Collectors.toList());
            this.throwz = info.getTypeDescriptor()
                    .getThrowsSignatures().stream().map(t -> new TypeRef(t))
                    .collect(Collectors.toList());
            this.returns = new TypeRef(info.getTypeDescriptor().getResultType());
        }
    }

    private static class MethodParameter {
        private final MethodParameterInfo info;

        @Expose
        private final TypeRef type;

        MethodParameter(MethodParameterInfo info) {
            this.info = info;
            this.type = new TypeRef(info.getTypeSignature());
        }
    }

    private static class TypeParam {
        private final TypeParameter info;

        @Expose
        private final String name;

        TypeParam(TypeParameter info) {
            this.info = info;
            this.name = info.getName();
        }
    }

    private static class TypeRef {
        private final TypeSignature info;

        @Expose
        private final TypeRefKind type;

        @Expose
        private final String value;

        TypeRef(TypeSignature info) {
            this.info = info;
            if (info != null) {
                this.type = TypeRefKind.of(info.getClass().getName());
                this.value = info.toString();
            } else {
                this.type = null;
                this.value = null;
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

    public enum TypeRefKind {
        UNKNOWN(0),
        A(1),
        B(2),
        L(3),
        G(4);

        private final int code;

        TypeRefKind(int code) {
            this.code = code;
        }

        public static TypeRefKind of(String s) {
            switch (s) {
                case "io.github.classgraph.ClassRefTypeSignature":
                    return L;
                case "io.github.classgraph.BaseTypeSignature":
                    return B;
                case "io.github.classgraph.ArrayTypeSignature":
                    return A;
                case "io.github.classgraph.TypeVariableSignature":
                    return G;
                default:
                    System.err.println("Unknown typerefkind: " + s);
                    System.exit(1);
                    return UNKNOWN;
            }
        }
    }

    public static void main(String[] args) throws FileNotFoundException {
        if (args.length == 0) {
            System.err.println("USAGE: $0 [file.jar]+\n");
            System.exit(1);
        }

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
            inputFiles.add(arg);
        }
        if (inputFiles.isEmpty()) {
            throw new IllegalArgumentException("malformed usage: no input files provided");
        }

        JarIndexer indexer = new JarIndexer();

        if (outputFile != null) {
            try (PrintWriter out = new PrintWriter(outputFile)) {
                for (String inputFile : inputFiles) {
                    System.out.println("JarIndex " + inputFile);
                    Index index = indexer.index(inputFile);
                    out.println(index);
                }
            }
        } else {
            for (String inputFile : inputFiles) {
                Index index = indexer.index(inputFile);
                System.out.println(index.toString());
            }
        }
    }

}
