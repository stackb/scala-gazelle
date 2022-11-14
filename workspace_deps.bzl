load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive", "http_file", "http_jar")

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
    node_bin_darwin_x64()
    node_bin_darwin_arm64()
    node_bin_linux_x64()
    node_bin_windows_x64()

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
    viz_js_lite()

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
        sha256 = "661081f106ebdc9592543223887de999d2a2b6229bd1aa22b1376ba6b695675d",
        strip_prefix = "package",
        build_file_content = """
filegroup(
    name = "module",
    srcs = ["index.js"],
    visibility = ["//visibility:public"],
)
        """,
        urls = ["https://registry.npmjs.org/scalameta-parsers/-/scalameta-parsers-4.4.17.tgz"],
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
        patches = ["//third_party/bazelbuild/bazel-gazelle:rule-attrassignment-api.patch"],
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
    # Release: v2.0.1
    # TargetCommitish: master
    # Date: 2022-10-20 02:38:27 +0000 UTC
    # URL: https://github.com/stackb/rules_proto/releases/tag/v2.0.1
    # Size: 2071295 (2.1 MB)
    http_archive(
        name = "build_stack_rules_proto",
        sha256 = "ac7e2966a78660e83e1ba84a06db6eda9a7659a841b6a7fd93028cd8757afbfb",
        strip_prefix = "rules_proto-2.0.1",
        urls = ["https://github.com/stackb/rules_proto/archive/v2.0.1.tar.gz"],
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

def viz_js_lite():
    # HTTP/2.0 200 OK
    # Content-Type: application/javascript; charset=utf-8
    # Date: Sat, 26 Mar 2022 00:35:46 GMT
    # Last-Modified: Mon, 04 May 2020 16:17:44 GMT
    # Size: 1439383 (1.4 MB)
    _maybe(
        http_file,
        name = "cdnjs_cloudflare_com_ajax_libs_viz_js_2_1_2_lite_render_js",
        sha256 = "1344fd45812f33abcb3de9857ebfdd599e57f49e3d0849841e75e28be1dd6959",
        urls = ["https://cdnjs.cloudflare.com/ajax/libs/viz.js/2.1.2/lite.render.js"],
    )

def bun_darwin():
    _maybe(
        http_archive,
        name = "bun_darwin",
        sha256 = "8976309239260f8089377980cf9399e99a6e352f22878b59fc9804e7a8b98b7b",
        strip_prefix = "bun-darwin-x64",
        build_file_content = """
filegroup(
    name = "bin",
    srcs = ["bun"],
    visibility = ["//visibility:public"],
)
        """,
        urls = [
            "https://github.com/oven-sh/bun/releases/download/bun-v0.2.1/bun-darwin-x64.zip",
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

def node_bin_darwin_x64():
    _maybe(
        http_archive,
        name = "node_bin_darwin_x64",
        urls = ["https://nodejs.org/dist/latest/node-v19.1.0-darwin-x64.tar.gz"],
        sha256 = "63f4284fa1474b779f0e4fa93985ddc2efa227484476f33d923ae44922637080",
        strip_prefix = "node-v19.1.0-darwin-x64",
        build_file_content = """
filegroup(
    name = "node",
    srcs = ["bin/node"],
    visibility = ["//visibility:public"],
)
        """,
    )

def node_bin_darwin_arm64():
    _maybe(
        http_archive,
        name = "node_bin_darwin_arm64",
        urls = ["https://nodejs.org/dist/latest/node-v19.1.0-darwin-arm64.tar.gz"],
        sha256 = "d05a4a3c9f081c7fbab131f447714fa708328c5c1634c278716adfbdbae0ff26",
        strip_prefix = "node-v19.1.0-darwin-arm64",
        build_file_content = """
filegroup(
    name = "node",
    srcs = ["bin/node"],
    visibility = ["//visibility:public"],
)
        """,
    )

def node_bin_linux_x64():
    _maybe(
        http_archive,
        name = "node_bin_linux_x64",
        urls = ["https://nodejs.org/dist/latest/node-v19.1.0-linux-x64.tar.gz"],
        sha256 = "1a42a67beb3e07289da2ad22a58717801c6ab80d09668e2da6b1c537b2a80a5e",
        strip_prefix = "node-v19.1.0-linux-x64",
        build_file_content = """
filegroup(
    name = "node",
    srcs = ["bin/node"],
    visibility = ["//visibility:public"],
)
        """,
    )

def node_bin_windows_x64():
    _maybe(
        http_archive,
        name = "node_bin_windows_x64",
        urls = ["https://nodejs.org/dist/latest/node-v19.1.0-win-x64.zip"],
        sha256 = "9ca998da2063fd5b374dc889ee1937ada5a1e1f4fb50b5f989412dda7c6bb357",
        strip_prefix = "node-v19.1.0-win-x64",
        build_file_content = """
filegroup(
    name = "node",
    srcs = ["node.exe"],
    visibility = ["//visibility:public"],
)
        """,
    )
