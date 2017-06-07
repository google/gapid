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
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/google/gapid/core/app/benchmark"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/image"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/client"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
	"github.com/google/gapid/gapis/stringtable"
)

const (
	gapisLink = "gapis"
	traceLink = "trace"
)

type session struct {
	ctx       context.Context
	bench     *Benchmark
	tracefile string
	runIdx    int

	client  client.Client
	capture *path.Capture
	device  *path.Device

	atoms []atom.Atom

	features        []string
	stringTables    []*stringtable.StringTable
	resourceBundles interface{}
	report          interface{}
	contexts        interface{}
	commandTree     interface{}
}

func fullRun(ctx context.Context, bench *Benchmark) (err error) {
	// Update Gapis link.
	g, err := newGapisLink(bench, bench.Input.Gapis.Get().Bundle)
	if err != nil {
		return err
	}
	bench.Input.Gapis = g

	bench.Links[traceLink] = bench.Input.Trace
	bench.Links[gapisLink] = bench.Input.Gapis

	bench.Meta.HostName, _ = os.Hostname()

	traceFile, isTemp, err := bench.Input.Trace.Get().DiskFile()
	if isTemp {
		defer os.Remove(traceFile)
	}

	bench.Meta.DateStarted = time.Now()
	for runIdx := 0; runIdx < bench.Input.Runs; runIdx++ {
		log.I(ctx, "Benchmark run: %d/%d", (runIdx + 1), bench.Input.Runs)
		if err := singleRun(ctx, bench, runIdx, traceFile); err != nil {
			return log.Err(ctx, err, "singleRun")
		}
	}
	bench.TotalTimeTaken.Set(time.Since(bench.Meta.DateStarted))

	bench.Fit = stringOrEmpty(bench.Samples.IndexedMultisamples().Analyse(
		func(m IndexedMultisample) (int, time.Duration) { return int(m.Index), m.Values.Median() }))
	bench.FitByFrame = stringOrEmpty(bench.Samples.IndexedMultisamples().Analyse(
		func(m IndexedMultisample) (int, time.Duration) {
			return bench.AtomIndicesToFrames[m.Index], m.Values.Median()
		}))

	return err
}

func stringOrEmpty(s fmt.Stringer) string {
	if s == nil {
		return ""
	} else {
		return s.String()
	}
}

func singleRun(ctx context.Context, bench *Benchmark, runIdx int, tracefile string) error {
	s := &session{ctx: ctx, bench: bench, tracefile: tracefile, runIdx: runIdx}
	if err := s.gapisConnect(); err != nil {
		return err
	}
	defer s.client.Close()

	start := time.Now()
	var actions []func() error
	switch bench.Input.BenchmarkType {
	case "state", "frames":
		actions = []func() error{
			s.getDevices,
			s.loadCapture,
			s.getAtoms,
			s.maybeBeginProfile,
			s.grabSamples,
			s.maybeSaveProfileData,
		}
	case "startup":
		actions = []func() error{
			s.maybeBeginProfile,
			s.getStringTables,
			s.getDevices,
			s.loadCapture,
			s.getAtoms,
			func() error {
				return s.get("Resources", s.capture.Resources(), &(s.resourceBundles))
			},
			func() error {
				return s.get("Report", s.capture.Report(nil, nil), &(s.report))
			},
			func() error {
				return s.get("Contexts", s.capture.Contexts(), &(s.contexts))
			},
			func() error {
				return s.get("CommandTree", s.capture.CommandTree(nil), &(s.commandTree))
			},
			s.maybeSaveProfileData,
		}
	default:
		return fmt.Errorf("Invalid benchmark type: %s", bench.Input.BenchmarkType)
	}

	for _, action := range actions {
		if err := action(); err != nil {
			return err
		}
	}
	s.bench.Metric("Actions", time.Since(start))

	// TODO(valbulescu): Allow averaging counter values.
	if runIdx == 0 {
		counterData, err := s.client.GetPerformanceCounters(ctx)
		if err != nil {
			return err
		}
		bench.Counters = benchmark.NewCounters()
		if err = json.Unmarshal(counterData, bench.Counters); err != nil {
			return err
		}
	}

	return nil
}

func (s *session) maybeBeginProfile() error {
	if s.bench.Input.EnableCPUProfile {
		return s.client.BeginCPUProfile(s.ctx)
	}
	return nil
}

func (s *session) maybeSaveProfileData() error {
	if s.bench.Input.EnableCPUProfile {
		data, err := s.client.EndCPUProfile(s.ctx)
		if err != nil {
			return err
		}
		if err := s.saveProfileDataEntry(data, "cpu"); err != nil {
			return err
		}
	}
	if s.bench.Input.EnableHeapProfile {
		data, err := s.client.GetProfile(s.ctx, "heap", 0)
		if err != nil {
			return err
		}
		if err := s.saveProfileDataEntry(data, "heap"); err != nil {
			return err
		}
	}
	return nil
}

func (s *session) saveProfileDataEntry(data []byte, key string) error {
	link, err := s.bench.Root().NewLink(&DataEntry{
		DataSource: ByteSliceDataSource(data),
		Name:       fmt.Sprintf("%s/%s/%d.pprof", s.bench.Input.Name, key, s.runIdx),
		Bundle:     true,
	})
	if err != nil {
		return err
	}
	s.bench.Links[fmt.Sprintf("%s/%d", key, s.runIdx)] = link
	return nil
}

type sampleGrabber func(context.Context, *session, *path.Command) error

func (s *session) gapisConnect() error {
	log.I(s.ctx, "Connecting to GAPIS...")
	start := time.Now()
	ctx := log.PutFilter(s.ctx, log.SeverityFilter(log.Info))
	client, err := client.Connect(ctx, client.Config{})
	if err != nil {
		return fmt.Errorf("Failed to connect to the GAPIS server: %v", err)
	}
	s.bench.Metric("Connect", time.Since(start))
	s.client = client
	return nil
}

func (s *session) getDevices() error {
	log.I(s.ctx, "Getting devices...")
	start := time.Now()
	devices, err := s.client.GetDevices(s.ctx)
	if err != nil {
		return log.Err(s.ctx, err, "GetDevices")
	}
	s.bench.Metric("GetDevices", time.Since(start))
	if len(devices) != 0 {
		s.device = devices[0]
	}
	return nil
}

func (s *session) getStringTables() error {
	log.I(s.ctx, "Getting string tables...")

	start := time.Now()
	stringTableInfos, err := s.client.GetAvailableStringTables(s.ctx)
	if err != nil {
		return log.Err(s.ctx, err, "GetAvailableStringTables")
	}
	s.bench.Metric("GetAvailableStringTables", time.Since(start))

	s.stringTables = make([]*stringtable.StringTable, len(stringTableInfos))
	for i := range s.stringTables {
		stringTable, err := s.client.GetStringTable(s.ctx, stringTableInfos[i])
		if err != nil {
			return err
		}
		s.stringTables[i] = stringTable
	}
	s.bench.Metric("GetStringTables", time.Since(start))
	return nil
}

func (s *session) loadCapture() error {
	log.I(s.ctx, "Loading capture file %s...", s.tracefile)
	start := time.Now()
	capture, err := s.client.LoadCapture(s.ctx, s.tracefile)
	if err != nil {
		return log.Err(s.ctx, err, "Failed to load the capture file")
	}
	s.bench.Metric("LoadCapture", time.Since(start))
	s.capture = capture
	return nil
}

func (s *session) get(what string, p path.Node, dest *interface{}) error {
	metric := fmt.Sprintf("Get%s", what)
	log.I(s.ctx, "Getting: %s...", what)
	start := time.Now()
	var err error
	*dest, err = s.client.Get(s.ctx, p.Path())
	if err != nil {
		return log.Err(s.ctx, err, metric)
	}
	s.bench.Metric(metric, time.Since(start))
	return nil
}

func (s *session) getAtoms() error {
	var result interface{}
	if err := s.get("Commands", s.capture.Commands(), &result); err != nil {
		return err
	}
	s.atoms = result.(*atom.List).Atoms
	return nil
}

func getAtomIndicesAndSampleGrabber(bench *Benchmark, session *session) (err error, arr indices, grab sampleGrabber) {
	arr = newConsecutiveIndices(len(session.atoms))

	switch bench.Input.SampleOrder {
	case "ordered":
	case "random":
		arr = arr.randomize()
	case "reverse":
		arr = arr.reverse()
	default:
		return fmt.Errorf("Invalid sample order: %s", bench.Input.SampleOrder), indices{}, nil
	}

	switch bench.Input.BenchmarkType {
	case "frames":
		arr, grab = arr.filter(func(idx int) bool {
			return session.atoms[idx].AtomFlags().IsEndOfFrame()
		}), getFrame
	case "state":
		grab = getStateAfter
	default:
		return fmt.Errorf("Invalid benchmark type: %s", bench.Input.BenchmarkType), indices{}, nil
	}

	arr = arr.take(bench.Input.MaxSamples)
	return nil, arr, grab
}

func (s *session) grabSamples() error {
	ctx := s.ctx
	if s.bench.Input.Timeout > 0 {
		ctx, _ = task.WithTimeout(ctx, s.bench.Input.Timeout)
	}

	rand.Seed(s.bench.Input.Seed)
	err, as, grabSample := getAtomIndicesAndSampleGrabber(s.bench, s)
	if err != nil {
		return err
	}

	log.I(ctx, "Getting samples...")
	start := time.Now()

	// Map atom indices to frames.
	interesting := map[int]bool{}
	for _, index := range as {
		interesting[index] = true
	}
	currentFrame := 0
	for index, atom := range s.atoms {
		if atom.AtomFlags().IsEndOfFrame() {
			currentFrame++
		}
		if interesting[index] {
			s.bench.AtomIndicesToFrames[int64(index)] = currentFrame
		}
	}

	for i, index := range as {
		log.I(ctx, "Index %d [%d/%d]", index, (i + 1), len(as))
		if task.Stopped(ctx) {
			break
		}

		start := time.Now()
		if err := grabSample(ctx, s, s.capture.Command(uint64(index))); err != nil {
			return err
		}
		s.bench.Samples.Add(int64(index), time.Since(start))
	}
	s.bench.Metric("GrabSamples", time.Since(start))
	log.I(ctx, "Benchmark complete.")
	return nil
}

func getStateAfter(ctx context.Context, session *session, cmd *path.Command) error {
	_, err := session.client.Get(ctx, cmd.StateAfter().Path())
	return err
}

func getFrame(ctx context.Context, session *session, cmd *path.Command) error {
	settings := &service.RenderSettings{
		MaxWidth:  uint32(session.bench.Input.MaxFrameWidth),
		MaxHeight: uint32(session.bench.Input.MaxFrameHeight),
	}
	imgInfoPath, err := session.client.GetFramebufferAttachment(
		ctx,
		session.device,
		cmd,
		gfxapi.FramebufferAttachment_Color0,
		settings,
		nil,
	)
	if err != nil {
		return err
	}

	imgInfo, err := session.client.Get(ctx, imgInfoPath.Path())
	if err != nil {
		return err
	}

	ii := imgInfo.(*image.Info)
	_, err = session.client.Get(ctx, path.NewBlob(ii.Bytes.ID()).Path())

	if err != nil {
		return err
	}
	if ii.Width == 0 || ii.Height == 0 {
		return fmt.Errorf("Framebuffer at atom %d was %x x %x", cmd.Indices, ii.Width, ii.Height)
	}

	return nil
}
