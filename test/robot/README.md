# Robot - automated integration testing system.

## Components

### Master

The master is a process that controls the entire, distributed Robot system.

The master is the main coordination point, all other service are discovered
through the master.
It's main job is to manage and control the set of satellites.
A satellite is a server that offers any of the other robot services.

A master instance can be started with:

```./do run robot start master```

By default, this verb will also start a local satellite for locally attached devices.

### Satellite

An instance running on a host machine (Windows / MacOS / Linux) that connects to a master
and runs workers.

A satellite instance can be started with:

```./do run robot start worker```

TODO: Rename verb or satellite!

### Worker

A worker runs on a satellite and executes task for a given target device.

Trace, Report, Replay are all workers.


### Package

A package is a zip file containing GAPID build artifacts (`gapis`, `gapir`, etc.)

A package can be complete (as in it contains binaries for all hosts, all Android ABIs), or sparse.
Obviously Robot will only be able to execute tasks on devices it has binaries for.

Packages can be uploaded to the master with:

```./do run robot upload```

or with

```./do upload```

#### TODO:
`./do upload` is supposed to generate the .zip for you, but it seems this functionality [was removed by this CL](https://github.com/google/gapid/commit/8c7d48133268cfdf458e24b6f0622d3d3d8a271f).

We need some way of regenerating the zip files, locally and on the build machines.

The layout for a package much match the expected layout in [test/robot/build/artifact.go](https://github.com/google/gapid/blob/cfde04afa4d6f4c384412b669d1aa85f608c9b02/test/robot/build/artifact.go#L83-L107).

### Subject

A subject is a tracable application (currently only APKs are supported).

They can be uploaded to the master with:

```./do upload subject```


### Tracks

A track is a sequence of packages, usually ordered by changelist SHA.
There doesn't however need to be a 1:1 package <-> CL mapping in a track. This allows a track to
have entries for Gerrit-style patch-sets.

There can be multiple tracks and each track has a HEAD package which allows for track forking, much like branches in a git repository.


## Running guide

* Craft a package zip file (see above).
* Attach an Android device, make sure it shows up with `adb devices`.
* In the terminal type:

```
./do run robot --log-level verbose start master
./do run robot upload build package.zip
./do run robot upload subject my-app.apk
```

<point browser to localhost:8080>

