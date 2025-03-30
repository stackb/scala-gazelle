"""workspace_deps.bzl declares dependencies for the workspace
"""

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive", "http_jar")

def _maybe(repo_rule, name, **kwargs):
    if name not in native.existing_rules():
        repo_rule(name = name, **kwargs)

def language_scala_deps():
    """language_scala_deps loads a subset of dependencies

    when @build_stack_scala_gazelle//language/scala is used from another
    repository.
    """
    protobuf_java_jar()
    classgraph_jar()
    scalameta_parsers()
    node_binaries()

def workspace_deps():
    """workspace_deps loads all dependencies for the workspace
    """
    rules_proto()  # via <TOP>
    io_bazel_rules_go()  # via bazel_gazelle
    language_scala_deps()
    bazel_gazelle()  # via <TOP>
    build_stack_rules_proto()
    rules_jvm_external()
    io_bazel_rules_scala()
    protobuf_core_deps()
    hermetic_cc_toolchain()
    plantuml_jar()

def protobuf_core_deps():
    bazel_skylib()  # via com_google_protobuf
    rules_python()  # via com_google_protobuf
    zlib()  # via com_google_protobuf
    com_google_protobuf()  # via <TOP>

def io_bazel_rules_go():
    # Release: v0.35.0
    # TargetCommitish: release-0.35
    # Date: 2022-09-11 15:59:49 +0000 UTC
    # URL: https://github.com/bazelbuild/rules_go/releases/tag/v0.35.0
    # Size: 931734 (932 kB)
    _maybe(
        http_archive,
        name = "io_bazel_rules_go",
        sha256 = "cc027f11f98aef8bc52c472ced0714994507a16ccd3a0820b2df2d6db695facd",
        strip_prefix = "rules_go-0.35.0",
        urls = ["https://github.com/bazelbuild/rules_go/archive/v0.35.0.tar.gz"],
    )

def scalameta_parsers():
    _maybe(
        http_archive,
        name = "scalameta_parsers",
        sha256 = "c419383f9fe63da14104416cfc4ba3200d52ad6bddc1d0a9a2058c2a4349f691",
        strip_prefix = "package",
        build_file_content = """
filegroup(
    name = "module",
    srcs = ["main.js"],
    visibility = ["//visibility:public"],
)
        """,
        urls = ["https://registry.npmjs.org/scalameta-parsers/-/scalameta-parsers-4.13.4.tgz"],
    )

def bazel_gazelle():
    # Branch: master
    # Commit: 2d1002926dd160e4c787c1b7ecc60fb7d39b97dc
    # Date: 2022-11-14 04:43:02 +0000 UTC
    # URL: https://github.com/bazelbuild/bazel-gazelle/commit/2d1002926dd160e4c787c1b7ecc60fb7d39b97dc
    #
    # fix updateStmt makeslice panic (#1371)
    # Size: 1859745 (1.9 MB)
    _maybe(
        http_archive,
        name = "bazel_gazelle",
        patch_args = ["-p1"],
        patches = ["@build_stack_scala_gazelle//third_party/bazelbuild/bazel-gazelle:pr-1394.patch"],
        sha256 = "5ebc984c7be67a317175a9527ea1fb027c67f0b57bb0c990bac348186195f1ba",
        strip_prefix = "bazel-gazelle-2d1002926dd160e4c787c1b7ecc60fb7d39b97dc",
        urls = ["https://github.com/bazelbuild/bazel-gazelle/archive/2d1002926dd160e4c787c1b7ecc60fb7d39b97dc.tar.gz"],
    )

def local_bazel_gazelle():
    _maybe(
        native.local_repository,
        name = "bazel_gazelle",
        path = "/Users/i868039/go/src/github.com/bazelbuild/bazel-gazelle",
    )

def rules_proto():
    # Commit: f7a30f6f80006b591fa7c437fe5a951eb10bcbcf
    # Date: 2021-02-09 14:25:06 +0000 UTC
    # URL: https://github.com/bazelbuild/rules_proto/commit/f7a30f6f80006b591fa7c437fe5a951eb10bcbcf
    #
    # Merge pull request #77 from Yannic/proto_descriptor_set_rule
    #
    # Create proto_descriptor_set
    # Size: 14397 (14 kB)
    _maybe(
        http_archive,
        name = "rules_proto",
        sha256 = "9fc210a34f0f9e7cc31598d109b5d069ef44911a82f507d5a88716db171615a8",
        strip_prefix = "rules_proto-f7a30f6f80006b591fa7c437fe5a951eb10bcbcf",
        urls = ["https://github.com/bazelbuild/rules_proto/archive/f7a30f6f80006b591fa7c437fe5a951eb10bcbcf.tar.gz"],
    )

def build_stack_rules_proto():
    # Branch: master
    # Commit: aa380e4421057b35228544bc234f816bb6b72c1c
    # Date: 2022-12-08 05:19:32 +0000 UTC
    # URL: https://github.com/stackb/rules_proto/commit/aa380e4421057b35228544bc234f816bb6b72c1c
    #
    # use distinct impLang for scala proto exports (#304)
    #
    # * use distinct impLang for scala proto exports
    # * fix test
    # Size: 2074364 (2.1 MB)
    http_archive(
        name = "build_stack_rules_proto",
        sha256 = "820dc71f2e265a50104671d323caba53790dfe20e9f7249a0e6beeaee39b4597",
        strip_prefix = "rules_proto-aa380e4421057b35228544bc234f816bb6b72c1c",
        urls = ["https://github.com/stackb/rules_proto/archive/aa380e4421057b35228544bc234f816bb6b72c1c.tar.gz"],
    )

def rules_jvm_external():
    _maybe(
        http_archive,
        name = "rules_jvm_external",
        sha256 = "b17d7388feb9bfa7f2fa09031b32707df529f26c91ab9e5d909eb1676badd9a6",
        strip_prefix = "rules_jvm_external-4.5",
        urls = [
            "https://github.com/bazelbuild/rules_jvm_external/archive/4.5.zip",
        ],
    )

def io_bazel_rules_scala():
    _maybe(
        http_archive,
        name = "io_bazel_rules_scala",
        sha256 = "0701ee4e1cfd59702d780acde907ac657752fbb5c7d08a0ec6f58ebea8cd0efb",
        strip_prefix = "rules_scala-2437e40131072cadc1628726775ff00fa3941a4a",
        urls = [
            "https://github.com/bazelbuild/rules_scala/archive/2437e40131072cadc1628726775ff00fa3941a4a.tar.gz",
        ],
    )

def classgraph_jar():
    # bzl use jar https://repo1.maven.org/maven2/io/github/classgraph/classgraph/4.8.149/classgraph-4.8.149.jar
    # Last-Modified: Wed, 06 Jul 2022 04:30:32 GMT
    # X-Checksum-Md5: 7fca2eb70908395af9ac43858b428c35
    # X-Checksum-Sha1: 4bc2f188bc9001473d4a26ac488c2ae1a3e906de
    # Size: 558272 (558 kB)
    http_jar(
        name = "classgraph_jar",
        sha256 = "ece8abfe1277450a8b95e57fc56991dca1fd42ffefdad88f65fe171ac576f604",
        url = "https://repo1.maven.org/maven2/io/github/classgraph/classgraph/4.8.149/classgraph-4.8.149.jar",
    )

def protobuf_java_jar():
    # bzl use jar https://repo1.maven.org/maven2/com/google/protobuf/protobuf-java/3.21.8/protobuf-java-3.21.8.jar
    # Last-Modified: Tue, 18 Oct 2022 19:48:19 GMT
    # X-Checksum-Md5: 39d238b47a0278795884e92e1c966796
    # X-Checksum-Sha1: 2a1eebb74b844d9ccdf1d22eb2f57cec709698a9
    # Size: 1671407 (1.7 MB)
    http_jar(
        name = "protobuf_java_jar",
        sha256 = "0b8581ad810d2dfaefd0dcfbf1569b1450448650238d7e2fd6b176c932d08c95",
        url = "https://repo1.maven.org/maven2/com/google/protobuf/protobuf-java/3.21.8/protobuf-java-3.21.8.jar",
    )

def bazel_skylib():
    _maybe(
        http_archive,
        name = "bazel_skylib",
        sha256 = "ebdf850bfef28d923a2cc67ddca86355a449b5e4f38b0a70e584dc24e5984aa6",
        strip_prefix = "bazel-skylib-f80bc733d4b9f83d427ce3442be2e07427b2cc8d",
        urls = [
            "https://github.com/bazelbuild/bazel-skylib/archive/f80bc733d4b9f83d427ce3442be2e07427b2cc8d.tar.gz",
        ],
    )

def rules_python():
    _maybe(
        http_archive,
        name = "rules_python",
        sha256 = "8cc0ad31c8fc699a49ad31628273529ef8929ded0a0859a3d841ce711a9a90d5",
        strip_prefix = "rules_python-c7e068d38e2fec1d899e1c150e372f205c220e27",
        urls = [
            "https://github.com/bazelbuild/rules_python/archive/c7e068d38e2fec1d899e1c150e372f205c220e27.tar.gz",
        ],
    )

def zlib():
    _maybe(
        http_archive,
        name = "zlib",
        sha256 = "c3e5e9fdd5004dcb542feda5ee4f0ff0744628baf8ed2dd5d66f8ca1197cb1a1",
        strip_prefix = "zlib-1.2.11",
        urls = [
            "https://mirror.bazel.build/zlib.net/zlib-1.2.11.tar.gz",
            "https://zlib.net/zlib-1.2.11.tar.gz",
        ],
        build_file = "@build_stack_rules_proto//third_party:zlib.BUILD",
    )

def com_google_protobuf():
    _maybe(
        http_archive,
        name = "com_google_protobuf",
        sha256 = "d0f5f605d0d656007ce6c8b5a82df3037e1d8fe8b121ed42e536f569dec16113",
        strip_prefix = "protobuf-3.14.0",
        urls = [
            "https://github.com/protocolbuffers/protobuf/archive/v3.14.0.tar.gz",
        ],
    )

def node_binaries():
    # see https://nodejs.org/dist/ to update
    versions = {
        "linux-x64": struct(
            executable = "bin/node",
            sha256 = "1a42a67beb3e07289da2ad22a58717801c6ab80d09668e2da6b1c537b2a80a5e",
            type = "tar.gz",
            version = "v19.1.0",
        ),
        "darwin-arm64": struct(
            executable = "bin/node",
            sha256 = "d05a4a3c9f081c7fbab131f447714fa708328c5c1634c278716adfbdbae0ff26",
            type = "tar.gz",
            version = "v19.1.0",
        ),
        "darwin-x64": struct(
            executable = "bin/node",
            sha256 = "63f4284fa1474b779f0e4fa93985ddc2efa227484476f33d923ae44922637080",
            type = "tar.gz",
            version = "v19.1.0",
        ),
        "win-x64": struct(
            executable = "node.exe",
            sha256 = "9ca998da2063fd5b374dc889ee1937ada5a1e1f4fb50b5f989412dda7c6bb357",
            type = "zip",
            version = "v19.1.0",
        ),
    }
    for os_arch, data in versions.items():
        url = "https://nodejs.org/dist/{version}/node-{version}-{os_arch}.{type}".format(
            os_arch = os_arch,
            type = data.type,
            version = data.version,
        )
        _maybe(
            http_archive,
            name = "node_bin_" + os_arch,
            urls = [url],
            sha256 = data.sha256,
            strip_prefix = "node-{version}-{os_arch}".format(
                os_arch = os_arch,
                version = data.version,
            ),
            type = data.type,
            build_file_content = """
filegroup(
    name = "node",
    srcs = ["{executable}"],
    visibility = ["//visibility:public"],
)
            """.format(executable = data.executable),
        )

def hermetic_cc_toolchain():
    HERMETIC_CC_TOOLCHAIN_VERSION = "v3.1.0"
    _maybe(
        http_archive,
        name = "hermetic_cc_toolchain",
        sha256 = "df091afc25d73b0948ed371d3d61beef29447f690508e02bc24e7001ccc12d38",
        urls = [
            "https://mirror.bazel.build/github.com/uber/hermetic_cc_toolchain/releases/download/{0}/hermetic_cc_toolchain-{0}.tar.gz".format(HERMETIC_CC_TOOLCHAIN_VERSION),
            "https://github.com/uber/hermetic_cc_toolchain/releases/download/{0}/hermetic_cc_toolchain-{0}.tar.gz".format(HERMETIC_CC_TOOLCHAIN_VERSION),
        ],
    )

def plantuml_jar():
    _maybe(
        http_jar,
        name = "plantuml_jar",
        url = "https://github.com/plantuml/plantuml/releases/download/v1.2024.6/plantuml-1.2024.6.jar",
        sha256 = "5a8dc3b37fe133a4744e55be80caf6080a70350aba716d95400a0f0cbd79e846",
    )
