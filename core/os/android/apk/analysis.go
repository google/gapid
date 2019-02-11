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

package apk

import (
	"archive/zip"
	"context"
	"path/filepath"

	"github.com/google/gapid/core/log"
)

// engineSignatures is used to identify the middleware engine used based on
// files found in the APK.
var engineSignatures = map[string]string{
	"libunity.so":         "unity",
	"libUnrealEngine3.so": "unreal3",
}

// Analyze parses the APK file and returns the APK's information.
func Analyze(ctx context.Context, apkData []byte) (*Information, error) {
	files, err := Read(ctx, apkData)
	if err != nil {
		return nil, err
	}
	m, err := GetManifest(ctx, files)
	if err != nil {
		return nil, err
	}

	activity, action, err := m.MainActivity(ctx)
	if err != nil {
		return nil, log.Err(ctx, err, "Finding launch activity")
	}
	return &Information{
		Name:        m.Package, // TODO
		VersionCode: int32(m.VersionCode),
		VersionName: m.VersionName,
		Package:     m.Package,
		Activity:    activity,
		Action:      action,
		Engine:      engine(files),
		ABI:         GatherABIs(files),
		Debuggable:  m.Application.Debuggable,
	}, nil
}

func engine(files []*zip.File) string {
	for _, file := range files {
		_, name := filepath.Split(file.Name)
		for sig, engine := range engineSignatures {
			if name == sig {
				return engine
			}
		}
	}
	return "<unknown>"
}

func (i *Information) URI() string {
	var uri string
	if i.Action != "" {
		uri = i.Action + ":"
	}
	uri += i.Package
	if i.Activity != "" {
		uri += "/" + i.Activity
	}
	return uri
}
