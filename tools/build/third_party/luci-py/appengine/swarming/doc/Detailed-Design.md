# Detailed design

Lightweight distribution server with a legion of dumb bots.

**One line description**: Running all tests concurrently, not be impacted by
network or device flakiness.


## Overview

### Server

The server runs on [AppEngine](https://developers.google.com/appengine/) and is
the only communication point for any client or bot that wants to use Swarming.
Clients send task requests to the server and receive a task ID. It's up to the
client to poll the server to know when a task is done.

The server uses its DB to store the tasks, the bots' state and stdout from the
tasks. It exposes the web frontend UI, client JSON REST API and bot JSON API. It
uses OAuth2 (or optionally IP whitelisting) to authenticate clients.

Task requests have a set of dimensions associated to it. It's the server that
matches the request's dimensions to find a bot which has the same set of
dimensions.

The server serializes all its state in the DB. There is **no** continuously
running thread. This makes upgrading the server and rolling back trivial. When
the bot code is affected by the server upgrade, the bots seamlessly upgrade
after their currently running task completes.


### APIs

The client API is implemented via Cloud Endpoints over HTTPS.

  - API can be browsed online at
    https://chromium-swarm.appspot.com/_ah/api/explorer in the [swarming
    API](https://apis-explorer.appspot.com/apis-explorer/?base=https://chromium-swarm.appspot.com/_ah/api#p/swarming/v1/)
    section.
  - Implementation is in [handlers_endpoints.py](../handlers_endpoints.py).
  - Messages are defined in [swarming_rpcs.py](../swarming_rpcs.py).
  - Authentication is done via OAuth2. The ACL groups are defined via
    [auth_service](../../auth_service/) integration.

**All the APIs are idempotent**: retrying a call in case of error is always
safe. This permits transparent retry in case of failure. The exception is
/tasks/new, which creates a new task, in case of failure, it may leave an orphan
task.

The bot API is an implementation detail and doesn't provide any compatibility
guarantee.


### Server configuration

The server has a few major configuration points;
  - Authentication, usually delegated to the [auth_service](../../auth_service/)
    instance.
  - [config_service](../../config_service) manages
  - [bootstrap.py](../swarming_bot/config/bootstrap.py) which permits automatic
    single-command bot bootstrapping.
  - [bot_config.py](../swarming_bot/config/bot_config.py)
    which permits Swarming server specific global configuration. It can hook
    what the bots do after running a task, their dimensions, etc. See the file
    itself for the APIs. The Bot interface is provided by
    [bot.py](../swarming/swarming_bot/api/bot.py).


### Tasks

Each task is represented by a `TaskRequest` and a `TaskProperties` described in
[task_request.py](../server/task_request.py). The `TaskRequest` represents the
meta data about a task, who, when, expiration timestamp, etc. The
`TaskProperties` contains the actual details to run a task, commands,
environment variables, execution timeout, etc. This separation makes it possible
to dedupe the task requests when the exact same `.isolated` file is ran multiple
times, so that task-deduplication can be eventually implemented.

A task also has a `TaskResultSummary` to describe its pending state and a tiny
`TaskToRun` entity for the actual scheduling. They are respectively defined in
[task_result.py](../server/task_result.py) and
[task_to_run.py](../server/task_to_run.py).

The task ID is the milliseconds since epoch plus low order random bits and the
last byte set to 0. The last byte is used to differentiate between each try.


#### Priority task queues

The server implements a Priority queue based on the creation timestamp of
request. The priority is a 0-255 value with lower is higher priority. The
priority enforces ordering, higher priority (lower value) tasks are run first.
Then tasks *with the same priority* are run in either FIFO or LIFO order,
depending on the server's configuration.

Technically speaking, it's possible to do more elastic priority scheduling, like
old pending requests have their priority slowly increasing over time but the
code to implement this was not committed since there was no immediate need.


#### Assignment

When a bot polls the server for work, the server assigns the first available
matching task available.

Matching is done via the dimensions of the request vs the dimensions of the bot.
The bot must have all the dimensions listed on the request to be given the task.
For example, it could be "os=Windows-Vista-SP2; gpu=15ad:0405".

To make the process efficient, the dimensions are MD5 hashed and only the first
32 bits are saved so integer comparison can be used. This greatly reduce the
size of the hot `TaskToRun` entities used for task scheduling and the amount of
memory necessary on the frontend server.

Once a bot is assigned to a task, a `TaskRunResult` is created. If the task is
actually a retry, multiple `TaskRunResult` can be created for a single
`TaskRequest`.


#### Task execution

During execution, the bot streams back the stdout and a heartbeat over HTTPS
requests every 10 seconds. This works around stable long-lived network
connectivity, as a failing HTTPS POST will simply be retried.


#### Task success

Swarming distributes tasks but it doesn't care much about the task itself. A
task is marked as `COMPLETED_SUCCESS` when the exit code is 0.


#### Orphaned task

If a task stops being updated by its bot after 5 minutes, a cron job will abort
the task with BOT_DIED. This condition is **masked by retrying the task
transparently on the next available bot**. Only **one** retry is permitted to
not overflow the infrastructure with potentially broken tasks.

If any part of the scheduling, execution or processing of results fails, this is
considered an infrastructure failure.


### Task deduplication

If a task is marked as idempotent, e.g. `--idempotent` is used, the client
certifies that the task do not have side effects. This means that running the
task twice shall return the same results (pending flakiness).

The way it works internally is by calculating the SHA256 of `TaskProperties`
when marked as idempotent. When a `TaskResultSummary` succeeds that was also
idempotent, it sets a property to tell that its values can be reused.

When a new request comes in, it looks for a `TaskResultSummary` that has
`properties_hash` set. If it finds one, the results are reused as-is and served
to the client immediately, without ever scheduling a task.

**Efficient task deduplication requires a deterministic build and no side
effects in the tasks themselves**. On the other hand, successful task
deduplication can results is large infrastructure savings.

☞ See [the user guide about idempotency](User-Guide.md#idempotency) for more
information.


### Caveats of running on AppEngine

   - Reliability. The main caveat of running on AppEngine is that it is ~99.99%
     stable. A simple task scheduling services that is running on a single host
     would never have to care about this. This forces the code and client to be
     *extremely defensive*.
   - No "process" or "thread" makes simple things difficult; message passing has
     to be DB based, cannot be only in-memory. Have to use memcache instead of
     in-memory lookup, which causes some overhead.
   - No long lived TCP connections makes it hard to have push based design.
   - DB operations scale horizontally but are vertically slow.
   - It's pretty rare that MySQL or Postgres would save half of the entities in
     a DB.put_multi() call. AppEngine does this all the time.
   - Transactions have to be avoided as much as possible. This forces the DB
     schema to be of a particular style.
   - Increased latency due to polling based design.

We accepted these caveats as we found the benefits outweighed, and by far, the
caveats. The main issue has been coding defensively up-front, which represented
a sunk cost in coding time.


### Handling flakiness

Running on AppEngine forced Swarming to make every subsystem to support
flakiness;

   - The server tolerates DB failure. In this case, it usually returns HTTP 500.
   - The client and the bot handles HTTP 500 by automatically retrying with
     exponential backoff. This is fine because the REST APIs are safe to retry.
   - No long lived TCP connection is ever created, so a network switch reset or
     flakiness network conditions are transparently handled.
   - The task scheduler handles flaky bots by retrying the task when the bot
     stops sending heartbeats.


## Bot

Each Swarming bot is intended to be extremely dumb and replaceable. These
workers have a very limited understanding of the system and access the server
via a JSON API. Each bot polls the server for a task. If the server hands a
task, the bot runs the associated commands and then pipe the output back to the
server. Once done, it starts polling again.


### Bootstrapping

Only two basic assumptions are:

   - The bot must be able to access the server through HTTPS.
   - python 2.7 must be installed.

The bot's code is served directly from the server as a self-packaged
`swarming_bot.zip`. The server generates it on the fly and embeds its own URL in
it. The server can also optionally have a custom
[bootstrap.py](../swarming_bot/config/bootstrap.py) to further automate the bot
bootstrapping process.


### Self updating

The bot keeps itself up to date with what the server provides.

   - At each poll, the bot hands to the server the SHA256 of the contents of
     `swarming_bot.zip`. If it mismatches what the server expects, it is told to
     auto-update;
     - The bot downloads the new bot code to `swarming_bot.2.zip` or
       `swarming_bot.1.zip`, depending on the currently running version and
       alternates between both names.
   - `swarming_bot.zip` is generated by the server and includes 2 generated
     files:
      - [bot_config.py](../swarming_bot/config/bot_config.py) is
        user-configurable and contains hooks to be run on bot startup, shutdown
        and also before and after task execution.
      - [config.json](../swarming_bot/config/config.json) contains the URL of
        the server itself.
   - When a bot runs a task, it locks itself into the server version it started
     the task with. This permits to do breaking bot API change safely. This
     implies two side-effects:
      - A server version must not be deleted on AppEngine until all bot locked
        into this version completed their task. It's normally below one hour.
      - A server shouldn't be updated in-place, in particular if it modifies the
        bot API. Use a new server version name when changing the server or bot
        code.

Since the bot version calculation is done solely by the hash, the bot will also
roll back to earlier versions if the server is rolled back. All the bot's code
is inside the zip, this greatly reduces issues like a partial update, update
failure when there's no free space available, etc.

The bot also keeps a `LKGBC` copy (Last Known Good Bot Code):

   - Upon startup, if the bot was executed via `swarming_bot.zip`;
     - It copies itself to swarming_bot.1.zip and starts itself back, e.g.
       execv().
   - After successfully running a task, it looks if `swarming_bot.zip` is not
     the same version as the currently running version, if so;
     - It copies itself (`swarming.1.zip` or `swarming.2.zip`) back to
       `swarming_bot.zip`. This permits that at the next VM reboot, the most
       recent LKGBC version will be used right away.

The bot code has been tested on Linux, Mac and Windows, Chrome OS' crouton and
Raspbian.


### Bot dimensions

The bot publishes a dictionary of *dimensions*, which is a dict(key,
list(values)), where each value can have multiple values. For example, a Windows
XP bot would have `'os': ['Windows', 'Windows-10-15063']`. This permits broad or
scoped selection of bot type.

For desktop OSes, it's about the OS and hardware properties. For devices, it's
about the device, not about the host driving it.

These "dimensions" are used to associate tasks with the bots. See below.


### Multiple bots on a single host

Multiple bots can run on a host simultaneously, as long as each bot has its own
base directory. So for example, one could be located in `/b/s/bot1` and a second
in `/b/s/bot2`.

In the scenario of multiple bots running on a host, make sure to never call
[Bot.host_reboot()](../swarming_bot/api/bot.py).


### Device Bot

For bots that represent a device (Android, iOS, ChromeOS, Fuchsia), a bot can
"own" all the devices connected to the host (generally via USB) or each bot can
be in a docker container to own a single device.

In the case of devices that are communicated through IP, it's up to
[bot_config.py](../swarming_bot/config/bot_config.py) to decide what is "owned"
by this bot. In some cases this can be determined by hardware (like when the
host has two ethernet cards and devices are connected on the second), vlan
proximity or hard coded host names in
[bot_config.py](../swarming_bot/config/bot_config.py).

In the USB case, a prototype recipe to create
[udev](http://en.wikipedia.org/wiki/Udev) rules to fire up the bot upon
connection is included. The general idea is to reduce sysadmin overhead to its
minimum, configure the host once, then connect devices. No need to touch the
host again afterward. The server tolerates devices going Missing In Action or
the host rebooting, forcibly killing the on-going tasks. The server will retry
these task in this case, since it is an *infrastructure failure*.

The only sysadmin overhead remaining is to look for dead devices once in a while
via the client tools or server monitoring functionality.


## Client

Clients trigger tasks and requests results via a [Cloud Endpoints JSON REST
API](#apis).

It is not possible for the client to access bots directly, no interactivity is
provided _by design_.

See [APIs](#apis) above to write your own client.


### Requesting a task

When a client wishes to run something on Swarming, they can use the REST API or
use the client script `swarming.py trigger`. It's a simple HTTPS POST with the
`TaskRequest` and `TaskProperties` serialized as JSON.

The request message is `NewTaskRequest` as defined in
[swarming_rpcs.py](../swarming_rpcs.py).


### Task -> Bot assignment

The bot selection process is inverted. It's the bot that polls for tasks. It
looks at all the products of all the `dimensions` it has and look at the oldest
task with highest priority that has `dimensions` which are also defined on the
bot.


### Authentication

   - The Web UI is implemented in Polymer and uses the same API as the client,
     both are authenticated via OAuth2.
   - Bots using the REST APIs can optionally be whitelisted by IPs.
   - The ACLs can be federated through a third party AppEngine instance
     [auth_service](../../auth_service/).


### Access control and Federation

The access control groups are optionally federated via
[auth_service](../../auth_service/) via a master-replica model. This presents a
coherent view on all the associated services.


## Testing Plan

Swarming is tested by python tests in the following ways:
   - Pre-commit testing:
      - Unit tests
      - Smoke test
   - Canarying on the chromium infrastructure, to ensure the code works before
     deploying to prod.
