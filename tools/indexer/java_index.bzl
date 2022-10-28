"""java_index.bzl provides the java_index rule.
"""

load(":aspect.bzl", "JarIndexerAspect", "java_indexer_aspect")

def build_jarindex(ctx, label, basename, jar):
    """Builds a single jarindex.

    Args:
        ctx: the context object
        label: label to use for the jar
        basename: a string representing a filename prefix for generated files
        jar: the File to parse
    Returns:
        the output File of the index.
    """

    ijar = ctx.actions.declare_file(jar.short_path.replace("/", "-"))

    # input_file = ctx.actions.declare_file(basename + ".jar.json")
    output_file = ctx.actions.declare_file(basename + ".jarindex.json")

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

    # ctx.actions.write(
    #     content = json.encode(struct(
    #         filename = ijar.path,
    #     )),
    #     output = input_file,
    # )

    # args = [
    #     "--input_file",
    #     input_file.path,
    #     "--output_file",
    #     output_file.path,
    # ]

    # if False:
    #     ctx.actions.run(
    #         mnemonic = "JarIndexer",
    #         progress_message = "Parsing jar file symbols",
    #         executable = ctx.executable._jarindexer,
    #         arguments = args,
    #         inputs = [input_file, ijar],
    #         outputs = [output_file],
    #     )
    # else:
    ctx.actions.run(
        mnemonic = "JarIndexer2",
        progress_message = "Indexing jar " + ijar.basename,
        executable = ctx.executable._jarindexer2,
        arguments = [
            "--label",
            str(label),
            "--output_file",
            output_file.path,
            ijar.path,
        ],
        inputs = [ijar],
        outputs = [output_file],
    )

    return output_file

def build_mergeindex(ctx, jarindex_files):
    """Builds the merged index for all jarindexes.

    Args:
        ctx: the context object
        jarindex_files: a sequence of File representing the jarindex files
    Returns:
        the output File of the merged file.
    """

    output_file = ctx.actions.declare_file(ctx.label.name + ".json")

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

    return output_file

def _java_index_impl(ctx):
    """Collect deps from our aspect."""

    # List[File]
    direct_files = []

    # List[Depset[File]]
    transitive_jarindex_files = []

    # List[File]
    jarindex_files = []

    for dep in ctx.attr.deps:
        info = dep[JarIndexerAspect]
        jarindex_files.extend(info.jar_index_files.to_list())
        transitive_jarindex_files += [info.info_file, info.jar_index_files]

    for i, jar in enumerate(ctx.files.platform_jars):
        label = ctx.attr.platform_jars[i].label
        basename = ctx.label.name + "." + str(i)
        jarindex_files.append(build_jarindex(ctx, label, basename, jar))

    index_file = build_mergeindex(ctx, jarindex_files)
    direct_files.append(index_file)

    return [DefaultInfo(
        files = depset(direct_files),
    ), OutputGroupInfo(
        index_file = [index_file],
        jarindex_files = depset(transitive = transitive_jarindex_files),
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
            default = Label("//cmd/mergeindex"),
            cfg = "exec",
            executable = True,
        ),
        # "_jarindexer": attr.label(
        #     default = Label("//cmd/jarindexer"),
        #     cfg = "exec",
        #     executable = True,
        # ),
        "_jarindexer2": attr.label(
            default = Label("//cmd/jarindexer:jarindexer2"),
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
