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

// +build integration

package service_test

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/capture"
	gapis "github.com/google/gapid/gapis/client"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/replay"
	"github.com/google/gapid/gapis/server"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
	"github.com/google/gapid/gapis/stringtable"
	"github.com/google/gapid/test/integration/replay/gles/samples"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// adapter convertes a GapidServer to a GapidClient interface (drops the opts
// args and uses a new context)
type adapter struct{ service.GapidServer }

func (a adapter) GetServerInfo(ctx context.Context, in *service.GetServerInfoRequest, opts ...grpc.CallOption) (*service.GetServerInfoResponse, error) {
	return a.GapidServer.GetServerInfo(context.Background(), in)
}
func (a adapter) Get(ctx context.Context, in *service.GetRequest, opts ...grpc.CallOption) (*service.GetResponse, error) {
	return a.GapidServer.Get(context.Background(), in)
}
func (a adapter) Set(ctx context.Context, in *service.SetRequest, opts ...grpc.CallOption) (*service.SetResponse, error) {
	return a.GapidServer.Set(context.Background(), in)
}
func (a adapter) Follow(ctx context.Context, in *service.FollowRequest, opts ...grpc.CallOption) (*service.FollowResponse, error) {
	return a.GapidServer.Follow(context.Background(), in)
}
func (a adapter) BeginCPUProfile(ctx context.Context, in *service.BeginCPUProfileRequest, opts ...grpc.CallOption) (*service.BeginCPUProfileResponse, error) {
	return a.GapidServer.BeginCPUProfile(context.Background(), in)
}
func (a adapter) EndCPUProfile(ctx context.Context, in *service.EndCPUProfileRequest, opts ...grpc.CallOption) (*service.EndCPUProfileResponse, error) {
	return a.GapidServer.EndCPUProfile(context.Background(), in)
}
func (a adapter) GetPerformanceCounters(ctx context.Context, in *service.GetPerformanceCountersRequest, opts ...grpc.CallOption) (*service.GetPerformanceCountersResponse, error) {
	return a.GapidServer.GetPerformanceCounters(context.Background(), in)
}
func (a adapter) GetProfile(ctx context.Context, in *service.GetProfileRequest, opts ...grpc.CallOption) (*service.GetProfileResponse, error) {
	return a.GapidServer.GetProfile(context.Background(), in)
}
func (a adapter) GetSchema(ctx context.Context, in *service.GetSchemaRequest, opts ...grpc.CallOption) (*service.GetSchemaResponse, error) {
	return a.GapidServer.GetSchema(context.Background(), in)
}
func (a adapter) GetAvailableStringTables(ctx context.Context, in *service.GetAvailableStringTablesRequest, opts ...grpc.CallOption) (*service.GetAvailableStringTablesResponse, error) {
	return a.GapidServer.GetAvailableStringTables(context.Background(), in)
}
func (a adapter) GetStringTable(ctx context.Context, in *service.GetStringTableRequest, opts ...grpc.CallOption) (*service.GetStringTableResponse, error) {
	return a.GapidServer.GetStringTable(context.Background(), in)
}
func (a adapter) ImportCapture(ctx context.Context, in *service.ImportCaptureRequest, opts ...grpc.CallOption) (*service.ImportCaptureResponse, error) {
	return a.GapidServer.ImportCapture(context.Background(), in)
}
func (a adapter) LoadCapture(ctx context.Context, in *service.LoadCaptureRequest, opts ...grpc.CallOption) (*service.LoadCaptureResponse, error) {
	return a.GapidServer.LoadCapture(context.Background(), in)
}
func (a adapter) GetDevices(ctx context.Context, in *service.GetDevicesRequest, opts ...grpc.CallOption) (*service.GetDevicesResponse, error) {
	return a.GapidServer.GetDevices(context.Background(), in)
}
func (a adapter) GetDevicesForReplay(ctx context.Context, in *service.GetDevicesForReplayRequest, opts ...grpc.CallOption) (*service.GetDevicesForReplayResponse, error) {
	return a.GapidServer.GetDevicesForReplay(context.Background(), in)
}
func (a adapter) GetFramebufferAttachment(ctx context.Context, in *service.GetFramebufferAttachmentRequest, opts ...grpc.CallOption) (*service.GetFramebufferAttachmentResponse, error) {
	return a.GapidServer.GetFramebufferAttachment(context.Background(), in)
}

func setup(t *testing.T) (log.Context, server.Server) {
	ctx := log.Testing(t)

	m, r := replay.New(ctx), bind.NewRegistry()
	ctx = replay.PutManager(ctx, m)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	ctx = bind.PutRegistry(ctx, r)

	r.AddDevice(ctx, bind.Host(ctx))

	return ctx, gapis.New(adapter{server.NewGapidServer(ctx, cfg)})
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
	drawAtomIndex   uint64
)

func init() {
	check := func(err error) {
		if err != nil {
			panic(err)
		}
	}
	ctx := log.Background()
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	atoms, draw := samples.DrawTexturedSquare(ctx)
	p, err := capture.ImportAtomList(ctx, "sample", atoms)
	check(err)
	buf := bytes.Buffer{}
	check(capture.Export(ctx, p, &buf))
	testCaptureData, drawAtomIndex = buf.Bytes(), uint64(draw)
}

func TestGetServerInfo(t *testing.T) {
	ctx, server := setup(t)
	got, err := server.GetServerInfo(ctx)
	assert.With(ctx).ThatError(err).Succeeded()
	assert.With(ctx).That(got).DeepEquals(cfg.Info)
}

func TestGetSchema(t *testing.T) {
	ctx, server := setup(t)
	got, err := server.GetSchema(ctx)
	assert.With(ctx).ThatError(err).Succeeded()
	assert.With(ctx).That(got).IsNotNil()
}

func TestGetAvailableStringTables(t *testing.T) {
	ctx, server := setup(t)
	got, err := server.GetAvailableStringTables(ctx)
	assert.With(ctx).ThatError(err).Succeeded()
	assert.With(ctx).That(got).DeepEquals([]*stringtable.Info{stringtables[0].Info})
}

func TestGetStringTable(t *testing.T) {
	ctx, server := setup(t)
	got, err := server.GetStringTable(ctx, stringtables[0].Info)
	assert.With(ctx).ThatError(err).Succeeded()
	assert.With(ctx).That(got).DeepEquals(stringtables[0])
}

func TestImportCapture(t *testing.T) {
	ctx, server := setup(t)
	got, err := server.ImportCapture(ctx, "test-capture", testCaptureData)
	assert.With(ctx).ThatError(err).Succeeded()
	assert.With(ctx).That(got).IsNotNil()
}

func TestGetDevices(t *testing.T) {
	ctx, server := setup(t)
	got, err := server.GetDevices(ctx)
	assert.With(ctx).ThatError(err).Succeeded()
	assert.With(ctx).ThatSlice(got).IsNotEmpty()
}

func TestGetDevicesForReplay(t *testing.T) {
	ctx, server := setup(t)
	capture, err := server.ImportCapture(ctx, "test-capture", testCaptureData)
	assert.With(ctx).ThatError(err).Succeeded()
	assert.With(ctx).That(capture).IsNotNil()
	got, err := server.GetDevicesForReplay(ctx, capture)
	assert.With(ctx).ThatError(err).Succeeded()
	assert.With(ctx).ThatSlice(got).IsNotEmpty()
}

func TestGetFramebufferAttachment(t *testing.T) {
	ctx, server := setup(t)
	capture, err := server.ImportCapture(ctx, "test-capture", testCaptureData)
	assert.With(ctx).ThatError(err).Succeeded()
	assert.With(ctx).That(capture).IsNotNil()
	devices, err := server.GetDevices(ctx)
	assert.With(ctx).ThatError(err).Succeeded()
	assert.With(ctx).ThatSlice(devices).IsNotEmpty()
	after := capture.Commands().Index(drawAtomIndex)
	attachment := gfxapi.FramebufferAttachment_Color0
	settings := &service.RenderSettings{}
	got, err := server.GetFramebufferAttachment(ctx, devices[0], after, attachment, settings)
	assert.With(ctx).ThatError(err).Succeeded()
	assert.With(ctx).That(got).IsNotNil()
}

func TestGet(t *testing.T) {
	ctx, server := setup(t)
	capture, err := server.ImportCapture(ctx, "test-capture", testCaptureData)
	assert.With(ctx).ThatError(err).Succeeded()
	assert.With(ctx).That(capture).IsNotNil()
	T, any := reflect.TypeOf, reflect.TypeOf(struct{}{})
	for _, test := range []struct {
		path path.Node
		ty   reflect.Type
	}{
		{capture, T((*service.Capture)(nil))},
		{capture.Contexts(), T([]*service.Context{})},
		{capture.Commands(), T((*atom.List)(nil))},
		{capture.Commands().Index(drawAtomIndex), T((*atom.Atom)(nil)).Elem()},
		{capture.Commands().Index(drawAtomIndex).StateAfter(), any},
		{capture.Commands().Index(drawAtomIndex).MemoryAfter(0, 0x1000, 0x1000), T((*service.MemoryInfo)(nil))},
		{capture.Commands().Index(drawAtomIndex).Mesh(false), T((*gfxapi.Mesh)(nil))},
		{capture.Hierarchies(), T([]*service.Hierarchy{})},
		{capture.Report(nil), T((*service.Report)(nil))},
		{capture.Resources(), T((*service.Resources)(nil))},
	} {
		ctx := ctx.S("path", test.path.Text())
		got, err := server.Get(ctx, test.path.Path())
		assert.With(ctx).ThatError(err).Succeeded()
		if test.ty.Kind() == reflect.Interface {
			assert.With(ctx).That(got).Implements(test.ty)
		} else if test.ty != any {
			assert.With(ctx).That(reflect.TypeOf(got)).Equals(test.ty)
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
	ctx, server := setup(t)
	err := server.BeginCPUProfile(ctx)
	assert.With(ctx).ThatError(err).Succeeded()
	data, err := server.EndCPUProfile(ctx)
	assert.With(ctx).ThatError(err).Succeeded()
	assert.With(ctx).That(data).IsNotNil()
}

func TestGetPerformanceCounters(t *testing.T) {
	ctx, server := setup(t)
	data, err := server.GetPerformanceCounters(ctx)
	assert.With(ctx).ThatError(err).Succeeded()
	assert.With(ctx).That(data).IsNotNil()
}
