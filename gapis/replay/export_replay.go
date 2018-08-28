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

package replay

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
	gapir "github.com/google/gapid/gapir/client"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/replay/builder"
	"github.com/google/gapid/gapis/resolve/initialcmds"
	"github.com/google/gapid/gapis/service/path"
)

// ExportReplay write replay commands and assets to path.
func ExportReplay(ctx context.Context, pCapture *path.Capture, pDevice *path.Device, outDir string) error {
	if pDevice == nil {
		return log.Errf(ctx, nil, "Unable to produce replay on unknown device.")
	}

	ctx = capture.Put(ctx, pCapture)
	c, err := capture.Resolve(ctx)
	if err != nil {
		return err
	}

	// Capture can use multiple APIs.
	// Iterate the APIs in use looking for those that support replay generation.
	var generator Generator
	for _, a := range c.APIs {
		if a, ok := a.(Generator); ok {
			generator = a
			break
		}
	}

	if generator == nil {
		return log.Errf(ctx, nil, "Unable to find replay API.")
	}

	d := bind.GetRegistry(ctx).Device(pDevice.ID.ID())
	if d == nil {
		return log.Errf(ctx, nil, "Unknown device %v", pDevice.ID.ID())
	}

	ctx = log.V{
		"capture": pCapture.ID.ID(),
		"device":  d.Instance().GetName(),
	}.Bind(ctx)

	cml := c.Header.ABI.MemoryLayout
	ctx = log.V{"capture memory layout": cml}.Bind(ctx)

	deviceABIs := d.Instance().GetConfiguration().GetABIs()
	if len(deviceABIs) == 0 {
		return log.Err(ctx, nil, "Replay device doesn't list any ABIs")
	}

	replayABI := findABI(cml, deviceABIs)
	if replayABI == nil {
		log.I(ctx, "Replay device does not have a memory layout matching device used to trace")
		replayABI = deviceABIs[0]
	}
	ctx = log.V{"replay target ABI": replayABI}.Bind(ctx)

	b := builder.New(replayABI.MemoryLayout)

	_, ranges, err := initialcmds.InitialCommands(ctx, pCapture)

	generatorReplayTimer.Time(func() {
		err = generator.Replay(
			ctx,
			Intent{pDevice, pCapture},
			Config(&struct{}{}),
			[]RequestAndResult{{
				Request: Request(generator.(interface{ ExportReplayRequest() Request }).ExportReplayRequest()),
				Result:  func(val interface{}, err error) {},
			}},
			d.Instance(),
			c,
			&adapter{
				state:   c.NewUninitializedState(ctx, ranges),
				builder: b,
			})
	})

	if err != nil {
		return log.Err(ctx, err, "Replay returned error")
	}

	var payload gapir.Payload
	var handlePost builder.PostDataHandler
	var handleNotification builder.NotificationHandler
	builderBuildTimer.Time(func() { payload, handlePost, handleNotification, err = b.Build(ctx) })
	if err != nil {
		return log.Err(ctx, err, "Failed to build replay payload")
	}

	err = os.MkdirAll(outDir, os.ModePerm)
	if err != nil {
		return log.Errf(ctx, err, "Failed to create output directory: %v", outDir)
	}

	payloadBytes, err := proto.Marshal(&payload)
	if err != nil {
		return log.Errf(ctx, err, "Failed to serialize replay payload.")
	}
	err = ioutil.WriteFile(gopath.Join(outDir, "payload.bin"), payloadBytes, 0644)

	ar := archive.New(gopath.Join(outDir, "resources"))
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
