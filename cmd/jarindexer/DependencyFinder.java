import java.net.URI;
import java.util.HashSet;
import java.util.Map;
import java.util.Set;

import io.github.classgraph.ClassGraph;
import io.github.classgraph.ClassInfo;
import io.github.classgraph.ClassInfoList;
import io.github.classgraph.ScanResult;

public class DependencyFinder {

  private final String clazz;

  public DependencyFinder(String clazz) {
    this.clazz = clazz;
  }

  public Set<URI> process() {
    ScanResult scanResult = new ClassGraph()
        .whitelistPackages()
        .enableInterClassDependencies()
        .scan();

    ClassInfo rootClass = scanResult.getClassInfo(clazz);
    Map<ClassInfo, ClassInfoList> dependencyMap = scanResult.getClassDependencyMap();

    Set<URI> results = new HashSet<>();
    Set<ClassInfo> seen = new HashSet<>();

    accumulateJars(new HashSet<>(dependencyMap.get(rootClass)), dependencyMap, results, seen);

    return results;
  }

  private void accumulateJars(Set<ClassInfo> roots, Map<ClassInfo, ClassInfoList> dependencies, Set<URI> accumulated,
      Set<ClassInfo> seen) {
    Set<ClassInfo> nextRoots = new HashSet<>();

    for (ClassInfo info : roots) {
      if (seen.contains(info)) {
        continue;
      }

      accumulated.add(info.getClasspathElementURI());
      seen.add(info);

      nextRoots.addAll(dependencies.get(info));
    }

    if (nextRoots.size() > 0) {
      accumulateJars(nextRoots, dependencies, accumulated, seen);
    }
  }

  public static void main(String[] args) {
    if (args.length == 0) {
      System.err.println("USAGE: $0 CLASSNAME\n");
      System.exit(1);
    }
    String src = args[0];
    DependencyFinder app = new DependencyFinder(src);

    System.out.format("src: %s\n", src);
    for (URI uri : app.process()) {
      System.out.format("dst: %s\n", uri);
    }
  }
}
