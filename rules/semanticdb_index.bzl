"""semanticdb_index.bzl provides the semanticdb_index rule.
"""

SemanticDbIndexInfo = provider("provider that carries the index file", fields = {
    "index": "the index file",
})

def _merge(ctx, output_file):
    indexes = [dep[SemanticDbIndexInfo].index for dep in ctx.attr.deps]

    args = ctx.actions.args()
    args.use_param_file("@%s", use_always = True)
    args.add("--output_file", output_file)
    args.add_all(ctx.files.jars)
    args.add_all(indexes)

    ctx.actions.run(
        mnemonic = "SemanticDbIndex",
        progress_message = "Building semanticdb index: " + str(ctx.label),
        executable = ctx.executable._mergetool,
        arguments = [args],
        inputs = ctx.files.jars + indexes,
        outputs = [output_file],
    )

def _extract(ctx, jar_file):
    output_file = ctx.actions.declare_file(jar_file.basename + ".textdocuments.json", sibling = jar_file)
    args = ctx.actions.args()
    args.use_param_file("@%s", use_always = False)
    args.add("--jar_file", jar_file)
    args.add("--output_file", output_file)

    ctx.actions.run(
        mnemonic = "SemanticDbExtract",
        progress_message = "Extracting semanticdb textdocuments from jar: " + jar_file.short_path,
        executable = ctx.executable._extracttool,
        arguments = [args],
        inputs = [jar_file],
        outputs = [output_file],
    )

    return output_file

def _semanticdb_index_impl(ctx):
    _merge(ctx, ctx.outputs.index)
    _merge(ctx, ctx.outputs.json)
    textdocuments = [
        _extract(ctx, jar_file)
        for jar_file in ctx.files.jars
    ]

    return [
        DefaultInfo(
            files = depset([ctx.outputs.index]),
        ),
        OutputGroupInfo(
            json = depset([ctx.outputs.json]),
            textdocuments = depset(textdocuments),
        ),
        SemanticDbIndexInfo(
            index = ctx.outputs.index,
        ),
    ]

semanticdb_index = rule(
    implementation = _semanticdb_index_impl,
    attrs = {
        "jars": attr.label_list(
            doc = "list of scala jars to be indexed",
            allow_files = True,
        ),
        "deps": attr.label_list(
            doc = "list of child semanticdb_index rules to be merged",
            providers = [SemanticDbIndexInfo],
        ),
        "kinds": attr.string_list(
            doc = "list of scala rule kinds to collect",
            mandatory = False,
        ),
        "_mergetool": attr.label(
            default = Label("@build_stack_scala_gazelle//cmd/semanticdbmerge"),
            cfg = "exec",
            executable = True,
            doc = "the semanticdb merge tool",
        ),
        "_extracttool": attr.label(
            default = Label("@build_stack_scala_gazelle//cmd/semanticdbextract"),
            cfg = "exec",
            executable = True,
            doc = "the semanticdb extract tool",
        ),
    },
    outputs = {
        "index": "%{name}.pb",
        "json": "%{name}.json",
    },
)
