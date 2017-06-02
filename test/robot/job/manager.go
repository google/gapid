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

package job

import (
	"context"

	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/test/robot/search"
)

// DeviceHandler is a function used to consume a stream of Devices.
type DeviceHandler func(context.Context, *Device) error

// WorkerHandler is a function used to consume a stream of Workers.
type WorkerHandler func(context.Context, *Worker) error

// Manager is the abstract interface to the job manager.
type Manager interface {
	// SearchDevices delivers matching workers to the supplied handler.
	SearchDevices(ctx context.Context, query *search.Query, handler DeviceHandler) error
	// SearchWorkers delivers matching workers to the supplied handler.
	SearchWorkers(ctx context.Context, query *search.Query, handler WorkerHandler) error
	// GetWorker finds or adds a worker.
	GetWorker(ctx context.Context, host *device.Instance, target *device.Instance, op Operation) (*Worker, error)
}
