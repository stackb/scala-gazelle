
[![CI](https://github.com/stackb/scala-gazelle/actions/workflows/ci.yaml/badge.svg)](https://github.com/stackb/scala-gazelle/actions/workflows/ci.yaml) [![Go Reference](https://pkg.go.dev/badge/github.com/stackb/scala-gazelle.svg)](https://pkg.go.dev/github.com/stackb/scala-gazelle)

<table border="0">
  <tr>
    <td><img src="https://upload.wikimedia.org/wikipedia/en/thumb/7/7d/Bazel_logo.svg/1920px-Bazel_logo.svg.png" height="120"/></td>
    <td><img src="https://www.scala-lang.org/resources/img/frontpage/scala-spiral.png" height="120"/></td>
    <td><img src="https://user-images.githubusercontent.com/50580/141892423-5205bbfd-8487-442b-81c7-f56fa3d1f69e.jpeg" height="120"/></td>
  </tr>
  <tr>
    <td>bazel</td>
    <td>scala</td>
    <td>gazelle</td>
  </tr>
</table>

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
    - [`source`](#source)
    - [`maven`](#maven)
    - [`java`](#java)
    - [`protobuf`](#protobuf)
    - [Custom Known Import Provider](#custom-known-import-provider)
    - [CanProvide](#canprovide)
    - [Split Packages](#split-packages)
  - [Cache](#cache)
  - [Profiling](#profiling)
    - [CPU](#cpu)
    - [Memory](#memory)
  - [Directives](#directives)
    - [`gazelle:scala_rule`](#gazellescala_rule)
    - [`gazelle:resolve`](#gazelleresolve)
    - [`gazelle:resolve_with`](#gazelleresolve_with)
    - [`gazelle:resolve_kind_rewrite_name`](#gazelleresolve_kind_rewrite_name)
    - [`gazelle:annotate`](#gazelleannotate)
      - [`imports`](#imports)
      - [`unresolved_deps`](#unresolved_deps)
- [Import Resolution Procedure](#import-resolution-procedure)
  - [How Required Imports are Calculated](#how-required-imports-are-calculated)
  - [How Required Imports are Resolved](#how-required-imports-are-resolved)
- [Help](#help)

# Overview

This is an experimental gazelle extension for scala.  It has the following
design characteristics:

- It only works on scala rules that already exist in a `BUILD` file.  You are
  responsible for manually creating `scala_library`, `scala_binary`, and
  `scala_test` targets in their respective packages.
- It only manages compile-time scala `deps`; you are responsible for
  `runtime_deps`.
- Existing scala rules are evaluated for the contents of their `srcs`. Globs are
  interpreted the same as bazel starlark (unless there is a a bug 😱).
- Source files named in the `srcs` are parsed for their import statements and
  exportable symbols (classes, traits, objects, ...).
- Dependencies are resolved by matching required imports against their providing
  rule labels.  The resolution procedure is configurable.

# Installation

## Primary Dependency

Add `build_stack_scala_gazelle` as an external workspace:

```bazel
# Branch: master
# Commit: 476615560157f22919e5c936dc84b7ddf3278ca4
# Date: 2022-12-08 20:23:01 +0000 UTC
# URL: https://github.com/stackb/scala-gazelle/commit/476615560157f22919e5c936dc84b7ddf3278ca4
# 
# Add license (#53)
# Size: 149692 (150 kB)
http_archive(
    name = "build_stack_scala_gazelle",
    sha256 = "e903f248ba5921b5f6891ada8134760984749478ee350b11c64302205b04001a",
    strip_prefix = "scala-gazelle-476615560157f22919e5c936dc84b7ddf3278ca4",
    urls = ["https://github.com/stackb/scala-gazelle/archive/476615560157f22919e5c936dc84b7ddf3278ca4.tar.gz"],
)
```

> Update to latest, the version in the readme is probably out-of-date!

## Transitive Dependencies

Load corresponding transitive dependencies in your `WORKSPACE` as follows:

```bazel
load("@build_stack_scala_gazelle//:workspace_deps.bzl", "language_scala_deps")

language_scala_deps()

load("@build_stack_scala_gazelle//:go_repos.bzl", build_stack_scala_gazelle_gazelle_extension_deps = "gazelle_extension_deps")

build_stack_scala_gazelle_gazelle_extension_deps()
```

## Gazelle Binary

Include the language/scala extension in your `gazelle_binary` rule.  For
example:

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

The extension needs to know which rules it should manage (parse `srcs`/resolve
`deps`).  This is done using `gazelle:scala_rule` directives.

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

> This reads as "create a rule provider configuration named 'scala_library'
> whose provider implementation is registered under the name
> '@io_bazel_rules_scala//scala:scala.bzl%scala_library'

### Custom Existing Rule Providers

You may have your own scala rule macros that look like a `scala_library` or
`scala_binary`, but have their own rule kinds and loads.  To register these
rules/macros as provider implementations, use the
`-existing_scala_rule=LOAD%KIND` flag.  For example:

```bazel
gazelle(
    name = "gazelle",
    args = [
        "-existing_scala_rule=//bazel_tools:scala.bzl%scala_app",
        ...
    ],
    ...
)
```

```bazel
# gazelle:scala_rule scala_app implementation //bazel_tools:scala.bzl%scala_app
```

### Custom Rule Provider

An advanced use-case would involve writing your own `scalarule.Provider`
implementation.

To register it:

```go
import "github.com/stackb/scala-gazelle/pkg/scalarule"

func init() {
  scalarule.GlobalProviderRegistry().RegisterProvider(
    "@foo//rules/scala.bzl:my_scala_library",
    newMyScalaLibrary(),
  )
}
```

Enable the rule provider configuration:

```bazel
# gazelle:scala_rule my_scala_library implementation @foo//rules/scala.bzl:my_scala_library
```

## Known Import Providers

At the core of the import resolution process is a trie structure where the keys
of the trie are parts of an import statement and the values are
`*resolver.Symbol` structs.

For example, for the import `io.grpc.Status`, the trie would contain the
following:

- `io`: (`nil`)
  - `grpc` (type `PACKAGE`, from `@maven//:io_grpc_grpc_core`)
    - `Status` (type `CLASS`, from `@maven//:io_grpc_grpc_core`)

When resolving the import `io.grpc.Status.ALREADY_EXISTS`, the longest prefix
match would find the `io.grpc.Status` `CLASS` and the label
`@maven//:io_grpc_grpc_core` would be added to the rule `deps`.

The trie is populated by `resolver.SymbolProvider` implementations. Each
implementation provides symbols from a different data source.

Known import providers:

- Have a canonical name.
- Must be enabled with the `-scala_import_provider` flag.
- Manage their own flags; check the provider source code for details.

### `source`

The `source` provider is responsible for indexing importable symbols from
`.scala` source files during the rule generation phase.

Source files that are listed in the `srcs` of existing scala rules are parsed.
The discovered `object`, `class`, `trait` types are provided to the known import
trie such that they can be resolved by other rules.

The extension wouldn't do much without this provider, but it still needs to be
enabled in `args`:

```bazel
gazelle(
    name = "gazelle",
    args = [
        "-scala_import_provider=source",
    ],
)
```

### `maven`

This provider reads `maven_install.json` files that are produced from pinned
`maven_install` repository rules.

As of https://github.com/bazelbuild/rules_jvm_external/pull/716 (`Add index of
packages in jar files when pinning`), `@rules_jvm_external` indexes the package
names that jars provide.

The `maven` provider reads these package names and populates the trie
accordingly.  Note that since only package names are known, maven dependency
resolution via this mechanism alone is _coarse-grained_.

To configure the `maven` provider, use the `-maven_install_json_file` flag (can
be repeated if you have more than one `maven_install` rule):

```bazel
gazelle(
    name = "gazelle",
    args = [
        "-scala_import_provider=source",
        "-scala_import_provider=maven",
        "-maven_install_json_file=$(location //:maven_install.json)",
        "-maven_install_json_file=$(location //:artifactory_install.json)",
    ],
    data = [
        "//:maven_install.json",
        "//:artifactory_install.json",
    ],
)
```

### `java`

The `java` provider indexes symbols from java-related dependencies in the bazel
graph.  It relies on an index file produced by the `java_index` rule:

```bazel
load("@build_stack_scala_gazelle//rules:java_index.bzl", "java_index")

java_index(
    name = "java_index",
    jars = [
        "@maven//:io_grpc_grpc_context",
        "@maven//:io_grpc_grpc_core",
    ],
    out_json = "java_index.json",
    out_proto = "java_index.pb",
    platform_jars = ["@bazel_tools//tools/jdk:platformclasspath"],
)
```

> NOTE: Use `bazel build //:java_index --output_groups=json` to produce the JSON
> file if you want to inspect it.

The `jars` attribute names dependencies that you want indexed at a _fine-grained_
level.  Any label that provides `JavaInfo` will satisfy.

The `platform_jars` is special: it indexes jars that are provided by the
platform and do not need to be resolved to a label in rule `deps`.  For example,
if you import `java.util.Map`, no additional bazel label is required to use it.
The `@bazel_tools//tools/jdk:platformclasspath` is the bazel rule that supplies
these symbols.  You can also add things like
`@maven//:org_scala_lang_scala_library` or other toolchain-provided jars that
never need to be explicitly stated in `deps`.

To enable it:

```bazel
gazelle(
    name = "gazelle",
    args = [
        "-scala_import_provider=source",
        "-scala_import_provider=java",
        "-java_index_file=$(location //:java_index.pb)",
        # the flag order is significant: put fine-grained providers (java)
        # before coarse-grained ones (maven)
        "-scala_import_provider=maven",
        ...
    ],
    data = [
        "//:java_index.pb",
    ],
)
```

### `protobuf`

The `protobuf` providers works in conjuction with the
[stackb/rules_proto](https://github.com/stackb/rules_proto) gazelle extension.

That extension parses proto files and supplies scala imports for proto
`message`, `enum`, and `service` classes.

To resolve scala dependencies to protobuf rules, enable as follows:

```bazel
gazelle(
    name = "gazelle",
    args = [
        "-scala_import_provider=source",
        "-scala_import_provider=protobuf",
        ...
    ],
)
```

> TODO: provide an example repo showing the full configuration of these two
> extensions.

### Custom Known Import Provider

If your organization has an additional database or mechanism for import
tracking, you can implement the `resolver.SymbolProvider` interface and
register it with the global registry.

For example, if your organization uses https://github.com/johnynek/bazel-deps,
you might implement something like:

```go
package provider

import (
	"flag"
	"fmt"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"

	"github.com/stackb/scala-gazelle/pkg/collections"
	"github.com/stackb/scala-gazelle/pkg/resolver"
)

func init() {
  resolver.
    GlobalSymbolProviderRegistry().
    AddSymbolProvider(newBazelDepsProvider())
}

// bazelDepsProvider is a provider of symbols for the
// johnynek/bazel-deps.
type bazelDepsProvider struct {
	bazelDepsYAMLFiles collections.StringSlice
}

// newBazelDepsProvider constructs a new provider.
func newBazelDepsProvider() *bazelDepsProvider {
	return &bazelDepsProvider{}
}

// Name implements part of the resolver.Scope interface.
func (p *bazelDepsProvider) Name() string {
	return "bazel-deps"
}

// RegisterFlags implements part of the resolver.Scope interface.
func (p *bazelDepsProvider) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {
	fs.Var(&p.bazelDepsYAMLFiles, "bazel_deps_yaml_file", "path to bazel_deps.yaml")
}

// CheckFlags implements part of the resolver.Scope interface.
func (p *bazelDepsProvider) CheckFlags(fs *flag.FlagSet, c *config.Config, importRegistry resolver.Scope) error {
	for _, filename := range p.bazelDepsYAMLFiles {
		if err := p.loadFile(c.WorkDir, filename, importRegistry); err != nil {
			return err
		}
	}
	return nil
}

func (p *bazelDepsProvider) loadFile(dir string, filename string, importRegistry resolver.Scope) error {
	return fmt.Errorf("Implement me; Supply symbols to the given importRegistry!")
}

// CanProvide implements part of the resolver.Scope interface.
func (p *bazelDepsProvider) CanProvide(dep label.Label, knownRule func(from label.Label) (*rule.Rule, bool)) bool {
	if dep.Repo == "bazel_deps" {
		return true
	}
	return false
}

// OnResolve implements part of the resolver.Scope interface.
func (p *bazelDepsProvider) OnResolve() {
}
```

### CanProvide

The `resolver.Scope.CanProvide` function is used to determine if
this provider is capable of providing a given dependency label.  When rule deps
are resolved, the existing deps list is cleared of those labels it can find a
provider for.  For example, given the rule:

```bazel
scala_library(
  name = "lib",
  srcs = glob(["*.scala"]),
  deps = [
    "//src/main/scala:scala",
    "@foo//:scala",
    "@maven//:com_google_gson_gson",
  ],
)
```

The configured providers are checked to see which labels can be re-resolved. So,
the intermediate state of the rule before deps resolution actually happens looks
like:

```diff
scala_library(
  name = "lib",
  srcs = glob(["*.scala"]),
  deps = [
-    "//src/main/scala:scala",  # can be resolved to a source rule - delete it!
    "@foo//:scala",  # don't know anything about @foo - leave it alone!
-    "@maven//:com_google_gson_gson",  # can be resolved by maven provider - delete it!
  ],
)
```

So, if the scala-gazelle extension is not confident that a label can be
re-resolved, it will leave the dependency alone, even without `# keep`
directives.

### Split Packages

Issues can occur when more than one jar provides the same package name.  This
situation is known as a "split package".  The `io.grpc` namespace is a classic
example (see [discussion](https://github.com/grpc/grpc-java/issues/3522)).  The
`io.grpc.Context` is in `@maven//:io_grpc_grpc_context`, but other classes like
`io.grpc.Status` are in `@maven//:io_grpc_grpc_core`.  Both advertise the
package `io.grpc`.

To help avoid issues with split packages:

- Use the `java` provider to supply fine-grained deps for selected artifacts.
- Avoid wildcard imports that involve split packages.

## Cache

Parsing scala source files for a large repository is expensive.  A cache can be
enabled via the `-scala_gazelle_cache_file` flag.  If present, the extension
will read and write to this file.

```bazel
gazelle(
    name = "gazelle",
    args = [
        "-scala_gazelle_cache_file=${BUILD_WORKING_DIRECTORY}/.scala-gazelle-cache.pb",
    ],
)
```

The cache stores a sha256 hash of each source file; it will use cached state if
the hash matches the source file.

> - Environment variables are expanded.
> - To use a JSON cache (for example, to inspect it, change the extension to
> `.json`)
> - Bonus: the cache also records the total number of packages and enables a
> nice progress bar.

## Profiling

Gazelle can be slow for large repositories.  To get a better sense of what's
going on, cpu and memory profiling can be enabled:

```bazel
gazelle(
    name = "gazelle",
    args = [
        "-cpuprofile_file=./gazelle.cprof",
        "-memprofile_file=./gazelle.mprof",
    ],
)
```

### CPU

Use `bazel run @go_sdk//:bin/go -- tool pprof ./gazelle.cprof` to analyze it
(try the  commands `top10` or `web`).

### Memory

Use `bazel run @go_sdk//:bin/go -- tool mprof ./gazelle.mprof` to analyze it
(try commands `top10` or `web`)

## Directives

This extension supports the following directives:

### `gazelle:scala_rule`

Instantiates a named rule provider configuration (enabled by default once
instantiated):

```bazel
# gazelle:scala_rule scala_library implementation @io_bazel_rules_scala//scala:scala.bzl%scala_library
```

To enable/disable the configuration in a subpackage:

```bazel
# gazelle:scala_rule scala_library enabled false
# gazelle:scala_rule scala_library enabled true
```

### `gazelle:resolve`

This is the core gazelle directive not implemented here but is applicable to
this one.

Use something like the following to override dependency resolution to a
hard-coded label:

```bazel
# gazelle:resolve scala scala.util @maven//:org_scala_lang_scala_library
```

### `gazelle:resolve_with`

Use this directive to co-resolve dependencies that, while not explicitly stated
in the source file, are needed for compilation.  Example:

```bazel
# gazelle:resolve_with scala com.typesafe.scalalogging.LazyLogging org.slf4j.Logger
```

> This is referred to as an "implicit" dependency internally.

These are included transitively.

### `gazelle:resolve_kind_rewrite_name`

The `resolve_kind_rewrite_name` is required for the following scenario:

1. You have a custom existing rule implemented as a macro, for example
   `my_scala_app`.
2. The `my_scala_app` macro declares a "real" `scala_library` using a name like
   `%{name}_lib`.

In this case the extension would parse a `my_scala_app` rule at
`//src/main/scala/com/foo:scala`; other rules that import symbols from this rule
would resolve to `//src/main/scala/com/foo:scala`.  However, there is no such
actual `scala_library` at `:scala`, it really should be
`//src/main/scala/com/foo:scala_lib`.

This can be dealt with as follows:

```bazel
# gazelle:resolve_kind_rewrite_name my_scala_app %{name}_lib
```

This tells the extension _"if you find a rule with kind `my_scala_app`, rewrite
the label name to name + `"_lib"`, using the magic token `%{name}` as a
placeholder."_


### `gazelle:annotate`

The `annotate` directive is a debugging aid that adds comments to the generated
rules detailing what the symbols are and how they resolved.

#### `imports`

This adds a list of comments to the `srcs` attribute detailing the required
imports and how they resolved.  For example:

```
# gazelle:annotate imports
```

Generates:

```bazel
scala_binary(
    name = "app",
    srcs =
    # ❌ AbstractServiceBase<ERROR> import not found (EXTENDS of foo.allocation.Main)
    # ✅ akka.NotUsed<CLASS> @maven//:com_typesafe_akka_akka_actor_2_12<jarindex> (DIRECT of BusinessFlows.scala)
    # ✅ java.time.format.DateTimeFormatter<CLASS> NO-LABEL<java> (DIRECT of RequestHandler.scala)
    # ✅ scala.concurrent.ExecutionContext<PACKAGE> @maven//:org_scala_lang_scala_library<maven> (DIRECT of RequestHandler.scala)
    glob(["src/main/**/*.scala"]),
    main_class = "foo.allocation.Main",
)
```

#### `unresolved_deps`

Similar to above, but only the `import not found` would be included.

# Import Resolution Procedure

## How Required Imports are Calculated

The imports are calculated as a union of the following for all `.scala` source
files in the rule:

1. All imports explicitly named in an import statement (`DIRECT` import). Nested
   imports are included and resolved in a best-effort basis.
1. The `main_class`, if the rule has that attribute (`MAIN_CLASS` import).
1. All symbols named in an `extends` clause (`EXTENDS` import).
1. Any additional imports matching a `gazelle:resolve_with` directive
   (`IMPLICIT` import).

## How Required Imports are Resolved

The resolution procedure works as follows:

1. Is the import provided by the containing rule?  If yes, `NoLabel` is
   required.  Stop ✅.
2. Is the import named in a `gazelle:resolve` override?  If yes, stop ✅.
3. Does the import satisfy a longest prefix match in the known import trie?  If
   yes, stop ✅.
4. Does the gazelle "rule index" and "cross-resolve" mechanism find a result for
   the import?  If yes, stop ✅.
5. No label was found.  Mark as `import not found` and move on ❌. 

# Help

For general help, please raise an [github
issue](https://github.com/stackb/scala-gazelle/issues) or ask on the bazel slack
in the `#gazelle` channel.

If you need dedicated help integrating `scala-gazelle` into your repository or
want additional features, please reach out to `pcj@stack.build` to assist on a
part-time contractual basis.
