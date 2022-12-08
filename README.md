
![github-ci](https://github.com/stackb/scala-gazelle/actions/workflows/ci.yaml/badge.svg)


- [Overview](#overview)
- [Installation](#installation)
- [Configuration](#configuration)
  - [Flags](#flags)
  - [Directives](#directives)
  - [Known Import Providers](#known-import-providers)
    - [`scalaparse`](#scalaparse)
    - [`jarindex`](#jarindex)
    - [`github.com/stackb/rules_proto`](#githubcomstackbrules_proto)
    - [`github.com/bazelbuild/rules_jvm_external`](#githubcombazelbuildrules_jvm_external)
- [Usage](#usage)

# Overview

This is an experimental gazelle extension for scala.  It has the following design characteristics:

- it only manages scala dependencies.  You are responsible for manually creating
  `scala_library`, `scala_binary`, and `scala_test` targets.
- existing scala rules are inspected for the contents of their `srcs`.  
  Globs are interpreted the same as bazel starlark (unless a bug exists).
- source files named in the `srcs` are parsed for their import statements.
- dependencies are gathered from three possible locations:
    1. override directives (e.g. `gazelle:resolve scala com.typesafe.scalalogging.LazyLogging @maven_@maven//:com_typesafe_scala_logging_scala_logging_2_12`)
    2. so-called "import providers".  See below for details.
    3. cross-resolution for language `scala`.  This is only relevant if you have
       a custom extension that implement a custom CrossResolver for scala
       imports. 

#  Installation

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
        "-maven_install_json_file=$(location maven_install.json)",
    ],
    data = ["//:maven_install.json"],
    gazelle = ":gazelle-scala",
)
```

> NOTE: -maven_install_json_file can be a comma-separated list of 
> @{EXTERNAL_MAVEN_WORKSPACE_NAME}_install.json files.

# Configuration

Configure the scala rules you want dependencies to be resolved for.  Often this 
is done in the root `BUILD.bazel` file, but it can be elsewhere:

## Flags

## Directives

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

## Known Import Providers

The scala-gazelle extension maintains a trie of "known imports".  For example, the
trie may know that `com.google.gson.Gson` resolves to a `CLASS` provided by
`@maven//:com_google_code_gson_gson`, and that the import `com.google.gson._`
resolves to the `PACKAGE` `com.google.gson`, also from
`@maven//:com_google_code_gson_gson`.  

Similarly, the import
`io.grpc.Status.UNIMPLEMENTED` would resolve to the longest trie prefix as
`io.grpc.Status`.

Various `resolver.KnownImportProvider` implementations can be configured to
populate the known import trie.  Each import provider has a canonical name and
are enabled via the `-scala_import_provider=NAME` flag.  

The order of `-scala_import_provider` determines the resolution ordering, so put more fine-grained providers (e.g `jarindex`) before more coarse-grained ones (e.g. `github.com/bazelbuild/rules_jvm_external`, which only provides package-level imports).

Provider implementations manage their own flags, so please check the source file for the most up-to-date documentation on the flags used by different import providers.

### `scalaparse`

A provider that parses scala source files and populates the trie with classes, objects, traits, etc that are discovered during the rule generation phase.

### `jarindex`

A provider that reads jar index files and populates the trie with classes, objects, traits, etc that are listed in the file.  An index is produced by  the `@build_stack_scala_gazelle//rules:jar_class_index.bzl%jar_class_index` build rule.

### `github.com/stackb/rules_proto`

A provider that gathers imports from `proto_scala_library` and `grpc_scala_library`.

### `github.com/bazelbuild/rules_jvm_external`

A provider that reads pinned `maven_install.json` files produced by the `@rules_jvm_external//:defs.bzl%maven_install` repository rule.

# Usage

Invoke gazelle as per typical usage:

```sh
$ bazel run //:gazelle
```

