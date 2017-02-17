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
	"io"
	"sync"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/fault/cause"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/core/os/file"
)

func init() {
	const name = "gapir"
	// Search directory that this executable is in.
	if path, err := file.FindExecutable(file.ExecutablePath().Parent().Join(name).System()); err == nil {
		GapirPath = path
		//the layer's manifest file should be placed at the same directory as
		//gapir exectuable.
		VirtualSwapchainLayerPath = GapirPath.Parent()
		return
	}
	// Search $PATH.
	if path, err := file.FindExecutable(name); err == nil {
		GapirPath = path
		//the layer's manifest file should be placed at the same directory as
		//gapir exectuable.
		VirtualSwapchainLayerPath = GapirPath.Parent()
		return
	}
}

// Client is interface used to connect to GAPIR instances on devices.
type Client struct {
	mutex    sync.Mutex
	sessions map[deviceArch]*session
}

// New returns a newly construct Client.
func New(ctx log.Context) *Client {
	c := &Client{sessions: map[deviceArch]*session{}}
	app.AddCleanup(ctx, c.shutdown)
	return c
}

type deviceArch struct {
	d bind.Device
	a device.Architecture
}

// Connect opens a connection to the replay device.
func (c *Client) Connect(ctx log.Context, d bind.Device, abi *device.ABI) (io.ReadWriteCloser, error) {
	s, isNew, err := c.getOrCreateSession(ctx, d, abi)
	if err != nil {
		return nil, err
	}

	if isNew {
		if err := s.init(ctx, d, abi); err != nil {
			return nil, err
		}
	}

	return s.connect(ctx)
}

func (c *Client) getOrCreateSession(ctx log.Context, d bind.Device, abi *device.ABI) (*session, bool, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.sessions == nil {
		return nil, false, cause.Explain(ctx, nil, "Client has been shutdown")
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

func (c *Client) shutdown() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	for _, s := range c.sessions {
		s.close()
	}
	c.sessions = nil
}
