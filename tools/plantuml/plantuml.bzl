"""plantuml_diagram.bzl provides the plantuml_diagram rule.
"""

def _plantuml_diagram_impl(ctx):
    args = ctx.actions.args()
    args.add(ctx.file.src.path)
    args.add(ctx.outputs.dst.path)

    ctx.actions.run(
        mnemonic = "PlantUML",
        progress_message = "Generating PlantUML file: %s" % ctx.outputs.dst.basename,
        executable = ctx.executable._plantuml_tool,
        arguments = [args],
        inputs = [ctx.file.src],
        outputs = [ctx.outputs.dst],
    )

    return [DefaultInfo(
        files = depset([ctx.outputs.dst]),
    )]

plantuml_diagram = rule(
    implementation = _plantuml_diagram_impl,
    attrs = {
        "src": attr.label(
            doc = "the plantuml source file",
            allow_single_file = True,
        ),
        "_plantuml_tool": attr.label(
            default = Label("@build_stack_scala_gazelle//tools/plantuml"),
            cfg = "exec",
            executable = True,
            doc = "the mergeindex tool",
        ),
        "dst": attr.output(
            mandatory = True,
            doc = "the output file for the diagram",
        ),
    },
)

def plantuml_diagram_genrule(name, srcs, format = "png", visibility = None):
    native.genrule(
        name = name,
        srcs = srcs,
        cmd = "java -jar $(location @plantuml_jar//jar) -t%s -o $@ $(SRCS)" % format,
        outs = [name + ".sh"],
        executable = True,
        tools = ["@plantuml_jar//jar"],
        visibility = visibility,
    )
