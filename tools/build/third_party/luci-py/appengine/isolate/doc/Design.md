# Isolated Server

High performance Infrastructure to run an executable with all its dependencies
mapped in.

**One line description:** How to push 10000 files worth 2Gb in total to tens of
bots within a few seconds, *even on Windows or Android*.


## Objective

Enables running an executable on an arbitrary machine by describing the
executable's runtime dependency efficiently. Archival and retrieval of the
needed files scale sub-linearly. Stale files are evicted from the cache
automatically.


## Background

To be able to run an executable along with all its dependencies fast, the
required files must be transferred. Many techniques can be used, like:
  - zip the executable along all the data files and unzip on the target machine
  - use virtual disk snapshot to clone and map on the target machine
  - use a faulty file system that remote-fetches files on demand
  - store all the files in a Content Addressed Storage (CAS) and push files
    required ahead of time
  - or more simply, share the files on a network share like NFS

*Using the Chromium infrastructure as a real-world example*, the Chromium
Continuous Integration (CI) infrastructure uses a set of "builders" and
testers". The "builders" compiles and archive the generated unit tests. The
"testers" run one or multiple tests, or a fraction of a test, i.e. a shard.

Testers have historically required a full checkout to run the tests since they
have no information about which data files are read by which test, even less if
they are accessed at all. In addition, the executables were transferred as a
large zip. That has became problematic as the zip grew larger than 2gb.

Syncing multiple tens of GB of sources is a constant cost that doesn't scale
well when the tests are sharded across multiple testers. As we want to spread
the test execution on more bots, the need to reduce constant costs grows, since
it's becoming the major cost of running a test on a bot.

Authors looked at using a faulty file system instead, e.g. a copy-on-write
compile plus mounting the partition on the testers to run the test from the
checkout. The problem is that it puts a significant burden on the hardware
providing these partitions and the round-trip latency is worsened, since the
underlying infrastructure has no idea what data will be needed upfront. This has
the net effect of multiplying the latency on the end-to-end execution duration.

For the Chromium team to scale properly, a more deterministic approach needs to
be applied to the way tests are run. Guessing what is needed to run a test,
manually keeping a list of executables to zip in a python script to send them
over to testers is not scalable. So a totally different approach is to make the
tests to run in a "pure" environment. Make sure running the tests is idempotent
from the surrounding environment. From a directory-tree point of view, the best
way to do it is the map the test executable into a temporary directory before
running it.

We highly recommend reading the Google engtools blog posts about building in the
cloud. In particular, read [Testing at the speed and scale of
Google](http://google-engtools.blogspot.com/2011/06/testing-at-speed-and-scale-of-google.html)
and [Build in the Cloud: Distributing Build
Steps](http://google-engtools.blogspot.com/2011/09/build-in-cloud-distributing-build-steps.html)
for background information about why you want to do that.


## Overview

The process works as follow:
  - A `.isolated` file is generated. See IsolateDesign for tools that generate
    these files. It contains the `relative directory` -> `SHA-1` mapping.
  - The dependencies and the `.isolated` file are archived on the isolate server
    with `isolateserver.py`.
  - `run_isolated.py` is provided the SHA-1 of the `.isolated` file, retrieves
    the dependencies and maps them all in a temporary directory. It runs the
    executable and delete the temporary directory afterward.


## Infrastructure

`isolateserver.py` interacts with a Content Addressed Cache file datastore
hosted on AppEngine named isolate-server. `isolateserver.py` can optionally
archive on an NFS/Samba share or locally albeit at the loss of cache eviction
functionality.

`isolate-server` is the main infrastructure that acts as a very efficient file
cache.


## Detailed Design

### `.isolated` file format

**Goal:** specifies "how to run a command with all its runtime dependencies".
It's more than just specifying the dependencies, it also describes how to run a
command, which includes the relative working directory to start the executable
from and the actual command line to execute.

A `.isolated` file is `.isolate` file processed by `isolate.py`. It lists the
exact content of each file that needs to be mapped in the relative path, the
content being addressed by it's SHA-1. It also lists the command that should be
used to run the executable and the relative directory to start the command in.

`.isolated` files are JSON file, unlike `.isolate` which are python files.


#### Description

The root is a dictionary with the following keys:
  - `algo`: Hashing algorithm used to hash the content. Normally `'sha-1'`.
  - `command`: exact command to run as a list.
  - `files`: list of dictionary, each key being the relative file path, and the
    entry being a dict determining the properties of the file. Exactly one of
    `h` or `l` must be present. `m` must be present only on POSIX systems.
    - `h`: file content's SHA-1
    - `l`: link destination iff a symlink
    - `m`: POSIX file mode (required on POSIX, ignored on non-POSIX).
    - `s`: file size iff not a symlink
    - `t`: type of the file iff not the default of `basic`
  - `includes`: references another `.isolated` file for additional files or to
    provide the command. In practice, this is used to reduce `.isolated` file
    size by moving rarely changed test data files in a separate `.isolated`
    file.
  - `read_only`: boolean to specify if all the files should be read-only. This
    will be eventually enforced.
  - `relative_cwd`: relative directory inside the temporary directory tree when
    the command should be executed from.
  - `version`: version of the file format. Increment the minor version for non
    breaking changes and the major version if code written for a previous
    version shouldn't be able to parse it.


##### File types

There are the following file types;
  - `basic`: All normal files, the default type.
  - `ar`: An [ar](https://en.wikipedia.org/wiki/Ar_(Unix)) archive containing a
    large number of small files.
  - `tar`: An [tar](https://en.wikipedia.org/wiki/Tar_(Unix)) archive
    containing a large number of small files.


#### Arbitrary split vs recursive trees

The `.isolated` format supports the `includes` key to split and merge back list
of files in separate `.isolated` files. It is in stark contrast with more
traditional tree of trees structure like the git tree object rooted to a git
commit object.

The reason is to leave a lot of room to the tool generating the `.isolated`
files to be able to package low-churn files versus high-churn files into a small
number of `.isolated` files. As a practical example, we can state that test data
files are low churn, as an example the rate of change of every `.png` or `.js`
file is low. The files in <(PRODUCT_DIR) are high-churn files since they are
usually different at each build. It's up to the tool generating the `.isolated`
files to generate the most optimal setup.

That's the primary design decision to use a flat list and not a tree-based
storage for the file mapping like the git tree objects. The reason is that
splitting the high-churn files from the low-churn files is not necessarily
directly describable in term of directories.

It's important to clarify here that `includes` for `.isolate` are not related in
any way to the `includes` in `.isolated` files. Reread the sentence if
necessary. The first one is to reuse a common list of runtime dependencies for
multiple targets, the second is to optimize the overall size of the `.isolated`
files to archive and load for bots.


### Isolate Server

**Goal:** performance tiered Content Addressed Cache with native compression
support. That is, it caches data, with each entry's key being the hash of the
entry's content. The performance tier of each entry is specified at each entry
upload time, so higher performance can be stored in higher performance backend
for faster retrieval.

As such, it is ill named, it doesn't know about the `.isolate` or `.isolated`
file formats. It is a **cache**, not a permanent data store, so while the
semantics can be similar to a [Content Addressed
Storage](http://en.wikipedia.org/wiki/Content-addressable_storage), it is much
more restricted in its usage and optimized for very fast lookups.

Non goals:
- Be able to remove an entry from the cache. The cache manages items' lifetime.
- Long term archival. It is a cache CAC, not a CAS.


#### Object eviction (GC)

Each item's timestamp is refreshed on *storage* or *request for presence*.
Fetching, like running `run_isolated.py`, never updates the timestamp. Only
storing does, like running `isolateserver.py archive`.

The cache uses a global 7 days eviction policy by default so objects are deleted
automatically if not tested for presence.


#### Namespaces

To help future-proof the server, all the objects are stored in a specified
namespace. The namespace is used as a signal to specify which hashing algorithm
is used (defaults to SHA-1) and if the objects are stored in their plain format
or transformed (compressed). The namespace logic also has a special case for
temporary objects, any "temporary" namespace is evicted after 1 day instead of
the default 7 days.

It is interesting to compare the choice of embedding the hashing algorithm in
the namespace instead of each key, like how [camlistore](http://camlistore.org/)
does. It slightly reduces the strings overhead and simplifies sending the hashes
as binary bytes. A single request handling several items doesn't have to switch
of hashing algorithm per item. It is a requirement and is implicitly enforced
that a single `.isolated` has all its items referenced in the same namespace.

As such, there is a close relationship between the `.isolated` and the
namespace, since both must use the exact same hash algorithm.

*TODO*: It could be valuable to use 2 layers of namespaces instead of one, so
that the hash algorithm and compression algorithm can be specified
independently.


#### Priorities

Some files are more important that others. In particular, `.isolated` files must
have much lower fetch latency than the other ones since they are the bottleneck
to fetch more data, i.e. all the dependencies. These high-priority files are
stored in
[memcache](https://developers.google.com/appengine/docs/python/memcache/) in
addition to the datastore, so the retrieve operation can complete with a lower
latency.


#### Object sizes

To optimize small object retrieval, small objects (with a current cut off at
20kb, heuristics needs to be done to select a better value) are stored directly
inline in the datastore instead of the AppEngine BlobStore or Cloud Storage to
reduce inefficient I/O for small objects.


#### Explicit compression support

Like most SCM like git and hg but unlike most CAS, `isolate-server` supports
on-the-wire and in-storage compression while using the uncompressed data to
calculate the hash key. Unlike git, `isolate-server` doesn't recompress on the
fly and do not do inter-file compression.

The reason for the on-the-wire compressed transfer is to greatly reduce the
network I/O. It is based on the assumption that most objects are build outputs,
usually executables, so they are usually both large and highly compressible. It
is important for that the `.isolated` files do not need to be modified to switch
from the non-compressed namespace to a compressed one so the key is the same for
the compressed and uncompressed version but they are stored in different
namespaces.


#### Optimized for warm store, warm fetch

The server is optimized for warm cache usage; the most frequent use case is that
a large number of files are already in the cache on store operation. The way to
do this is to batch requests for presence at 100 items per HTTP request, greatly
reducing the network overhead and latency. Then for each cache miss, the item is
uploaded as separate HTTP POST.

The actual algorithm used to do this is a bit more involved, files with recent
timestamps are more likely to be not present on the server so they are looked up
first. Same wise, larger files are looked up first, since they will incur the
largest latency to be uploaded. The batching of requests is gradual, the first
request specify a low number of highly probably cache misses, and as the cache
misses lower, the batches are larger.


#### URL endpoints

The number of supported requests is designed to be limited for its specific
intended use case. See the current API by visiting a live instance with path
`/_ah/api/explorer` to see the generated documentation.


#### Comparison to a few off-the-shelf CAS solutions

It's interesting to look at the trade offs with a few Content Addressed Storage
systems. Note that the other CAS compared here are not caches but real
datastores but the comparison is still useful from an optimization stand-point.
Using git (a source control system), bup (a backup software based on git),
camlistore (a one-size-fits-all datastore) as comparison.

| Feature  vs  Tool | git | bup | camlistore | isolateserver |
| ------------------|-----|-----|------------|---------------|
| Transparent compression | yes | yes | no | yes (but explicit) |
| Inter-file compression | yes | yes | no (but reuse chunks with same rolling hash) | no |
| Independent files request | no (supported through gitweb but inefficient I/O wise due to inter-file compression. Optimized to fetch a whole commit and its history) | no (supported through bup web but not optimal I/O wise due to rolling hash split chunks) | yes | yes |
| Efficient binary files support | no | yes (rolling hash) | yes | yes |
|Access control | external (usually ssh) | external (like git) | yes (AppEngine or native) | limited (IP or AppEngine) |
| Per file ACL | no, everything is visible | no | yes, a subset can be shared | no |
| Permanent reference to mutated objects | yes (tag) | no (each backup is independent?) |yes (permanode) | no |
| Tree of objects | explicit (root tree of a commit) | explicit (like git) | explicit | implicit (isolated files) |
| Automatic eviction policy | yes (GC unreferenced objects) | no | yes (GC unreferenced objects) | yes (explicit LRU, independent of reference tree) |
| Easy to delete older versions of an object | no (shallow clones aren't efficient) | no (near impossible to delete anything efficiently) | Yes (? not sure) | yes |
| Designed to work in distributed setup | yes | yes | yes | no |
| Hash algorithm can be changed | no | no | yes | yes |
| Priority support / fresh object memcaching | no | no | no | yes (explicit) |
| Fast remote object lookup | yes (by git commit only) | yes (like git) | no | yes (arbitrary) |

Note that the server double-checks the SHA-1 of the content uploaded, and will
discard the data if there is a mismatch.


## Project information

  - The whole project is written in python.
  - The isolate server code is subsumed by the Swarming project to make task
    distribution efficient.
  - The code is all contained in the repository
    https://chromium.googlesource.com/infra/luci/luci-py.git.
  - The primary consumer project is the Chromium project. As such some
    chromium-specific assumptions still remain throughout the code base but it
    is designed by the team to get rid of them.


## Caveats

  - The json format is not a determinist format per se. So the generator must
    always use the same json encoding so the content hash always match.


## Latency

Every step is optimized for the "warm" use case.

  - Files already present on Isolate Server are not uploaded again, reducing
    network I/O. Experience shows the hit rate is above 95%.
  - `isolateserver.py` lookups hundreds files at a time on Isolate Server to
    look for presence, and multiple HTTP requests for lookup are done
    simultaneously. The files are sorted via an heuristics to query the most
    likely cache misses first.
  - `run_isolated.py` keeps a LRU-based local cache to reduce network I/O, so
    only new files need to be fetched.
  - `run_isolated.py` uses hardlinks on all OSes to reduce file I/O when
    creating a temporary tree. A multiple thousands tree can be mapped in mere
    seconds.
  - `run_isolated.py` fetches multiple files simultaneously to reduce the
    overall effect of HTTP fetching latency.
  - Isolate Server keeps `.isolated` files in memcache for higher performance.
    Since they are a bottleneck to fetch the remaining dependencies, these files
    needs to be fetched first before fetching any other file.


## Scalability

To achieve better scalability, this project enables being able to confine each
test to a limited view of the available files. In practice, the bottleneck of a
Continuous Integration infrastructure will become:
  - Source tree checkout and build performance.
  - Speed of archival of the dependencies.
  - Network bandwidth to download the dependencies on the bots.

Isolated Server is designed to run on AppEngine so it can be considered a
"single distributed server".


## Redundancy and Reliability

There is no redundancy in the Isolate Server, as it is running on App Engine.


## Security Considerations

Isolate Server require a valid GAIA account to access the content. An IP
whitelist table is also available.


## Testing Plan

  - The isolate server code is unit, smoke and canary tested. Since most of the
    isolate server code is OS-independent and written in python, testing is
    relatively easy.
  - Support for hardlinks, symlinks and native path case need OS-specific code
    which can be tested itself on Swarming to get coverage across OSes.
  - A canary Continuous Integration master is run by the chromium team at
    http://build.chromium.org/p/chromium.swarm/waterfall.
