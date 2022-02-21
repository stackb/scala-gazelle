"""java_index.bzl provides the java_index rule.
"""

load(":aspect.bzl", "JarIndexerAspect", "java_indexer_aspect")

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
        "_mergeindex": attr.label(
            default = Label("//cmd/mergeindex"),
            cfg = "exec",
            executable = True,
        ),
    },
)
