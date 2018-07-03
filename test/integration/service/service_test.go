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

package service_test

import (
	"bytes"
	"context"
	"reflect"
	"testing"

	"github.com/google/gapid/core/app/auth"
	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/net/grpcutil"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/core/os/device/host"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/capture"
	gapis "github.com/google/gapid/gapis/client"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/replay"
	"github.com/google/gapid/gapis/server"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
	"github.com/google/gapid/gapis/stringtable"
	"github.com/google/gapid/test/integration/gles/snippets"
	"google.golang.org/grpc"
)

func startServerAndGetGrpcClient(ctx context.Context, config server.Config) (service.Service, error, func()) {
	l := grpcutil.NewPipeListener("pipe:servicetest")

	schan := make(chan *grpc.Server, 1)
	go server.NewWithListener(ctx, l, config, schan)
	svr := <-schan

	conn, err := grpcutil.Dial(ctx, "pipe:servicetest",
		grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(auth.ClientInterceptor(config.AuthToken)),
		grpc.WithDialer(grpcutil.GetDialer(ctx)),
	)
	if err != nil {
		return nil, log.Err(ctx, err, "Dialing GAPIS"), nil
	}
	client := gapis.Bind(conn)

	return client, nil, func() {
		client.Close()
		svr.GracefulStop()
	}
}

func setup(t *testing.T) (context.Context, server.Server, func()) {
	ctx := log.Testing(t)
	r := bind.NewRegistry()
	ctx = bind.PutRegistry(ctx, r)
	m := replay.New(ctx)
	ctx = replay.PutManager(ctx, m)
	ctx = database.Put(ctx, database.NewInMemory(ctx))

	r.AddDevice(ctx, bind.Host(ctx))

	client, err, shutdown := startServerAndGetGrpcClient(ctx, cfg)
	assert.For(ctx, "err").ThatError(err).Succeeded()

	return ctx, client, shutdown
}

func text(text string) *stringtable.Node {
	return &stringtable.Node{Node: &stringtable.Node_Text{Text: &stringtable.Text{Text: text}}}
}

var (
	stringtables = []*stringtable.StringTable{
		&stringtable.StringTable{
			Info: &stringtable.Info{
				CultureCode: "animals",
			},
			Entries: map[string]*stringtable.Node{
				"fish": text("glub"),
				"dog":  text("barks"),
				"cat":  text("meows"),
				"fox":  text("?"),
			},
		},
	}
	cfg = server.Config{
		Info: &service.ServerInfo{
			Name:         "testbot2000",
			VersionMajor: 123,
			VersionMinor: 456,
			Features:     []string{"moo", "meow", "meh"},
		},
		StringTables: stringtables,
	}
	testCaptureData []byte
	drawCmdIndex    uint64
	swapCmdIndex    uint64
)

func init() {
	check := func(err error) {
		if err != nil {
			panic(err)
		}
	}
	ctx := context.Background()

	deviceScanDone, onDeviceScanDone := task.NewSignal()
	onDeviceScanDone(ctx)
	cfg.DeviceScanDone = deviceScanDone

	ctx = database.Put(ctx, database.NewInMemory(ctx))
	dev := host.Instance(ctx)

	b := snippets.NewBuilder(ctx, dev)
	b.CreateContext(128, 128, false, false)
	draw, swap := b.DrawTexturedSquare(ctx)

	buf := bytes.Buffer{}
	check(capture.Export(ctx, b.Capture(ctx, "test-capture"), &buf))
	testCaptureData, drawCmdIndex, swapCmdIndex = buf.Bytes(), uint64(draw), uint64(swap)
}

func TestGetServerInfo(t *testing.T) {
	ctx, server, shutdown := setup(t)
	defer shutdown()
	got, err := server.GetServerInfo(ctx)
	assert.For(ctx, "err").ThatError(err).Succeeded()
	assert.For(ctx, "got").That(got).DeepEquals(cfg.Info)
}

func TestGetAvailableStringTables(t *testing.T) {
	ctx, server, shutdown := setup(t)
	defer shutdown()
	got, err := server.GetAvailableStringTables(ctx)
	assert.For(ctx, "err").ThatError(err).Succeeded()
	assert.For(ctx, "got").ThatSlice(got).DeepEquals([]*stringtable.Info{stringtables[0].Info})
}

func TestGetStringTable(t *testing.T) {
	ctx, server, shutdown := setup(t)
	defer shutdown()
	got, err := server.GetStringTable(ctx, stringtables[0].Info)
	assert.For(ctx, "err").ThatError(err).Succeeded()
	assert.For(ctx, "got").That(got).DeepEquals(stringtables[0])
}

func TestImportCapture(t *testing.T) {
	ctx, server, shutdown := setup(t)
	defer shutdown()
	got, err := server.ImportCapture(ctx, "test-capture", testCaptureData)
	assert.For(ctx, "err").ThatError(err).Succeeded()
	assert.For(ctx, "got").That(got).IsNotNil()
}

func TestGetDevices(t *testing.T) {
	ctx, server, shutdown := setup(t)
	defer shutdown()
	got, err := server.GetDevices(ctx)
	assert.For(ctx, "err").ThatError(err).Succeeded()
	assert.For(ctx, "got").ThatSlice(got).IsNotEmpty()
}

func TestGetDevicesForReplay(t *testing.T) {
	ctx, server, shutdown := setup(t)
	defer shutdown()
	capture, err := server.ImportCapture(ctx, "test-capture", testCaptureData)
	assert.For(ctx, "err").ThatError(err).Succeeded()
	assert.For(ctx, "capture").That(capture).IsNotNil()
	got, err := server.GetDevicesForReplay(ctx, capture)
	assert.For(ctx, "err").ThatError(err).Succeeded()
	assert.For(ctx, "got").ThatSlice(got).IsNotEmpty()
}

func TestGetFramebufferAttachment(t *testing.T) {
	ctx, server, shutdown := setup(t)
	defer shutdown()
	capture, err := server.ImportCapture(ctx, "test-capture", testCaptureData)
	assert.For(ctx, "err").ThatError(err).Succeeded()
	assert.For(ctx, "capture").That(capture).IsNotNil()
	devices, err := server.GetDevices(ctx)
	assert.For(ctx, "err").ThatError(err).Succeeded()
	assert.For(ctx, "devices").ThatSlice(devices).IsNotEmpty()
	after := capture.Command(swapCmdIndex)
	attachment := api.FramebufferAttachment_Color0
	settings := &service.RenderSettings{}
	renderSettings := &service.ReplaySettings{Device: devices[0]}
	got, err := server.GetFramebufferAttachment(ctx, renderSettings, after, attachment, settings, nil)
	assert.For(ctx, "err").ThatError(err).Succeeded()
	assert.For(ctx, "got").That(got).IsNotNil()
}

func TestGet(t *testing.T) {
	ctx, server, shutdown := setup(t)
	defer shutdown()
	capture, err := server.ImportCapture(ctx, "test-capture", testCaptureData)
	assert.For(ctx, "err").ThatError(err).Succeeded()
	assert.For(ctx, "capture").That(capture).IsNotNil()
	T, any := reflect.TypeOf, reflect.TypeOf(struct{}{})

	for _, test := range []struct {
		path path.Node
		ty   reflect.Type
	}{
		{capture, T((*service.Capture)(nil))},
		{capture.Contexts(), T((*service.Contexts)(nil))},
		{capture.Commands(), T((*service.Commands)(nil))},
		{capture.Command(swapCmdIndex), T((*api.Command)(nil))},
		// TODO: box.go doesn't currently support serializing structs this big.
		// See bug https://github.com/google/gapid/issues/1761
		// panic: reflect.nameFrom: name too long
		// {capture.Command(swapCmdIndex).StateAfter(), any},
		{capture.Command(swapCmdIndex).MemoryAfter(0, 0x1000, 0x1000), T((*service.Memory)(nil))},
		{capture.Command(drawCmdIndex).Mesh(nil), T((*api.Mesh)(nil))},
		{capture.CommandTree(nil), T((*service.CommandTree)(nil))},
		{capture.Report(nil, nil), T((*service.Report)(nil))},
		{capture.Resources(), T((*service.Resources)(nil))},
	} {
		ctx = log.V{"path": test.path}.Bind(ctx)
		got, err := server.Get(ctx, test.path.Path())
		assert.For(ctx, "err").ThatError(err).Succeeded()
		if test.ty.Kind() == reflect.Interface {
			assert.For(ctx, "got").That(got).Implements(test.ty)
		} else if test.ty != any {
			assert.For(ctx, "ty").That(reflect.TypeOf(got)).Equals(test.ty)
		}
	}
}

func TestSet(t *testing.T) {
	// TODO
}

func TestFollow(t *testing.T) {
	// TODO
}

func TestCPUProfile(t *testing.T) {
	ctx, server, shutdown := setup(t)
	defer shutdown()
	err := server.BeginCPUProfile(ctx)
	assert.For(ctx, "err").ThatError(err).Succeeded()
	data, err := server.EndCPUProfile(ctx)
	assert.For(ctx, "err").ThatError(err).Succeeded()
	assert.For(ctx, "data").That(data).IsNotNil()
}

func TestGetPerformanceCounters(t *testing.T) {
	ctx, server, shutdown := setup(t)
	defer shutdown()
	data, err := server.GetPerformanceCounters(ctx)
	assert.For(ctx, "err").ThatError(err).Succeeded()
	assert.For(ctx, "data").That(data).IsNotNil()
}
