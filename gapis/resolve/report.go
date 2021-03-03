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

package resolve

import (
	"context"

	"github.com/google/gapid/core/app/analytics"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/replay"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
	"github.com/google/gapid/gapis/stringtable"
)

// Report resolves the report for the given path.
func Report(ctx context.Context, p *path.Report, r *path.ResolveConfig) (*service.Report, error) {
	obj, err := database.Build(ctx, &ReportResolvable{Path: p, Config: r})
	if err != nil {
		return nil, err
	}
	return obj.(*service.Report), nil
}

func (r *ReportResolvable) newReportItem(s log.Severity, c uint64, m *stringtable.Msg) *service.ReportItemRaw {
	var cmd *path.Command
	if c != uint64(api.CmdNoID) {
		cmd = r.Path.Capture.Command(c)
	}
	return service.WrapReportItem(&service.ReportItem{
		Severity: service.Severity(s),
		Command:  cmd, // TODO: Subcommands
	}, m)
}

// Resolve implements the database.Resolver interface.
func (r *ReportResolvable) Resolve(ctx context.Context) (interface{}, error) {
	ctx = SetupContext(ctx, r.Path.Capture, r.Config)

	c, err := capture.ResolveGraphics(ctx)
	if err != nil {
		return nil, err
	}

	defer analytics.SendTiming("resolve", "report")(analytics.Size(len(c.Commands)))

	builder := service.NewReportBuilder()

	var currentCmd uint64
	items := []*service.ReportItemRaw{}
	state := c.NewState(ctx)
	state.NewMessage = func(s log.Severity, m *stringtable.Msg) uint32 {
		items = append(items, r.newReportItem(s, currentCmd, m))
		return uint32(len(items) - 1)
	}
	state.AddTag = func(i uint32, t *stringtable.Msg) {
		items[i].Tags = append(items[i].Tags, t)
	}

	issues := map[api.CmdID][]replay.Issue{}

	if r.Path.Device != nil {
		// Request is for a replay report too.
		intent := replay.Intent{
			Capture: r.Path.Capture,
			Device:  r.Path.Device,
		}

		mgr := replay.GetManager(ctx)

		// Capture can use multiple APIs.
		// Iterate the APIs in use looking for those that support the
		// QueryIssues interface. Call QueryIssues for each of these APIs.
		hints := &path.UsageHints{Background: true}
		for _, a := range c.APIs {
			if qi, ok := a.(replay.QueryIssues); ok {
				apiIssues, err := qi.QueryIssues(ctx, intent, mgr, r.Path.DisplayToSurface, hints)
				if err != nil {
					issue := replay.Issue{
						Command:  api.CmdNoID,
						Severity: service.Severity_ErrorLevel,
						Error:    err,
					}
					issues[api.CmdNoID] = append(issues[api.CmdNoID], issue)
					continue
				}
				for _, issue := range apiIssues {
					issues[issue.Command] = append(issues[issue.Command], issue)
				}
			}
		}
	}

	// Start with issues that are not specific to a command, like
	// replay connection errors.
	for _, issue := range issues[api.CmdNoID] {
		item := r.newReportItem(log.Severity(issue.Severity), uint64(issue.Command), messages.ErrReplayDriver(issue.Error.Error()))
		builder.Add(ctx, item)
	}

	// Gather report items from the state mutator, and collect together all the
	// APIs in use.
	api.ForeachCmd(ctx, c.Commands, true, func(ctx context.Context, id api.CmdID, cmd api.Cmd) error {
		items, currentCmd = items[:0], uint64(id)

		if as := cmd.Extras().Aborted(); as != nil && as.IsAssert {
			items = append(items, r.newReportItem(log.Fatal, uint64(id),
				messages.ErrTraceAssert(as.Reason)))
		}

		if err := cmd.Mutate(ctx, id, state, nil /* builder */, nil /* watcher */); err != nil {
			if !api.IsErrCmdAborted(err) {
				items = append(items, r.newReportItem(log.Error, uint64(id),
					messages.ErrInternalError(err.Error())))
			}
		}

		for _, item := range items {
			item.Tags = append(item.Tags, getCommandNameTag(cmd))
			builder.Add(ctx, item)
		}
		for _, issue := range issues[id] {
			item := r.newReportItem(log.Severity(issue.Severity), uint64(issue.Command),
				messages.ErrReplayDriver(issue.Error.Error()))
			if int(issue.Command) < len(c.Commands) {
				item.Tags = append(item.Tags, getCommandNameTag(c.Commands[issue.Command]))
			}
			builder.Add(ctx, item)
		}
		return nil
	})

	return builder.Build(), nil
}

func getCommandNameTag(cmd api.Cmd) *stringtable.Msg {
	return messages.TagCommandName(cmd.CmdName())
}
