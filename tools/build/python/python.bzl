load("@rules_python//python:defs.bzl", _py_test = "py_test")
load("@pip//:requirements.bzl", "requirement")

def _lint_pylint(name, srcs, deps = [], args = [], **kwargs):
    _py_test(
        name = name,
        srcs = [
            "//tools/build/python:lint_pylint.py",
        ] + srcs,
        main = "//tools/build/python:lint_pylint.py",
        args = [
            "--rcfile=$(location //tools/build/python:pylintrc)"
        ] + args + ["$(location :%s)" % x for x in srcs],
        python_version = "PY3",
        srcs_version = "PY3",
        data = [ "//tools/build/python:pylintrc" ],
        deps = deps + [
            requirement("pylint"),
        ],
        **kwargs
    )

def _lint_mypy(name, srcs, deps = [], args = [], **kwargs):
    _py_test(
        name = name,
        srcs = [
            "//tools/build/python:lint_mypy.py",
        ] + srcs,
        main = "//tools/build/python:lint_mypy.py",
        args = [
            "--config-file=$(location //tools/build/python:mypy.ini)"
        ] + args + ["$(location :%s)" % x for x in srcs],
        python_version = "PY3",
        srcs_version = "PY3",
        data = [ "//tools/build/python:mypy.ini" ],
        deps = deps + [
            requirement("mypy"),
        ],
        **kwargs
    )

def _lint_flake8(name, srcs, deps = [], args = [], **kwargs):
    _py_test(
        name = name,
        srcs = [
            "//tools/build/python:lint_flake8.py",
        ] + srcs,
        main = "//tools/build/python:lint_flake8.py",
        args = [
            "--config=$(location //tools/build/python:flake8)"
        ] + args + ["$(location :%s)" % x for x in srcs],
        python_version = "PY3",
        srcs_version = "PY3",
        data = [ "//tools/build/python:flake8" ],
        deps = deps + [
            requirement("flake8"),
        ],
        **kwargs
    )

def py_lint(name, srcs, deps = [], args = [], **kwargs):
    _lint_mypy(name + "_mypy", srcs, deps, args, **kwargs)
    _lint_pylint(name + "_pylint", srcs, deps, args, **kwargs)
    _lint_flake8(name + "_flake8", srcs, deps, args, **kwargs)

    native.test_suite(
        name=name,
        tests=[
            name + "_flake8",
            name + "_mypy",
            name + "_pylint",
        ]
    )

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
