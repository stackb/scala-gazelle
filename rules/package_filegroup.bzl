def package_filegroup(**kwargs):
    srcs = kwargs.pop("srcs", [])
    deps = kwargs.pop("deps", [])
    native.filegroup(
        srcs = srcs + deps,
        **kwargs
    )
