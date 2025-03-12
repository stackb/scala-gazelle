"autokeep.bzl provides a rule to run autokeep"
script_content = """#!/usr/bin/env bash
{autokeep_path} \
    --cache_file={cache_file} \
    --only_rules={only_rules} \

"""

def autokeep_impl(ctx):
    script = ctx.outputs.executable
    ctx.actions.write(
        output = script,
        is_executable = True,
        content = script_content.format(
            autokeep_path = ctx.executable._tool.short_path,
            cache_file = ctx.file.cache_file.short_path,
            only_rules = ",".join(ctx.attr.only_rules),
        ),
    )

    runfiles = ctx.runfiles([
        script,
        ctx.file.cache_file,
        ctx.executable._tool,
    ], collect_data = True, collect_default = True)

    return [
        DefaultInfo(
            files = depset([script]),
            runfiles = runfiles,
        ),
    ]

autokeep = rule(
    implementation = autokeep_impl,
    attrs = {
        "cache_file": attr.label(
            doc = "the scala-gazelle cache file to read",
            mandatory = True,
            allow_single_file = True,
        ),
        "only_rules": attr.string_list(
            doc = "list of rules to limit autokeep processing to",
        ),
        "_tool": attr.label(
            default = "//cmd/autokeep",
            executable = True,
            cfg = "exec",
        ),
    },
    executable = True,
)
