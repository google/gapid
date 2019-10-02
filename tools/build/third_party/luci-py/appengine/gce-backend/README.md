# GCE Backend

A GCE backend for Machine Provider.

GCE Backend is composed of a series of cron jobs. Each cron runs independently
of the others and performs idempotent operations. Many of the jobs are only
eventually consistent, converging on the desired state over multiple calls.


## Setting up

*   Visit http://console.cloud.google.com and create a project. Replace
    `<appid>` below with your project id.
*   Visit Google Cloud Console,
    *   IAM & Admin, click `Add Member` and add someone else so you can safely
        be hit by a bus.
    *   Compute Engine, to enable Compute Engine.
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
*   Configure the Machine Provider with the
    [config service](https://github.com/luci/luci-py/blob/master/appengine/gce-backend/proto/config.proto)
    or the `instance\_url` property of the
    components.machine\_provider.utils.MachineProviderConfiguration entity in
    the datastore.
*   TODO(smut): Simplify the above.


TODO(smut): Move the following into doc/ as applicable.


## Config

The configuration defines the desired instance templates, and the desired
instance group managers created from those templates. Configured in the config
service as templates.cfg and managers.cfg respectively.


## import-config

Responsible for importing the current GCE configuration from the config service.
When a new config is detected, the local copy of the config in the datastore is
updated. This task ensures that the valid manager and template configurations
are updated synchronously.


## process-config

Responsible for creating an entity structure corresponding to the config.

A models.InstanceTemplate is created for each template defined in templates.cfg
which has a globally unique base\_name. Any reuse of base\_name, even when the
configured project is different, is considered to be another revision of the
same instance template. A revision of an instance template is one set of fields
for it, and a corresponding models.InstanceTemplateRevision is created. Each
time a template in templates.cfg with the same name has its other fields changed
a new models.InstanceTemplateRevision is created for it.

A models.InstanceGroupManager is created for each manager defined in
managers.cfg which refers to an existing template by template\_base\_name. At
most one instance group manager in each zone may exist for each template.


## create-instance-templates

Ensures a GCE instance template exists for each template in the config. Updates
models.InstanceTemplateRevisions with the URL of the created instance template.


## create-instance-group-managers

Ensures a GCE instance group manager exists for each manager in the config.
Updates models.InstanceGroupManagers with the URL of the created instance group
manager. Waits for an instance template to exist for the template the manager
is configured to use before attempting to create the instance group manager.


## fetch-instances

Fetches the list of instances created by each instance group manager, creating
a models.Instance for each one. Waits for the instance group manager to exist
before attempting to fetch the list of instances.


## catalog-instances

Adds instances to the Machine Provider catalog. Any instance not cataloged and
not pending deletion is added to the catalog.


## update-cataloged-instances

Checks the state of the instance in the Machine Provider catalog. If the Machine
Provider has leased the instance, that instance is marked as leased. If the
Machine Provider has reclaimed the instance after lease expiration, schedules
an operation to delete the GCE instance.


## delete-instances-pending-deletion

Deletes GCE instances for each models.Instance with pending\_deletion set.


## resize-instance-groups

Embiggens a GCE managed instance group whose size has fallen below the minimum
configured size.


## remove-cataloged-instances

Removes each models.Instance from the Machine Provider catalog that wasn't
created from an instance template currently referenced in the config and sets
pending\_deletion.


## delete-instance-group-managers

Deletes GCE instance group managers that aren't found in the config and have no
instances created from them.


## delete-instance-templates

Deletes GCE instance templates that aren't found in the config and have no
instance group managers configured to use them.


## cleanup-entities

Deletes entities that are no longer needed.

