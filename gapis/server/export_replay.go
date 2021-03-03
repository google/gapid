// Copyright (C) 2018 Google Inc.
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

// Package server implements the rpc gpu debugger service, queriable by the
// clients, along with some helpers exposed via an http listener.
package server

import (
	"context"
	"io/ioutil"
	"os"
	gopath "path"

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/archive"
	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/replay"
	"github.com/google/gapid/gapis/resolve"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

type mockDevice struct {
	bind.Simple
}

func (*mockDevice) CanTrace() bool {
	return false
}

func exportReplay(ctx context.Context, c *path.Capture, d *path.Device, out string, opts *service.ExportReplayOptions) error {
	cap, err := capture.ResolveGraphicsFromPath(ctx, c)

	if d == nil {
		instance := *cap.Header.Device
		instance.Name = "mock-" + instance.Name
		instance.GenID()
		dev := &mockDevice{}
		dev.To = &instance
		bind.GetRegistry(ctx).AddDevice(ctx, dev)
		d = path.NewDevice(dev.Instance().ID.ID())
	}

	intent := replay.Intent{d, c}

	var queries []func(mgr replay.Manager) error
	switch {
	case opts.Report != nil && len(opts.FramebufferAttachments) > 0 && opts.GetTimestampsRequest != nil:
		return log.Errf(ctx, nil, "at most one of the request should be specified")
	case opts.FramebufferAttachments != nil:
		r := &path.ResolveConfig{ReplayDevice: d}
		changes, err := resolve.FramebufferChanges(ctx, c, r)
		if err != nil {
			return err
		}

		for _, req := range opts.FramebufferAttachments {
			req := req
			fbInfo, err := changes.Get(ctx, req.After, req.Index)
			if err != nil {
				return err
			}

			for _, a := range cap.APIs {
				a, ok := a.(replay.QueryFramebufferAttachment)
				if !ok {
					continue
				}
				queries = append(queries, func(mgr replay.Manager) error {
					_, err := a.QueryFramebufferAttachment(
						ctx,                         // context.Context
						intent,                      // Intent
						mgr,                         // Manager
						req.After.Indices,           // after []uint64
						fbInfo.Width,                // width uint32
						fbInfo.Height,               // height uint32
						fbInfo.Type,                 // Index uint32
						req.Index,                   // uint32
						req.RenderSettings.DrawMode, // service.DrawMode
						true,                        // disableReplayOptimization bool
						false,                       // displayToSurface bool
						nil,                         // hints *service.UsageHints
					)
					return err
				})
			}
		}
	case opts.GetTimestampsRequest != nil:
		for _, a := range cap.APIs {
			a, ok := a.(replay.QueryTimestamps)
			if !ok {
				continue
			}
			queries = append(queries, func(mgr replay.Manager) error {
				return a.QueryTimestamps(ctx, intent, mgr, nil, nil)
			})
		}
	case opts.Report != nil:
		// TODO(hysw): Add a simple replay request that output commands as
		// captured. For now, if there are no frame query, force an issue query.
		// Since otherwise the trace will not be generated.
		fallthrough
	default:
		for _, a := range cap.APIs {
			a, ok := a.(replay.QueryIssues)
			if !ok {
				continue
			}
			queries = append(queries, func(mgr replay.Manager) error {
				_, err := a.QueryIssues(ctx, intent, mgr, opts.DisplayToSurface, nil)
				return err
			})
		}
	}

	exporter := replay.NewExporter()

	errs := make(chan error, len(queries))
	for _, q := range queries {
		q := q
		go func() {
			errs <- q(exporter)
		}()
	}

	payload, err := exporter.Export(ctx, len(queries))
	if err != nil {
		return err
	}

	for range queries {
		err := <-errs
		if err != nil {
			// TODO(hysw): Ignore error for now, since empty postback will upset some
			// of the postback handlers.
		}
	}

	err = os.MkdirAll(out, os.ModePerm)
	if err != nil {
		return log.Errf(ctx, err, "Failed to create output directory: %v", out)
	}

	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		return log.Errf(ctx, err, "Failed to serialize replay payload.")
	}
	err = ioutil.WriteFile(gopath.Join(out, "payload.bin"), payloadBytes, 0644)

	ar := archive.New(gopath.Join(out, "resources"))
	defer ar.Dispose()

	db := database.Get(ctx)
	for _, ri := range payload.Resources {
		rID, err := id.Parse(ri.Id)
		if err != nil {
			return log.Errf(ctx, err, "Failed to parse resource id: %v", ri.Id)
		}
		obj, err := db.Resolve(ctx, rID)
		if err != nil {
			return log.Errf(ctx, err, "Failed to parse resource id: %v", ri.Id)
		}
		ar.Write(ri.Id, obj.([]byte))
	}

	return nil
}
