"""java_index.bzl provides the java_index rule.
"""

load(":providers.bzl", "JarIndexerAspectInfo")
load(":java_indexer_aspect.bzl", "java_indexer_aspect")

def build_jarfile_index(ctx, label, basename, jar):
    """Builds a single jarfile index.

    Args:
        ctx: the context object
        label: label to use for the jar
        basename: a string representing a filename prefix for generated files
        jar: the File to parse
    Returns:
        the output File of the index.
    """

    ijar = ctx.actions.declare_file(jar.short_path.replace("/", "-"))
    ctx.actions.run(
        executable = ctx.executable._ijar,
        inputs = [jar],
        outputs = [ijar],
        arguments = [
            "--target_label",
            str(label),
            jar.path,
            ijar.path,
        ],
        mnemonic = "Ijar",
    )

    output_file = ctx.actions.declare_file(basename + ".jarindex.bin")
    print("indexing jar %s -> %s" % (label, output_file.path))
    ctx.actions.run(
        mnemonic = "JarIndexer",
        progress_message = "Indexing jar " + ijar.basename,
        executable = ctx.executable._jarindexer,
        arguments = [
            "--label",
            str(label),
            "--output_file",
            output_file.path,
            jar.path,
        ],
        inputs = [ijar, jar],
        outputs = [output_file],
    )

    return output_file

def build_mergeindex(ctx, output_file, jarindex_files):
    """Builds the merged index for all jarindexes.

    Args:
        ctx: the context object
        output_file: the output File of the merged file.
        jarindex_files: a sequence of File representing the jarindex files
    Returns:
    """

    args = ctx.actions.args()
    args.use_param_file("@%s", use_always = False)
    args.add("--output_file", output_file)
    args.add_joined("--predefined", [str(Label(lbl)) for lbl in ctx.attr.predefined], uniquify = True, join_with = ",")
    args.add_joined("--preferred", [str(Label(lbl)) for lbl in ctx.attr.preferred], uniquify = True, join_with = ",")
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

def _jar_class_index_impl(ctx):
    """Implementation that collects symbols from jars."""

    # List[File]
    direct_files = []

    # List[Depset[File]]
    transitive_jarindex_files = []

    # List[File]
    jarindex_files = []

    for dep in ctx.attr.deps:
        info = dep[JarIndexerAspectInfo]
        jarindex_files.extend(info.jar_index_files.to_list())
        transitive_jarindex_files += [info.info_file, info.jar_index_files]

    for i, jar in enumerate(ctx.files.jars):
        label = ctx.attr.jars[i].label
        basename = jarindex_basename(ctx, label)
        jarindex_files.append(build_jarfile_index(ctx, label, basename, jar))

    for i, jar in enumerate(ctx.files.platform_jars):
        label = ctx.attr.platform_jars[i].label
        basename = jarindex_basename(ctx, label)
        jarindex_files.append(build_jarfile_index(ctx, label, basename, jar))

    output_proto = ctx.actions.declare_file(ctx.label.name + ".jarindex.bin")
    output_json = ctx.actions.declare_file(ctx.label.name + ".jarindex.json")

    build_mergeindex(ctx, output_proto, jarindex_files)
    build_mergeindex(ctx, output_json, jarindex_files)

    direct_files.append(output_proto)

    return [DefaultInfo(
        files = depset(direct_files),
    ), OutputGroupInfo(
        index_proto = [output_proto],
        index_json = [output_json],
        jarindex_files = depset(transitive = transitive_jarindex_files),
    )]

jar_class_index = rule(
    implementation = _jar_class_index_impl,
    attrs = {
        "deps": attr.label_list(
            # TODO(pcj): make JavaInfo a requirement here?  Currently the aspect looks for it if present.
            # providers = [JavaInfo],
            aspects = [java_indexer_aspect],
            doc = "list of java deps to be indexed",
        ),
        "jars": attr.label_list(
            aspects = [java_indexer_aspect],
            doc = "list of jars to be indexed",
        ),
        "platform_jars": attr.label_list(
            doc = "list of jar files to be indexed without a JarSpec.Label, typically [@bazel_tools//tools/jdk:platformclasspath]",
            allow_files = True,
        ),
        "predefined": attr.string_list(
            doc = "list of labels that do not need to be included in deps",
        ),
        "preferred": attr.string_list(
            doc = """A list of labels that should be chosen in the case of a resolve ambiguity.
E.g. ["@maven//:io_grpc_grpc_api"] means, "in the case where io.grpc.CallCredentials resolves to multiple labels, always choose @maven//:io_grpc_grpc_api"
""",
        ),
        "_mergeindex": attr.label(
            default = Label("@build_stack_scala_gazelle//cmd/mergeindex"),
            cfg = "exec",
            executable = True,
        ),
        "_jarindexer": attr.label(
            default = Label("@build_stack_scala_gazelle//cmd/jarindexer:jarindexer_bin"),
            cfg = "exec",
            executable = True,
        ),
        "_ijar": attr.label(
            default = Label("@bazel_tools//tools/jdk:ijar"),
            executable = True,
            cfg = "exec",
            allow_files = True,
        ),
    },
)
