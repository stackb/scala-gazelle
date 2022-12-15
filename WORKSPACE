workspace(name = "build_stack_scala_gazelle")

load(":workspace_deps.bzl", "workspace_deps")

workspace_deps()

# ----------------------------------------------------
# @rules_proto
# ----------------------------------------------------

load("@rules_proto//proto:repositories.bzl", "rules_proto_dependencies")

rules_proto_dependencies()

# ----------------------------------------------------
# @io_bazel_rules_go
# ----------------------------------------------------

load(
    "@io_bazel_rules_go//go:deps.bzl",
    "go_register_toolchains",
    "go_rules_dependencies",
)

go_rules_dependencies()

go_register_toolchains(version = "1.18.2")

# ----------------------------------------------------
# @bazel_gazelle
# ----------------------------------------------------
# gazelle:repository_macro go_repos.bzl%go_repositories

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")

gazelle_dependencies()

# ----------------------------------------------------
# @build_stack_rules_proto
# ----------------------------------------------------

register_toolchains("@build_stack_rules_proto//toolchain:standard")

# defining go_repository for @org_golang_google_grpc this in the WORKSPACE
# It must occur before gazelle_protobuf_extension_go_deps() macro call.
# I don't understand this one...  Despite 'bazel query //external:org_golang_google_grpc --output build'
# saying it's still coming from go_repos.bzl, that does not appear to be the case.
# Without this override org_golang_google_grpc is falling back to 1.27.0.
go_repository(
    name = "org_golang_google_grpc",
    build_file_proto_mode = "disable_global",
    importpath = "google.golang.org/grpc",
    sum = "h1:XT2/MFpuPFsEX2fWh3YQtHkZ+WYZFQRfaUgLZYj/p6A=",
    version = "v1.42.0",
)

load("@build_stack_rules_proto//:go_deps.bzl", "gazelle_protobuf_extension_go_deps")

gazelle_protobuf_extension_go_deps()

load("@build_stack_rules_proto//deps:go_core_deps.bzl", "go_core_deps")

go_core_deps()

load("//:go_repos.bzl", "go_repositories")

go_repositories()

# ----------------------------------------------------
# @maven
#
# Note: maven dependencies should only be required for
# tests.
# ----------------------------------------------------

load("@rules_jvm_external//:defs.bzl", "maven_install")

maven_install(
    artifacts = [
        "com.google.code.gson:gson:2.8.9",
        "com.google.errorprone:error_prone_annotations:2.11.0",
        "com.google.guava:guava:30.1.1-jre",
    ],
    maven_install_json = "//:maven_install.json",
    repositories = [
        "https://repo1.maven.org/maven2",
        "https://repo.maven.apache.org/maven2",
    ],
)

load("@maven//:defs.bzl", "pinned_maven_install")

pinned_maven_install()

# required by @com_google_protobuf//java/util:util
bind(
    name = "error_prone_annotations",
    actual = "@maven//:com_google_errorprone_error_prone_annotations",
)

# required by @com_google_protobuf//java/util:util
bind(
    name = "gson",
    actual = "@maven//:com_google_code_gson_gson",
)

# required by @com_google_protobuf//java/util:util
bind(
    name = "guava",
    actual = "@maven//:com_google_guava_guava",
)

# ----------------------------------------------------
# @io_bazel_rules_scala
# ----------------------------------------------------

load("@io_bazel_rules_scala//:scala_config.bzl", "scala_config")

scala_config(scala_version = "2.13.2")

load("@io_bazel_rules_scala//scala:scala.bzl", "scala_repositories")

scala_repositories()

load("@io_bazel_rules_scala//scala:toolchains.bzl", "scala_register_toolchains")

scala_register_toolchains()
