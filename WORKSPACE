workspace(name = "build_stack_scala_gazelle")

load(":workspace_deps.bzl", "workspace_deps")

workspace_deps()

# ----------------------------------------------------
# @rules_proto
# ----------------------------------------------------

load("@rules_proto//proto:repositories.bzl", "rules_proto_dependencies", "rules_proto_toolchains")

rules_proto_dependencies()

rules_proto_toolchains()

# ----------------------------------------------------
# Go
# ----------------------------------------------------

load(
    "@io_bazel_rules_go//go:deps.bzl",
    "go_register_toolchains",
    "go_rules_dependencies",
)

go_rules_dependencies()

go_register_toolchains(version = "1.18.2")

# ----------------------------------------------------
# Gazelle
# ----------------------------------------------------
# gazelle:repository_macro go_repos.bzl%go_repositories

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")

gazelle_dependencies()

# defining this in the WORKSPACE despite 'bazel query //external:org_golang_google_grpc --output build' saying it's
# still coming from go_repos.bzl.  I don't understand this one...  Without it org_golang_google_grpc
# is falling back to 1.27.0.
go_repository(
    name = "org_golang_google_grpc",
    build_file_proto_mode = "disable_global",
    importpath = "google.golang.org/grpc",
    sum = "h1:XT2/MFpuPFsEX2fWh3YQtHkZ+WYZFQRfaUgLZYj/p6A=",
    version = "v1.42.0",
)

# ----------------------------------------------------
# @build_stack_rules_proto
# ----------------------------------------------------

register_toolchains("@build_stack_rules_proto//toolchain:standard")

load("@build_stack_rules_proto//:go_deps.bzl", "gazelle_protobuf_extension_go_deps")

gazelle_protobuf_extension_go_deps()

# ----------------------------------------------------
# Go
# ----------------------------------------------------

load("@build_stack_rules_proto//deps:go_core_deps.bzl", "go_core_deps")

go_core_deps()

load("//:go_repos.bzl", "go_repositories")

go_repositories()

# ----------------------------------------------------
# NodeJS
# ----------------------------------------------------

load("@build_bazel_rules_nodejs//:index.bzl", "node_repositories")

node_repositories()

register_toolchains("//tools/toolchains:nodejs")

# ----------------------------------------------------
# Maven
#
# Note: maven dependencies should only be required for
# tests.
# ----------------------------------------------------

# load("@com_google_protobuf//:protobuf_deps.bzl", "PROTOBUF_MAVEN_ARTIFACTS", "protobuf_deps")

# protobuf_deps()

load("@rules_jvm_external//:defs.bzl", "maven_install")

maven_install(
    artifacts = [
        "com.google.caliper:caliper:1.0-beta-3",
        "com.google.code.findbugs:jsr305:3.0.2",
        "com.google.code.gson:gson:2.8.9",
        "com.google.errorprone:error_prone_annotations:2.5.1",
        "com.google.j2objc:j2objc-annotations:1.3",
        "com.google.guava:guava:31.1-jre",
        "com.google.guava:guava-testlib:31.1-jre",
        "com.google.truth:truth:1.1.2",
        "junit:junit:4.13.2",
        "org.mockito:mockito-core:4.3.1",
    ],
    # maven_install_json = "//:maven_install.json",
    repositories = [
        "https://repo1.maven.org/maven2",
        "https://repo.maven.apache.org/maven2",
    ],
)

load("@maven//:defs.bzl", "pinned_maven_install")

pinned_maven_install()

bind(
    name = "error_prone_annotations",
    actual = "@maven//:com_google_errorprone_error_prone_annotations",
)

bind(
    name = "gson",
    actual = "@maven//:com_google_code_gson_gson",
)

bind(
    name = "guava",
    actual = "@maven//:com_google_guava_guava",
)

# ----------------------------------------------------
# Scala
# ----------------------------------------------------

load("@io_bazel_rules_scala//:scala_config.bzl", "scala_config")

scala_config(scala_version = "2.13.2")

load("@io_bazel_rules_scala//scala:scala.bzl", "scala_repositories")

scala_repositories()

load("@io_bazel_rules_scala//scala:toolchains.bzl", "scala_register_toolchains")

scala_register_toolchains()
