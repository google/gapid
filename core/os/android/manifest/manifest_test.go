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

package manifest_test

import (
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/android/manifest"
)

const xml = `<manifest xmlns:android="http://schemas.android.com/apk/res/android"
          package="com.bobgames.bobsgame"
          android:versionCode="11"
          android:versionName="1.0">
    <application android:label="@string/app_name"
                 android:icon="@drawable/ic_launcher"
                 android:banner="@drawable/tv_banner"
                 android:allowBackup="true"
                 android:isGame="true"
                 android:theme="@android:style/Theme.NoTitleBar.Fullscreen">
        <activity android:name="BobsGame"
                  android:label="@string/app_name"
                  android:screenOrientation="sensorLandscape"
                  android:configChanges="orientation|keyboardHidden|screenSize">
            <intent-filter>
                <action android:name="android.intent.action.MAIN" />
                <category android:name="android.intent.category.LAUNCHER" />
            </intent-filter>
        </activity>
        <activity android:name="BobsGameTvActivity"
                  android:label="@string/app_name"
                  android:screenOrientation="landscape"
                  android:configChanges="orientation|keyboardHidden|screenSize">
        </activity>
    </application>
    <uses-sdk android:minSdkVersion="15" android:targetSdkVersion="21" />
    <uses-permission android:name="android.permission.INTERNET" />
    <uses-permission android:name="android.permission.ACCESS_NETWORK_STATE" />
    <uses-permission android:name="android.permission.NFC" />
    <uses-feature android:glEsVersion="0x00020000" />
    <uses-feature android:name="android.hardware.gamepad" android:required="false"/>
    <uses-feature android:name="android.hardware.touchscreen" android:required="true" />
</manifest>
`

func TestParseManifest(_t *testing.T) {
	ctx := log.Testing(_t)
	got, err := manifest.Parse(ctx, xml)
	assert.For(ctx, "err").ThatError(err).Succeeded()
	expected := manifest.Manifest{
		Package:     "com.bobgames.bobsgame",
		VersionCode: 11,
		VersionName: "1.0",
		Application: manifest.Application{
			Activities: []manifest.Activity{
				{
					Name: "BobsGame",
					IntentFilters: []manifest.IntentFilter{
						{
							Action: manifest.Action{
								Name: "android.intent.action.MAIN",
							},
							Categories: []manifest.Category{
								{
									Name: "android.intent.category.LAUNCHER",
								},
							},
						},
					},
				},
				{
					Name: "BobsGameTvActivity",
				},
			},
		},
		Features: []manifest.Feature{
			{
				GlEsVersion: "0x00020000",
			},
			{
				Name: "android.hardware.gamepad",
			},
			{
				Name:     "android.hardware.touchscreen",
				Required: true,
			},
		},
		Permissions: []manifest.Permission{
			{Name: "android.permission.INTERNET"},
			{Name: "android.permission.ACCESS_NETWORK_STATE"},
			{Name: "android.permission.NFC"},
		},
	}
	assert.For(ctx, "got").That(got).DeepEquals(expected)
}
