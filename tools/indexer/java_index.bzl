"""java_index.bzl provides the java_index rule.
"""

load(":aspect.bzl", "JarIndexerAspect", "java_indexer_aspect")

def _java_index_impl(ctx):
    """Collect deps from our aspect."""
    outputs = depset()

    for dep in ctx.attr.deps:
        info = dep[JarIndexerAspect]

        # TODO: this depset construction can't be right
        outputs = depset(direct = outputs.to_list(), transitive = [info.info_file, info.jar_index_files])

    return [DefaultInfo(
        files = outputs,
    )]

java_index = rule(
    implementation = _java_index_impl,
    attrs = {
        "deps": attr.label_list(
            providers = [JavaInfo],
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
