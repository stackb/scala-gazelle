"""custom scala rules
"""

load("@io_bazel_rules_scala//scala:scala.bzl", "scala_library")

def scala_helper_library(**kwargs):
    """scala_helper_library is a contrived rule that helps demonstrate the resolve_kind_rewrite_name directive.

    Args:
        **kwargs: must contain 'name' and 'srcs'
    """

    name = kwargs.pop("name")
    srcs = kwargs.pop("srcs")
    helper_name = name + "_helper"

    filegroup(
        name = name,
        srcs = srcs,
    )

    scala_library(
        name = helper_name,
        srcs = srcs,
    )
