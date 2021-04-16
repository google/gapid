// Copyright (C) 2017 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package adb_test

import (
	"context"
	"fmt"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/os/android/adb"
	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/core/os/shell"
	"github.com/google/gapid/core/os/shell/stub"
)

var (
	adbPath = file.Abs("/adb")

	validDevices = stub.RespondTo(adbPath.System()+` devices`, `
List of devices attached
adb server version (36) doesn't match this client (35); killing...
* daemon not running. starting it now on port 5037 *
* daemon started successfully *
debug_device                unknown
debug_device2               unknown
dumpsys_device              offline
error_device                device
install_device              unauthorized
invalid_device              unknown
logcat_device               unauthorized
no_pgrep_no_ps_device       unknown
no_pgrep_ok_ps_device       offline
ok_pgrep_no_ps_device       device
ok_pgrep_ok_ps_device       unauthorized
production_device           unknown
pull_device                 offline
push_device                 device
rooted_device               unauthorized
run_device                  unknown
screen_off_locked_device    offline
screen_off_unlocked_device  offline
screen_on_locked_device     offline
screen_on_unlocked_device   device
serial_do_match             device
`)
	emptyDevices = stub.RespondTo(adbPath.System()+` devices`, `
List of devices attached
* daemon not running. starting it now on port 5037 *
* daemon started successfully *
`)
	invalidDevices = stub.RespondTo(adbPath.System()+` devices`, `
List of devices attached
* daemon not running. starting it now on port 5037 *
* daemon started successfully *
production_device        unauthorized invalid
`)
	invalidStatus = stub.RespondTo(adbPath.System()+` devices`, `
List of devices attached
* daemon not running. starting it now on port 5037 *
* daemon started successfully *
production_device        invalid
`)
	notDevices = stub.RespondTo(adbPath.System()+` devices`, ``)
	devices    = &stub.Delegate{Handlers: []shell.Target{validDevices}}
)

func init() {
	adb.ADB = file.Abs("/adb")

	shell.LocalTarget = stub.OneOf(
		devices,
		stub.RespondTo(adbPath.System()+` -s dumpsys_device shell dumpsys package`, `
Activity Resolver Table:
  Non-Data Actions:
    android.intent.action.MAIN:
      43178558 com.google.foo/.FooActivity filter 4327f110
      12345678 com.google.qux/.QuxActivity filter 1256e899
    com.google.android.FOO:
      43178558 com.google.foo/.FooActivity filter 431d7db8
    android.intent.action.SEARCH:
      43178558 com.google.foo/.FooActivity filter 4327cc40

Packages:
  Package [com.google.foo] (ffffffc):
    userId=12345
    primaryCpuAbi=armeabi-v7a
    secondaryCpuAbi=null
    versionCode=902107 minSdk=14 targetSdk=15
    flags=[ HAS_CODE ALLOW_CLEAR_USER_DATA ALLOW_BACKUP ]
  Package [com.google.qux] (cafe0000):
    userId=34567
    primaryCpuAbi=armeabi-v7a
    secondaryCpuAbi=null
    versionCode=123456 targetSdk=15
    flags=[ DEBUGGABLE HAS_CODE ALLOW_CLEAR_USER_DATA ALLOW_BACKUP ]
`),

		// Screen state queries
		stub.RespondTo(adbPath.System()+` -s screen_off_locked_device shell dumpsys window`, `
mHasSoftInput=true
mAwake=false
mScreenOnEarly=true mScreenOnFully=true
mDockLayer=268435456 mStatusBarLayer=-1
mShowingDream=false mDreamingLockscreen=true mDreamingSleepToken=null
mFocusedWindow=Window{7f1fdc4 u0 com.google.gapid.gltests/com.google.gapid.gltests.MainActivity}
...`),
		stub.RespondTo(adbPath.System()+` -s screen_off_unlocked_device shell dumpsys window`, `
mHomePressed=false
mAwake=falsemScreenOnEarly=true mScreenOnFully=true
mKeyguardDrawComplete=true mWindowManagerDrawComplete=true
mDockLayer=268435456 mStatusBarLayer=0
mShowingDream=false mShowingLockscreen=false mDreamingSleepToken=null
mStatusBar=Window{5033a83 u0 StatusBar} isStatusBarKeyguard=false
...`),
		stub.RespondTo(adbPath.System()+` -s screen_on_locked_device shell dumpsys window`, `
mHasSoftInput=true
mAwake=true
mScreenOnEarly=true mScreenOnFully=true
mDockLayer=268435456 mStatusBarLayer=-1
mShowingDream=false mDreamingLockscreen=true mDreamingSleepToken=null
mFocusedWindow=Window{7f1fdc4 u0 com.google.gapid.gltests/com.google.gapid.gltests.MainActivity}
...`),
		stub.RespondTo(adbPath.System()+` -s screen_on_unlocked_device shell dumpsys window`, `
mHomePressed=false
mAwake=truemScreenOnEarly=true mScreenOnFully=true
mKeyguardDrawComplete=true mWindowManagerDrawComplete=true
mDockLayer=268435456 mStatusBarLayer=0
mShowingDream=false mShowingLockscreen=false mDreamingSleepToken=null
mStatusBar=Window{5033a83 u0 StatusBar} isStatusBarKeyguard=false
...`),

		// Pid queries.
		stub.Regex(`adb -s ok_pgrep_\S*device shell pgrep .* com.google.foo`, stub.Respond("")),
		stub.Regex(`adb -s ok_pgrep\S*device shell pgrep -n -f com.google.bar`, stub.Respond("2778")),
		stub.RespondTo(adbPath.System()+` -s no_pgrep_ok_ps_device shell ps`, `
u0_a11    21926 5061  1976096 42524 SyS_epoll_ 0000000000 S com.google.android.gms
u0_a111   2778  5062  1990796 59268 SyS_epoll_ 0000000000 S com.google.bar
u0_a69    22841 5062  1255788 88672 SyS_epoll_ 0000000000 S com.example.meh`),
		stub.Regex(`adb -s \S*no_ps\S*device shell ps`, stub.Respond("/system/bin/sh: ps: not found")),
		stub.Regex(`adb -s \S*no_pgrep\S*device shell pgrep \S+`, stub.Respond("/system/bin/sh: pgrep: not found")),

		stub.RespondTo(adbPath.System()+` -s invalid_device shell dumpsys window`, `not a normal response`),

		// Root command responses
		stub.RespondTo(adbPath.System()+` -s production_device root`, `adbd cannot run as root in production builds`),
		&stub.Sequence{
			stub.RespondTo(adbPath.System()+` -s debug_device root`, `restarting adbd as root`),
			stub.RespondTo(adbPath.System()+` -s debug_device root`, `some random output`),
			stub.RespondTo(adbPath.System()+` -s debug_device root`, `adbd is already running as root`),
		},
		stub.RespondTo(adbPath.System()+` -s debug_device2 root`, `* daemon not running. starting it now at tcp:5036 *
* daemon started successfully *`),
		stub.RespondTo(adbPath.System()+` -s rooted_device root`, `adbd is already running as root`),
		stub.RespondTo(adbPath.System()+` -s invalid_device root`, `not a normal response`),
		stub.Match(adbPath.System()+` -s error_device root`, &stub.Response{WaitErr: fmt.Errorf(`not a normal response`)}),

		// SELinuxEnforcing command responses
		stub.RespondTo(adbPath.System()+` -s production_device shell getenforce`, `Enforcing`),
		stub.RespondTo(adbPath.System()+` -s debug_device shell getenforce`, `Permissive`),
		stub.Match(adbPath.System()+` -s error_device shell getenforce`, &stub.Response{WaitErr: fmt.Errorf(`not a normal response`)}),

		// Logcat command responses
		stub.RespondTo(adbPath.System()+` -s logcat_device logcat -v long -T 0 GAPID:V *:W`, `
[ 03-29 15:16:29.514 24153:24153 V/AndroidRuntime ]
>>>>>> START com.android.internal.os.RuntimeInit uid 0 <<<<<<


[ 03-29 15:16:29.518 24153:24153 D/AndroidRuntime ]
CheckJNI is OFF


[ 03-29 15:16:29.761 31608:31608 I/Finsky   ]
[1] PackageVerificationReceiver.onReceive: Verification requested, id = 331

[ 03-29 15:16:32.205 31608:31655 W/qtaguid  ]
Failed write_ctrl(u 48) res=-1 errno=22

[ 03-29 15:16:32.205 31608:31655 E/NetworkManagementSocketTagger ]
untagSocket(48) failed with errno -22

[ 03-29 15:16:32.219 31608:31608 F/Finsky   ]
[1] PackageVerificationReceiver.onReceive: Verification requested, id = 331
`),

		// Common responses to all devices
		stub.Regex(`adb -s .* shell getprop ro\.build\.product`, stub.Respond("flame")),
		stub.Regex(`adb -s .* shell getprop ro\.build\.version\.release`, stub.Respond("10")),
		stub.Regex(`adb -s .* shell getprop ro\.build\.description`, stub.Respond("flame-user 10 QQ1A.191003.005 5926727 release-keys")),
		stub.Regex(`adb -s .* shell getprop ro\.product\.cpu\.abi`, stub.Respond("arm64-v8a")),
		stub.Regex(`adb -s .* shell getprop ro\.build\.version\.sdk`, stub.Respond("29")),
		stub.Regex(`adb -s .* shell setprop persist\.traced\.enable 1`, stub.Respond("")),

		stub.Regex(`adb -s .* shell input .*`, stub.Respond("")),
	)
}

// expectedCommand uses the standard response for an unexpected command to the stub in order to check the command itself
// was as expected.
func expectedCommand(ctx context.Context, expect string, err error) {
	assert.For(ctx, "Expected an unmatched command").
		ThatError(err).HasMessage(fmt.Sprintf(`Failed to start process
   Cause: unmatched:%s`, expect))
}
