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
	"flag"
	"os"
	"reflect"
	"testing"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/data/binary"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device/bind"
	gapir "github.com/google/gapid/gapir/client"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/replay/builder"
	"github.com/google/gapid/gapis/replay/executor"
	"github.com/google/gapid/gapis/replay/value"
)

func TestMain(m *testing.M) {
	flag.Parse()
	ctx, cancel := task.WithCancel(context.Background())
	code := m.Run()
	cancel()
	app.WaitForCleanup(ctx)
	os.Exit(code)
}

func doReplay(t *testing.T, f func(*builder.Builder)) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))

	device := bind.Host(ctx)
	client := gapir.New(ctx)
	abi := device.Instance().GetConfiguration().PreferredABI(nil)
	connection, err := client.Connect(ctx, device, abi)
	if err != nil {
		t.Errorf("Failed to connect to '%v': %v", device, err)
		return
	}

	b := builder.New(abi.MemoryLayout)

	f(b)

	payload, decoder, err := b.Build(ctx)
	if err != nil {
		t.Errorf("Build failed with error: %v", err)
	}

	err = executor.Execute(ctx, payload, decoder, connection, abi.MemoryLayout)
	if err != nil {
		t.Errorf("Executor failed with error: %v", err)
	}
}

func TestPostbackString(t *testing.T) {
	expected := "γειά σου κόσμος"

	done := make(chan struct{})

	doReplay(t, func(b *builder.Builder) {
		ptr := b.String(expected)
		b.Post(ptr, uint64(len(expected)), func(r binary.Reader, err error) error {
			defer close(done)
			if err != nil {
				t.Errorf("Postback returned error: %v", err)
				return err
			}
			data := make([]byte, len(expected))
			r.Data(data)
			err = r.Error()
			if err != nil {
				t.Errorf("Postback returned error: %v", err)
				return err
			}
			if expected != string(data) {
				t.Errorf("Postback data was not as expected. Expected: %v. Got: %v", expected, data)
			}
			return err
		})
	})

	<-done
}

func TestMultiPostback(t *testing.T) {
	done := make(chan struct{})

	doReplay(t, func(b *builder.Builder) {
		ptr := b.AllocateTemporaryMemory(8)
		b.Push(value.Bool(false))
		b.Store(ptr)
		b.Post(ptr, 1, func(r binary.Reader, err error) error {
			expected := false
			if err != nil {
				t.Errorf("Postback returned error: %v", err)
				return err
			}
			data := r.Bool()
			err = r.Error()
			if err != nil {
				t.Errorf("Postback returned error: %v", err)
				return err
			}
			if !reflect.DeepEqual(expected, data) {
				t.Errorf("Postback data was not as expected. Expected: %v. Got: %v", expected, data)
			}
			return err
		})

		b.Push(value.Bool(true))
		b.Store(ptr)
		b.Post(ptr, 1, func(r binary.Reader, err error) error {
			expected := true
			if err != nil {
				t.Errorf("Postback returned error: %v", err)
				return err
			}
			data := r.Bool()
			err = r.Error()
			if err != nil {
				t.Errorf("Postback returned error: %v", err)
				return err
			}
			if !reflect.DeepEqual(expected, data) {
				t.Errorf("Postback data was not as expected. Expected: %v. Got: %v", expected, data)
			}
			return err
		})

		b.Push(value.F64(123.456))
		b.Store(ptr)
		b.Post(ptr, 8, func(r binary.Reader, err error) error {
			expected := float64(123.456)
			if err != nil {
				t.Errorf("Postback returned error: %v", err)
				return err
			}
			data := r.Float64()
			err = r.Error()
			if err != nil {
				t.Errorf("Postback returned error: %v", err)
				return err
			}
			if !reflect.DeepEqual(expected, data) {
				t.Errorf("Postback data was not as expected. Expected: %v. Got: %v", expected, data)
			}
			close(done)
			return err
		})
	})

	<-done
}
