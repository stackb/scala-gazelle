"""scala_index.bzl provides the scala_index rule.
"""

load(":aspect.bzl", "JarIndexerAspect", "scala_indexer_aspect")

def build_scalaindex(ctx, jarindex_files):
    """Builds the merged index for all jarindexes."""

    output_file = ctx.actions.declare_file(ctx.label.name + ".json")

    args = [
        "--output_file",
        output_file.path,
    ] + [f.path for f in jarindex_files]

    ctx.actions.run(
        mnemonic = "ScalaIndex",
        progress_message = "Creating scalaindex files: " + str(ctx.label),
        executable = ctx.executable._scalaindex,
        arguments = args,
        inputs = jarindex_files,
        outputs = [output_file],
    )

    return output_file

def _scala_index_impl(ctx):
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

    index_file = build_scalaindex(ctx, jarindex_files)
    direct_files.append(index_file)

    return [DefaultInfo(
        files = depset(direct_files),
    ), OutputGroupInfo(
        index_file = [index_file],
        jarindex_files = depset(transitive = transitive_jarindex_files),
    )]

scala_index = rule(
    implementation = _scala_index_impl,
    attrs = {
        "deps": attr.label_list(
            providers = [DefaultInfo],
            aspects = [scala_indexer_aspect],
            doc = "list of java deps to be indexed",
        ),
        "_scalaindex": attr.label(
            default = Label("//cmd/scalaindex"),
            cfg = "exec",
            executable = True,
        ),
    },
)
