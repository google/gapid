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
	"github.com/google/gapid/core/os/device"
)

const (
	AutoVideo VideoType = iota
	SxsVideo
	RegularVideo
	IndividualFrames
	SxsFrames
)

const (
	Json PackagesOutput = iota
	Proto
	ProtoString
	SimpleList
)

const (
	ExportPlain ExportMode = iota
	ExportOnScreen
	ExportDiagnostics
	ExportFrames
	ExportTimestamps
)

const (
	ModeMetrics PerfettoMode = iota
	ModeInteractive
	ModeList
)

const (
	OutputDefault PerfettoOutputFormat = iota
	OutputText
	OutputJson
)

type VideoType uint8

var videoTypeNames = map[VideoType]string{
	AutoVideo:        "auto",
	SxsVideo:         "sxs",
	RegularVideo:     "regular",
	IndividualFrames: "frames",
	SxsFrames:        "sxs-frames",
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

type ExportMode uint8

var exportModeNames = map[ExportMode]string{
	ExportPlain:       "plain",
	ExportOnScreen:    "on-screen",
	ExportDiagnostics: "diagnostics",
	ExportFrames:      "frames",
	ExportTimestamps:  "timestamps",
}

func (v *ExportMode) Choose(c interface{}) {
	*v = c.(ExportMode)
}
func (v ExportMode) String() string {
	return exportModeNames[v]
}

type PerfettoMode uint8

var perfettoModeNames = map[PerfettoMode]string{
	ModeMetrics:     "metrics",
	ModeInteractive: "interactive",
	ModeList:        "list",
}

func (v *PerfettoMode) Choose(c interface{}) {
	*v = c.(PerfettoMode)
}
func (v PerfettoMode) String() string {
	return perfettoModeNames[v]
}

type PerfettoOutputFormat uint8

var PerfettoOutputFormatNames = map[PerfettoOutputFormat]string{
	OutputDefault: "default",
	OutputText:    "text",
	OutputJson:    "json",
}

func (v *PerfettoOutputFormat) Choose(c interface{}) {
	*v = c.(PerfettoOutputFormat)
}
func (v PerfettoOutputFormat) String() string {
	return PerfettoOutputFormatNames[v]
}

type (
	CaptureFileFlags struct {
		CaptureID bool `help:"if true then interpret the capture file argument as a capture ID that is already loaded in gapis"`
	}
	CommandFilterFlags struct {
	}
	ObservationFlags struct {
		Ranges            bool `help:"if true then display the read and write ranges made by each command."`
		Data              bool `help:"if true then display the bytes read and written by each command. Implies Ranges."`
		TypedObservations bool `help:"if true then display the bytes read and written by each command as resolved types"`
	}
	DeviceFlags struct {
		Device string            `help:"Device to use. Either 'host' or the friendly name of the device"`
		Serial string            `help:"Serial of the device to use."`
		Os     string            `help:"Os of the device to use."`
		Env    flags.StringSlice `help:"List of environment variables to set, X=Y"`
		Ssh    struct {
			Config string `help:"The ssh config to use for finding remote devices"`
		}
	}
	DevicesFlags struct {
		Gapis GapisFlags
		OS    device.OSKind `help:"Only display devices of the given OS kind"`
	}
	ProfileFlags struct {
		Pprof string `help:"_produce a pprof file"`
		Trace string `help:"_produce a trace file"`
	}
	GapisFlags struct {
		Profile    ProfileFlags
		Port       int    `help:"gapis tcp port to connect to, 0 means start new instance."`
		Args       string `help:"_The arguments to be passed to gapis"`
		Token      string `help:"_The auth token to use when connecting to an existing server."`
		DisableLog bool   `help:"_Disable the log output"`
	}
	GapirFlags struct {
		DeviceFlags
		NoFallback bool   `help:"Do not fallback to another device if the requested one could not be found"`
		Args       string `help:"_The arguments to be passed to gapir"`
	}
	GapiiFlags struct {
		DeviceFlags
	}
	ReportFlags struct {
		Gapis            GapisFlags
		Gapir            GapirFlags
		Out              string `help:"output report path"`
		DisplayToSurface bool   `help:"display the frames rendered in the replay back to the surface"`
		CaptureFileFlags
	}
	ExportReplayFlags struct {
		Gapis          GapisFlags
		Gapir          GapirFlags
		OriginalDevice bool       `help:"export replay for the original device"`
		Out            string     `help:"output directory for commands and assets"`
		Mode           ExportMode `help:"generate special purposed trace"`
		Apk            string     `help:"(experimental) name of the stand-alone APK created to perform the replay. This name must be <app_package>.apk (e.g. com.example.replay.apk)"`
		SdkPath        string     `help:"Path to Android SDK directory (default: ANDROID_SDK_HOME environment variable)"`
		LoopCount      int        `help:"_The number of times to loop the trace. (experimental)"`
		CommandFilterFlags
		CaptureFileFlags
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
		Text     string    `help:"_summary prefix (use '║' for aligned columns, '¶' for new line)"`
		Commands bool      `help:"Treat every command as its own frame"`
		Frames   struct {
			Start   int `help:"frame to start capture from"`
			Count   int `help:"number of frames after Start to capture: -1 for all frames"`
			Minimum int `help:"_return error when less than this number of frames is found"`
		}
		NoOpt bool `help:"disables optimization of the replay stream"`
		CommandFilterFlags
		CaptureFileFlags
	}
	DumpShadersFlags struct {
		Gapis GapisFlags
		Gapir GapirFlags
		At    int `help:"command index to dump the resources after, e.g. '1234'"`
		CaptureFileFlags
	}
	DumpFBOFlags struct {
		Gapis GapisFlags
		Gapir GapirFlags
		Out   string `help:"output framebuffer directory path"`
		CommandFilterFlags
		CaptureFileFlags
	}
	DumpFlags struct {
		Gapis          GapisFlags
		Gapir          GapirFlags
		Raw            bool `help:"if true then the value of constants, instead of their names, will be dumped."`
		ShowDeviceInfo bool `help:"if true then show originating device information."`
		ShowABIInfo    bool `help:"if true then show information of the ABI used for the trace."`
		Observations   ObservationFlags
		CaptureFileFlags
	}
	CommandsFlags struct {
		Gapis                GapisFlags
		Gapir                GapirFlags
		Raw                  bool   `help:"if true then the value of constants, instead of their names, will be dumped."`
		Name                 string `help:"Filter to commands and groups with the specified name."`
		MaxChildren          int    `help:"_Maximum children per tree node."`
		GroupByAPI           bool   `help:"Group commands by api"`
		GroupByThread        bool   `help:"Group commands by thread"`
		GroupByDrawCall      bool   `help:"Group commands by draw call"`
		GroupByFrame         bool   `help:"Group commands by frame"`
		GroupByUserMarkers   bool   `help:"Group commands by user markers"`
		GroupBySubmission    bool   `help:"Group commands by submissions"`
		AllowIncompleteFrame bool   `help:"_Make a group for incomplete frames"`
		OnlyExecutedDraws    bool   `help:"Only show executed draw calls from within command buffers"`
		Observations         ObservationFlags
		CommandFilterFlags
		CaptureFileFlags
	}
	ReplaceResourceFlags struct {
		Gapis                GapisFlags
		Gapir                GapirFlags
		Handle               string `help:"required. handle or ID of the resource to replace"`
		ResourcePath         string `help:"file path for the new resource"`
		At                   int    `help:"command index to replace the resource(s) at, e.g. '1234'"`
		UpdateResourceBinary string `help:"shaders only. binary to run for every shader; consumes resource data from standard input and writes to standard output"`
		OutputTraceFile      string `help:"file name for the updated trace"`
		SkipOutput           bool   `help:"skip writing the modified trace to a file"`
		CaptureFileFlags
	}
	ServerPerformanceFlags struct {
		Gapis          GapisFlags
		Gapir          GapirFlags
		OriginalDevice bool `help:"export replay for the original device"`
		LoopCount      int  `help:"_The number of times to loop the trace. (experimental)"`
		CommandFilterFlags
		CaptureFileFlags
	}
	StateFlags struct {
		Gapis  GapisFlags
		Gapir  GapirFlags
		At     flags.U64Slice    `help:"command/subcommand index to get the state after, e.g. '[123, 0, 0, 4]'. 0 for first command. Empty for last"`
		Depth  int               `help:"How many nodes deep should the state tree be displayed. -1 for all"`
		Filter flags.StringSlice `help:"Which path (e.g. '[root, Devices]') through the tree should we filter to, default All"`
		CaptureFileFlags
	}
	StressTestFlags struct {
		Gapis GapisFlags
		Gapir GapirFlags
		CaptureFileFlags
	}
	TraceFlags struct {
		DeviceFlags
		Gapis          GapisFlags
		For            time.Duration `help:"duration to trace for"`
		Out            string        `help:"the file to generate"`
		AdditionalArgs string        `help:"additional arguments to pass to the application"`
		WorkingDir     string        `help:"working directory for the application"`
		URI            string        `help:"uri of the application to trace"`
		Observe        struct {
			Frames uint `help:"capture the framebuffer every n frames (0 to disable)"`
			Draws  uint `help:"capture the framebuffer every n draws (0 to disable)"`
		}
		Disable struct {
			Unknown struct {
				Extensions bool `help:"Hide unknown extensions from the application."`
			}
			CoherentMemoryTracker bool `help:"_disables the coherent memory tracker so it won't interfere with gdb during trace"`
		}
		Record struct {
			TraceTimes bool `help:"record trace timing into the capture"`
		}
		Clear struct {
			Cache bool `help:"clear package data before running it"`
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
		No struct {
			Buffer bool `help:"Do not buffer the output, this helps if the application crashes"`
		}
		API   string `help:"only capture the given API valid options are vulkan, angle and perfetto"`
		Local struct {
			Port int `help:"connect to an application already running on the server using this port"`
		}
		PipeName            string `help:"The name of the pipe to connect/listen to."`
		Perfetto            string `help:"File containing the Perfetto configuration proto."`
		WaitForDebugger     bool   `help:"Make GAPII wait for a debugger to attach"`
		ProcessName         string `help:"Name of the process to capture. Default to empty, i.e. capture any process. Useful for games that fork processes."`
		LoadValidationLayer bool   `help:"Load Vulkan validation layer at capture time, under the spy layer, to debug spy bugs. Android only."`
	}
	BenchmarkFlags struct {
		Gapis      GapisFlags
		Gapir      GapirFlags
		NumDraws   int              `help:"the number of draw calls to select"`
		Paths      []flags.U64Slice `help:"the (partial) paths to the draw calls to use"`
		Secondary  bool             `help:"whether to expect secondary command buffers inside the renderpasses"`
		Seed       int64            `help:"seed to use when making RNG decisions"`
		DumpTrace  string           `help:"dump a combined systrace of gapit and gapis"`
		SummaryOut string           `help:"filename to write summary to, default is stdout"`
		CsvOut     string           `help:"dump a csv file with the detailed stats"`
		DotOut     string           `help:"dump a dot file with the detailed stats"`
	}

	StatusFlags struct {
		Gapis                GapisFlags
		StatusUpdateInterval int `help:"Provides status updates at the given interval (in ms)"`
		MemoryUpdateInterval int `help:"Provides memory updates at the given interval (in ms)"`
	}

	PackagesFlags struct {
		DeviceFlags
		Gapis       GapisFlags
		Icons       bool           `help:"if true then package icons are also dumped."`
		IconDensity float64        `help:"_scale multiplier on icon density."`
		Format      PackagesOutput `help:"output format"`
		Out         string         `help:"output file, standard output if none"`
		DataHeader  string         `help:"marker to write before package data"`
		ADB         string         `help:"Path to the adb executable; leave empty to search the environment"`
	}
	ScreenshotFlags struct {
		Gapis         GapisFlags
		Gapir         GapirFlags
		At            []flags.U64Slice `help:"command/subcommand index (e.g. '[123, 0, 0, 4]') for the screenshot (repeatable)"`
		Frame         []int            `help:"frame index for the screenshot (repeatable). Empty for last"`
		Draws         bool             `help:"create a screenshot of every draw call in the requested frame(s) (only honored if using -frame)"`
		ExecutedDraws int              `help:"create a screenshot at every (total-draw-calls / this-flag-value) drawcall, e.g. for 100 draw calls, '-executeddraws 4' takes a screenshot at every 25th call"`
		Out           string           `help:"output image file (default 'screenshot.png')"`
		NoOpt         bool             `help:"disables optimization of the replay stream"`
		Attachment    string           `help:"the attachment to show (0-3 for color, d for depth, s for stencil)"`
		Overdraw      bool             `help:"renders the overdraw instead of the color framebuffer"`
		Max           struct {
			Overdraw int `help:"the amount of overdraw to map to white in the output"`
		}
		DisplayToSurface bool `help:"display the frames rendered in the replay back to the surface"`
		CommandFilterFlags
		CaptureFileFlags
	}
	UnpackFlags struct {
		Verbose bool `help:"if true, then output will not be truncated"`
	}

	MemoryFlags struct {
		Gapis GapisFlags
		At    flags.U64Slice `help:"command/subcommand index to get the memory after, e.g. '[123, 0, 0, 4]'. Empty for last"`
		CaptureFileFlags
	}
	PipelineFlags struct {
		Gapis GapisFlags
		At    flags.U64Slice `help:"command/subcommand index to get the pipeline after, e.g. '[123, 0, 0, 4]'. Empty for last"`
		Print struct {
			Shaders bool `help:"print the disassembled shaders along with the bound descriptor values"`
		}
		Compute bool `help:"print out the most recently bound compute pipeline instead of graphics pipeline"`
		CaptureFileFlags
	}
	TrimFlags struct {
		Gapis         GapisFlags
		Gapir         GapirFlags
		Commands      bool           `help:"Treat every command as its own frame"`
		ExtraCommands flags.U64Slice `help:"Additional commands to include (along with their dependencies)"`
		Frames        struct {
			Start int `help:"first frame to include (default 0)"`
			Count int `help:"number of frames to include: -1 for all frames (default -1)"`
		}
		Out string `help:"gfxtrace file to save the trimmed capture"`
		CommandFilterFlags
		CaptureFileFlags
	}
	TrimStateFlags struct {
		Gapis GapisFlags
		Gapir GapirFlags
		Out   string `help:"gfxtrace file to save the trimmed capture"`
		CaptureFileFlags
	}
	GetTimestampsFlags struct {
		Gapis     GapisFlags
		Gapir     GapirFlags
		LoopCount int    `help:"_The number of times to loop the trace. (experimental)"`
		Out       string `help:"output file to save the profiling result"`
	}

	GpuProfileFlags struct {
		Gapis        GapisFlags
		Gapir        GapirFlags
		Out          string           `help:"Output file (optional, if none then output goes to stdout)"`
		Json         bool             `help:"Return replay profiling data as JSON instead of text"`
		DisabledCmds []flags.U64Slice `help:"command/subcommand index (e.g. '[123, 0, 0, 4]') for disabling a draw call (repeatable)"`
		DisableAF    bool             `help:"Disable Anisotropic Filtering for all samplers"`
	}

	CreateGraphVisualizationFlags struct {
		Gapis  GapisFlags
		Out    string `help:"path to save graph visualization"`
		Format string `help:"output format of the graph: 'pbtxt' (Tensorboard) or 'dot' (Graphviz)"`
	}

	FramegraphFlags struct {
		Gapis GapisFlags
		Dot   string `help:"Store the framegraph in Graphviz dot format in this file"`
		Json  string `help:"Store the framegraph in JSON format in this file"`
	}

	SmokeTestsFlags struct {
	}

	Trace2apkFlags struct {
		Gapis GapisFlags
		Gapir GapirFlags
		CommandFilterFlags
		CaptureFileFlags
	}

	MakeDocFlags struct {
	}

	ValidateGpuProfilingFlags struct {
		DeviceFlags
		Gapis GapisFlags
	}

	PerfettoFlags struct {
		Gapis      GapisFlags
		Mode       PerfettoMode         `help:"Run mode: {metrics|interactive}. Default: metrics."`
		In         string               `help:"Input file. Refer to documentation for file format."`
		Categories string               `help:"Comma separated list of metric categories from the input file. Only valid if 'metrics' mode is selected."`
		Out        string               `help:"Output file."`
		Format     PerfettoOutputFormat `help:"Output file format: {text|json}."`
	}

	SplitFlags struct {
		Gapis GapisFlags
		Gapir GapirFlags
		CaptureFileFlags
		From uint64 `help:"The inclusive start index of the command range. Default: 0 (first command)"`
		To   uint64 `help:"The exclusive end index of the command range. Default: 0 (last command)"`
		Out  string `help:"Output file."`
	}
)
