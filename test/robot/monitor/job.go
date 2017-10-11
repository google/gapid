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

	"github.com/google/gapid/test/robot/job"
)

// Device is the in memory representation/wrapper for a job.Device
type Device struct {
	job.Device
}

// Devices is the type that manages a set of Device objects.
type Devices struct {
	entries []*Device
}

// Worker is the in memory representation/wrapper for a job.Worker
type Worker struct {
	job.Worker
}

// Workers is the type that manages a set of Worker objects.
type Workers struct {
	entries []*Worker
}

// All returns the complete set of Device objects we have seen so far.
func (d *Devices) All() []*Device {
	return d.entries
}

// FindDevice searches the device list for one that matches the supplied id.
func (data *Data) FindDevice(id string) *Device {
	for _, d := range data.Devices.entries {
		if d.Device.Id == id {
			return d
		}
	}
	return nil
}

func (o *DataOwner) updateDevice(ctx context.Context, device *job.Device) error {
	o.Write(func(data *Data) {
		for i, e := range data.Devices.entries {
			if device.Id == e.Id {
				data.Devices.entries[i].Device = *device
				return
			}
		}
		data.Devices.entries = append(data.Devices.entries, &Device{Device: *device})
	})
	return nil
}

// All returns the complete set of Worker objects we have seen so far.
func (w *Workers) All() []*Worker {
	return w.entries
}

func (o *DataOwner) updateWorker(ctx context.Context, worker *job.Worker) error {
	o.Write(func(data *Data) {
		for i, e := range data.Workers.entries {
			if worker.Host == e.Host && worker.Target == e.Target {
				data.Workers.entries[i].Worker = *worker
				return
			}
		}
		data.Workers.entries = append(data.Workers.entries, &Worker{Worker: *worker})
	})
	return nil
}
