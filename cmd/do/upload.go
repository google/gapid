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

package main

import (
	"context"
	"fmt"
	"runtime"
)

func getPlatform() string {
	return map[string]string{
		"darwin":  "osx",
		"windows": "windows",
		"linux":   "linux",
	}[runtime.GOOS]
}

func doUpload(ctx context.Context, cfg Config, options UploadOptions, args ...string) {
	pkg := cfg.out().Join(fmt.Sprintf("%s-%s.zip", "gapid", getPlatform()))
	robotArgs := []string{"upload", "package"}
	if options.CL != "" {
		robotArgs = append(robotArgs, "-cl", options.CL)
	}
	if options.Description != "" {
		robotArgs = append(robotArgs, "-description", options.Description)
	}
	if options.Tag != "" {
		robotArgs = append(robotArgs, "-tag", options.Tag)
	}
	if options.Track != "" {
		robotArgs = append(robotArgs, "-track", options.Track)
	}
	if options.Uploader != "" {
		robotArgs = append(robotArgs, "-uploader", options.Uploader)
	}
	if options.BuilderAbi != "" {
		robotArgs = append(robotArgs, "-builderabi", options.BuilderAbi)
	}
	if options.ArtifactPath != "" {
		robotArgs = append(robotArgs, "-artifactpath", options.ArtifactPath)
	}
	robotArgs = append(robotArgs, pkg.String())
	robotArgs = append(robotArgs, args...)
	doRobot(ctx, cfg, options.RobotOptions, robotArgs...)
}
