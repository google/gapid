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
	"fmt"
	"sync"
	"time"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/app/status"
	"github.com/google/gapid/core/context/keys"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/gapir"
)

type tyLaunchArgsKey string

const (
	// LaunchArgsKey is the bind device property key used to control the command
	// line arguments when launching GAPIR. The property must be of type []string.
	LaunchArgsKey     tyLaunchArgsKey = "gapir-launch-args"
	connectTimeout                    = time.Second * 10
	heartbeatInterval                 = time.Millisecond * 500
)

type clientInfo struct {
	device               bind.Device
	arch                 device.Architecture
	abi                  *device.ABI
	deviceConnectionInfo deviceConnectionInfo
	connection           gapir.Connection
	bgConnection         *backgroundConnection
}

type deviceArch struct {
	device bind.Device
	arch   device.Architecture
}

// ConnectionKey is used by manager to obtain a connection
type ConnectionKey deviceArch

// Client handles connections to GAPIR instances on devices.
// A single Client can handle multiple connections.
type Client struct {
	// Mutex is needed due to the risk that reconnect may happen in another thread
	mutex       sync.Mutex
	clientInfos map[ConnectionKey]clientInfo
}

// New returns a newly construct Client.
func New(ctx context.Context) *Client {
	client := &Client{clientInfos: map[ConnectionKey]clientInfo{}}
	app.AddCleanup(ctx, func() {
		client.shutdown(ctx)
	})
	return client
}

// Connect opens a connection to the replay device.
func (client *Client) Connect(ctx context.Context, device bind.Device, abi *device.ABI) (*ConnectionKey, error) {
	client.mutex.Lock()
	defer client.mutex.Unlock()

	ctx = status.Start(ctx, "Connect")
	defer status.Finish(ctx)

	if client.clientInfos == nil {
		return nil, log.Err(ctx, nil, "Client has been shutdown")
	}

	deviceArch := deviceArch{device: device, arch: abi.GetArchitecture()}
	key := ConnectionKey(deviceArch)

	if _, ok := client.clientInfos[key]; ok {
		return &key, nil
	}

	launchArgs, _ := bind.GetRegistry(ctx).DeviceProperty(ctx, device, LaunchArgsKey).([]string)
	newDeviceConnectionInfo, err := initDeviceConnection(ctx, device, abi, launchArgs)
	if err != nil {
		return nil, err
	}

	log.I(ctx, "Waiting for connection to GAPIR...")

	connection, err := newConnection(fmt.Sprintf("localhost:%d", newDeviceConnectionInfo.port), newDeviceConnectionInfo.authToken, connectTimeout)
	if err != nil {
		return nil, log.Err(ctx, err, "Timeout waiting for connection")
	}

	crash.Go(func() { client.heartbeat(ctx, heartbeatInterval, key) })

	log.I(ctx, "Heartbeat connection setup done")

	bgConnection, err := client.makeBackgroundConnection(ctx, device, connection)
	if err != nil {
		return nil, log.Err(ctx, err, "Background connection error")
	}

	client.clientInfos[key] = clientInfo{
		deviceConnectionInfo: *newDeviceConnectionInfo,
		connection:           connection,
		device:               device,
		arch:                 abi.Architecture,
		abi:                  abi,
		bgConnection:         bgConnection}
	return &key, nil
}

func (client *Client) makeBackgroundConnection(ctx context.Context, device bind.Device, conn gapir.Connection) (*backgroundConnection, error) {
	bgc := &backgroundConnection{conn: conn, OS: device.Instance().GetConfiguration().GetOS()}

	connected := make(chan error)
	cctx := keys.Clone(context.Background(), ctx)
	crash.Go(func() {
		// This shouldn't be sitting on this context
		cctx := status.PutTask(cctx, nil)
		cctx = status.StartBackground(cctx, "Handle Replay Communication")
		defer status.Finish(cctx)

		// Kick the communication handler
		err := conn.HandleReplayCommunication(cctx, bgc, connected)
		if err != nil {
			log.E(cctx, "Error communication with gapir: %v", err)
		}

		bgc.HandleFinished(ctx, err)
	})
	err := <-connected
	if err != nil {
		return nil, err
	}
	return bgc, nil
}

func (client *Client) closeConnection(ctx context.Context, key ConnectionKey) {
	clientInfo, found := client.clientInfos[key]
	if !found {
		log.Err(ctx, nil, "Connection could not be found!")
	}

	clientInfo.connection.Shutdown(ctx)
	clientInfo.deviceConnectionInfo.cleanupFunc()
	clientInfo.connection.Close()
}

func (client *Client) removeConnection(ctx context.Context, key ConnectionKey) {
	client.mutex.Lock()
	defer client.mutex.Unlock()

	client.closeConnection(ctx, key)
	delete(client.clientInfos, key)
}

func (client *Client) shutdown(ctx context.Context) {
	client.mutex.Lock()
	defer client.mutex.Unlock()

	for key := range client.clientInfos {
		client.closeConnection(ctx, key)
	}

	client.clientInfos = nil
}

func (client *Client) reconnect(ctx context.Context, key ConnectionKey) {
	clientInfo := client.clientInfos[key]
	device := clientInfo.device
	abi := clientInfo.abi

	client.removeConnection(ctx, key)
	client.Connect(ctx, device, abi)
}

func (client *Client) ping(ctx context.Context, connection gapir.Connection) (time.Duration, error) {
	if connection == nil {
		return time.Duration(0), log.Errf(ctx, nil, "cannot ping without gapir connection")
	}

	start := time.Now()
	err := connection.Ping(ctx)
	if err != nil {
		return 0, err
	}
	return time.Since(start), nil
}

func (client *Client) heartbeat(ctx context.Context, pingInterval time.Duration, key ConnectionKey) {
	for {
		select {
		case <-task.ShouldStop(ctx):
			return
		case <-time.After(pingInterval):
			_, err := client.ping(ctx, client.clientInfos[key].connection)
			if err != nil {
				log.E(ctx, "Error sending keep-alive ping. Error: %v", err)
				client.reconnect(ctx, key)
				return
			}
		}
	}
}

func (client *Client) BeginReplay(ctx context.Context, conn *ConnectionKey, payload string, dependent string) error {
	return client.clientInfos[*conn].bgConnection.BeginReplay(ctx, payload, dependent)
}

func (client *Client) SetReplayExecutor(ctx context.Context, conn *ConnectionKey, executor ReplayExecutor) (func(), error) {
	return client.clientInfos[*conn].bgConnection.SetReplayExecutor(ctx, executor)
}

func (client *Client) PrewarmReplay(ctx context.Context, conn *ConnectionKey, payload string, cleanup string) error {
	return client.clientInfos[*conn].bgConnection.PrewarmReplay(ctx, payload, cleanup)
}
