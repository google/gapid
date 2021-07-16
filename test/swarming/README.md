# Swarming: integration tests on real Android devices

This is a collection of scripts to run AGI tests on real Android devices.

We can access devices through Swarming, which is part of
[LUCI](https://chromium.googlesource.com/infra/infra/+/master/doc/users/services/about_luci.md),
hence the name "Swarming tests".

A Swarming test typically consists in installing the APK containing some GPU
workloads (e.g. a Vulkan demo), and running some `gapit` commands to test AGI
features on this workload. As we don't want to include APKs in this repo, we
store them on x20.

Googlers: see also [go/agi-doc-swarming](https://go/agi-doc-swarming).

## Scripts running order

The scripts are primarly designed to run as part of Kokoro Linux builds. The
running order is:

1. The Kokoro build script `kokoro/linux/build.sh` uses
   `test/swarming/trigger.py` to schedule Swarming tests.

2. The Swarming bot runs `test/swarming/bot-harness.py`, which wraps one of the
   scripts unders `test/swarming/bot-scripts/` and makes sure to **always** turn
   off the device screen at the end of the task, even upon failure or time
   out. It is important to always turn the screen off to make sure devices can
   cool-down, otherwise the Swarming bot may become unusable.

3. The Kokoro build script uses `test/swarming/collect.py` to retrieve the
   Swarming tests results.

## How to run a Swarming test manually, outside of Kokoro

Requirements:

- LUCI `isolate` and `swarming` tools installed in a directory pointed at with
the `LUCI_ROOT` environment variable. Search for "cipd" in
`kokoro/linux/build.sh` to check how to install these tools using chromium's
[CIPD](https://chromium.googlesource.com/chromium/src.git/+/master/docs/cipd.md).

- valid Swarming/Isolate credentials: run `${LUCI_ROOT}/isolate login` to login,
and `${LUCI_ROOT}/isolate whoami` to check your local credentials.

You can trigger a Swarming test manually with:

1. Grab tests from x20: `x20/teams/android-graphics-tools/agi/kokoro/swarming`.
   Copy the `tests` folder under `test/swarming/`. For instance, a "foobar" test
   folder (typically containing an APK and a `params.json` file) should be under
   `test/swarming/tests/foobar`.

2. Use `./manual-run.sh tests/foobar` to trigger the foobar test in Swarming and
   collect its results.

## How to run a Swarming test on a local device

1. Grab tests from x20:
   `x20/teams/android-graphics-tools/agi/kokoro/swarming`. Copy the `tests`
   folder under `test/swarming/`.

2. Make sure you have only one Android device plugged into your host machine.

3. Use `./bot-harness.py  ${TIMEOUT} tests/foobar ${LOGDIR}` to run the foobar
   test on the device plugged in your host machine, storing logs in the
   `${LOGDIR}` directory, with a timeout limit of `${TIMEOUT}` seconds.

## Test format

A test is a folder that contains at least a `params.json` file that defines the
test parameters. This JSON must contain at least a "script" entry which value is
the name of the script to use, this script being found under
`test/swarming/bot-scripts`. Moreover, this JSON typically defines additional
parameters used by the test script.

As an example, this is a possible `params.json` file to use the
`test/swarming/bot-scripts/video.py` script:

```
{
  "script": "video.py",
  "apk": "com.example.myApp.apk",
  "package": "com.example.myApp",
  "activity": "com.example.myApp.myActivity",
  "startframe": "5",
  "numframe": "2",
  "setprop": [
    {
      "name": "debug.myApp.foobar",
      "value": "42"
    }
  ]
}
```

This JSON can also define parameters related to how the Swarming task is
actually triggered, see the details by reading `test/swarming/trigger.py` source
code. Some noteworthy parameters:

```
{
  ...
  # List of device on which the test must be run
  "devices": [ "flame", "coral", ... ],
  # Swarming priority: lower value is higher priority
  "priority": "100",
  # Swarming task global timeout (seconds)
  "priority": "300",
  # Swarming task expiration: how long to wait to be scheduled (seconds)
  "expiration": "1200",
  ...
}
```

## Nightly results

Nightly results are accumulated over nightly runs. To achieve this, the results
are stored on x20, and used as both input and output of a given nightly run: a
nightly run receives the exiting results as an input, adds their own results to
it, and returns this as a result artifact that is stored on x20 again.

Results are stored in a `results.json` file. The precise format of this JSON is
defined in `test/swarming/collect.py`.
