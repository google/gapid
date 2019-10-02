# Config

## Introduction

GCE Backend uses the [config service](../../config_service) to store its
server configuration. [config.proto](../../config.proto) defines the
protocol buffers used to store the config.

### settings.cfg

Set `enable\_ts\_monitoring` to true to enable [ts\_mon](../../third_party/gae_ts_mon)
integration, which reports the number of GCE VMs and other metrics defined in
[metrics.py](../../metrics.py). `Set mp\_server` to the address of your deployed
[Machine Provider](../../machine_provider) instance.

### templates.cfg

Defines GCE Instance Templates which should be created and maintained by the
GCE Backend. In GCE, an instance template is immutable, so making changes
to an existing template will cause the GCE Backend to create a new instance
template and gracefully halt usage of the old version of the template.

### managers.cfg

Defines GCE Instance Group Managers which should be created from the instance
templates defined in templates.cfg. Instance group managers are used to manage
a pool of GCE instances created from a particular instance template.

### Config changes

GCE instance templates and the instances themselves are immutable, so any config
changes cannot modify them directly. Instead, a new one is created and the old
one is drained, causing the old one to be deleted when possible. This ensures
that a VM is never deleted while in use by the Machine Provider.
