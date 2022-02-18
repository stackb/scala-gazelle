workspace(name = "build_stack_scala_gazelle")

load(":workspace_deps.bzl", "workspace_deps")

workspace_deps()

# ----------------------------------------------------
# Go
# ----------------------------------------------------

load(
    "@io_bazel_rules_go//go:deps.bzl",
    "go_register_toolchains",
    "go_rules_dependencies",
)

go_rules_dependencies()

go_register_toolchains(version = "1.16.2")

# ----------------------------------------------------
# Gazelle
# ----------------------------------------------------
# gazelle:repository_macro go_repos.bzl%go_repositories

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies")

gazelle_dependencies()

# ----------------------------------------------------
# build_stack_rules_proto
# ----------------------------------------------------

register_toolchains("@build_stack_rules_proto//toolchain:standard")

load("@build_stack_rules_proto//:go_deps.bzl", "gazelle_protobuf_extension_go_deps")

gazelle_protobuf_extension_go_deps()

load("//:go_repos.bzl", "go_repositories")

go_repositories()

# ----------------------------------------------------
# Antlr
# ----------------------------------------------------

load("@rules_antlr//antlr:repositories.bzl", "rules_antlr_dependencies")
load("@rules_antlr//antlr:lang.bzl", "GO")

rules_antlr_dependencies("4.8", GO)

# ----------------------------------------------------
# Scala
# ----------------------------------------------------

load("@io_bazel_rules_scala//:scala_config.bzl", "scala_config")

scala_config(scala_version = "2.13.2")

load("@io_bazel_rules_scala//scala:scala.bzl", "scala_repositories")

scala_repositories()

load("@io_bazel_rules_scala//scala:toolchains.bzl", "scala_register_toolchains")

scala_register_toolchains()

# ----------------------------------------------------
# NodeJS
# ----------------------------------------------------

load("@build_bazel_rules_nodejs//:index.bzl", "node_repositories")

node_repositories()

register_toolchains("//tools/toolchains:nodejs")

# ----------------------------------------------------
# Maven
# ----------------------------------------------------

load("@rules_jvm_external//:defs.bzl", "maven_install")

# bazel run @maven_scala_gazelle//:pin, but first comment out the "maven_install_json"
# (put it back once pinned again)
maven_install(
    name = "maven_scala_gazelle",
    artifacts = [
        "org.scala-lang:scala-compiler:2.13.2",
        "org.parboiled:parboiled_2.13:2.3.0",
        "org.scalameta:scalameta_2.13:4.4.35",
        # "org.scalariform:scalariform:0.2.10",
    ],
    fetch_sources = True,
    # maven_install_json = "//:maven_install.json",
    repositories = ["https://repo1.maven.org/maven2"],
)

load("@maven_scala_gazelle//:defs.bzl", "pinned_maven_install")

pinned_maven_install()

# ----------------------------------------------------
# External proto
# ----------------------------------------------------

load("@build_stack_rules_proto//rules/proto:proto_repository.bzl", "proto_repository")

proto_repository(
    name = "protoapis",
    build_directives = [
        "gazelle:exclude testdata",
        "gazelle:exclude google/protobuf/compiler/ruby",
        "gazelle:proto_language go enable true",
    ],
    build_file_expunge = True,
    build_file_proto_mode = "file",
    cfgs = ["//:rules_proto_config.yaml"],
    deleted_files = [
        "google/protobuf/unittest_custom_options.proto",
        "google/protobuf/map_lite_unittest.proto",
        "google/protobuf/map_proto2_unittest.proto",
        "google/protobuf/test_messages_proto2.proto",
        "google/protobuf/test_messages_proto3.proto",
    ],
    strip_prefix = "protobuf-9650e9fe8f737efcad485c2a8e6e696186ae3862/src",
    type = "zip",
    urls = ["https://codeload.github.com/protocolbuffers/protobuf/zip/9650e9fe8f737efcad485c2a8e6e696186ae3862"],
)

# Commit: cd69fc97f6107f2e0d05ba5ce847fb43f043e781
# Date: 2020-03-08 06:37:42 +0000 UTC
# URL: https://github.com/scalapb/ScalaPB/commit/cd69fc97f6107f2e0d05ba5ce847fb43f043e781
#
# Setting version to 0.10.1
# Size: 548859 (549 kB)
# provides @scalapbapis//scalapb:scalapb_proto (scalapb/scalapb.proto)
SCALAPB_VERSION = "0.10.1"

proto_repository(
    name = "scalapbapis",
    build_directives = [
        "gazelle:proto_language go enable true",
    ],
    build_file_expunge = True,
    build_file_proto_mode = "package",
    cfgs = ["//:rules_proto_config.yaml"],
    sha256 = "78ba48f2a4de5de16b95dee2f8f29d00b30c54b9af6694c19d4fac9667e2ecc5",
    strip_prefix = "ScalaPB-%s/protobuf" % SCALAPB_VERSION,
    urls = ["https://github.com/scalapb/ScalaPB/archive/refs/tags/v%s.tar.gz" % SCALAPB_VERSION],
)
