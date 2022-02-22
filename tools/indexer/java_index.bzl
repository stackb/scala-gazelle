"""java_index.bzl provides the java_index rule.
"""

load(":aspect.bzl", "JarIndexerAspect", "java_indexer_aspect")

def build_jarindex(ctx, basename, jar):
    input_file = ctx.actions.declare_file(basename + ".jar.json")
    output_file = ctx.actions.declare_file(basename + ".jarindex.json")

    ctx.actions.write(
        content = json.encode(struct(
            filename = jar.path,
        )),
        output = input_file,
    )

    args = [
        "--input_file",
        input_file.path,
        "--output_file",
        output_file.path,
    ]

    ctx.actions.run(
        mnemonic = "JarIndexer",
        progress_message = "Parsing jar file symbols",
        executable = ctx.executable._jarindexer,
        arguments = args,
        inputs = [input_file, jar],
        outputs = [output_file],
    )

    return output_file

def build_mergeindex(ctx, jarindex_files):
    """Builds the merged index for all jarindexes."""

    output_file = ctx.actions.declare_file(ctx.label.name + ".json")

    args = [
        "--output_file",
        output_file.path,
    ] + [f.path for f in jarindex_files]

    ctx.actions.run(
        mnemonic = "MergeIndex",
        progress_message = "Merging jarindex files: " + str(ctx.label),
        executable = ctx.executable._mergeindex,
        arguments = args,
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

        # transitive_jarindex_files.append(info.jar_index_files)
        transitive_jarindex_files += [info.info_file, info.jar_index_files]

    i = 0
    for jar in ctx.files.platform_jars:
        basename = ctx.label.name + "." + str(i)
        jarindex_files.append(build_jarindex(ctx, basename, jar))
        i += 1

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
            # providers = [JavaInfo],
            aspects = [java_indexer_aspect],
            doc = "list of java deps to be indexed",
        ),
        "platform_jars": attr.label_list(
            doc = "list of jar files to be indexed without a JarSpec.Label",
            allow_files = True,
        ),
        "_mergeindex": attr.label(
            default = Label("//cmd/mergeindex"),
            cfg = "exec",
            executable = True,
        ),
        "_jarindexer": attr.label(
            default = Label("//cmd/jarindexer"),
            cfg = "exec",
            executable = True,
        ),
    },
)
