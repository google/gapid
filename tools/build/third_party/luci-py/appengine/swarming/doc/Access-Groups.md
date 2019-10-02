# Overview of the access controls

## Introduction

There are two layers of access control in Swarming: the first is a global layer
that controls user permissions globally for the server, and the second is
a bot pool layer, that controls who exactly can trigger tasks on what bot pools.

The second layer is optional and by default it is turned off, meaning anyone
that has global "can trigger a task" permission can trigger tasks on any bots
connected to the server. Details of how to turn it on and how it behaves are
described at the end of this doc.

Global access control is implemented via roles that are defined by the
access control groups, and are managed by going to the `/auth` URL of
your app. In a fresh instance, group names default to
`administrators`, so only those can use the service. To get started:

* Configure the group names for each role in the configuration file;
  see the [`config_service`](../../config_service) for details, and
  AuthSettings in [proto/config.proto](../proto/config.proto) for the
  schema.
* Create the appropriate groups under `/auth`.
* Add relevant users and IPs to the groups. Make sure that users who
  have access to the Swarming server also have equivalent access to
  the isolate server.

When specifying members of the auth groups, you can refer to the whitelisted IPs
using `bots:*`. For individual user accounts simply use their email,
e.g. `user@example.org`. All users in a domain can be specified with a glob,
e.g. `*@chromium.org`.


## Groups that define global server ACLs

### `users_group`

Members of this group can:

*   Trigger a task.
*   Query tasks the user triggered and get results.
*   List the tasks they triggered.
*   Cancel their own tasks.

Members have limited visibility over the whole system, and cannot view other user
tasks or bots.

Make sure members of this group are also member of `isolate-access`.

### `privileged_users_group`

Members of this group can do everything that members of the `users_group` can do
plus:

*   See other people's tasks.
*   See all the bots connected.

### `bot_bootstrap_group`

Members of this group can fetch Swarming bot code and bootstrap bots.

### `admins_group`

Members of this group can do all of the above plus:

*   Cancel anyone's task.
*   Delete bots.
*   Update `bootstrap.py` and `bot_config.py`.


## Pool ACLs

A bot pool is a named collection of bots usually dedicated to some single
kind of workload or a single project. Swarming assigns bots to pools by forcibly
setting `pool:<name>` dimension on them based on configuration defined in
`bots.cfg` config file (see [proto/bots.proto](../proto/bots.proto)).

Pool ACLs specify who is allowed to trigger tasks in a pool. This is a second
layer of ACLs that is consulted when triggering new tasks (and only then) after
the request passes the first server-global layer.

Only the triggering authorization can be controlled with a finer precision,
everything else is controlled through *server global groups* described above.

Pool ACLs are defined in `pools.cfg` config file (see
[proto/pools.proto](../proto/pools.proto)). If this file is missing or empty,
pool ACLs are completely disabled for the service, meaning only the
server-global ACLs are consulted.
