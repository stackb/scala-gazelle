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

# ----------------------------------------------------
# Antlr
# ----------------------------------------------------

load("@rules_antlr//antlr:repositories.bzl", "rules_antlr_dependencies")
load("@rules_antlr//antlr:lang.bzl", "GO")

rules_antlr_dependencies("4.8", GO)
