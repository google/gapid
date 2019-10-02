# Overview of the access controls

## Introduction

This is the list of the access control groups. By default in a fresh instance
all groups are empty and no one can do anything. Add relevant users and IPs in
the following groups. Make sure that users that have access to the isolate
server also have equivalent access to the swarming server.


### isolate-access

Members of this group can:

*   Download files if they know the SHA-1.
*   Browse content online.
*   Push new content.

All users of swarming (both clients and bots) should also be in this group.


### isolate-readonly-access

Members of this group can:

*   Download files if they know the SHA-1.
*   Browse content online.

This is generally of use for external services that are purely doing analysis.
