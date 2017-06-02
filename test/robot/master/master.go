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

package master

import (
	"context"

	"github.com/google/gapid/test/robot/search"
)

type SatelliteHandler func(context.Context, *Satellite) error
type CommandHandler func(context.Context, *Command) error

// Master is the interface to a master implementation.
// It abstracts away whether the master is remote or local.
type Master interface {
	// Search returns a iterator of matching satellites from the store.
	Search(context.Context, *search.Query, SatelliteHandler) error
	// Orbit adds a satellite to the set being managed by the master.
	// The master will use the returned command stream to control the satellite.
	Orbit(context.Context, ServiceList, CommandHandler) error
	// Shutdown is called to ask the master to send shutdown requests to satellites.
	Shutdown(context.Context, *ShutdownRequest) (*ShutdownResponse, error)
}
