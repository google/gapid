# Authentication and user groups

## Introduction

Instances of Swarming and Isolate services can be shared among multiple teams or
used by a large team where full administrative access by all team members is not
desirable. Thus some sort of access control is required. For example, a task
launched by a member of team A should be visible only to other members of team
A, but not team B. Or Swarming bots can be decommissioned or moved to quarantine
only by members of some small group of machine administrators.

A related problem is to ensure that only bots can call bot related HTTP
endpoints, only known users can post tasks, only known admins can modify
critical settings, only allowed GAE apps can call Swarming APIs, etc.

Swarming and Isolate services use components.auth library that provides the
following set of features:
  - User authentication via OAuth2 or GAE cookies.
  - Bot authentication via IP whitelist (TODO: use OAuth2 too).
  - GAE apps authentication via X-Appengine-Inbound-Appid header.
  - Groups system for access control.
  - External groups import.
  - Service for central group management across multiple services.


## Identities and authentication

Any HTTP request handled by components.auth library is associated with a caller
identity that initiated it. An identity is represented as a string
**type**:**id** where **type** identifies what **id** means and indirectly
corresponds to a method used to authenticate the caller.

Supported identity types:
  - user:**email** - users using OAuth2 or GAE cookies to authenticate, e.g.
    user:joe@example.com.
  - bot:**hostname** - bots that use special machine tokens for auth.
  - bot:whitelisted-ip - bots with whitelisted IPs (deprecated).
  - service:**app-id** - some GAE application, e.g. service:isolate-server.
  - anonymous:anonymous - no authentication information was provided.

On all UI pages user: prefix is considered default and omitted, other prefixes
are still visible.


### OAuth2

Standard OAuth2 authentication flow for native apps (e.g. scripts like
swarming.py) consists of following high-level steps and configuration:
  1. The script knows OAuth2 client_id (that publicly identifies the script) and
     client_secret (that supposedly only this script knows).
  2. The script uses oauth2client library to transform client_id, client_secret
     and a list of scopes (e.g. https://www.googleapis.com/auth/userinfo.email)
     into a HTTP link to a consent screen.
  3. The user opens the link in the browser, logs in into their account, and
     sees "Application **app title** would like to access **list of stuff**.
     Accept?".
  4. Once user accepts the request, new refresh_token is passed from the browser
     back to the script (manually, by asking user to copy-paste it or via a
     redirect to http://localhost) where it is silently used to mint
     access_token.
  5. access_token is put into Authorization HTTP header with every request sent.
  6. components.auth library verifies that access_token was minted with required
     scope, and then extracts client_id and user's email used to create it.
  7. Requests from unknown non-whitelisted client_id's are rejected.
  8. Otherwise the authentication is complete.

swarming.py (and other client side script) slightly deviate from that scheme: we
decided not to hardcode client_id and client_secret in the script itself.
There's no way to keep a secret really secret under such circumstances anyway.

Instead swarming.py fetches client_id and client_secret from the service
(effectively relying on HTTPS for server authentication). That allows us to move
installation specific configuration into Datastore, instead of hardcoding it in
the code or in some other place.

Primary client_id and client_secret (fetched by swarming.py) as well as a list
of additional accepted client_ids is stored as Auth service configuration in
Datastore and it has to be set before OAuth2 can be used.

TODO: link to setup instructions.


## Groups system

components.auth manages a list of groups. Each group has a unique name and
contains:
  - List of identities that belong to that group.
  - List of names of nested groups.
  - List of glob-like patterns to match against identity string, e.g. user:`*`.

Group management UI and REST API is accessible only to members of
'administrators' group.


### Builtin groups

Some known built in groups:
  - administrators - can view and edit everything, across all services, absolute
    power.
  - ereporter2-reports - will receive exception reports on email.
  - ereporter2-viewers - can view ereporter2 errors via web UI.

TODO: link to Swarming groups.

TODO: link to Isolate groups.


### Admin bootstrap

When a new instance of service that uses components.auth is brought to life its
initial set of groups is empty, including 'administrators' group. There's a
bootstrap mechanism to add a first member to 'administrators' group: the first
Appengine level administrator (as specified in GAE Permissions page) to visit
/auth page would be automatically added to 'administrators' group. After that
its their responsibility to configure the rest of the groups.


## Central auth service

By default a service that uses components.auth is completely independent and has
its own set of groups and OAuth2 configuration. It's OK for one or two
instances, but managing multiple groups across multiple related services quickly
becomes cumbersome.

components.auth supports a mode where service's groups and other related
configuration is managed from a single central authentication service via
Primary `->` Replica synchronization mechanism.

Any service that uses components.auth can run in standalone mode or as a replica
of central auth_service.

auth_service is a separate service designed just for that. It knows how to push
group and configuration changes to linked replicas. There's no way to link two
arbitrary regular services together without using a separate central
auth_service.


### Replication mechanism

Internally all information about groups, OAuth2 configuration and shared secrets
is stored in a single versioned entity group. Whenever it changes, version
number is bumped and entire entity group is serialized and sent to replicas that
are behind. This process is retried until all replicas report that they have the
latest version.

Primary service authenticates replicas via SSL. Replicas authenticate primary
via X-Appengine-Inbound-Appid header and RSA+SHA256 signature of the serialized
blob.

Replicas provide access to groups and configuration in read only mode. Auth web
pages on replicas redirect to corresponding pages on Primary (since it's where
the source of truth is stored).


### Linking process

To start using auth_service as source of data about groups, a service should be
switched from Standalone to Replica mode and Primary and Replicas should perform
initial handshake.

In a nutshell the linking process looks like this:
  1. Administrator of auth_service generates a linking ticket via web UI, by
     providing GAE app id of an application they want to convert to a replica.
  2. The ticked is actually a URL to a special endpoint on the app being linked.
     The endpoint is accessible only to Appengine level admins of application
     being linked.
  3. When the ticket is used, the app being linked contacts auth_service (with
     two way authentication) and auth_service registers it as a new replica.
  4. Any local groups state on linked app is erased and replaced with current
     version of groups information from auth_server.


### External groups support

auth_service can optionally fetch groups from an external tar.gz tarball using
its own service account for authentication (via OAuth2).

It is enough to implement simple pipeline for external groups import:
  1. Some cron job fetches local groups from whatever source it wants (e.g.
     LDAP).
  2. Packs groups listings into a tarball and uploads it somewhere (e.g. Google
     Storage).
  3. auth_service fetches the tarball, and imports groups.

Names of imported groups by convention start with group system prefix, e.g.
'ldap/all-users'. Such groups are not editable by auth service UI or REST API,
but otherwise behave as regular groups (e.g. they can be used as nested groups,
they gets replicated to all replicas, etc.).
