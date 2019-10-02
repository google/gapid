# isolateserver.py

Archive and run your isolated test.


## Introduction

`isolateserver.py` is the client side code relating to isolated executable.
`isolate.py` is a frontend to `isolateserver.py` which understands `.isolate`
files. Use `isolateserver.py` directly for low-level archival and retrieval.


## Overview

  - "`isolateserver.py help`" gives you all the help you need so only a quick
    overview is given here.
  - "`isolate.py archive`" is a shortcut to compile a `.isolate` into a
    `.isolated` file *and* archive the tree to an Isolate Server; e.g.
    "`isolate.py check`" plus "`isolateserver.py archive`".


### Archiving

If you want to upload a test to the isolate server, you want the first command:

  - Archive an .isolate tree: "`isolate.py archive --isolated <.isolated> -o
    https://host`"
  - Archive a single file: "`isolateserver.py archive <file> --isolate-server
    https://host`"


### Downloading

A common use case is when a Continuous Integration builder or a Pre Commit
builder archived a build that you want to investigate locally.


#### CLI

  - Fetch the whole isolated tree : "`isolateserver.py download --isolated
    <.isolated> --isolate-server https://host` "
  - Fetch a single file: "`isolateserver.py download --file <hash> <destination>
    --isolate-server https://host`"


#### Cache

"`isolateserver.py download`" supports the flag `--cache` which keeps a local
cache, to speed up repetitive downloads.


#### Web

  - Browsing any file via `https://host/browse`.


### Running an isolated test

  - Run test archived on an isolate server: "`run_isolated.py --hash <.isolated
    sha1> --isolate-server https://host`"
  - Use `run_isolated.py --help` for more information.

Note that `run_isolated.py` will automatically create a local cache on your
behalf, similar to the optional `--cache` flag to `isolateserver.py download`.
*If you do not want this*, do not use `run_isolated.py`. Use "`isolateserver.py
download`" and run the command manually.
