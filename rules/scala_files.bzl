"""scala_files.bzl provides the scala_files rule.
"""

ScalaSourceInfo = provider("provider that carries the index file", fields = {
    "index": "File: the generated index file from parsing all the files",
    "files": "List[File]: list of source files",
})

def _merge(ctx, output_file):
    indexes = [dep[ScalaSourceInfo].index for dep in ctx.attr.deps]

    args = ctx.actions.args()
    args.use_param_file("@%s", use_always = True)
    args.add("--output_file", output_file)
    args.add_all(indexes)

    ctx.actions.run(
        mnemonic = "ScalaSourceFileMerge",
        progress_message = "Merging %d source rules: " % len(indexes),
        executable = ctx.executable._merger,
        arguments = [args],
        inputs = indexes,
        outputs = [output_file],
    )

def _parse(ctx, files, output_file):
    args = ctx.actions.args()
    args.use_param_file("@%s", use_always = True)
    args.set_param_file_format("multiline")

    args.add("--rule_kind=%s" % "scala_files")
    args.add("--rule_label=%s" % str(ctx.label))
    args.add("--output_file", output_file)
    args.add_all(files)

    ctx.actions.run(
        mnemonic = "ParseScala",
        progress_message = "Parsing %s (%d files)" % (str(ctx.label), len(files)),
        execution_requirements = {
            "supports-workers": "1",
            "requires-worker-protocol": "proto",
        },
        executable = ctx.executable._parser,
        arguments = [args],
        inputs = files,
        outputs = [output_file],
    )

    return output_file

def _scala_files_impl(ctx):
    if len(ctx.files.srcs) == 0:
        _merge(ctx, ctx.outputs.pb)
        _merge(ctx, ctx.outputs.json)
    else:
        _parse(ctx, ctx.files.srcs, ctx.outputs.pb)
        _parse(ctx, ctx.files.srcs, ctx.outputs.json)

    return [
        DefaultInfo(
            files = depset([ctx.outputs.pb]),
        ),
        OutputGroupInfo(
            json = depset([ctx.outputs.json]),
        ),
        ScalaSourceInfo(
            index = ctx.outputs.pb,
            files = ctx.files.srcs,
        ),
    ]

scala_files = rule(
    implementation = _scala_files_impl,
    attrs = {
        "srcs": attr.label_list(
            doc = "list of scala srcs to be indexed",
            allow_files = [".scala"],
        ),
        "deps": attr.label_list(
            doc = "list of child scala_files rules to be merged",
            providers = [ScalaSourceInfo],
        ),
        "kinds": attr.string_list(
            doc = "list of file kinds (for scala_fileset)",
        ),
        "_merger": attr.label(
            default = Label("@build_stack_scala_gazelle//cmd/scalafilemerge"),
            cfg = "exec",
            executable = True,
            doc = "the merger tool",
        ),
        "_parser": attr.label(
            default = Label("@build_stack_scala_gazelle//cmd/scalafileextract"),
            cfg = "exec",
            executable = True,
            doc = "the parser tool",
        ),
    },
    outputs = {
        "pb": "%{name}.pb",
        "json": "%{name}.json",
    },
)

# renamed to support two different use cases for gazelle
scala_fileset = scala_files
