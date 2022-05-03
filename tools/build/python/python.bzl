load("@rules_python//python:defs.bzl", _py_test = "py_test")
load("@pip//:requirements.bzl", "requirement")

def py_test(name, srcs, deps = [], args = [], **kwargs):
    _py_test(
        name = name,
        srcs = [
            "//tools/build/python:py_test.py",
        ] + srcs,
        main = "//tools/build/python:py_test.py",
        args = [
            "--capture=no",
        ] + args + ["$(location :%s)" % x for x in srcs],
        python_version = "PY3",
        srcs_version = "PY3",
        deps = deps + [
            requirement("pytest"),
        ],
        **kwargs
    )
