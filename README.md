# scala-gazelle

https://github.com/stackb/scala-gazelle/actions/workflows/ci.yaml/badge.svg

This is an experimental gazelle extension for scala.  It has the following design characteristics:

- it only manages scala dependencies.  You are responsible for manually creating
  `scala_library`, `scala_binary`, and `scala_test` targets.
- existing scala rules are inspected for the contents of their `srcs`.  
  Globs are interpreted the same as bazel starlark (unless a bug exists).
- source files named in the `srcs` are parsed for their import statements.
- dependencies are gathered from three possible locations:
    - first-party scala rules in the same workspace.
    - third-party scala rules @rules_jvm_external maven dependencies (relies on a pinned `maven_install.json` file).
    - @build_stack_rules_proto `proto_scala_library` and `grpc_scala_library` rules.  
      That gazelle extension publishes scala imports that can resolve as scala dependencies.

# Installation

Add the `build_stack_scala_gazelle` as an external workspace For example:

```bazel
    native.local_repository(
        name = "build_stack_scala_gazelle",
        path = "/Users/you/go/src/github.com/stackb/scala-gazelle",
    )
```

> NOTE: local_repository is in the README to indicate how experimental this extension
> currently is.  If you're using it, you're probably actively developing it, too.

Load corresponding dependencies in your `WORKSPACE`:

```bazel
load("@build_stack_scala_gazelle//:workspace_deps.bzl", "language_scala_deps")

language_scala_deps()

load("@build_stack_scala_gazelle//:go_repos.bzl", build_stack_scala_gazelle_gazelle_extension_deps = "gazelle_extension_deps")

build_stack_scala_gazelle_gazelle_extension_deps()
```

Include the language/scala extension in your gazelle_binary rule.  For example:

```bazel
gazelle_binary(
    name = "gazelle-scala",
    languages = [
        "@bazel_gazelle//language/proto:go_default_library",
        "@bazel_gazelle//language/go:go_default_library",
        "@build_stack_rules_proto//language/protobuf",
        "@build_stack_scala_gazelle//language/scala",
    ],
)

gazelle(
    name = "gazelle",
    args = [
        "-maven_install_file=./maven_install.json",
    ],
    gazelle = ":gazelle-scala",
)
```

# Configuration

Configure the scala rules you want dependencies to be resolved for.  Often this 
is done in the root `BUILD.bazel` file, but it can be elsewhere:

```bazel
# --- gazelle language "scala" ---
# gazelle:scala_rule scala_library implementation @io_bazel_rules_scala//scala:scala.bzl%scala_library
# gazelle:scala_rule scala_library enabled true
# gazelle:scala_rule scala_binary implementation @io_bazel_rules_scala//scala:scala.bzl%scala_binary
# gazelle:scala_rule scala_binary enabled true
# gazelle:scala_rule scala_test implementation @io_bazel_rules_scala//scala:scala.bzl%scala_test
# gazelle:scala_rule scala_test enabled true
# gazelle:scala_rule scala_macro_library implementation @io_bazel_rules_scala//scala:scala.bzl%scala_macro_library
# gazelle:scala_rule scala_macro_library enabled true
```

# Usage

Invoke gazelle as per typical usage:


```sh
$ bazel run //:gazelle
```


