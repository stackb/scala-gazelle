"""semanticdb_index.bzl provides the semanticdb_index rule.
"""

def _merge(ctx, output_file):
    args = ctx.actions.args()
    args.use_param_file("@%s", use_always = True)
    args.add("--output_file", output_file)
    args.add_all(ctx.files.jars)

    ctx.actions.run(
        mnemonic = "SemanticDbIndex",
        progress_message = "Building semanticdb index: " + str(ctx.label),
        executable = ctx.executable._mergetool,
        arguments = [args],
        inputs = ctx.files.jars,
        outputs = [output_file],
    )

def _semanticdb_index_impl(ctx):
    _merge(ctx, ctx.outputs.index)

    return [
        DefaultInfo(
            files = depset([ctx.outputs.index]),
        ),
    ]

semanticdb_index = rule(
    implementation = _semanticdb_index_impl,
    attrs = {
        "jars": attr.label_list(
            doc = "list of scala jars to be indexed",
            allow_files = True,
            mandatory = True,
        ),
        "kinds": attr.string_list(
            doc = "list of scala rule kinds to collect",
            mandatory = True,
        ),
        "_mergetool": attr.label(
            default = Label("@build_stack_scala_gazelle//cmd/semanticdbmerge"),
            cfg = "exec",
            executable = True,
            doc = "the semanticdb merge tool",
        ),
    },
    outputs = {
        "index": "%{name}.pb",
    },
)
