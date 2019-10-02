# Isolate Server

An AppEngine service to efficiently cache large set of large files over the
internet with high level of duplication. The cache is content addressed and uses
[Cloud Storage](https://cloud.google.com/storage/) as its backing store.

Isolate enables sending _temporary_ files around. It is a pure cache, _files_
will be deleted.

Isolate can be used standalone when only files need to be transfered but no task
scheduler is needed.

[Documentation](doc)


## Setting up

*   Visit http://console.cloud.google.com and create a project. Replace
    `<appid>` below with your project id.
*   Visit Google Cloud Console
    *   App Engine
        *   Enable it and choose `us-central`.
    *   IAM & Admin > IAM
        *   Click `Add Member` and add someone else so you can safely be hit by
            a bus.
    *   IAM & Admin > Service accounts
        *   Click `Create service account`.
        *   Service account name `server`
        *   Service account ID `server`
        *   Click `Create`.
        *   Click `Continue`, no need to add a permission.
        *   Click `Create key`.
        *   Select `P12` and click `Create`.
            *   TODO(vadimsh): switch to JSON keys.
        *   Click `Done`.
    *   Storage
        *   Click `Create bucket`.
        *   Name it with the same <appid>. Do not use any pre-created bucket,
            they won't work.
        *   Choose Multi-Regional, United States.
        *   Click `Create`.
        *   Click `Permissions` tab.
        *   Click `Add members`.
        *   Enter the user `server@<appid>.iam.gserviceaccount.com`.
        *   Select Roles in `Storage Legacy` group: `Storage Legacy Bucket
            Writer` and `Storage Legacy Object Reader`.
        *   Click `Add`.
    *   Pub/Sub
        *   Click `Enable API`.
    *   App Engine > Memcache
        *   Click `Change`.
        *   Chose `Dedicated`.
        *   Set the cache to Dedicated 5Gb.
        *   Wait a day of steady state usage.
        *   Set the limit to be lower than the value read at "Total cache
            size" in "Memcache Viewer".
    *   App Engine > Settings
        *   Click `Edit`:
        *   Set Google login Cookie expiration to: 2 weeks.
        *   Click `Save`.
*   Upload the code with: `./tools/gae upl -x -A <appid>`
*   Run [setup_bigquery.sh](setup_bigquery.sh) to create the BigQuery
    `isolated.stat` table and grant write access to the AppEngine app. The cron
    job will fail meanwhile.
*   If you plan to use an [auth_service](../auth_service),
    *   Make sure it is setup already.
    *   [Follow instructions
        here](../auth_service#linking-other-services-to-auth_service).
*   _Else visit "_https://\<appid\>.appspot.com/auth/bootstrap_" and click
    `Proceed`.
*   Visit "_https://\<appid\>.appspot.com/auth/groups_"
    *   Create [access groups](doc/Access-Groups.md) as relevant. Visit the "_IP
        Whitelists_" tab and add bot external IP addresses if needed.
*   Visit "_https://\<appid\>.appspot.com/restricted/config_"
    *   Follow the on-screen instructions to generate the base64 encoded DER
        private key.
    *   Click `Submit`.
*   If you plan to use a [config service](../config_service) (the normal case):
    *   Make sure it is setup already.
    *   Make sure you set
        [SettingsCfg.ui_client_id](https://chromium.googlesource.com/infra/luci/luci-py/+/master/appengine/swarming/proto/config.proto)
        to be `server@<appid>.iam.gserviceaccount.com`.
    *   [Follow instruction
        here](../components/components/config/#linking-to-the-config-service).
*   If you are not using a config service, see [Configuring using FS
    mode](https://chromium.googlesource.com/infra/luci/luci-py/+/master/appengine/components/components/config/README.md#fs-mode).
    You'll need to add an entry to settings.cfg like `ui_client_id:
    "server@<appid>.iam.gserviceaccount.com"`


## Stats

Use prpc CLI client from https://go.chromium.org/luci/grpc/cmd/prpc:

```
echo '{"resolution":"MINUTE","limit":20}' | prpc call -verbose <host> isolated.Isolated.Stats
```
