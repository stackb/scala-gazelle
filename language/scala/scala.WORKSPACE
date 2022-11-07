local_repository(
    name = "build_stack_scala_gazelle",
    path = "../build_stack_scala_gazelle",
)

load("@build_stack_scala_gazelle//:workspace_deps.bzl", "workspace_deps")

workspace_deps()

load("@rules_jvm_external//:defs.bzl", "maven_install")

maven_install(
    name = "maven",
    artifacts = [
        "xml-apis:xml-apis:1.4.01",
    ],
    generate_compat_repositories = True,
    # maven_install_json = "//:maven_install.json",
    repositories = [
        "https://repo.maven.apache.org/maven2/",
    ],
    version_conflict_policy = "pinned",
)

maven_install(
    name = "atlassian",
    artifacts = [
        "aopalliance:aopalliance:1.0",
    ],
    generate_compat_repositories = True,
    # maven_install_json = "//:atlassian_install.json",
    repositories = [
        "https://packages.atlassian.com/maven-external",
    ],
    version_conflict_policy = "pinned",
)

load("@maven//:compat.bzl", "compat_repositories")

compat_repositories()

# @atlassian-public workspace used to differentiate deps with @maven
maven_install(
    name = "atlassian-public",
    artifacts = [
        "junit:junit:4.9",
    ],
    repositories = ["https://packages.atlassian.com/maven-external"],
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
