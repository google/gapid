# Configuration service

*   Stores and imports config files from repositories, such as Gitiles.
*   Provides read-only access to config files and encapsulates their location.
*   Stores a registry of LUCI services.
*   Stores a registry of projects that use LUCI services.

[Documentation](doc)


## Setting up

*   Visit http://console.cloud.google.com and create a project. Replace
    `<appid>` below with your project id.
*   Visit Google Cloud Console, IAM & Admin, click Add Member and add someone
    else so you can safely be hit by a bus.
*   Follow the instructions in [ui/README.md](ui/README.md) to build the UI.
*   Upload the code with: `./tools/gae upl -x -A <appid>`
*   Ensure that the import location has been properly configured for access by
    the service account of the config service.
*   Visit https://\<appid\>.appspot.com/auth/bootstrap and click `Proceed`.
*   Set the import location and type using the Administration API's
    `globalConfig` setting call:
    *   `https://apis-explorer.appspot.com/apis-explorer/?base=https://<appid>.appspot.com/_ah/api#p/admin/v1/admin.globalConfig`
    *   `services_config_location` specifies the source location.
    *   `services_config_storage_type` specifies the source type
         (e.g. GITILES).
*   If you plan to use an [auth_service](../auth_service),
    *   Make sure it is setup already.
    *   [Follow instructions
        here](../auth_service#linking-other-services-to-auth_service).

