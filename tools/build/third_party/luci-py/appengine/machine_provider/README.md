# Machine Provider

Service which provides machines using a leasing mechanism where a client can
request a machine with certain characteristics for a desired duration.


[Documentation](doc)


## Setting up

*   Visit https://console.cloud.google.com and create a project. Replace
    `<appid>` below with your project id.
*   Visit Google Cloud Console,
    *   IAM & Admin, click `Add Member` and add someone else so you can safely
        be hit by a bus.
    *   IAM & Admin, change the role for `App Engine default service account`
        from `Editor` to `Owner`.
    *   Pub/Sub, click `Enable API`.
*   Upload the code with: `./tools/gae upl -x -A <appid>`
*   Visit https://\<appid\>.appspot.com/auth/bootstrap and click `Proceed`.
*   If you plan to use a [config service](../config_service),
    *   Make sure it is setup already.
    *   [Follow instruction
        here](../components/components/config/#linking-to-the-config-service).
*   If you plan to use an [auth_service](../auth_service),
    *   Make sure it is setup already.
    *   [Follow instructions
        here](../auth_service#linking-isolate-or-swarming-to-auth_service).
*   Visit "_https://\<appid\>.appspot.com/auth/groups_":
    *   Create [access groups](doc/Access-Groups.md) as relevant.

