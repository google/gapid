// Copyright (C) 2020 Google Inc.
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

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/app/auth"
	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/app/crash/reporting"
	"github.com/google/gapid/core/app/status"
	"github.com/google/gapid/core/context/keys"
	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/gapir"
	replaysrv "github.com/google/gapid/gapir/replay_service"
	"github.com/google/gapid/gapis/database"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	// gapirAuthTokenMetaDataName is the key of the Context metadata pair that
	// contains the authentication token. This token is common knowledge shared
	// between GAPIR client (which is GAPIS) and GAPIR server (which is GAPIR
	// device).
	gapirAuthTokenMetaDataName = "gapir-auth-token"
)

// replayer stores data related to a single GAPIR instance.
type replayer struct {
	device               bind.Device
	abi                  *device.ABI
	deviceConnectionInfo deviceConnectionInfo
	executor             ReplayExecutor
	conn                 *grpc.ClientConn
	rpcClient            replaysrv.GapirClient
	rpcStream            replaysrv.Gapir_ReplayClient
}

// closeConnection properly terminates the replayer
func (replayer *replayer) closeConnection(ctx context.Context) {
	// Call Shutdown RCP on the replayer
	if replayer.rpcClient != nil {
		// Use a clean context, since ctx is most likely already cancelled.
		sdCtx := attachAuthToken(context.Background(), replayer.deviceConnectionInfo.authToken)
		_, err := replayer.rpcClient.Shutdown(sdCtx, &replaysrv.ShutdownRequest{})
		if err != nil {
			log.E(ctx, "Sending replayer Shutdown request: %v", err)
		}
	}
	replayer.rpcClient = nil

	if replayer.rpcStream != nil {
		replayer.rpcStream.CloseSend()
	}
	replayer.rpcStream = nil

	if replayer.conn != nil {
		replayer.conn.Close()
	}
	replayer.conn = nil

	replayer.deviceConnectionInfo.cleanupFunc()
}

// ping uses the Ping RPC to make sure a GAPIR instance is alive.
func (replayer *replayer) ping(ctx context.Context) error {
	if replayer.rpcClient == nil {
		return log.Errf(ctx, nil, "cannot ping without gapir connection")
	}

	ctx = attachAuthToken(ctx, replayer.deviceConnectionInfo.authToken)
	r, err := replayer.rpcClient.Ping(ctx, &replaysrv.PingRequest{})
	if err != nil {
		return log.Err(ctx, err, "Sending ping")
	}
	if r == nil {
		return log.Err(ctx, nil, "No response for ping")
	}

	return nil
}

// startReplayCommunicationHandler launches a background task which creates
// the Replay RPC stream and starts to listen to it.
func (replayer *replayer) startReplayCommunicationHandler(ctx context.Context) error {
	connected := make(chan error)
	cctx := keys.Clone(context.Background(), ctx)
	crash.Go(func() {
		// This shouldn't be sitting on this context
		cctx := status.PutTask(cctx, nil)
		cctx = status.StartBackground(cctx, "Handle Replay Communication")
		defer status.Finish(cctx)

		// Kick the communication handler
		err := replayer.handleReplayCommunication(cctx, connected)
		if err != nil {
			log.E(cctx, "Error communication with gapir: %v", err)
		}

		if replayer.executor == nil {
			log.Err(ctx, nil, "No active replay executor to HandleFinish")
			return
		}
		err = replayer.executor.HandleFinished(ctx, err)
		if err != nil {
			log.Err(cctx, err, "In cleaning up after HandleReplayCommunication returned")
		}
	})
	err := <-connected
	return err
}

// handleReplayCommunication handles the communication with the GAPIR device on
// a replay stream connection. It creates the replay connection stream and then
// enters a loop where it listens to messages from GAPIR and dispatches them to
// the relevant handlers.
func (replayer *replayer) handleReplayCommunication(
	ctx context.Context,
	connected chan error) error {
	ctx = log.Enter(ctx, "HandleReplayCommunication")
	if replayer.conn == nil || replayer.rpcClient == nil {
		return log.Errf(ctx, nil, "Gapir not connected")
	}
	// One Connection is only supposed to be used to handle replay communication
	// in one thread. Initiating another replay communication with a connection
	// which is handling another replay communication will mess up the package
	// order.
	if replayer.rpcStream != nil {
		err := log.Errf(ctx, nil, "Replayer: %v is handling another replay communication stream in another thread. Initiating a new replay on this replayer will mess up the package order for both the existing replay and the new replay", replayer)
		connected <- err
		return err
	}

	ctx = attachAuthToken(ctx, replayer.deviceConnectionInfo.authToken)
	replayStream, err := replayer.rpcClient.Replay(ctx)
	if err != nil {
		return log.Err(ctx, err, "Getting replay stream client")
	}
	replayer.rpcStream = replayStream
	connected <- nil
	defer func() {
		if replayer.rpcStream != nil {
			replayer.rpcStream.CloseSend()
			replayer.rpcStream = nil
		}
	}()
	for {
		if replayer.rpcStream == nil {
			return log.Errf(ctx, nil, "No replayer connection stream")
		}
		r, err := replayer.rpcStream.Recv()
		if err != nil {
			return log.Errf(ctx, err, "Replayer connection lost")
		}
		switch r.Res.(type) {
		case *replaysrv.ReplayResponse_PayloadRequest:
			if err := replayer.handlePayloadRequest(ctx, r.GetPayloadRequest().GetPayloadId()); err != nil {
				return log.Errf(ctx, err, "Handling replay payload request")
			}
		case *replaysrv.ReplayResponse_ResourceRequest:
			if err := replayer.handleResourceRequest(ctx, r.GetResourceRequest()); err != nil {
				return log.Errf(ctx, err, "Handling replay resource request")
			}
		case *replaysrv.ReplayResponse_CrashDump:
			if err := replayer.handleCrashDump(ctx, r.GetCrashDump()); err != nil {
				return log.Errf(ctx, err, "Handling replay crash dump")
			}
			// No valid replay response after crash dump.
			return log.Errf(ctx, nil, "Replay crash")
		case *replaysrv.ReplayResponse_PostData:
			if replayer.executor == nil {
				return log.Err(ctx, nil, "Got an out-of-band PostData response")
			}
			if err := replayer.executor.HandlePostData(ctx, r.GetPostData()); err != nil {
				return log.Errf(ctx, err, "Handling post data")
			}
		case *replaysrv.ReplayResponse_Notification:
			if replayer.executor == nil {
				// Ignore any out-of-band notifications (e.g. from the prewarm).
				log.W(ctx, "Got an out-of-band notification: %v", r.GetNotification())
			} else if err := replayer.executor.HandleNotification(ctx, r.GetNotification()); err != nil {
				return log.Errf(ctx, err, "Handling notification")
			}
		case *replaysrv.ReplayResponse_Finished:
			if replayer.executor == nil {
				log.I(ctx, "Got a replay finished response for an already finished replay.")
			} else if err := replayer.executor.HandleFinished(ctx, nil); err != nil {
				return log.Errf(ctx, err, "Handling finished")
			}
		case *replaysrv.ReplayResponse_FenceReadyRequest:
			if replayer.executor == nil {
				return log.Err(ctx, nil, "No replay executor to HandleFenceReadyRequest")
			}
			fenceReq := r.GetFenceReadyRequest()
			if err := replayer.executor.HandleFenceReadyRequest(ctx, fenceReq); err != nil {
				return log.Errf(ctx, err, "Handling replay fence ready request")
			}
			if err := replayer.sendFenceReady(ctx, fenceReq.GetId()); err != nil {
				return log.Errf(ctx, err, "connection SendFenceReady failed")
			}

		default:
			return log.Errf(ctx, nil, "Unhandled ReplayResponse type")
		}
	}
}

// sendFenceReady signals the device to continue a replay.
func (replayer *replayer) sendFenceReady(ctx context.Context, id uint32) error {
	fenceReadyReq := replaysrv.ReplayRequest{
		Req: &replaysrv.ReplayRequest_FenceReady{
			FenceReady: &replaysrv.FenceReady{
				Id: id,
			},
		},
	}
	err := replayer.rpcStream.Send(&fenceReadyReq)
	if err != nil {
		return log.Errf(ctx, err, "Sending replay fence %v ready", id)
	}
	return nil
}

// handleResourceRequest sends back the requested resources.
func (replayer *replayer) handleResourceRequest(ctx context.Context, req *gapir.ResourceRequest) error {
	ctx = status.Start(ctx, "Resources Request (count: %d)", len(req.GetIds()))
	defer status.Finish(ctx)

	ctx = log.Enter(ctx, "handleResourceRequest")

	// Process the request
	if req == nil {
		return log.Err(ctx, nil, "Cannot handle nil resource request")
	}
	ids := req.GetIds()
	totalExpectedSize := req.GetExpectedTotalSize()
	totalReturnedSize := uint64(0)
	response := make([]byte, 0, totalExpectedSize)
	db := database.Get(ctx)
	for _, idStr := range ids {
		rID, err := id.Parse(idStr)
		if err != nil {
			return log.Errf(ctx, err, "Failed to parse resource id: %v", idStr)
		}
		obj, err := db.Resolve(ctx, rID)
		if err != nil {
			return log.Errf(ctx, err, "Failed to parse resource id: %v", idStr)
		}
		objData := obj.([]byte)
		response = append(response, objData...)
		totalReturnedSize += uint64(len(objData))
	}
	if totalReturnedSize != totalExpectedSize {
		return log.Errf(ctx, nil, "Total resource size mismatch. expected: %v, got: %v", totalExpectedSize, totalReturnedSize)
	}

	// Send the resources
	resReq := replaysrv.ReplayRequest{
		Req: &replaysrv.ReplayRequest_Resources{
			Resources: &replaysrv.Resources{Data: response},
		},
	}
	if err := replayer.rpcStream.Send(&resReq); err != nil {
		return log.Err(ctx, err, "Sending resources")
	}
	return nil
}

// handleCrashDump uploads the received crash dump the crash tracking service.
func (replayer *replayer) handleCrashDump(ctx context.Context, dump *gapir.CrashDump) error {
	if dump == nil {
		return log.Err(ctx, nil, "Nil crash dump")
	}
	filepath := dump.GetFilepath()
	crashData := dump.GetCrashData()
	OS := replayer.device.Instance().GetConfiguration().GetOS()
	// TODO(baldwinn860): get the actual version from GAPIR in case it ever goes out of sync
	if res, err := reporting.ReportMinidump(reporting.Reporter{
		AppName:    "GAPIR",
		AppVersion: app.Version.String(),
		OSName:     OS.GetName(),
		OSVersion:  fmt.Sprintf("%v %v.%v.%v", OS.GetBuild(), OS.GetMajorVersion(), OS.GetMinorVersion(), OS.GetPointVersion()),
	}, filepath, crashData); err != nil {
		return log.Err(ctx, err, "Failed to report GAPIR crash")
	} else if res != "" {
		log.I(ctx, "Crash Report Uploaded; ID: %v", res)
		file.Remove(file.Abs(filepath))
	}
	return nil
}

// handlePayloadRequest sends back the requested payload.
func (replayer *replayer) handlePayloadRequest(ctx context.Context, payloadID string) error {
	ctx = status.Start(ctx, "Payload Request")
	defer status.Finish(ctx)

	pid, err := id.Parse(payloadID)
	if err != nil {
		return log.Errf(ctx, err, "Parsing payload ID")
	}
	boxed, err := database.Resolve(ctx, pid)
	if err != nil {
		return log.Errf(ctx, err, "Getting replay payload")
	}
	if payload, ok := boxed.(*gapir.Payload); ok {
		if replayer.conn == nil || replayer.rpcClient == nil {
			return log.Err(ctx, nil, "Gapir not connected")
		}
		if replayer.rpcStream == nil {
			return log.Err(ctx, nil, "Replay Communication not initiated")
		}
		payloadReq := replaysrv.ReplayRequest{
			Req: &replaysrv.ReplayRequest_Payload{
				Payload: payload,
			},
		}
		err := replayer.rpcStream.Send(&payloadReq)
		if err != nil {
			return log.Err(ctx, err, "Sending replay payload")
		}
		return nil
	}
	return log.Errf(ctx, err, "Payload type is unexpected: %T", boxed)
}

// attachAuthToken attaches authentication token to the context as metadata, if
// the authentication token is not empty, and returns the new context. If the
// authentication token is empty, returns the original context.
func attachAuthToken(ctx context.Context, authToken auth.Token) context.Context {
	if len(authToken) != 0 {
		return metadata.NewOutgoingContext(ctx,
			metadata.Pairs(gapirAuthTokenMetaDataName, string(authToken)))
	}
	return ctx
}
