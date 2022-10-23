SourceIndexerAspect = provider(
    "a provider for Scala Source Indexing",
    fields = {
        "index": "The index json",
        "source_files": "The File objects used to create the index",
    },
)

_scala_rules = [
    "scala_library",
    "scala_binary",
]

def build_sourceindex(ctx, target, source_files):
    """Builds the manifest for the given source files."""

    output_file = ctx.actions.declare_file(target.label.name + ".index.json")

    args = ["-o", output_file.path, "-l", str(target.label)]
    args += [f.short_path for f in source_files]

    ctx.actions.run(
        mnemonic = "ScalaIndexer",
        progress_message = "Extracting top-level-symbols " + str(target.label),
        executable = ctx.executable._sourceindexer,
        arguments = args,
        inputs = source_files,
        outputs = [output_file],
    )

    return output_file

def _scala_indexer_aspect_impl(target, ctx):
    deps = []
    if hasattr(ctx.rule.attr, "deps"):
        deps.extend(ctx.rule.attr.deps)

    transitive_indexes = []
    transitive_source_files = []
    for dep in deps:
        if SourceIndexerAspect not in dep:
            continue
        transitive_indexes.append(dep[SourceIndexerAspect].index)
        transitive_source_files.append(dep[SourceIndexerAspect].source_files)

    if ctx.rule.kind not in _scala_rules:
        return [
            SourceIndexerAspect(
                index = depset(transitive = transitive_indexes),
                source_files = depset(transitive = transitive_source_files),
            ),
            OutputGroupInfo(
                scala_index_files = depset(transitive = transitive_indexes),
            ),
        ]

    if DefaultInfo not in target:
        fail("target does not provide DefaultInfo: " + str(target.label))

    source_files = [f for f in ctx.rule.files.srcs if f.path.endswith(".scala")]
    if len(source_files) == 0:
        return [
            SourceIndexerAspect(
                index = depset(transitive = transitive_indexes),
                source_files = depset(transitive = transitive_source_files),
            ),
            OutputGroupInfo(
                scala_index_files = depset(transitive = transitive_indexes),
            ),
        ]

    # print("source_files:", source_files)

    index = build_sourceindex(ctx, target, source_files)

    return [
        SourceIndexerAspect(
            index = depset(direct = [index], transitive = transitive_indexes),
            source_files = depset(direct = source_files, transitive = transitive_source_files),
        ),
        OutputGroupInfo(
            scala_index_files = depset(direct = [index], transitive = transitive_indexes),
        ),
    ]

scala_indexer_aspect = aspect(
    attr_aspects = ["deps"],
    attrs = {
        "_sourceindexer": attr.label(
            default = Label("//cmd/sourceindexer"),
            cfg = "exec",
            executable = True,
        ),
    },
    provides = [SourceIndexerAspect],
    implementation = _scala_indexer_aspect_impl,
    apply_to_generating_rules = True,
)

def merge_action(ctx, index_files):
    """Builds the merged index for all jarindexes."""

    output_file = ctx.actions.declare_file(ctx.label.name + ".json")

    args = [
        "--output_file",
        output_file.path,
    ] + [f.path for f in index_files]

    ctx.actions.run(
        mnemonic = "MergeSourceIndex",
        progress_message = "Merging sourceindex files: " + str(ctx.label),
        executable = ctx.executable._mergetool,
        arguments = args,
        inputs = index_files,
        outputs = [output_file],
    )

    return output_file

def _scala_source_index_impl(ctx):
    """Collect deps from our aspect."""

    # List[Depset[File]]
    transitive_index_files = []

    # List[File]
    index_files = []

    for dep in ctx.attr.deps:
        info = dep[SourceIndexerAspect]
        index_files.extend(info.index.to_list())
        transitive_index_files.append(info.index)

    source_index = merge_action(ctx, index_files)

    return [DefaultInfo(
        files = depset(direct = [source_index]),
    ), OutputGroupInfo(
        source_index = [source_index],
        source_index_files = depset(transitive = transitive_index_files),
    )]

scala_source_index = rule(
    implementation = _scala_source_index_impl,
    attrs = {
        "deps": attr.label_list(
            providers = [DefaultInfo],
            aspects = [scala_indexer_aspect],
            doc = "list of scala deps to be indexed",
        ),
        "implied": attr.string_dict(
            doc = """implied import symbol dependencies (e.g. "com.typesafe.scalalogging.Logger": "org.slf4j.Logger")""",
        ),
        "_mergetool": attr.label(
            default = Label("//cmd/sourceindex_merger"),
            cfg = "exec",
            executable = True,
        ),
    },
)
