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
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/app/status"
	"github.com/google/gapid/core/event/task"
	img "github.com/google/gapid/core/image"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/client"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
	"github.com/google/gapid/gapis/stringtable"
)

type benchmarkVerb struct {
	BenchmarkFlags
	startTime            time.Time
	beforeStartTraceTime time.Time
	traceInitializedTime time.Time
	traceDoneTime        time.Time
	traceSizeInBytes     int64
	traceFrames          int
	gapisInteractiveTime time.Time
	gapisCachingDoneTime time.Time
	interactionStartTime time.Time
	interactionDoneTime  time.Time
}

var BenchmarkName = "benchmark.gfxtrace"

func init() {
	verb := &benchmarkVerb{}

	app.AddVerb(&app.Verb{
		Name:      "benchmark",
		ShortHelp: "Runs a set of benchmarking tests on an application",
		Action:    verb,
	})
}

// We wnat to write our some of our own tracing data
type profileTask struct {
	Name      string `json:"name,omitempty"`
	Pid       uint64 `json:"pid"`
	Tid       uint64 `json:"tid"`
	EventType string `json:"ph"`
	Ts        int64  `json:"ts"`
	S         string `json:"s,omitempty"`
}

type u64List []uint64

// Len is the number of elements in the collection.
func (s u64List) Len() int { return len(s) }

// Less reports whether the element with
// index i should sort before the element with index j.
func (s u64List) Less(i, j int) bool { return s[i] < s[j] }

// Swap swaps the elements with indexes i and j.
func (s u64List) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

func (verb *benchmarkVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	oldCtx := ctx
	ctx = status.Start(ctx, "Initializing GAPIS")

	if verb.NumFrames == 0 {
		verb.NumFrames = 100
	}

	verb.startTime = time.Now()

	client, err := getGapis(ctx, verb.Gapis, verb.Gapir)
	if err != nil {
		return log.Err(ctx, err, "Failed to connect to the GAPIS server")
	}
	defer func() {
		if err := client.Close(); err != nil {
			log.E(ctx, "Error closing client: %v", err)
		}
	}()

	var writeTrace func(path string, gapisTrace, gapitTrace *bytes.Buffer) error

	if verb.DumpTrace != "" {
		gapitTrace := &bytes.Buffer{}
		gapisTrace := &bytes.Buffer{}
		stopGapitTrace := status.RegisterTracer(gapitTrace)
		stopGapisTrace, err := client.Profile(ctx, nil, gapisTrace, 1)
		if err != nil {
			return err
		}

		defer func() {
			stopGapitTrace()
			stopGapisTrace()
			// writeTrace may not be initialized yet
			if writeTrace != nil {
				if err := writeTrace(verb.DumpTrace, gapisTrace, gapitTrace); err != nil {
					log.E(ctx, "Failed to write trace: %v", err)
				}
			}
		}()
	}

	stringTables, err := client.GetAvailableStringTables(ctx)
	if err != nil {
		return log.Err(ctx, err, "Failed get list of string tables")
	}

	var stringTable *stringtable.StringTable
	if len(stringTables) > 0 {
		// TODO: Let the user pick the string table.
		stringTable, err = client.GetStringTable(ctx, stringTables[0])
		if err != nil {
			return log.Err(ctx, err, "Failed get string table")
		}
	}
	_ = stringTable

	status.Finish(ctx)

	if flags.NArg() > 0 {
		traceURI := flags.Arg(0)
		verb.doTrace(ctx, client, traceURI)
		verb.traceDoneTime = time.Now()
	}

	s, err := os.Stat(BenchmarkName)
	if err != nil {
		return err
	}

	verb.traceSizeInBytes = s.Size()
	status.Event(ctx, status.GlobalScope, "Trace Size %+v", verb.traceSizeInBytes)

	ctx = status.Start(oldCtx, "Initializing Capture")
	c, err := client.LoadCapture(ctx, BenchmarkName)
	if err != nil {
		return err
	}

	devices, err := client.GetDevicesForReplay(ctx, c)
	if err != nil {
		panic(err)
	}
	if len(devices) == 0 {
		panic("No devices")
	}

	resolveConfig := &path.ResolveConfig{
		ReplayDevice: devices[0],
	}
	device := devices[0]

	wg := sync.WaitGroup{}

	var resources *service.Resources
	wg.Add(1)
	go func() {
		ctx := status.Start(oldCtx, "Resolving Resources")
		defer status.Finish(ctx)
		boxedResources, err := client.Get(ctx, c.Resources().Path(), resolveConfig)
		if err != nil {
			panic(err)
		}
		resources = boxedResources.(*service.Resources)

		wg.Done()
	}()

	wg.Add(1)
	go func() {
		ctx := status.Start(oldCtx, "Getting Report")
		defer status.Finish(ctx)

		_, err := client.Get(ctx, c.Commands().Path(), resolveConfig)
		if err != nil {
			panic(err)
		}

		_, err = client.Get(ctx, c.Report(device, false).Path(), resolveConfig)
		wg.Done()
	}()

	var commandToClick *path.Command

	wg.Add(1)

	var events []*service.Event
	go func() {
		ctx := status.Start(oldCtx, "Getting Thumbnails")
		defer status.Finish(ctx)
		var e error
		events, e = getEvents(ctx, client, &path.Events{
			Capture:                 c,
			AllCommands:             false,
			FirstInFrame:            false,
			LastInFrame:             true,
			FramebufferObservations: false,
			IncludeTiming:           true,
		})
		if e != nil {
			panic(e)
		}
		verb.traceFrames = len(events)

		gotThumbnails := sync.WaitGroup{}
		//Get thumbnails
		settings := &path.RenderSettings{
			MaxWidth:                  uint32(256),
			MaxHeight:                 uint32(256),
			DisableReplayOptimization: verb.NoOpt,
			DisplayToSurface:          false,
		}
		numThumbnails := 10
		if len(events) < 10 {
			numThumbnails = len(events)
		}
		commandToClick = events[len(events)-1].Command
		for i := len(events) - numThumbnails; i < len(events); i++ {
			gotThumbnails.Add(1)
			hints := &path.UsageHints{Preview: true}
			go func(i int) {
				fbPath := &path.FramebufferAttachment{
					After:          events[i].Command,
					Index:          0,
					RenderSettings: settings,
					Hints:          hints,
				}
				iip, err := client.Get(ctx, fbPath.Path(), resolveConfig)

				iio, err := client.Get(ctx, iip.(*service.FramebufferAttachment).GetImageInfo().Path(), resolveConfig)
				if err != nil {
					panic(log.Errf(ctx, err, "Get frame image.Info failed"))
				}
				ii := iio.(*img.Info)
				dataO, err := client.Get(ctx, path.NewBlob(ii.Bytes.ID()).Path(), resolveConfig)
				if err != nil {
					panic(log.Errf(ctx, err, "Get frame image data failed"))
				}
				_, _, _ = int(ii.Width), int(ii.Height), dataO.([]byte)
				gotThumbnails.Done()
			}(i)
		}
		gotThumbnails.Wait()
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		ctx := status.Start(oldCtx, "Resolving Command Tree")

		filter := &path.CommandFilter{}

		treePath := c.CommandTree(filter)
		treePath.GroupByApi = true
		treePath.GroupByDrawCall = true
		treePath.GroupByFrame = true
		treePath.GroupByUserMarkers = true
		treePath.GroupBySubmission = true
		treePath.AllowIncompleteFrame = true
		treePath.MaxChildren = int32(2000)

		boxedTree, err := client.Get(ctx, treePath.Path(), resolveConfig)
		if err != nil {
			panic(log.Err(ctx, err, "Failed to load the command tree"))
		}
		tree := boxedTree.(*service.CommandTree)

		boxedNode, err := client.Get(ctx, tree.Root.Path(), resolveConfig)
		if err != nil {
			panic(log.Errf(ctx, err, "Failed to load the node at: %v", tree.Root.Path()))
		}

		n := boxedNode.(*service.CommandTreeNode)
		numChildren := 30
		if n.NumChildren < 30 {
			numChildren = int(n.NumChildren)
		}
		gotThumbnails := sync.WaitGroup{}
		gotNodes := sync.WaitGroup{}
		settings := &path.RenderSettings{
			MaxWidth:                  uint32(64),
			MaxHeight:                 uint32(64),
			DisableReplayOptimization: verb.NoOpt,
			DisplayToSurface:          false,
		}
		hints := &path.UsageHints{Background: true}
		tnCtx := status.Start(oldCtx, "Resolving Command Thumbnails")
		for i := 0; i < numChildren; i++ {
			gotThumbnails.Add(1)
			gotNodes.Add(1)
			go func(i int) {
				defer gotThumbnails.Done()
				boxedChild, err := client.Get(ctx, tree.Root.Child(uint64(i)).Path(), resolveConfig)
				if err != nil {
					panic(err)
				}
				child := boxedChild.(*service.CommandTreeNode)
				gotNodes.Done()
				fbPath := &path.FramebufferAttachment{
					After:          child.Representation,
					Index:          0,
					RenderSettings: settings,
					Hints:          hints,
				}
				iip, err := client.Get(tnCtx, fbPath.Path(), resolveConfig)

				iio, err := client.Get(tnCtx, iip.(*service.FramebufferAttachment).GetImageInfo().Path(), resolveConfig)
				if err != nil {
					return
				}
				ii := iio.(*img.Info)
				dataO, err := client.Get(tnCtx, path.NewBlob(ii.Bytes.ID()).Path(), resolveConfig)
				if err != nil {
					panic(log.Errf(tnCtx, err, "Get frame image data failed"))
				}
				_, _, _ = int(ii.Width), int(ii.Height), dataO.([]byte)
			}(i)
		}

		gotNodes.Wait()
		status.Finish(ctx)
		verb.gapisInteractiveTime = time.Now()

		gotThumbnails.Wait()
		status.Finish(tnCtx)
		wg.Done()
	}()
	// Done initializing capture
	wg.Wait()
	verb.gapisCachingDoneTime = time.Now()

	// At this point we are Interactive. All pre-loading is done:
	// Next we have to actually handle an interaction
	status.Finish(ctx)

	status.Event(ctx, status.GlobalScope, "Load done, interaction starting %+v", verb.traceSizeInBytes)

	// Sleep for 20 seconds so that the server is idle before we do
	// the last part of the benchmark. When we open a trace we, in the
	// background, generate the Dependency Graph. If we start making
	// requests before that is done, we will skew the benchmarking
	// results for 2 reasons:
	//
	//  1. Because the CPU will be under load for building the Dep
	//  graph
	//
	//  2. Because requests that normally use the dep graph (getting
	//  the framebuffer observations in this case) won't take
	//  advantage of it.
	time.Sleep(20 * time.Second)

	ctx = status.Start(oldCtx, "Interacting with frame")
	// One interaction done
	verb.interactionStartTime = time.Now()

	interactionWG := sync.WaitGroup{}
	// Get the framebuffer
	interactionWG.Add(1)
	go func() {
		ctx = status.Start(oldCtx, "Getting Framebuffer")
		defer status.Finish(ctx)
		defer interactionWG.Done()
		hints := &path.UsageHints{Primary: true}
		settings := &path.RenderSettings{
			MaxWidth:                  uint32(0xFFFFFFFF),
			MaxHeight:                 uint32(0xFFFFFFFF),
			DisableReplayOptimization: verb.NoOpt,
			DisplayToSurface:          false,
		}
		fbPath := &path.FramebufferAttachment{
			After:          commandToClick,
			Index:          0,
			RenderSettings: settings,
			Hints:          hints,
		}
		iip, err := client.Get(ctx, fbPath.Path(), resolveConfig)

		iio, err := client.Get(ctx, iip.(*service.FramebufferAttachment).GetImageInfo().Path(), resolveConfig)
		if err != nil {
			return
		}
		ii := iio.(*img.Info)
		dataO, err := client.Get(ctx, path.NewBlob(ii.Bytes.ID()).Path(), resolveConfig)
		if err != nil {
			panic(log.Errf(ctx, err, "Get frame image data failed"))
		}
		_, _, _ = int(ii.Width), int(ii.Height), dataO.([]byte)
	}()

	// Get state tree
	interactionWG.Add(1)
	go func() {
		ctx = status.Start(oldCtx, "Resolving State Tree")
		defer status.Finish(ctx)
		defer interactionWG.Done()
		//commandToClick
		boxedTree, err := client.Get(ctx, commandToClick.StateAfter().Tree().Path(), resolveConfig)
		if err != nil {
			panic(log.Err(ctx, err, "Failed to load the state tree"))
		}
		tree := boxedTree.(*service.StateTree)

		boxedRoot, err := client.Get(ctx, tree.Root.Path(), resolveConfig)
		if err != nil {
			panic(log.Err(ctx, err, "Failed to load the state tree"))
		}
		root := boxedRoot.(*service.StateTreeNode)

		gotNodes := sync.WaitGroup{}
		numChildren := 30
		if root.NumChildren < 30 {
			numChildren = int(root.NumChildren)
		}
		for i := 0; i < numChildren; i++ {
			gotNodes.Add(1)
			go func(i int) {
				defer gotNodes.Done()
				boxedChild, err := client.Get(ctx, tree.Root.Index(uint64(i)).Path(), resolveConfig)
				if err != nil {
					panic(err)
				}
				child := boxedChild.(*service.StateTreeNode)

				if child.Preview != nil {
					if child.Constants != nil {
						_, _ = getConstantSet(ctx, client, child.Constants)
					}
				}
			}(i)
		}
		gotNodes.Wait()
	}()

	// Get the mesh
	interactionWG.Add(1)
	go func() {
		ctx = status.Start(oldCtx, "Getting Mesh")
		defer status.Finish(ctx)
		defer interactionWG.Done()
		meshOptions := path.NewMeshOptions(false)
		_, _ = client.Get(ctx, commandToClick.Mesh(meshOptions).Path(), resolveConfig)
	}()

	// GetMemory
	interactionWG.Add(1)
	go func() {
		ctx = status.Start(oldCtx, "Getting Memory")
		defer status.Finish(ctx)
		defer interactionWG.Done()
		observationsPath := &path.Memory{
			Address:         0,
			Size:            uint64(0xFFFFFFFFFFFFFFFF),
			Pool:            0,
			After:           commandToClick,
			ExcludeData:     true,
			ExcludeObserved: true,
		}
		allMemory, err := client.Get(ctx, observationsPath.Path(), resolveConfig)
		if err != nil {
			panic(err)
		}
		memory := allMemory.(*service.Memory)
		var mem *service.MemoryRange
		if len(memory.Reads) > 0 {
			mem = memory.Reads[0]
		} else if len(memory.Writes) > 0 {
			mem = memory.Writes[0]
		} else {
			log.I(ctx, "No memory observations.")
			return
		}
		client.Get(ctx, commandToClick.MemoryAfter(0, mem.Base, 64*1024).Path(), resolveConfig)
	}()

	// Get Resource Data (For each texture, and shader)
	interactionWG.Add(1)
	go func() {
		ctx = status.Start(oldCtx, "Getting Resources")
		defer status.Finish(ctx)
		defer interactionWG.Done()
		gotResources := sync.WaitGroup{}
		for _, types := range resources.GetTypes() {
			for ii, v := range types.GetResources() {
				if (types.Type == api.ResourceType_TextureResource ||
					types.Type == api.ResourceType_ShaderResource ||
					types.Type == api.ResourceType_ProgramResource) &&
					ii < 30 {
					gotResources.Add(1)
					go func(id *path.ID) {
						defer gotResources.Done()
						resourcePath := commandToClick.ResourceAfter(id)
						_, _ = client.Get(ctx, resourcePath.Path(), resolveConfig)
					}(v.ID)
				}
			}
		}
		gotResources.Wait()
	}()

	interactionWG.Wait()
	verb.interactionDoneTime = time.Now()
	status.Finish(ctx)

	m, err := client.Get(ctx, c.Messages().Path(), nil)
	if err != nil {
		return err
	}
	messages := m.(*service.Messages)

	boxedVal, err := client.Get(ctx, (&path.Stats{
		Capture:  c,
		DrawCall: false,
	}).Path(), nil)
	if err != nil {
		return err
	}
	traceStartTimestamp := boxedVal.(*service.Stats).TraceStart

	frameTimes := []uint64{}

	stateBuildTime := int64(0)
	stateBuildStartTime := traceStartTimestamp
	stateBuildEndTime := traceStartTimestamp
	hasStateSerialization := false
	frameRe := regexp.MustCompile("Frame Number: [\\d]*")
	for _, m := range messages.List {
		if m.Message == "State serialization started" {
			hasStateSerialization = true
			stateBuildStartTime = m.Timestamp
		} else if m.Message == "State serialization finished" {
			stateBuildEndTime = m.Timestamp
			stateBuildTime = int64(stateBuildEndTime - stateBuildStartTime)
		} else if !hasStateSerialization && frameRe.MatchString(m.Message) {
			frameTimes = append(frameTimes, m.Timestamp)
		}
	}

	if len(events) < 1 {
		panic("No events")
	}
	lastFrameEvent := events[len(events)-1]
	frameCaptureTime := lastFrameEvent.Timestamp - stateBuildEndTime
	// Convert nanoseconds to milliseconds
	frameTime := float64(frameCaptureTime / uint64(len(events)))
	stateTime := float64(stateBuildTime)
	traceMaxMemory := int64(0)

	nonLoadingFrameTime := uint64(0)
	// We assume that the last 20% of frames come from a non-loading screen
	if hasStateSerialization {
		nFrames := len(frameTimes) / 5
		stableStart := frameTimes[len(frameTimes)-nFrames-1]
		stableEnd := frameTimes[len(frameTimes)-1]
		nonLoadingFrameTime = (stableEnd - stableStart) / uint64(nFrames)
	}

	ctx = oldCtx
	writeOutput := func() {
		preMecFramerate := float64(stateBuildStartTime - traceStartTimestamp)
		if verb.StartFrame > 0 {
			preMecFramerate = preMecFramerate / float64(verb.StartFrame)
		}
		if verb.OutputCSV {
			csvWriter := csv.NewWriter(os.Stdout)
			header := []string{
				"Trace Time (ms)", "Trace Size", "Trace Frames", "State Serialization (ms)", "Trace Frame Time (ms)", "Interactive (ms)",
				"Caching Done (ms)", "Interaction (ms)", "Max Memory", "Before MEC Frame Time (ms)", "Trailing Frame Time (ms)"}
			csvWriter.Write(header)
			record := []string{
				fmt.Sprint(float64(verb.traceDoneTime.Sub(verb.beforeStartTraceTime).Nanoseconds()) / float64(time.Millisecond)),
				fmt.Sprint(verb.traceSizeInBytes),
				fmt.Sprint(verb.traceFrames),
				fmt.Sprint(stateTime / float64(time.Millisecond)),
				fmt.Sprint(frameTime / float64(time.Millisecond)),
				fmt.Sprint(float64(verb.gapisInteractiveTime.Sub(verb.traceDoneTime).Nanoseconds()) / float64(time.Millisecond)),
				fmt.Sprint(float64(verb.gapisCachingDoneTime.Sub(verb.traceDoneTime).Nanoseconds()) / float64(time.Millisecond)),
				fmt.Sprint(float64(verb.interactionDoneTime.Sub(verb.interactionStartTime).Nanoseconds()) / float64(time.Millisecond)),
				fmt.Sprint(traceMaxMemory),
				fmt.Sprint(preMecFramerate / float64(time.Millisecond)),
				fmt.Sprint(float64(nonLoadingFrameTime) / float64(time.Millisecond)),
			}
			csvWriter.Write(record)
			csvWriter.Flush()
		} else {
			w := tabwriter.NewWriter(os.Stdout, 4, 4, 3, ' ', 0)
			fmt.Fprintln(w, "Trace Time\tTrace Size\tTrace Frames\tState Serialization\tTrace Frame Time\tInteractive")
			fmt.Fprintln(w, "----------\t----------\t------------\t-------------------\t----------------\t-----------")
			fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\t%v\n",
				verb.traceDoneTime.Sub(verb.beforeStartTraceTime),
				verb.traceSizeInBytes,
				verb.traceFrames,
				time.Duration(stateTime)*time.Nanosecond,
				time.Duration(frameTime)*time.Nanosecond,
				verb.gapisInteractiveTime.Sub(verb.traceDoneTime),
			)
			w.Flush()
			fmt.Fprintln(os.Stdout, "")
			w = tabwriter.NewWriter(os.Stdout, 4, 4, 3, ' ', 0)
			fmt.Fprintln(w, "Caching Done\tInteraction\tMax Memory\tBefore MEC Frame Time\tTrailing Frame Time")
			fmt.Fprintln(w, "------------\t-----------\t----------\t---------------------\t-----------------")
			fmt.Fprintf(w, "%+v\t%+v\t%+v\t%+v\t%+v\n",
				verb.gapisCachingDoneTime.Sub(verb.traceDoneTime),
				verb.interactionDoneTime.Sub(verb.interactionStartTime),
				traceMaxMemory,
				time.Duration(preMecFramerate)*time.Nanosecond,
				time.Duration(nonLoadingFrameTime)*time.Nanosecond,
			)
			w.Flush()
		}
	}

	writeTrace = func(path string, gapisTrace, gapitTrace *bytes.Buffer) error {
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = f.Write(gapisTrace.Bytes())
		if err != nil {
			return err
		}
		// Skip the leading [
		_, err = f.Write(gapitTrace.Bytes()[1:])
		if err != nil {
			return err
		}
		// This is the entire profile except for what happened on the trace device.
		// This is now stored in the trace file.
		// We have all of the timing information for the trace file,
		// the last thing we have to do is sync the existing traces with our trace.
		// We need to find the point in the GAPIS trace where the trace was connected.
		timeOffsetInMicroseconds := int64(0)
		var prof interface{}
		err = json.Unmarshal([]byte(string(gapisTrace.Bytes()[:len(gapisTrace.Bytes())-1])+"]"), &prof)
		if prof, ok := prof.([]interface{}); ok {
			for _, d := range prof {
				if d, ok := d.(map[string]interface{}); ok {
					if n, ok := d["name"]; ok {
						if s, ok := n.(string); ok {
							if s == "Trace Connected" {
								if n, ok := d["ts"]; ok {
									if n, ok := n.(float64); ok {
										timeOffsetInMicroseconds = int64(n)
									}
								}
							} else if s == "periodic_interval" {
								d := d["args"].(map[string]interface{})
								d = d["dumps"].(map[string]interface{})
								d = d["process_totals"].(map[string]interface{})
								m := d["heap_in_use"].(string)
								b, _ := strconv.ParseInt("0x"+m, 0, 64)
								if b > traceMaxMemory {
									traceMaxMemory = b
								}
							}
						}
					}
				}
			}
		} else {
			panic(fmt.Sprintf("Could not read profile data: %+v", err))
		}
		traceStartTimestampInMicroseconds := (traceStartTimestamp / 1000)
		timeOffsetInMicroseconds = int64(traceStartTimestampInMicroseconds) - timeOffsetInMicroseconds
		// Manually write out some profiling data for the trace
		tsk := profileTask{
			Name:      "Tracing",
			Tid:       0,
			Pid:       1,
			Ts:        int64(traceStartTimestampInMicroseconds) - timeOffsetInMicroseconds,
			EventType: "B",
		}
		b, _ := json.Marshal(tsk)
		f.Write([]byte("\n"))
		f.Write(b)
		f.Write([]byte(","))

		startTime := traceStartTimestampInMicroseconds
		for i, m := range frameTimes {
			if m >= stateBuildStartTime {
				break
			}
			tsk.Name = fmt.Sprintf("Untracked Frame %+v", i)
			tsk.Ts = int64(startTime) - timeOffsetInMicroseconds
			tsk.EventType = "B"
			b, _ = json.Marshal(tsk)
			f.Write([]byte("\n"))
			f.Write(b)
			f.Write([]byte(","))

			tsk.Name = ""
			tsk.Ts = int64(m/1000) - timeOffsetInMicroseconds
			tsk.EventType = "E"
			b, _ = json.Marshal(tsk)
			f.Write([]byte("\n"))
			f.Write(b)
			f.Write([]byte(","))

			startTime = (m / 1000)
		}

		if stateBuildStartTime != stateBuildEndTime {
			tsk.Name = "State Serialization"
			tsk.Ts = int64(stateBuildStartTime/1000) - timeOffsetInMicroseconds
			tsk.EventType = "B"
			b, _ = json.Marshal(tsk)
			f.Write([]byte("\n"))
			f.Write(b)
			f.Write([]byte(","))

			tsk.Name = ""
			tsk.Ts = int64(stateBuildEndTime/1000) - timeOffsetInMicroseconds
			tsk.EventType = "E"
			b, _ = json.Marshal(tsk)
			f.Write([]byte("\n"))
			f.Write(b)
			f.Write([]byte(","))
		}

		startTime = (stateBuildEndTime / 1000)
		for i, e := range events {
			tsk.Name = fmt.Sprintf("Frame %+v", i)
			tsk.Ts = int64(startTime) - timeOffsetInMicroseconds
			tsk.EventType = "B"
			b, _ = json.Marshal(tsk)
			f.Write([]byte("\n"))
			f.Write(b)
			f.Write([]byte(","))

			tsk.Name = ""
			tsk.Ts = int64(e.Timestamp/1000) - timeOffsetInMicroseconds
			tsk.EventType = "E"
			b, _ = json.Marshal(tsk)
			f.Write([]byte("\n"))
			f.Write(b)
			f.Write([]byte(","))

			startTime = (e.Timestamp / 1000)
		}

		tsk.Name = ""
		tsk.Ts = int64(startTime) - timeOffsetInMicroseconds
		tsk.EventType = "E"
		b, _ = json.Marshal(tsk)
		f.Write([]byte("\n"))
		f.Write(b)
		f.Write([]byte("]"))

		writeOutput()
		return nil
	}

	if verb.DumpTrace == "" {
		writeOutput()
	}
	return nil
}

// This intentionally duplicates a lot of the gapit trace logic
// so that we can independently chnage how what we do to benchmark
// everything.
func (verb *benchmarkVerb) doTrace(ctx context.Context, client client.Client, traceURI string) error {
	ctx = status.Start(ctx, "Record Trace for %+v frames", verb.NumFrames)
	defer status.Finish(ctx)

	// Find the actual trace URI from all of the devices
	_, err := client.GetServerInfo(ctx)
	if err != nil {
		return err
	}

	devices, err := client.GetDevices(ctx)
	if err != nil {
		return err
	}

	devices, err = filterDevices(ctx, &verb.DeviceFlags, client)
	if err != nil {
		return err
	}
	if len(devices) == 0 {
		return fmt.Errorf("Could not find matching device")
	}

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

	out := BenchmarkName
	uri := found[0].uri
	traceDevice := found[0].device

	options := &service.TraceOptions{
		Device:                    traceDevice,
		Apis:                      []string{"Vulkan"},
		AdditionalCommandLineArgs: verb.AdditionalArgs,
		Cwd:                       verb.WorkingDir,
		Environment:               verb.Env,
		Duration:                  0,
		ObserveFrameFrequency:     0,
		ObserveDrawFrequency:      0,
		StartFrame:                uint32(verb.StartFrame),
		FramesToCapture:           uint32(verb.NumFrames),
		RecordErrorState:          false,
		DeferStart:                false,
		NoBuffer:                  false,
		HideUnknownExtensions:     true,
		RecordTraceTimes:          true,
		ClearCache:                false,
		ServerLocalSavePath:       out,
	}
	options.App = &service.TraceOptions_Uri{
		uri,
	}
	verb.beforeStartTraceTime = time.Now()
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
	verb.traceInitializedTime = time.Now()

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

		if status.Status == service.TraceStatus_Done {
			return true, nil
		}
		return false, nil
	})
}
