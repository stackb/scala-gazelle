"""java_index.bzl provides the java_index rule.
"""

load(":providers.bzl", "JarIndexerAspectInfo")
load(":java_indexer_aspect.bzl", "jarindexer_action", "java_indexer_aspect")

def merge_action(ctx, output_file, jarindex_files):
    args = ctx.actions.args()
    args.use_param_file("@%s", use_always = False)
    args.add("--output_file", output_file)
    args.add_joined("--predefined", [target.label for target in ctx.attr.platform_deps], uniquify = True, join_with = ",")
    for pkg, dep in ctx.attr.preferred_deps.items():
        args.add("--preferred=%s=%s" % (pkg, dep))

    args.add_all(jarindex_files)

    ctx.actions.run(
        mnemonic = "MergeIndex",
        progress_message = "Merging jarindex files: " + str(ctx.label),
        executable = ctx.executable._mergeindex,
        arguments = [args],
        inputs = jarindex_files,
        outputs = [output_file],
    )

def jarindex_basename(ctx, label):
    return "-".join([
        ctx.label.name,
        label.workspace_name if label.workspace_name else "default",
        label.package if label.package else "_",
        label.name,
    ])

def _java_index_impl(ctx):
    # List[Depset[File]]
    transitive_jarindex_files = []

    # List[File]
    jarindex_files = []

    for dep in ctx.attr.deps + ctx.attr.platform_deps:
        if JarIndexerAspectInfo in dep:
            info = dep[JarIndexerAspectInfo]
            jarindex_files.extend(info.jar_index_files.to_list())
            transitive_jarindex_files.append(info.jar_index_files)

    for i, jar in enumerate(ctx.files.platform_deps):
        label = ctx.attr.platform_deps[i].label
        jarindex_files.append(jarindexer_action(ctx, label, "bootclasspath", ctx.executable._jarindexer, jar))

    output_proto = ctx.outputs.proto
    output_json = ctx.outputs.json

    jarindex_depset = depset(direct = jarindex_files, transitive = transitive_jarindex_files)
    files = jarindex_depset.to_list()
    merge_action(ctx, output_proto, files)
    merge_action(ctx, output_json, files)

    # List[File]
    direct_files = [output_proto]

    return [DefaultInfo(
        files = depset(direct_files),
    ), OutputGroupInfo(
        proto = [output_proto],
        json = [output_json],
        jarindex_depset = jarindex_depset,
    )]

java_index = rule(
    implementation = _java_index_impl,
    attrs = {
        "deps": attr.label_list(
            aspects = [java_indexer_aspect],
            doc = "list of java deps to be indexed",
        ),
        "platform_deps": attr.label_list(
            doc = "list of java labels to be indexed without a JarSpec.Label, typically [@bazel_tools//tools/jdk:platformclasspath]",
            allow_files = True,
        ),
        "preferred_deps": attr.string_dict(
            doc = "mapping of package name to label that should be used for dependency resolution",
        ),
        "_mergeindex": attr.label(
            default = Label("@build_stack_scala_gazelle//cmd/mergeindex"),
            cfg = "exec",
            executable = True,
            doc = "the mergeindex tool",
        ),
        "_jarindexer": attr.label(
            default = Label("@build_stack_scala_gazelle//cmd/jarindexer:jarindexer_bin"),
            cfg = "exec",
            executable = True,
            doc = "the jarindexer tool",
        ),
        "_ijar": attr.label(
            default = Label("@bazel_tools//tools/jdk:ijar"),
            executable = True,
            cfg = "exec",
            allow_files = True,
            doc = "the ijar tool",
        ),
    },
    outputs = {
        "proto": "%{name}.pb",
        "json": "%{name}.json",
    },
)
