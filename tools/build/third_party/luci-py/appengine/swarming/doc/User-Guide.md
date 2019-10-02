# User guide

Important Swarming concepts are **Tasks** and **Bots**. A task is a step that
has inputs and generates outputs. A bot can run tasks. Simple, right?

*   [Task](#task): everything to understand Swarming tasks
    *   [Properties](#properties): the task request data to describe a task
        *   [Idempotency](#idempotency): flag to improve efficiency
    *   [Request](#request): the task request metadata to explain a task
    *   [Results](#results): the result of a task request
*   [Bot](#bot): everything to understand Swarming bots
*   [Interface](#interface): how to interact with the service
    *   [Web UI](#web-ui)
    *   [Python CLI](#python-cli)
    *   [HTTP API](#http-api)


## Task

There are 3 important classes of data:

*   **Task properties** defines inputs (files, command), a description of the
    dimensions a bot must have to run it, timeouts. It's the _functional_ part.
    *   It is the description of what requester wants to do. Properties act as
        the key in the cache of finished results (see
        [Idempotency](#idempotency)).
*   **Task request** defines the particular attempt to execute a task. It
    contains task properties (_what_ to execute), and accompanying metadata for
    bookkeeping tweaking (_who_ executes, _when_, _why_, whether it is ok to
    reuse existing cached result, etc).
*   **Task result** defines which bots ran it, timestamps, stdout, outputs if
    any, performance metadata, logs.

Task request (and task properties as part of it) are set when the task is
created and do not change.

A task is referenced to via a task ID. It looks like an hex number but should be
treated as a string.


### Properties

A Swarming task is conceptually a function that has some input and generates
outputs. The process can be simplified as _F(i, c, b)_ where _F()_ is the
Swarming task, _i_ is the input files, _c_ the command execution environment,
_b_ the bot selection (dimensions) description.

Inputs can be a mix of all 4 of:

*   Isolated tree: optionally includes a command. It's essentially a pointer
    (SHA1 digest) to a root of a [Merkle tree-like
    structure](https://github.com/luci/luci-py/blob/master/appengine/isolate/doc/Design.md#isolated-file-format)
    that stores files for the task.
*   CIPD package(s): archives to be mapped inside the work directory. This
    leverages [CIPD](https://github.com/luci/luci-go/tree/master/cipd).
*   Named cache(s): local cache directory(ies) to do incremental work to be
    mapped in the task working directory. This tells the worker to create
    directory(ies) to keep even _after_ the task is completed, so files can be
    reused by some later task that requests the exact same cache. Examples are
    local git clone caches, incremental build directories, etc. This adds
    transitivity to the task (non-determinism).
*   Secret bytes. This data is sent to the task via
    [LUCI_CONTEXT](../../../client/LUCI_CONTEXT.md) but is not otherwise
    retrievable.

Command execution environment is defined as:

*   Command line:
    *   From the isolated file. In this case, it is possible to extend with
        `extra_args` to specify arguments _appended_ to the command on a
        per-task basis. A good use is to shard test runs into multiple
        subshards.
    *   In the task properties. In this case, the actual command is listed as a
        list.
*   Idempotency flag to signify possibility of reusing previous succeeding
    results of a task with the exact same task properties.
*   Timeouts:
    *   Hard timeout: the maximum allocated time to run the task.
    *   I/O timeout: the maximum allowed time between output to stdout or
        stderr. For example, an I/O timeout of 20 minutes and a Hard timeout of
        60 minutes, a task sleeping for 20 minutes will be killed, even if still
        has allocated time. This catches hung tests faster.
*   Environment variables can be leveraged to create side effects, for example
    test sharding.

Bot selection is defined as:

*   A list of key:value dimensions. This is the core bot selection mechanism to
    select which bots are _allowed_ to run the task.

The dimensions are important. For example be clear upfront if you assume an
Intel processor, the OS distribution version, e.g. Windows-7-SP1 vs
Windows-Vista-SP2.


#### Idempotency

Idempotency is a mechanism to improve the efficiency of the infrastructure. When
a task is requested to the server, and a previous task with the _exact same
properties_ had previously succeeded on the Swarming server, previous results
can be returned as-is without ever running the task.

Not running anything is always faster than running something. This saves a lot
of time and infrastructure usage.

To effectively leverage idempotency, it is important for the inputs files to be
as "stable" as possible. For example efforts, see Debian's initiated effort
[reproducible builds](https://reproducible-builds.org/) and Chromium's
[deterministic
builds](https://www.chromium.org/developers/testing/isolated-testing/deterministic-builds)
effort.

To enable this feature, a task must be declared as idempotent. This tells the
server that this request fits the **contract** that the task implements a pure
function: same inputs **always** produce same outputs. Results of execution of
such tasks can be reused.

For a task to be idempotent, it **must** depend on **nothing else** than the
task inputs and the declared environment. This means the dimensions uniquely
describe the type of bot required; exact OS version, any other important detail
that can affect the task output.

Other things of note are:

*   No access to any remote service. This include HTTP(S), DNS lookup, etc. No
    file can be 'downloaded' or 'uploaded' by the task. They must be mapped in,
    content addressed, up front. Results must be inside `${ISOLATED_OUTDIR}`.
    *   This is also important from a performance PoV since `run_isolate.py`
        keeps a local content addressed cache.
*   No dependency on the time of the day or any other side-signal.

If any of the rule above does not hold, the task must *not* be marked as
idempotent since it is not reproducible by definition.


### Request

The request is the metadata around the task properties requested. This defines:

*   Who: authenticated account to create the task and on behalf of whom. For
    example, the Commit Queue creates a task on the behalf of a user.
*   When: creation timestamp, expiration delay
    *   The expiration delay is the maximum allowed time to wait for a bot to
        pickup the task. If no bot pickups the task in the allowed timeframe,
        the task is marked as `EXPIRED`.
*   What: the task properties. As described in the previous section, this
    defines all inputs for the task and the hash digest of this data is used as
    a key in a cache of available results when evaluating idempotent tasks.
*   Why: display name, tags. Tags are used to enable searching for tasks.
*   A numerical priority is associated to the tag between 0 and 255. Lower means
    most important. This is effectively a [FIFO or LIFO
    queue](Detailed-Design.md#priority-task-queues) of tasks.


### Result

The result is a collection of:

*   Where the execution process happened, which bot.
*   What are the results of the "function call". The stdout, the isolated
    outputs.
*   Metadata like the exit code, the timeout signal if any, timestamps.

Once the task is completed, results become immutable.

The result can also be a non-event: the task wasn't run at all. This results in
an `EXPIRED` task. This happens when there was no bot available to run the task
before the expiration delay.

An exceptional event can be `BOT_DIED`. This means that either the bot was lost
while the task ran or that the server had an internal failure during the task
execution.


## Bot

To understand how bot behaves, see [Bot.md](Bot.md). This section focuses from
the point of view of running a task.

Swarming tasks are normally running a isolated tree directly via
[run_isolated.py](https://github.com/luci/luci-py/tree/master/client/run_isolated.py).

Swarming is designed with inspiration from internal Google test distribution
mechanism. As such, it has a few assumptions baked in. A task shall:

*   Open input files as read-only, never for write.
*   Write files only to these two locations:
    *   The OS-specific temporary directory, e.g. `/tmp` or `%TEMP%` file files
        that are irrelevant after the task execution.
    *   `${ISOLATED_OUTDIR}` for files that are the output of this task.

Once the task completed, results are uploaded back and the tree is associated
with the task result.


## Interaction

There's 4 ways to interact with Swarming:

*   Swarming web UI. It is primarily a way to _view_ state, not change it. Two
    exceptions are cancelling a task and retrying a task. The Web UI doesn't
    provide a way to trigger a new task (ping the author if this feature is
    desirable).
*   CLI command line tool. There's a [python
    client](https://github.com/luci/luci-py/tree/master/client) and eventually a
    [Go
    client](https://github.com/luci/luci-go/tree/master/client/cmd/swarming).
    *   The luci-py/client repository subdirectory is cloned automatically to
        [client-py](https://github.com/luci/client-py) so you don't have to
        clone the whole server code if not desired. It's the same code.
    *   Once the Go client code is completed, you'll be able to `go get
        github.com/luci/luci-go/client/cmd/swarming`
*   Direct HTTP requests. You can browse the [API description
    online](https://apis-explorer.appspot.com/apis-explorer/?base=https://chromium-swarm.appspot.com/_ah/api#p/swarming/v1/).
    *    This is the interface to use to trigger a task via another server.


## Web UI

The Web UI has 4 purposes:

*   Present tasks and enable searching per tags and dimensions.
    *   Enable canceling or retrying individual task.
*   Present bots and enable querying per dimensions.
*   For administrators:
    *   Enable updating `bot_config.py` and `bootstrap.py`.
    *   Present a token to bootstrap new bots.
    *   View error reports.


## Python CLI

[swarming.py](https://github.com/luci/luci-py/blob/master/client/swarming.py) is
the client side script to manage Swarming tasks at the command line.

**Warning:** This doc is bound to become out of date. Here's one weird trick:
*   "`swarming.py help`" gives you all the help you need so only a quick
    overview is given here:


### Running a task synchronously

If you just want to run something remotely, you can use the `run` command. It is
going to block until the command has completely run remotely.

```
swarming.py run --swarming <host> --isolate-server <isolate_host> <isolated|hash>
```

The `<hash>` is what `isolate.py archive` gave you. See IsolatedUserGuide for
more information. A path to a `.isolated` file will work too.


### Running a task asynchronously

The buildbot slaves uses `trigger` + `collect`, so they can do multiple things
simultaneously. The general idea is that you trigger all the tests you want to
run immediately, then collect the results.


#### Triggering

Triggers a task and exits without waiting for it:
```
swarming.py trigger --swarming <host> --isolate-server <isolate_host> --task <name> <hash>
```

  - `<name>` is the name you want to give to the task, like "`base_unittests`".
  - `<hash>` is an `.isolated` hash.

Run `help trigger` for more information.


#### Collecting results

Collects results for a previously triggered task. The results can be collected
multiple times without problem until they are expired on the server. This means
you can collect again data run from a job triggered via `run`.

```
swarming.py collect --swarming <host> <name>
```


### Querying bot states

`swarming.py query` returns state about the known bots. More APIs will be added,
like returning tasks, once needed by someone. In the meantime the web frontend
shows these.


### More info

The client tools are self-documenting. Use "`swarming.py help`" for more
information.


## HTTP API

The API is implemented via Cloud Endpoints v1. It can be browsed at:
https://apis-explorer.appspot.com/apis-explorer/?base=https://chromium-swarm.appspot.com/_ah/api#p/swarming/v1/

Until the API is rewritten as proto files, each API struct description can be
read at
https://github.com/luci/luci-py/blob/master/appengine/swarming/swarming_rpcs.py
