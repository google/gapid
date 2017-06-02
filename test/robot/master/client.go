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
	"io"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/test/robot/search"
	"github.com/pkg/errors"
)

// Client is a wrapper over a Master object that provides client targeted features.
type Client struct {
	// Master is the master this client is talking to.
	// This should not be modifed, the results are undefined if you do.
	Master   Master
	shutdown Shutdown
	name     string
}

// NewClient returns a new master client object that talks to the provided Master.
func NewClient(ctx context.Context, m Master) *Client {
	return &Client{
		Master: m,
	}
}

// Orbit registers a satellite with the master.
// The function will only return when the connection is lost.
func (c *Client) Orbit(ctx context.Context, services ServiceList) (Shutdown, error) {
	err := c.Master.Orbit(ctx, services,
		func(ctx context.Context, command *Command) error {
			switch do := command.Do.(type) {
			case *Command_Ping:
				return nil
			case *Command_Identify:
				c.name = do.Identify.Name
				log.I(ctx, "Identified as %s", c.name)
				return nil
			case *Command_Shutdown:
				// abort the report stream
				c.shutdown = *do.Shutdown
				return io.EOF
			default:
				return log.Err(ctx, nil, "Unknown command type")
			}
		},
	)
	if errors.Cause(err) == io.EOF {
		err = nil
	}
	return c.shutdown, err
}

// Shutdown causes a graceful shutdown of the server.
func (c *Client) Shutdown(ctx context.Context, to ...string) error {
	_, err := c.Master.Shutdown(ctx, &ShutdownRequest{
		Shutdown: &Shutdown{
			Now:     false,
			Restart: false,
		},
		To: to,
	})
	return err
}

// Restart causes a graceful shutdown and restart of the server.
func (c *Client) Restart(ctx context.Context, to ...string) error {
	_, err := c.Master.Shutdown(ctx, &ShutdownRequest{
		Shutdown: &Shutdown{
			Now:     false,
			Restart: true,
		},
		To: to,
	})
	return err
}

// Kill causes an immediate shutdown of the server.
func (c *Client) Kill(ctx context.Context, to ...string) error {
	_, err := c.Master.Shutdown(ctx, &ShutdownRequest{
		Shutdown: &Shutdown{
			Now:     true,
			Restart: false,
		},
		To: to,
	})
	return err
}

// Search delivers the set of satellites that match the query to the supplied function.
func (c *Client) Search(ctx context.Context, query *search.Query, handler SatelliteHandler) error {
	return c.Master.Search(ctx, query, handler)
}
