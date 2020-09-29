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
	"math"
	"sync"
	"time"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/app/status"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/gapir"
	replaysrv "github.com/google/gapid/gapir/replay_service"
	"google.golang.org/grpc"
)

const (
	// LaunchArgsKey is the bind device property key used to control the command
	// line arguments when launching GAPIR. The property must be of type []string.
	LaunchArgsKey = "gapir-launch-args"
	// gRPCConnectTimeout is the time allowed to establish a gRPC connection.
	gRPCConnectTimeout = time.Second * 30
	// heartbeatInterval is the delay between heartbeat pings.
	heartbeatInterval = time.Second * 2
)

// ReplayExecutor must be implemented by replay executors to handle some live
// interactions with a running replay.
type ReplayExecutor interface {
	// HandlePostData handles the given post data message.
	HandlePostData(context.Context, *gapir.PostData) error
	// HandleNotification handles the given notification message.
	HandleNotification(context.Context, *gapir.Notification) error
	// HandleFinished is notified when the given replay is finished.
	HandleFinished(context.Context, error) error
	// HandleFenceReadyRequest handles when the replayer is waiting for the server
	// to execute the registered FenceReadyRequestCallback for fence ID provided
	// in the FenceReadyRequest.
	HandleFenceReadyRequest(context.Context, *gapir.FenceReadyRequest) error
}

// ReplayerKey is used to uniquely identify a GAPIR instance.
type ReplayerKey struct {
	device bind.Device
	arch   device.Architecture
}

// Client handles multiple GAPIR instances identified by ReplayerKey.
type Client struct {
	// mutex prevents data races when restarting replayers. All exported
	// functions must acquire this mutex upon start, to guard the lookup
	// in the replayers map which might be concurrently updated by a
	// replayer reconnection.
	mutex sync.Mutex
	// replayers stores the informations relater to GAPIR instances.
	replayers map[ReplayerKey]*replayer
}

// New returns a new Client with no replayers.
func New(ctx context.Context) *Client {
	client := &Client{replayers: map[ReplayerKey]*replayer{}}
	app.AddCleanup(ctx, func() {
		client.shutdown(ctx)
	})
	return client
}

// shutdown closes all replayer instances and makes the client invalid.
func (client *Client) shutdown(ctx context.Context) {
	client.mutex.Lock()
	defer client.mutex.Unlock()

	for _, replayer := range client.replayers {
		replayer.closeConnection(ctx)
	}
	client.replayers = nil
}

// Connect starts a GAPIR instance and return its ReplayerKey.
func (client *Client) Connect(ctx context.Context, device bind.Device, abi *device.ABI) (*ReplayerKey, error) {
	client.mutex.Lock()
	defer client.mutex.Unlock()

	ctx = status.Start(ctx, "Connect")
	defer status.Finish(ctx)

	if client.replayers == nil {
		return nil, log.Err(ctx, nil, "Client has been shutdown")
	}

	key := ReplayerKey{device: device, arch: abi.GetArchitecture()}

	if _, ok := client.replayers[key]; ok {
		return &key, nil
	}

	launchArgs, _ := bind.GetRegistry(ctx).DeviceProperty(ctx, device, LaunchArgsKey).([]string)
	newDeviceConnectionInfo, err := initDeviceConnection(ctx, device, abi, launchArgs)
	if err != nil {
		return nil, err
	}

	log.I(ctx, "Waiting for connection to GAPIR...")

	// Create gRPC connection
	conn, err := grpc.Dial(fmt.Sprintf("localhost:%d", newDeviceConnectionInfo.port),
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithTimeout(gRPCConnectTimeout),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(math.MaxInt32)))
	if err != nil {
		return nil, log.Err(ctx, err, "Timeout waiting for connection")
	}
	rpcClient := replaysrv.NewGapirClient(conn)

	replayer := &replayer{
		deviceConnectionInfo: *newDeviceConnectionInfo,
		device:               device,
		abi:                  abi,
		conn:                 conn,
		rpcClient:            rpcClient,
	}

	crash.Go(func() { client.heartbeat(ctx, replayer) })
	log.I(ctx, "Heartbeat connection setup done")

	err = replayer.startReplayCommunicationHandler(ctx)
	if err != nil {
		return nil, log.Err(ctx, err, "Error in startReplayCommunicationHandler")
	}

	client.replayers[key] = replayer
	return &key, nil
}

// removeReplayer closes and removes a GAPIR instance.
func (client *Client) removeReplayer(ctx context.Context, replayer *replayer) {
	client.mutex.Lock()
	defer client.mutex.Unlock()

	replayer.closeConnection(ctx)
	key := ReplayerKey{device: replayer.device, arch: replayer.abi.GetArchitecture()}
	delete(client.replayers, key)
}

// heartbeat regularly sends a ping to a replayer, and restarts it when it fails to reply.
func (client *Client) heartbeat(ctx context.Context, replayer *replayer) {
	for {
		select {
		case <-task.ShouldStop(ctx):
			return
		case <-time.After(heartbeatInterval):
			err := replayer.ping(ctx)
			if err != nil {
				log.E(ctx, "Error sending keep-alive ping. Error: %v", err)
				client.removeReplayer(ctx, replayer)
				return
			}
		}
	}
}

// getActiveReplayer returns the replayer identified by key only if this
// replayer has an active connection to a GAPIR instance.
func (client *Client) getActiveReplayer(ctx context.Context, key *ReplayerKey) (*replayer, error) {
	replayer, found := client.replayers[*key]
	if !found {
		return nil, log.Errf(ctx, nil, "Cannot find replayer for this key: %v", key)
	}

	if replayer.rpcClient == nil || replayer.conn == nil || replayer.rpcStream == nil {
		return nil, log.Err(ctx, nil, "Replayer has no active connection")
	}

	return replayer, nil
}

// BeginReplay sends a replay request to the replayer identified by key.
func (client *Client) BeginReplay(ctx context.Context, key *ReplayerKey, payload string, dependent string) error {
	client.mutex.Lock()
	defer client.mutex.Unlock()

	ctx = log.Enter(ctx, "Starting replay on gapir device")
	replayer, err := client.getActiveReplayer(ctx, key)
	if err != nil {
		return err
	}

	idReq := replaysrv.ReplayRequest{
		Req: &replaysrv.ReplayRequest_Replay{
			Replay: &replaysrv.Replay{
				ReplayId:    payload,
				DependentId: dependent,
			},
		},
	}
	err = replayer.rpcStream.Send(&idReq)
	if err != nil {
		return log.Err(ctx, err, "Sending replay id")
	}

	return nil
}

// SetReplayExecutor assigns a replay executor to the replayer identified by
// key. It returns a cleanup function to remove the executor once the replay
// is finished.
func (client *Client) SetReplayExecutor(ctx context.Context, key *ReplayerKey, executor ReplayExecutor) (func(), error) {
	client.mutex.Lock()
	defer client.mutex.Unlock()

	replayer, err := client.getActiveReplayer(ctx, key)
	if err != nil {
		return nil, err
	}

	if replayer.executor != nil {
		return nil, log.Err(ctx, nil, "Cannot set an executor while one is already present")
	}
	replayer.executor = executor
	return func() { replayer.executor = nil }, nil
}

// PrewarmReplay requests the GAPIR device to get itself into the given state
func (client *Client) PrewarmReplay(ctx context.Context, key *ReplayerKey, payload string, cleanup string) error {
	client.mutex.Lock()
	defer client.mutex.Unlock()

	replayer, err := client.getActiveReplayer(ctx, key)
	if err != nil {
		return log.Err(ctx, err, "Getting active replayer")
	}

	PrerunReq := replaysrv.ReplayRequest{
		Req: &replaysrv.ReplayRequest_Prewarm{
			Prewarm: &replaysrv.PrewarmRequest{
				PrerunId:  payload,
				CleanupId: cleanup,
			},
		},
	}
	err = replayer.rpcStream.Send(&PrerunReq)
	if err != nil {
		return log.Err(ctx, err, "Sending replay payload")
	}
	return nil
}
