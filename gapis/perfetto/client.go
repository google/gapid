// Copyright (C) 2019 Google Inc.
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

package perfetto

import (
	"context"
	"net"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/gapis/perfetto/client"
)

const (
	consumerService = "ConsumerPort"
)

// Client is a client ("consumer") of a Perfetto service.
type Client struct {
	conn    *client.Connection
	methods map[string]*client.Method
}

// NewClient returns a new client using the provided socket connection. The
// client takes ownership of the connection and invokes the provided cleanup
// on shutdown.
func NewClient(ctx context.Context, conn net.Conn, cleanup app.Cleanup) (*Client, error) {
	c, err := client.Connect(ctx, conn, cleanup)
	if err != nil {
		conn.Close()
		cleanup.Invoke(ctx)
		return nil, err
	}

	bind := client.NewBindSync(ctx)
	if err := c.Bind(ctx, consumerService, bind.Handler); err != nil {
		c.Close(ctx)
		return nil, err
	}
	methods, err := bind.Wait(ctx)
	if err != nil {
		c.Close(ctx)
		return nil, err
	}

	return &Client{
		conn:    c,
		methods: methods,
	}, nil
}

// Close closes the underlying connection to the Perfetto service of this client.
func (c *Client) Close(ctx context.Context) {
	c.conn.Close(ctx)
	c.conn = nil
}
