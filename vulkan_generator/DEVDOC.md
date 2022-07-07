# Vulkan Generator Developer Documentation
This document is to be able to build, run and test the Vulkan Generator in AGI.

## Python

## Pre-requirements:

Vulkan generator requires Python 3.9.2 Install Python 3.9.2 and pyenv.

Then add these to `.bashrc` or `.zshrc`

```
export PATH="$HOME/.pyenv/bin:$PATH"
eval "$(pyenv init -)"
eval "$(pyenv virtualenv-init -)"
```

if pyenv not installed to `$HOME/.pyenv` then use the install location.

All the linting/testing tools should be able to installed via pip.

## Build instructions

### Build target
`//vulkan_generator`

### Run target 
`//vulkan_generator:main`

### Lint target
`//lint`

This target should run as test e.g. `bazel test //:lint`

If you would like to integrate this to your IDE or use them via command line here are the tools and their versions:

```
pylint==2.13.8
mypy==0.950
flake8==4.0.1
```

### Test target
`tests-vulkan-generator`

This target includes linting.

If you would like to run tests in IDE or command line here is the testing tool and the version:

```
pytest==7.1.2
```

## Setup to run presubmit tests locally

Presubmit can be run from `./kokoro/presubmit/presubmit.sh`. This will require the Python formatter given below.

```
autopep8==1.6.0
```

## Command Line

All the config files are under `./tools/build/python/` folder. Each tool can be run with the config file:

Pylint:
`pylint --recursive=y --rcfile=tools/build/python/pylintrc vulkan_generator`

Flake8:
`flake8 --config tools/build/python/flake8 vulkan_generator`

Mypy
`mypy --config-file=tools/build/python/mypy.ini vulkan_generator`

Autopep8
`autopep8 --global-config=tools/build/python/pep8 -r --in-place vulkan_generator`

### Troubleshooting
If any command fails to run or does an unexpected behaviour: Please add `PYTHONPATH` at the beginning e.g.

`PYTHONPATH=. pylint --recursive=y --rcfile=tools/build/python/pylintrc vulkan_generator`

## IDE setup
This tools can be integrated into IDEs. An example will be given for `VSCode` and it should be fairly similar in other IDEs as well.

This script is for the local setting file under `.vscode` folder in the workspace. If you would like to add it to global settings
every path should include the workspace directory as `tools/build/python` is relative to the workspace.

```
 "python.linting.pylintEnabled": true,
    "python.linting.pylintArgs": [
        "--rcfile",
        "tools/build/python/pylintrc",
    ],
    "python.linting.mypyEnabled": true,
    "python.linting.mypyArgs": [
        "--config-file",
        "tools/build/python/mypy.ini"
    ],
    "python.linting.flake8Enabled": true,
    "python.linting.flake8Args": [
        "--config",
        "tools/build/python/flake8",
    ],
    "python.formatting.autopep8Args": [
        "--global-config",
        "tools/build/python/pep8",
    ],
    "python.linting.lintOnSave": true,
    "[python]": {
        "editor.formatOnSave": true,
    },

    "python.testing.cwd": "${workspaceFolder}",
    "python.testing.unittestEnabled": false,
    "python.testing.pytestEnabled": true,
    "python.testing.pytestArgs": [
        "--collect-only",
    ],
```

### Troubleshooting
VSCode is very slow to discover tests due to the project size(including rest of the AGI). You may need to disable go test discovery and even then
tests may not be able to discovered accordingly. You can try opening only the `vulkan_generator` folder for the tests or run them from the command line 
instead.
