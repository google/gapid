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

package scheduler

import (
	"context"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/test/robot/build"
	"github.com/google/gapid/test/robot/job"
	"github.com/google/gapid/test/robot/monitor"
)

type schedule struct {
	managers *monitor.Managers
	data     *monitor.Data
	pkg      *monitor.Package
	worker   *monitor.Worker
}

// Tick can be called to schedule new actions based on the current data set.
// It looks at the data to find holes that need actions to fill, and then picks which
// ones it thinks should be next based on the available workers.
// It is intended to be called in the update function of a monitor.
// See monitor.Run for more details.
// Blocking will prevent updates of the data store, so the function will try to schedule
// tasks to idle workers only returning quickly on the assumption it will be ticked again
// as soon as the data changes.
func Tick(ctx context.Context, managers *monitor.Managers, data *monitor.Data) []error {
	s := schedule{
		managers: managers,
		data:     data,
	}
	var errs []error
	// TODO: a real scheduler, not just try to do everything in any order
	for _, pkg := range data.Packages.All() {
		s.pkg = pkg
		for _, w := range data.Workers.All() {
			s.worker = w
			tools := s.getHostTools(ctx)
			if tools == nil {
				continue
			}
			for _, subj := range data.Subjects.All() {
				androidTools := s.getAndroidTools(ctx, subj)
				if androidTools == nil {
					continue
				}
				if err := s.doTrace(ctx, subj, tools, androidTools); err != nil {
					errs = append(errs, err)
				}
			}
			for _, t := range data.Traces.MatchPackage(s.pkg) {
				if t.Status != job.Succeeded {
					continue
				}
				if t.Output == nil {
					continue
				}
				if err := s.doReport(ctx, t, tools); err != nil {
					errs = append(errs, err)
				}
				if err := s.doReplay(ctx, t, tools); err != nil {
					errs = append(errs, err)
				}
			}
		}
	}
	return errs
}

func (s schedule) getHostTools(ctx context.Context) *build.ToolSet {
	ctx = log.V{"Host": s.worker.Host}.Bind(ctx)
	tools := s.pkg.FindTools(ctx, s.data.FindDevice(s.worker.Host))
	if tools == nil {
		return nil
	}
	if tools.Host.Gapit == "" {
		return nil
	}
	if tools.Host.Gapis == "" {
		return nil
	}
	if tools.Host.Gapir == "" {
		return nil
	}
	if tools.Host.VirtualSwapChainLib == "" {
		return nil
	}
	if tools.Host.VirtualSwapChainJson == "" {
		return nil
	}
	return tools
}

func (s schedule) getAndroidTools(ctx context.Context, subj *monitor.Subject) *build.AndroidToolSet {
	ctx = log.V{"target": s.worker.Target}.Bind(ctx)
	tools := s.pkg.FindToolsForAPK(ctx, s.data.FindDevice(s.worker.Host), s.data.FindDevice(s.worker.Target), subj.GetAPK())
	if tools == nil {
		return nil
	}
	if tools.GapidApk == "" {
		return nil
	}
	return tools
}
