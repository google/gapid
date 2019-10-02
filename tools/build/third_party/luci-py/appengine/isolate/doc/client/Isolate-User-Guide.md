# isolate.py user guide

Isolate your test.


## Introduction

  - `isolate.py` wraps the vast majority of the client side code relating to
    executable isolation.
  - "`isolate.py help`" gives you all the help you need so only a quick overview
    is given here.


## isolate.py

  - "`isolate.py`" wraps usage for tracing, compiling, archiving and even
    running a test isolated locally.
  - Look at the `isolate.py --help` page for more information.
  - `--isolate` is not necessary when `--isolated` is specified and the
    `.isolated` file exists. This is because the `.isolated.state` file saved
    beside the `.isolated` file contains a pointer back to the original
    `.isolate` file *and* persists the variables.


### Minimal .isolate file

Here is an example of a minimal .isolate file where an additional file is needed
only on Windows and the command there is different:
```
{
  1. Global level.
  'variables': {
    'files': [
      '<(PRODUCT_DIR)/foo_unittests<(EXECUTABLE_SUFFIX)',
      1. All files in a subdirectory will be included.
      '../test/data/',
    ],
  },

  1. Things that are configuration or OS specific.
  'conditions': [
    ['OS=="linux" or OS=="mac"', {
      'variables': {
        'command': [
          '<(PRODUCT_DIR)/foo_unittests<(EXECUTABLE_SUFFIX)',
        ],
      },
    }],

    ['OS=="android"', {
      'variables': {
        'command': [
          'setup_env.py',
          '<(PRODUCT_DIR)/foo_unittests<(EXECUTABLE_SUFFIX)',
        ],
        'files': [
          'setup_env.py',
        ],
      },
    }],
  ],
}
```


The `EXECUTABLE_SUFFIX` variable is automatically set to `".exe"` on Windows and
empty on the other OSes.

Working on Chromium? You are not done yet! You need to create a GYP target too,
check out http://dev.chromium.org/developers/testing/isolated-testing/for-swes


### Useful subcommands

  - "`isolate.py check`" compiles a `.isolate` into a `.isolated`.
  - "`isolate.py archive`" does the equivalent of `check`, then uses
    `isolateserver.py` to archive the isolated tree.
  - "`isolate.py run`" runs the test locally isolated, so you can verify for any
    failure specific to isolated testing.

Did I tell you about "`isolate.py help`" yet?


## FAQ

### I have a feature request / This looks awesome, can I contribute?

This project is 100% open source. See
[Contributing](https://github.com/luci/luci-py/wiki/Contributing) page for more
information.
