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
	"bytes"
	"context"
	"io/ioutil"
	"strings"

	"github.com/google/gapid/core/fault"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/android/binaryxml"
	"github.com/google/gapid/core/os/android/manifest"
	"github.com/google/gapid/core/os/device"
)

const (
	mainfestPath       = "AndroidManifest.xml"
	ErrMissingManifest = fault.Const("Couldn't find APK's manifest file.")
	ErrInvalidAPK      = fault.Const("File is not an APK.")
)

// Read parses the APK file, returning its contents.
func Read(ctx context.Context, apkData []byte) ([]*zip.File, error) {
	apkZip, err := zip.NewReader(bytes.NewReader(apkData), int64(len(apkData)))
	if err != nil {
		return nil, log.Err(ctx, ErrInvalidAPK, "")
	}
	return apkZip.File, nil
}

func GetManifestXML(ctx context.Context, files []*zip.File) (string, error) {
	manifestZipFile := findManifest(files)
	if manifestZipFile == nil {
		return "", log.Err(ctx, ErrMissingManifest, "")
	}
	manifestFile, err := manifestZipFile.Open()
	if err != nil {
		return "", log.Err(ctx, err, "Couldn't open APK's manifest")
	}
	defer manifestFile.Close()

	manifestData, _ := ioutil.ReadAll(manifestFile)

	return binaryxml.Decode(ctx, manifestData)
}

func GetManifest(ctx context.Context, files []*zip.File) (manifest.Manifest, error) {
	manifestXML, err := GetManifestXML(ctx, files)
	if err != nil {
		return manifest.Manifest{}, err
	}
	return manifest.Parse(ctx, manifestXML)
}

func findManifest(files []*zip.File) *zip.File {
	for _, file := range files {
		if file.Name == mainfestPath {
			return file
		}
	}
	return nil
}

// GatherABIs returns the list of ABI directories in the zip file.
func GatherABIs(files []*zip.File) []*device.ABI {
	abis := []*device.ABI{}
	seen := map[string]struct{}{}
	for _, f := range files {
		parts := strings.Split(f.Name, "/")
		if len(parts) >= 2 && parts[0] == "lib" {
			abiName := parts[1]
			if _, existing := seen[abiName]; !existing {
				abis = append(abis, device.AndroidABIByName(abiName))
				seen[abiName] = struct{}{}
			}
		}
	}
	return abis
}
