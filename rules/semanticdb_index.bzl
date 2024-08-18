"""semanticdb_index.bzl provides the semanticdb_index rule.
"""

load("@io_bazel_rules_scala//scala:semanticdb_provider.bzl", "SemanticdbInfo")

def _index(ctx, output_file):
    args = ctx.actions.args()
    args.use_param_file("@%s", use_always = False)
    args.add("--output_file", output_file)
    info_files = []

    for dep in ctx.attr.deps:
        if SemanticdbInfo in dep:
            info = dep[SemanticdbInfo]
            for sourcefile, info_file in info.info_file_map.items():
                args.add("--info_file=%s=%s" % (sourcefile, info_file.path))
                info_files.append(info_file)

    ctx.actions.run(
        mnemonic = "SemanticDbIndex",
        progress_message = "Building semanticdb index: " + str(ctx.label),
        executable = ctx.executable._indextool,
        arguments = [args],
        inputs = info_files,
        outputs = [output_file],
    )

    return info_files

def _semanticdb_index_impl(ctx):
    info_files = _index(ctx, ctx.outputs.index)
    test_file = ctx.actions.declare_file(ctx.label.name)
    ctx.actions.write(test_file, "")

    return [
        DefaultInfo(
            executable = test_file,
            files = depset(info_files + [ctx.outputs.index]),
        ),
    ]

semanticdb_index = rule(
    implementation = _semanticdb_index_impl,
    attrs = {
        "deps": attr.label_list(
            doc = "list of scala jars to be indexed",
            providers = [SemanticdbInfo],
            mandatory = True,
        ),
        "_indextool": attr.label(
            default = Label("@build_stack_scala_gazelle//cmd/semanticdbidx"),
            cfg = "exec",
            executable = True,
            doc = "the semanticdbidx tool",
        ),
    },
    outputs = {
        "index": "%{name}.pb",
    },
)
