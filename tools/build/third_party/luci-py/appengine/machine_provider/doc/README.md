# Lease requests

## Introduction

A lease request is an RPC which requests to lease a specific type of machine
from the Machine Provider. Only OAuth2-authenticated users may issue lease
requests. Requests are processed asynchronously, meaning that clients should
poll for the result of their request.

### Dimensions

The type of machine to lease is decribed in terms of
[dimensions](../../components/components/machine_provider/dimensions.py), a set
of key-value pairs which specify desired attributes.

### Request ID

For each request, the client picks a request ID which will be used to uniquely
identify the request. A request which has the same ID and dimensions as a
previous request from the same client will be considered a duplicate request.
Request ID reuse with different dimensions will be considered an error. Use a
different request ID to request multiple machines with the same dimensions,
and reissue a request to see its current fulfillment status.


# Config

## Introduction

Machine Provider uses the [config service](../../config_service) to store some
of its server configuration.

### config.proto

Defines SettingsCfg, a protocol buffer with the `enable\_ts\_monitoring` field.
Set to true to enable [ts\_mon](../../third_party/gae_ts_mon) integration and
report various lease statistics \(see [metrics.py](../../metrics.py)\).


# ACLs

## Introduction

Machine Provider uses the [auth service](../../auth_service) to store and manage
access control lists. Lease requests are only accepted from OAuth2-authenticated
users. Add users to the following groups to give them advanced permissions.

### machine-provider-catalog-administrators

Members of this group can perform the following for any backend service:

*   Add new machines.
*   Query the catalog for machines.
*   Update any CatalogEntry.
*   Modify the capacity of machines in the catalog.
*   Delete exsting machines.

### machine-provider-NAME-backend

The `NAME` identifier represents the name of the corresponding backend, such
as `gce`.

Members of this group can perform the following for backend service `NAME`:

*   Add new machines.
*   Query the catalog for machines.
*   Update any CatalogEntry.
*   Modify the capacity of machines in the catalog.
*   Delete exsting machines.
