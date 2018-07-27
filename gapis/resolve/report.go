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
	"github.com/google/gapid/core/app/status"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapil/executor"
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

func (r *ReportResolvable) newReportItem(s log.Severity, id api.CmdID, m *stringtable.Msg) *service.ReportItemRaw {
	var cmd *path.Command
	if id != api.CmdNoID {
		cmd = r.Path.Capture.Command(uint64(id))
	}
	return service.WrapReportItem(&service.ReportItem{
		Severity: service.Severity(s),
		Command:  cmd, // TODO: Subcommands
	}, m)
}

// Resolve implements the database.Resolver interface.
func (r *ReportResolvable) Resolve(ctx context.Context) (interface{}, error) {
	ctx = status.Start(ctx, "Report")
	defer status.Finish(ctx)

	ctx = SetupContext(ctx, r.Path.Capture, r.Config)

	c, err := capture.Resolve(ctx)
	if err != nil {
		return nil, err
	}

	env := c.Env().InitState().Execute().Build(ctx)
	defer env.Dispose()
	ctx = executor.PutEnv(ctx, env)

	defer analytics.SendTiming("resolve", "report")(analytics.Size(len(c.Commands)))

	sd, err := SyncData(ctx, r.Path.Capture)
	if err != nil {
		return nil, err
	}

	filter, err := buildFilter(ctx, r.Path.Capture, r.Path.Filter, sd, r.Config)
	if err != nil {
		return nil, err
	}

	builder := service.NewReportBuilder()

	cmdItems := map[api.CmdID][]*service.ReportItemRaw{}
	cmdIssues := map[api.CmdID][]replay.Issue{}

	state := env.State
	state.NewMessage = func(s log.Severity, m *stringtable.Msg) uint32 {
		cmdID := env.CmdID()
		items := cmdItems[cmdID]
		items = append(items, r.newReportItem(s, cmdID, m))
		cmdItems[cmdID] = items
		return uint32(len(items) - 1)
	}
	state.AddTag = func(i uint32, t *stringtable.Msg) {
		cmdID := env.CmdID()
		items := cmdItems[cmdID]
		items[i].Tags = append(items[i].Tags, t)
	}

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
		hints := &service.UsageHints{Background: true}
		for _, a := range c.APIs {
			if qi, ok := a.(replay.QueryIssues); ok {
				apiIssues, err := qi.QueryIssues(ctx, intent, mgr, r.Path.DisplayToSurface, hints)
				if err != nil {
					issue := replay.Issue{
						Command:  api.CmdNoID,
						Severity: service.Severity_ErrorLevel,
						Error:    err,
					}
					cmdIssues[api.CmdNoID] = append(cmdIssues[api.CmdNoID], issue)
					continue
				}
				for _, issue := range apiIssues {
					cmdIssues[issue.Command] = append(cmdIssues[issue.Command], issue)
				}
			}
		}
	}

	ctx = status.Start(ctx, "Execute")
	defer status.Finish(ctx)

	// Gather report items from the state mutator, and collect together all the
	// APIs in use.
	errs := env.ExecuteN(ctx, 0, c.Commands)

	api.ForeachCmd(ctx, c.Commands, func(ctx context.Context, id api.CmdID, cmd api.Cmd) error {
		if !filter(id, cmd, state) {
			return nil
		}

		items, issues := cmdItems[id], cmdIssues[id]

		if as := cmd.Extras().Aborted(); as != nil && as.IsAssert {
			items = append(items, r.newReportItem(log.Fatal, id, messages.ErrTraceAssert(as.Reason)))
		}

		if err := errs[id]; err != nil && !api.IsErrCmdAborted(err) {
			items = append(items, r.newReportItem(log.Error, id, messages.ErrInternalError(err.Error())))
		}

		for _, item := range items {
			item.Tags = append(item.Tags, getCommandNameTag(cmd))
			builder.Add(ctx, item)
		}

		for _, issue := range issues {
			item := r.newReportItem(log.Severity(issue.Severity), issue.Command,
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
