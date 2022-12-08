
[![CI](https://github.com/stackb/scala-gazelle/actions/workflows/ci.yaml/badge.svg)](https://github.com/stackb/scala-gazelle/actions/workflows/ci.yaml)

- [Overview](#overview)
- [Installation](#installation)
  - [Primary Dependency](#primary-dependency)
  - [Transitive Dependencies](#transitive-dependencies)
  - [Gazelle Binary](#gazelle-binary)
  - [Gazelle Rule](#gazelle-rule)
- [Usage](#usage)
- [Configuration](#configuration)
  - [Rule Providers](#rule-providers)
    - [Built-in Existing Rule Providers](#built-in-existing-rule-providers)
    - [Custom Existing Rule Providers](#custom-existing-rule-providers)
    - [Custom Rule Provider](#custom-rule-provider)
  - [Known Import Providers](#known-import-providers)
    - [`scalaparse` known import provider](#scalaparse-known-import-provider)
    - [`maven` known import provider](#maven-known-import-provider)
    - [`jarindex`](#jarindex)
  - [Extension Cache File](#extension-cache-file)

# Overview

This is an experimental gazelle extension for scala.  It has the following design characteristics:

- It only manages scala dependencies.  You are responsible for manually creating
  `scala_library`, `scala_binary`, and `scala_test` targets in their respective packages.
- Existing scala rules are evaluated for the contents of their `srcs`.  
  Globs are interpreted the same as bazel starlark (unless a bug exists).
- Source files named in the `srcs` are parsed for their import statements and exportable symbols (classes, traits, objects, ...).
- Dependencies are resolved be matching required imports against their providing rule labels.  The resolution procedure is configurable.


    1. override directives (e.g. `gazelle:resolve scala com.typesafe.scalalogging.LazyLogging @maven_@maven//:com_typesafe_scala_logging_scala_logging_2_12`)
    2. so-called "import providers".  See below for details.
    3. cross-resolution for language `scala`.  This is only relevant if you have
       a custom extension that implement a custom CrossResolver for scala
       imports. 

# Installation

Add the `build_stack_scala_gazelle` as an external workspace For example:

## Primary Dependency

```bazel
# Branch: master
# Commit: 7a74c78c24e4a4a1877fea854865be8687c87f2c
# Date: 2022-12-08 05:32:02 +0000 UTC
# URL: https://github.com/stackb/scala-gazelle/commit/7a74c78c24e4a4a1877fea854865be8687c87f2c
# 
# Redesign resolution strategy with `resolver.ImportResolver` (#51)
# Size: 160362 (160 kB)
http_archive(
    name = "build_stack_scala_gazelle",
    sha256 = "8229a7e5bc94fa07ef8700b1c89e4afe312d9608ff17523f044b274ea07b6233",
    strip_prefix = "scala-gazelle-7a74c78c24e4a4a1877fea854865be8687c87f2c",
    urls = ["https://github.com/stackb/scala-gazelle/archive/7a74c78c24e4a4a1877fea854865be8687c87f2c.tar.gz"],
)
```

## Transitive Dependencies

Load corresponding transitive dependencies in your `WORKSPACE` as follows:

```bazel
load("@build_stack_scala_gazelle//:workspace_deps.bzl", "language_scala_deps")

language_scala_deps()

load("@build_stack_scala_gazelle//:go_repos.bzl", build_stack_scala_gazelle_gazelle_extension_deps = "gazelle_extension_deps")

build_stack_scala_gazelle_gazelle_extension_deps()
```

## Gazelle Binary

Include the language/scala extension in your `gazelle_binary` rule.  For example:

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
```

## Gazelle Rule

Reference the binary in the gazelle rule:

```bazel
gazelle(
    name = "gazelle",
    args = [...],
    gazelle = ":gazelle-scala",
)
```

The `args` and `data` for this rule are discussed below. 

# Usage

Invoke gazelle as per typical usage:

```sh
$ bazel run //:gazelle
```

# Configuration

## Rule Providers

The extension needs to know which rules it should manage (parse imports/resolve
deps).  This is done using `gazelle:scala_rule` directives.

### Built-in Existing Rule Providers

A preset catalog of providers are available out-of-the-box:

- `@io_bazel_rules_scala//scala:scala.bzl%scala_binary`
- `@io_bazel_rules_scala//scala:scala.bzl%scala_library`
- `@io_bazel_rules_scala//scala:scala.bzl%scala_macro_library`
- `@io_bazel_rules_scala//scala:scala.bzl%scala_test`

To enable a provider, instantiate a "rule provider config":

```bazel
# gazelle:scala_rule scala_library implementation @io_bazel_rules_scala//scala:scala.bzl%scala_library
```

> This reads as "create a rule provider configuration named 'scala_library' whose provider implementation is registered under the name '@io_bazel_rules_scala//scala:scala.bzl%scala_library'

### Custom Existing Rule Providers

You may have your own scala rule macros that look like a `scala_library` or
`scala_binary`, but have their own rule kinds and loads.  To register these
rules/macros as provider implementations, use the
`-existing_scala_rule=LOAD%KIND` flag.  For example:

```bazel
gazelle(
    name = "gazelle",
    args = [
        "-existing_scala_rule=@io_bazel_rules_scala//scala:scala.bzl%_scala_library",
        "-existing_scala_rule=//bazel_tools:scala.bzl%scala_app",
        ...
    ],
    ...
)
```

This can then be instatiated as:

```bazel
# gazelle:scala_rule scala_app implementation //bazel_tools:scala.bzl%scala_app
# gazelle:scala_rule scala_app enabled false # optional if you wanted to disable it in the root
```

This rule could then be selectively enabled/disabled in subpackages as follows:

```bazel
# gazelle:scala_rule scala_app enabled true
```

### Custom Rule Provider

An advanced use-case would involve writing your own `scalarule.Provider`
implementation.

To register it:

```go
import "github.com/stackb/scala-gazelle/pkg/scalarule"

func init() {
  scalarule.GlobalProviderRegistry().RegisterProvider(
    "@foo//rules/scala.bzl:foo_scala_library",
    newFooScalaLibrary(),
  )
}
```

Enable the rule provider configuration:

```bazel
# gazelle:scala_rule foo_scala_library implementation @foo//rules/scala.bzl:foo_scala_library
```

## Known Import Providers

At the core of the import resolution process is a trie structure where the keys
of the trie are parts of an import statement and the values are
`*resolver.KnownImport` structs.

For example, for the import `io.grpc.Status`, the trie would contain the following:

- `io`: (`nil`)
  - `grpc`: type `PACKAGE`, from `@maven//:io_grpc_grpc_api`
    - `Status`: type `CLASS`, from `@maven//:io_grpc_grpc_api`

When resolving the import `io.grpc.Status.ALREADY_EXISTS`, the longest prefix
match would find the `CLASS io.grpc.Status` and the label
`@maven//:io_grpc_grpc_api` would be added to the rule `deps`.

The trie is populated by `resolver.KnownImportProvider` implementations. Each
implementation provides known imports from a different source.

Known import providers:

- Have a canonical name.
- Must be enabled with the `-scala_import_provider` flag.
- Manage their own flags; check the provider source code for details.

### `scalaparse` known import provider

The `scalaparse` provider is responsible for indexing importable symbols from
`.scala` source files during the rule generation phase.

Source files that are listed in the `srcs` of existing scala rules are parsed.
The discovered `object`, `class`, `trait` types are provided to the known import
trie such that they can be resolved by other rules.

The `scala-gazelle` extension would not do much without this provider, but it still needs to be enabled in `args`:

```bazel
-scala_import_provider=source
```

### `maven` known import provider

This provider reads `maven_install.json` files that are produced from pinned `maven_install` repository rules.

As of https://github.com/bazelbuild/rules_jvm_external/pull/716 (`Add index of
packages in jar files when pinning`), `@rules_jvm_external` indexes the package
names that jars provide.

The `maven` provider reads these package names and populates the trie accordingly.  Note that since only package names are known, maven dependency resolution via this mechanism alone is more "coarse-grained".

Issues can occur when more than one jar provides the same package name.  This situation is known as a "split package".  The `io.grpc` namespace is a classic example (see [discussion](https://github.com/grpc/grpc-java/issues/3522)).  The `io.grpc.Context` is in `@maven//:io_grpc_grpc_context`, but other classes like `io.grpc.Status` are in `@maven//:io_grpc_grpc_core`.  Both advertise the package `io.grpc`.

To help avoid issues with split packages:

- Use the `jarindex` provider to supply fine-grained deps for selected artifacts.
- Avoid wildcard imports that involve split packages.

### `jarindex` 

## Extension Cache File

If the extension cache file feature is enabled, 


