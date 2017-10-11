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

package monitor

import (
	"context"
	"sync"

	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/test/robot/build"
	"github.com/google/gapid/test/robot/job"
	"github.com/google/gapid/test/robot/master"
	"github.com/google/gapid/test/robot/replay"
	"github.com/google/gapid/test/robot/report"
	"github.com/google/gapid/test/robot/search"
	"github.com/google/gapid/test/robot/stash"
	"github.com/google/gapid/test/robot/subject"
	"github.com/google/gapid/test/robot/trace"
)

// Managers describes the set of managers to monitor for data changes.
type Managers struct {
	Master  master.Master
	Stash   *stash.Client
	Job     job.Manager
	Build   build.Store
	Subject subject.Subjects
	Trace   trace.Manager
	Report  report.Manager
	Replay  replay.Manager
}

// Data is the live store of data from the monitored servers.
// Entries with no live manager will not be updated.
type Data struct {
	mu   sync.Mutex
	cond *sync.Cond

	Gen *Generation

	Devices  Devices
	Workers  Workers
	Subjects Subjects
	Tracks   Tracks
	Packages Packages
	Traces   Traces
	Reports  Reports
	Replays  Replays
}

type DataOwner struct {
	data *Data
}

func NewDataOwner() DataOwner {
	data := &Data{
		Gen: NewGeneration(),
	}
	data.cond = sync.NewCond(&data.mu)
	return DataOwner{data}
}

func (o DataOwner) Read(rf func(d *Data)) {
	o.data.mu.Lock()
	defer o.data.mu.Unlock()
	rf(o.data)
}

func (o DataOwner) Write(wf func(d *Data)) {
	o.data.mu.Lock()
	defer func() {
		o.data.cond.Broadcast()
		o.data.mu.Unlock()
	}()
	wf(o.data)
}

func (data *Data) Wait() {
	data.cond.Wait()
}

// Run is used to run a new monitor.
// It will monitor the data from all the managers that are in the supplied managers, filling in the data structure
// with all the results it receives.
// Each time it receives a batch of updates it will invoke the update function passing in the manager set being
// monitored and the updated set of data.
func Run(ctx context.Context, managers Managers, owner DataOwner, update func(ctx context.Context, managers *Managers, data *Data) []error) error {
	// start all the data monitors we have managers for
	if err := monitor(ctx, &managers, owner); err != nil {
		return err
	}

	owner.Read(func(data *Data) {
		for {
			// Update generation
			data.Gen.Update()
			// Run the update
			if update != nil {
				if errs := update(ctx, &managers, data); len(errs) != 0 {
					log.E(ctx, "Error(s) during update: %v", errs)
				}
			}
			// Wait for new data
			data.Wait()
		}
	})
	return nil
}

func monitor(ctx context.Context, managers *Managers, owner DataOwner) error {
	initial := &search.Query{}
	// TODO: care about monitors erroring
	monitor := &search.Query{Monitor: true}
	if managers.Job != nil {
		if err := managers.Job.SearchDevices(ctx, initial, owner.updateDevice); err != nil {
			return err
		}
		crash.Go(func() { managers.Job.SearchDevices(ctx, monitor, owner.updateDevice) })
		if err := managers.Job.SearchWorkers(ctx, initial, owner.updateWorker); err != nil {
			return err
		}
		crash.Go(func() { managers.Job.SearchWorkers(ctx, monitor, owner.updateWorker) })
	}
	if managers.Build != nil {
		if err := managers.Build.SearchTracks(ctx, initial, owner.updateTrack); err != nil {
			return err
		}
		crash.Go(func() { managers.Build.SearchTracks(ctx, monitor, owner.updateTrack) })
		if err := managers.Build.SearchPackages(ctx, initial, owner.updatePackage); err != nil {
			return err
		}
		crash.Go(func() { managers.Build.SearchPackages(ctx, monitor, owner.updatePackage) })
	}
	if managers.Subject != nil {
		if err := managers.Subject.Search(ctx, initial, owner.updateSubject); err != nil {
			return err
		}
		crash.Go(func() { managers.Subject.Search(ctx, monitor, owner.updateSubject) })
	}
	if managers.Trace != nil {
		if err := managers.Trace.Search(ctx, initial, owner.updateTrace); err != nil {
			return err
		}
		crash.Go(func() { managers.Trace.Search(ctx, monitor, owner.updateTrace) })
	}
	if managers.Report != nil {
		if err := managers.Report.Search(ctx, initial, owner.updateReport); err != nil {
			return err
		}
		crash.Go(func() { managers.Report.Search(ctx, monitor, owner.updateReport) })
	}
	if managers.Replay != nil {
		if err := managers.Replay.Search(ctx, initial, owner.updateReplay); err != nil {
			return err
		}
		crash.Go(func() { managers.Replay.Search(ctx, monitor, owner.updateReplay) })
	}

	return nil
}
