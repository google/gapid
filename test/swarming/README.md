# Swarming: capture-replay tests on real Android devices

This is a collection of scripts to run AGI tests on real Android devices.

We can access devices through Swarming, which is part of
[LUCI](https://chromium.googlesource.com/infra/infra/+/master/doc/users/services/about_luci.md),
hence the name "swarming tests".

A swarming test typically consists in installing an APK using Vulkan, and
running some `gapit` commands to capture-replay this APK. As we don't want to
include APKs in this repo, we store them on x20.

## Scripts running order

The scripts are primarly designed to run as part of Kokoro Linux builds. The
running order is:

1. The Kokoro build script `kokoro/linux/build.sh` uses `test/swarming/trigger.py` to
   schedule Swarming tests.

2. The Swarming bot runs `test/swarming/bot-harness.sh`, which wraps
   `test/swarming/bot-task.sh` to make sure to always turn off the device screen
   at the end of the task, even upon failure or time out. It is important to
   always turn the screen off to make sure devices can cool-down, otherwise the
   Swarming bot may become unusable.

3. The Kokoro build script uses `test/swarming/collect.py` to retrieve the
   Swarming tests results.

## How to run a Swarming test manually, outside of Kokoro

Requirements:

- LUCI tools installed, and the `LUCI_CLIENT_ROOT` environment variable set
  accordingly.

- valid Swarming/Isolate credentials.

You can trigger a Swarming test manually with:

1. Grab tests from x20:
   `x20/teams/android-graphics-tools/agi/kokoro/swarming`. Copy the `tests`
   folder under `test/swarming/`.

2. Use `./manual-run.sh tests/foobar` to trigger the foobar test in Swarming and
   collect its results.

## How to run a Swarming test on a local device

1. Grab tests from x20:
   `x20/teams/android-graphics-tools/agi/kokoro/swarming`. Copy the `tests`
   folder under `test/swarming/`.

2. Make sure you have only one Android device plugged into your host machine.

3. Use `./bot-harness.sh tests/foobar ${LOGDIR} ${TIMEOUT}` to run the foobar
   test on the device plugged in your host machine, storing logs in the
   `${LOGDIR}` directory, with a timeout limit of `${TIMEOUT}` seconds.

## Test format

A test is a folder that contains at least a `env.sh` file, which sets various
environment variables. A typical test contains an APK and a `env.sh` files with
the information needed by `bot-task.sh`, e.g.:

```
SWARMING_APK=com_example-app_version42.apk
SWARMING_PACKAGE=com.example.myApp
SWARMING_ACTIVITY=com.example.myApp.myActivity
SWARMING_STARTFRAME=5
SWARMING_NUMFRAME=5
```

Moreover, some `SWARMING_*` environment variables can be overriden on a
test-specific level. Some interesting ones:

```
# Array of devices to run on (bash-array format, e.g.: '(dev1 dev2 dev3)' )
SWARMING_DEVICES=(flame)
# Priority: lower value is higher priority
SWARMING_PRIORITY=100
# Timeout: maximum number of seconds for the task to terminate
SWARMING_TIMEOUT=300
```

## Nightly results

Nightly results are accumulated over nightly runs. To achieve this, the results
file is receieved as a build input from the latest build on x20, new results are
added to it, and the new results are produced as a nightly build artifact.

The results are stored in the `results.json` file. The precise format of this
JSON is defined in `test/swarming/collect.py`.
