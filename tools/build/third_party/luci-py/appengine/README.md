# appengine/

This directory contains the AppEngine services needed for a LUCI infrastructure.

Every single microservice below is _optional_. Please mix and match based on
your needs.


## TL;DR

*   A minimal setup can be [isolate](isolate) + [swarming](swarming).
*   As infrastructure complexity scales up, [auth_service](auth_service),
    [config_service](config_service) can be added over time to reduce the
    management overhead.
*   Once automatic dynamic VM managment is needed, setup
    [machine_provider](machine_provider) and [gce-backend](gce-backend).
*   Go to http://console.cloud.google.com and register as many instances as
    needed then follow the service specific instructions.


## Services

*   [auth_service/](auth_service) Authentication Server. Provides centralized
    group management and group database replication across services.
*   [config_service/](config_service) is a project configuration distribution
    server that supports importing from repositories.
*   [gce-backend/](gce-backend) is a [Google Compute
    Engine](http://console.cloud.google.com/) backend for machine_provider.
*   [isolate/](isolate) Isolate Server is a Content-Addressed Cache running on
    AppEngine backed by Cloud Storage.
*   [machine_provider/](machine_provider) Machine provider is a service to
    lease VMs from a pool to users.
*   [swarming/](swarming) Swarming Server is a task distribution engine for
    highly hetegeneous fleet at high scale.


## Supporting code

*   [components/](components) contains the modules shared by all services in
    this repository. This includes the embeddable part of auth_service to act as
    a client for auth_service, ereporter2, machine_provider, tooling for testing
    and deployment.
*   [third_party/](third_party) contains shared third parties. Services using
    these should symlink the packages inside the root server directory so it
    becomes available in sys.path.
*   [third_party_local/](third_party_local) constains testing or tooling related
    third parties that are not meant to be ever used on a AppEngine server.


## Tooling

All services can be managed with `./tools/gae`, including running locally or
pushing a new version. Use `./tools/gae help` for an up to date list of commands
available.


### Pushing

Pushing new code to an AppEngine instance doesn't change the default version.

To push a new version of one of the services, do:

```
cd <server> # for example, config_service
./tools/gae upload -A <instance_name>
```

As described in the output by the tool, you can access it to
`<version>-dot-<name>.appspot.com` until you switch the default version.


### Changing the version

To make the new code _live_, you need to change the default version:

```
cd <server> # for example, config_service
./tools/gae switch -A <instance_name>
```

`gae` will propose the versions already uploaded and will propose the latest one
by default.


## External dependencies

luci-py leverages Chromium specific functionalities:
[CIPD](https://chromium.googlesource.com/infra/infra/+/master/cipd/)
(hermetic package management) and
[ts_mon](https://chromium.googlesource.com/infra/infra/+/master/infra_libs/ts_mon/)
(monitoring).  Neither are strictly required for operational purpose.
