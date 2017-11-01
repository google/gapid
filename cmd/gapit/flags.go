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
	"time"

	"github.com/google/gapid/core/app/flags"
	"github.com/google/gapid/core/os/file"
)

const (
	AutoVideo VideoType = iota
	SxsVideo
	RegularVideo
	IndividualFrames
)

const (
	Json PackagesOutput = iota
	Proto
	ProtoString
	SimpleList
)

type VideoType uint8

var videoTypeNames = map[VideoType]string{
	AutoVideo:        "auto",
	SxsVideo:         "sxs",
	RegularVideo:     "regular",
	IndividualFrames: "frames",
}

func (v *VideoType) Choose(c interface{}) {
	*v = c.(VideoType)
}
func (v VideoType) String() string {
	return videoTypeNames[v]
}

type PackagesOutput uint8

var packagesOutputNames = map[PackagesOutput]string{
	Json:        "json",
	Proto:       "proto",
	ProtoString: "proto-string",
	SimpleList:  "list",
}

func (v *PackagesOutput) Choose(c interface{}) {
	*v = c.(PackagesOutput)
}
func (v PackagesOutput) String() string {
	return packagesOutputNames[v]
}

type (
	CommandFilterFlags struct {
		Context int `help:"Filter to the i'th context."`
	}
	ObservationFlags struct {
		Ranges bool `help:"if true then display the read and write ranges made by each command."`
		Data   bool `help:"if true then display the bytes read and written by each command. Implies Ranges."`
	}
	DeviceFlags struct {
		Device string `help:"Device to spawn on. One of: 'host', 'android' or <device-serial>"`
	}
	DevicesFlags struct {
		Gapis GapisFlags
	}
	GapisFlags struct {
		Profile string `help:"produce a pprof file from gapis"`
		Port    int    `help:"gapis tcp port to connect to, 0 means start new instance."`
		Args    string `help:"The arguments to be passed to gapis"`
		Token   string `help:"The auth token to use when connecting to an existing server."`
	}
	GapirFlags struct {
		DeviceFlags
		Args string `help:"The arguments to be passed to gapir"`
	}
	GapiiFlags struct {
		DeviceFlags
	}
	InfoFlags struct {
	}
	ReportFlags struct {
		Gapis GapisFlags
		Gapir GapirFlags
		Out   string `help:"output report path"`
		CommandFilterFlags
	}
	VideoFlags struct {
		Gapis GapisFlags
		Gapir GapirFlags
		FPS   int    `help:"frames per second"`
		Out   string `help:"output video path"`
		Max   struct {
			Width  int `help:"maximum video width"`
			Height int `help:"maximum video height"`
		}
		Type     VideoType `help:"type of output to produce"`
		Text     string    `help:"summary prefix (use '║' for aligned columns, '¶' for new line)"`
		Commands bool      `help:"Treat every command as its own frame"`
		Frames   struct {
			Start int `help:"frame to start capture from"`
			Count int `help:"number of frames after Start to capture: -1 for all frames"`
		}
		CommandFilterFlags
	}
	DumpShadersFlags struct {
		Gapis GapisFlags
		Gapir GapirFlags
		At    int `help:"command index to dump the resources after"`
	}
	DumpFlags struct {
		Gapis          GapisFlags
		Gapir          GapirFlags
		Raw            bool `help:"if true then the value of constants, instead of their names, will be dumped."`
		ShowDeviceInfo bool `help:"if true then show originating device information."`
		ShowABIInfo    bool `help:"if true then show information of the ABI used for the trace."`
		Observations   ObservationFlags
	}
	CommandsFlags struct {
		Gapis                  GapisFlags
		Gapir                  GapirFlags
		Raw                    bool   `help:"if true then the value of constants, instead of their names, will be dumped."`
		Name                   string `help:"Filter to commands and groups with the specified name."`
		MaxChildren            int    `help:"Maximum children per tree node."`
		GroupByAPI             bool   `help:"Group commands by api"`
		GroupByContext         bool   `help:"Group commands by context"`
		GroupByThread          bool   `help:"Group commands by thread"`
		GroupByDrawCall        bool   `help:"Group commands by draw call"`
		GroupByFrame           bool   `help:"Group commands by frame"`
		GroupByUserMarkers     bool   `help:"Group commands by user markers"`
		IncludeNoContextGroups bool   `help:"Include no context groups"`
		AllowIncompleteFrame   bool   `help:"Make a group for incomplete frames"`
		Observations           ObservationFlags
		CommandFilterFlags
	}
	StateFlags struct {
		Gapis GapisFlags
		Gapir GapirFlags
		At    flags.U64Slice `help:"command/subcommand index to get the state after. Empty for last"`
	}
	StressTestFlags struct {
		Gapis GapisFlags
		Gapir GapirFlags
	}
	TraceFlags struct {
		Gapii GapiiFlags
		For   time.Duration `help:"duration to trace for"`
		Out   string        `help:"the file to generate"`
		Local struct {
			Port       int       `help:"capture a local program instead of using ADB"`
			App        file.Path `help:"a local program to trace"`
			Args       string    `help:"arguments to pass to the traced program"`
			WorkingDir string    `help:"working directory for the process"`
		}
		Android struct {
			Package  string `help:"the full package name"`
			Activity string `help:"the full activity name"`
			Action   string `help:"the full action name"`
			Attach   bool   `help:"attach to running instance of the specified package"`
			Logcat   bool   `help:"print the output of logcat while tracing"`
		}
		APK     file.Path `help:"the path to an apk to install"`
		Observe struct {
			Frames uint `help:"capture the framebuffer every n frames (0 to disable)"`
			Draws  uint `help:"capture the framebuffer every n draws (0 to disable)"`
		}
		Disable struct {
			PCS bool `help:"disable pre-compiled shaders"`
		}
		Record struct {
			Errors bool `help:"record device error state"`
			Inputs bool `help:"record the inputs to file"`
		}
		Clear struct {
			Cache bool `help:"clear package data before running it"`
		}
		Input struct {
			File string `help:"the file to use for recorded inputs"`
		}
		Replay struct {
			Inputs bool `help:"replay the inputs from file"`
		}
		Start struct {
			Defer bool `help:"defers the start of the trace until <enter> is pressed. Only valid for Vulkan."`
			At    struct {
				Frame int `help:"defers the start of the trace until given frame. Only valid for Vulkan. Not compatible with start-defer."`
			}
		}
		Capture struct {
			Frames int `help:"only capture the given number of frames. 0 for all"`
		}
		API string `help:"only capture the given API valid options are gles and vulkan"`
		ADB string `help:"Path to the adb executable; leave empty to search the environment"`
	}
	PackagesFlags struct {
		DeviceFlags
		Icons       bool           `help:"if true then package icons are also dumped."`
		IconDensity float64        `help:"scale multiplier on icon density."`
		Format      PackagesOutput `help:"output format"`
		Out         string         `help:"output file, standard output if none"`
		DataHeader  string         `help:"marker to write before package data"`
		ADB         string         `help:"Path to the adb executable; leave empty to search the environment"`
	}
	ScreenshotFlags struct {
		Gapis GapisFlags
		Gapir GapirFlags
		At    flags.U64Slice `help:"command/subcommand index for the screenshot. Empty for last"`
	}
	UnpackFlags struct {
		Verbose bool `help:"if true, then output will not be truncated"`
	}
)
