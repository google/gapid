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
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	perfetto_pb "protos/perfetto/config"

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

type traceVerb struct{ TraceFlags }

func init() {
	verb := &traceVerb{}
	verb.Disable.Unknown.Extensions = true

	app.AddVerb(&app.Verb{
		Name:      "trace",
		ShortHelp: "Captures a gfx trace from an application",
		Action:    verb,
	})
}

type target func(opts *service.TraceOptions)

func (verb *traceVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	if flags.NArg() > 1 {
		app.Usage(ctx, "Expected at most one argument.")
		return nil
	}

	if verb.API == "" {
		app.Usage(ctx, "The API is required.")
		return nil
	}
	api, err := verb.traceType()
	if err != nil {
		return err
	}

	traceURI := verb.URI
	if traceURI == "" && verb.Local.Port == 0 {
		if flags.NArg() != 1 {
			if api.traceType != service.TraceType_Perfetto &&
				api.traceType != service.TraceType_Fuchsia {
				app.Usage(ctx, "Expected application name.")
				return nil
			}
		} else {
			traceURI = flags.Arg(0)
		}
	} else if flags.NArg() != 0 {
		app.Usage(ctx, "Expected no arguments when a URI or port is specified.")
		return nil
	}

	if api.traceType == service.TraceType_Perfetto {
		if verb.Perfetto == "" {
			app.Usage(ctx, "The Perfetto config is required for System Profiles.")
			return nil
		}
		if verb.Local.Port != 0 {
			app.Usage(ctx, "-local-port is not supported for System Profiles.")
			return nil
		}
	}

	client, err := getGapis(ctx, verb.Gapis, GapirFlags{})
	if err != nil {
		return log.Err(ctx, err, "Failed to connect to the GAPIS server")
	}
	defer client.Close()

	out := "trace" + api.traceExt
	var target target

	if verb.Local.Port != 0 {
		serverInfo, err := client.GetServerInfo(ctx)
		if err != nil {
			return err
		}
		traceDevice := serverInfo.GetServerLocalDevice()
		if traceDevice.GetID() == nil {
			return fmt.Errorf("The server was not started with a local device for tracing")
		}
		target = func(opts *service.TraceOptions) {
			opts.Device = traceDevice
			opts.App = &service.TraceOptions_Port{
				Port: uint32(verb.Local.Port),
			}
		}
	} else {
		// Find the actual trace URI from all of the devices
		devices, err := filterDevices(ctx, &verb.DeviceFlags, client)
		if err != nil {
			return err
		}

		if len(devices) == 0 {
			return fmt.Errorf("Could not find matching device")
		}

		if len(devices) == 1 && strings.HasPrefix(traceURI, "port:") {
			target = func(opts *service.TraceOptions) {
				opts.Device = devices[0]
				opts.App = &service.TraceOptions_Uri{
					Uri: traceURI,
				}
			}
		} else if len(devices) == 1 && strings.HasPrefix(traceURI, "apk:") {
			data, err := ioutil.ReadFile(traceURI[4:])
			if err != nil {
				return log.Errf(ctx, err, "Failed to read APK at %s", traceURI[4:])
			}
			target = func(opts *service.TraceOptions) {
				opts.Device = devices[0]
				opts.App = &service.TraceOptions_UploadApplication{
					UploadApplication: data,
				}
			}
		} else if traceURI == "" {
			if len(devices) != 1 {
				return log.Errf(ctx, nil, "Found multiple matching devices, please specify the trace device")
			}
			target = func(opts *service.TraceOptions) {
				opts.Device = devices[0]
			}
		} else {
			type info struct {
				uri        string
				device     *path.Device
				deviceName string
				name       string
			}
			var found []info

			for _, dev := range devices {
				targets, err := client.FindTraceTargets(ctx, &service.FindTraceTargetsRequest{
					Device: dev,
					Uri:    traceURI,
				})
				if err != nil {
					continue
				}

				dd, err := client.Get(ctx, dev.Path(), nil)
				if err != nil {
					return err
				}
				d := dd.(*device.Instance)

				for _, target := range targets {
					name := target.Name
					switch {
					case target.FriendlyApplication != "":
						name = target.FriendlyApplication
					case target.FriendlyExecutable != "":
						name = target.FriendlyExecutable
					}

					found = append(found, info{
						uri:        target.Uri,
						deviceName: d.Name,
						device:     dev,
						name:       name,
					})
				}
			}

			if len(found) == 0 {
				return fmt.Errorf("Could not find %+v to trace on any device", traceURI)
			}

			if len(found) > 1 {
				sb := strings.Builder{}
				fmt.Fprintf(&sb, "Found %v candidates: \n", traceURI)
				for i, f := range found {
					if i == 0 || found[i-1].deviceName != f.deviceName {
						fmt.Fprintf(&sb, "  %v:\n", f.deviceName)
					}
					fmt.Fprintf(&sb, "    %v\n", f.uri)
				}
				return log.Errf(ctx, nil, "%v", sb.String())
			}

			fmt.Printf("Tracing %+v\n", found[0].uri)
			out = found[0].name + api.traceExt
			target = func(opts *service.TraceOptions) {
				opts.Device = found[0].device
				opts.App = &service.TraceOptions_Uri{
					Uri: found[0].uri,
				}
			}
		}
	}

	if verb.Out != "" {
		out = verb.Out
	}

	options := &service.TraceOptions{
		Type:                          api.traceType,
		AdditionalCommandLineArgs:     verb.AdditionalArgs,
		Cwd:                           verb.WorkingDir,
		Environment:                   verb.Env,
		Duration:                      float32(verb.For.Seconds()),
		ObserveFrameFrequency:         uint32(verb.Observe.Frames),
		StartFrame:                    uint32(verb.Start.At.Frame),
		FramesToCapture:               uint32(verb.Capture.Frames),
		DeferStart:                    verb.Start.Defer,
		IgnoreFrameBoundaryDelimiters: verb.IgnoreAndroidFrameBoundary,
		NoBuffer:                      verb.No.Buffer,
		HideUnknownExtensions:         verb.Disable.Unknown.Extensions,
		RecordTraceTimes:              verb.Record.TraceTimes,
		ClearCache:                    verb.Clear.Cache,
		ServerLocalSavePath:           out,
		PipeName:                      verb.PipeName,
		DisableCoherentMemoryTracker:  verb.Disable.CoherentMemoryTracker,
		WaitForDebugger:               verb.WaitForDebugger,
		ProcessName:                   verb.ProcessName,
		LoadValidationLayer:           verb.LoadValidationLayer,
	}
	target(options)

	if api.traceType == service.TraceType_Perfetto {
		data, err := ioutil.ReadFile(verb.Perfetto)
		if err != nil {
			return log.Errf(ctx, err, "Failed to read Perfetto config")
		}
		options.PerfettoConfig = &perfetto_pb.TraceConfig{}
		if err := proto.UnmarshalText(string(data), options.PerfettoConfig); err != nil {
			return log.Errf(ctx, err, "Failed to parse Perfetto config")
		}
		dur := uint32(verb.For.Seconds() * 1000)
		if dur == 0 {
			dur = 10 * 60 * 1000
		}
		options.PerfettoConfig.DurationMs = proto.Uint32(dur)
	}

	handler, err := client.Trace(ctx)
	if err != nil {
		return err
	}
	defer handler.Dispose(ctx)

	defer app.AddInterruptHandler(func() {
		handler.Dispose(ctx)
	})()

	status, err := handler.Initialize(ctx, options)
	if err != nil {
		return err
	}
	log.I(ctx, "Trace Status %+v", status)

	handlerInstalled := false
	return task.Retry(ctx, 0, time.Second*3, func(ctx context.Context) (retry bool, err error) {
		status, err = handler.Event(ctx, service.TraceEvent_Status)
		if err == io.EOF {
			return true, nil
		}
		if err != nil {
			log.I(ctx, "Error %+v", err)
			return true, err
		}
		if status == nil {
			return true, nil
		}

		if status.BytesCaptured > 0 {
			if !handlerInstalled {
				crash.Go(func() {
					reader := bufio.NewReader(os.Stdin)
					if options.DeferStart {
						println("Press enter to start capturing...")
						_, _ = reader.ReadString('\n')
						_, _ = handler.Event(ctx, service.TraceEvent_Begin)
					}
					println("Press enter to stop capturing...")
					_, _ = reader.ReadString('\n')
					handler.Event(ctx, service.TraceEvent_Stop)
				})
				handlerInstalled = true
			}
			log.I(ctx, "Captured bytes: %+v", status.BytesCaptured)
		}
		if status.Status == service.TraceStatus_Done {
			return true, nil
		}
		return false, nil
	})
}

type traceType struct {
	traceType service.TraceType
	traceExt  string
}

func (verb *traceVerb) traceType() (traceType, error) {
	switch verb.API {
	case "angle":
		return traceType{
			service.TraceType_ANGLE,
			".gfxtrace",
		}, nil
	case "vulkan":
		return traceType{
			service.TraceType_Graphics,
			".gfxtrace",
		}, nil
	case "perfetto":
		return traceType{
			service.TraceType_Perfetto,
			".perfetto",
		}, nil
	case "fuchsia":
		return traceType{
			service.TraceType_Fuchsia,
			".fxt",
		}, nil
	default:
		return traceType{}, fmt.Errorf("Unknown API '%s'", verb.API)
	}
}
