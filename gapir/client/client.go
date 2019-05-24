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

package client

import (
	"context"
	"sync"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/app/status"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/gapir"
)

type tyLaunchArgsKey string

// LaunchArgsKey is the bind device property key used to control the command
// line arguments when launching GAPIR. The property must be of type []string.
const LaunchArgsKey tyLaunchArgsKey = "<gapir-launch-args>"

// Client is interface used to connect to GAPIR instances on devices.
type Client struct {
	mutex    sync.Mutex
	sessions map[deviceArch]*session
}

// New returns a newly construct Client.
func New(ctx context.Context) *Client {
	c := &Client{sessions: map[deviceArch]*session{}}
	app.AddCleanup(ctx, func() {
		c.shutdown(ctx)
	})
	return c
}

type deviceArch struct {
	d bind.Device
	a device.Architecture
}

// Connect opens a connection to the replay device.
func (c *Client) Connect(ctx context.Context, d bind.Device, abi *device.ABI) (gapir.Connection, error) {
	ctx = status.Start(ctx, "Connect")
	defer status.Finish(ctx)

	s, isNew, err := c.getOrCreateSession(ctx, d, abi)
	if err != nil {
		return nil, err
	}

	if isNew {
		launchArgs, _ := bind.GetRegistry(ctx).DeviceProperty(ctx, d, LaunchArgsKey).([]string)
		if err := s.init(ctx, d, abi, launchArgs); err != nil {
			return nil, err
		}
	}

	return s.connect(ctx)
}

func (c *Client) getOrCreateSession(ctx context.Context, d bind.Device, abi *device.ABI) (*session, bool, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.sessions == nil {
		return nil, false, log.Err(ctx, nil, "Client has been shutdown")
	}

	key := deviceArch{d, abi.Architecture}
	s, existing := c.sessions[key]
	if existing {
		return s, false, nil
	}

	s = newSession(d)
	c.sessions[key] = s
	s.onClose(func() {
		c.mutex.Lock()
		defer c.mutex.Unlock()
		delete(c.sessions, key)
	})

	return s, true, nil
}

func (c *Client) shutdown(ctx context.Context) {
	for _, s := range c.getSessions() {
		s.close(ctx)
	}
}

func (c *Client) getSessions() []*session {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	r := []*session{}
	for _, s := range c.sessions {
		r = append(r, s)
	}
	return r
}
