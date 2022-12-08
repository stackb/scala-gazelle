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

def build_mergeindex(ctx, output_file, javaindex_files):
    """Builds the merged index for all jarindexes.

    Args:
        ctx: the context object
        output_file: the output File of the merged file.
        javaindex_files: a sequence of File representing the jarindex files
    Returns:
    """

    args = ctx.actions.args()
    args.use_param_file("@%s", use_always = False)
    args.add("--output_file", output_file)
    args.add_joined("--predefined", [target.label for target in ctx.attr.platform_jars], uniquify = True, join_with = ",")
    args.add_all(javaindex_files)

    ctx.actions.run(
        mnemonic = "MergeIndex",
        progress_message = "Merging jarindex files: " + str(ctx.label),
        executable = ctx.executable._mergeindex,
        arguments = [args],
        inputs = javaindex_files,
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
    """Implementation that collects symbols from jars."""

    # List[File]
    direct_files = []

    # List[Depset[File]]
    transitive_javaindex_files = []

    # List[File]
    javaindex_files = []

    for dep in ctx.attr.deps:
        info = dep[JarIndexerAspectInfo]
        javaindex_files.extend(info.jar_index_files.to_list())
        transitive_javaindex_files += [info.info_file, info.jar_index_files]

    for i, jar in enumerate(ctx.files.jars):
        label = ctx.attr.jars[i].label
        basename = jarindex_basename(ctx, label)
        javaindex_files.append(build_jarfile_index(ctx, label, basename, jar))

    for i, jar in enumerate(ctx.files.platform_jars):
        label = ctx.attr.platform_jars[i].label
        basename = jarindex_basename(ctx, label)
        javaindex_files.append(build_jarfile_index(ctx, label, basename, jar))

    output_proto = ctx.outputs.out_proto
    output_json = ctx.outputs.out_json

    build_mergeindex(ctx, output_proto, javaindex_files)
    build_mergeindex(ctx, output_json, javaindex_files)

    direct_files.append(output_proto)

    return [DefaultInfo(
        files = depset(direct_files),
    ), OutputGroupInfo(
        proto = [output_proto],
        json = [output_json],
        javaindex_files = depset(transitive = transitive_javaindex_files),
    )]

java_index = rule(
    implementation = _java_index_impl,
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
        "out_proto": attr.output(
            mandatory = True,
            doc = "the name of the proto output file",
        ),
        "out_json": attr.output(
            mandatory = True,
            doc = "the name of the json output file",
        ),
    },
)
