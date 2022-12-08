
[![CI](https://github.com/stackb/scala-gazelle/actions/workflows/ci.yaml/badge.svg)](https://github.com/stackb/scala-gazelle/actions/workflows/ci.yaml)

- [Overview](#overview)
- [Installation](#installation)
  - [Primary Dependency](#primary-dependency)
  - [Transitive Dependencies](#transitive-dependencies)
  - [Gazelle Binary](#gazelle-binary)
  - [Gazelle Rule](#gazelle-rule)
- [Configuration](#configuration)
  - [Rule Providers](#rule-providers)
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

# Configuration

## Rule Providers

The extension needs to know which rules it should manage (parse imports/resolve deps).  This is done using `gazelle:scala_rule` directives.  Example:

```bazel
# gazelle:scala_rule scala_library implementation @io_bazel_rules_scala//scala:scala.bzl%scala_library
```

> This reads as "create a rule provider configuration named 'scala_library' whose provider implementation is registered under the name '@io_bazel_rules_scala//scala:scala.bzl%scala_library'

There is a registry of implementations; a preset catalog of  [builtin implementations](https://github.com/stackb/scala-gazelle/blob/7a74c78c24e4a4a1877fea854865be8687c87f2c/language/scala/scala_existing_rule.go#L21-L24) is available out-of-the-box.

You may have your own scala rule macros that look like a `scala_library` or `scala_binary`, but have their own rule kind names and loads.  To register these rules/macros as  `scalaExistingRule` provider implementations, use the `-scala_existing_rule=LOAD%KIND` flag.  For example:

```bazel
gazelle(
    name = "gazelle",
    args = [
        "-scala_existing_rule=@io_bazel_rules_scala//scala:scala.bzl%_scala_library",
        "-scala_existing_rule=//bazel_tools:scala.bzl%scala_app",
        ...
    ],
    ...
)
```

This can then be instatiated as:

```bazel
# gazelle:scala_rule scala_app implementation //bazel_tools:scala.bzl%scala_app
# gazelle:scala_rule scala_app enabled false
```

This rule could then be selectively enabled/disabled in subpackages as follows:

```bazel
# gazelle:scala_rule scala_app enabled true
```

An advanced use-case would involve creating your own `Rule

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

