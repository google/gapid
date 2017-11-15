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

// Package executor contains the Execute function for sending a replay to a device.
package executor

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/app/crash/reporting"
	"github.com/google/gapid/core/data/binary"
	"github.com/google/gapid/core/data/endian"
	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/replay/builder"
	"github.com/google/gapid/gapis/replay/protocol"
)

type executor struct {
	payload      protocol.Payload
	decoder      builder.ResponseDecoder
	memoryLayout *device.MemoryLayout
	OS           *device.OS
}

// Execute sends the replay payload for execution on the target replay device
// communicating on connection.
// decoder will be used for decoding all postback reponses. Once a postback
// response is decoded, the corresponding handler in the handlers map will be
// called.
func Execute(
	ctx context.Context,
	payload protocol.Payload,
	decoder builder.ResponseDecoder,
	connection io.ReadWriteCloser,
	memoryLayout *device.MemoryLayout,
	os *device.OS) error {

	// The memoryLayout is specific to the ABI of the requested capture,
	// while the OS is not. Thus a device.Configuration is not applicable here.
	return executor{
		payload:      payload,
		decoder:      decoder,
		memoryLayout: memoryLayout,
		OS:           os,
	}.execute(ctx, connection)
}

func (e executor) execute(ctx context.Context, connection io.ReadWriteCloser) error {
	// Encode the payload
	// TODO: Make this a proto.
	buf := &bytes.Buffer{}
	w := endian.Writer(buf, e.memoryLayout.GetEndian())
	w.Uint32(e.payload.StackSize)
	w.Uint32(e.payload.VolatileMemorySize)
	w.Uint32(uint32(len(e.payload.Constants)))
	w.Data(e.payload.Constants)
	w.Uint32(uint32(len(e.payload.Resources)))
	for _, r := range e.payload.Resources {
		w.String(r.ID)
		w.Uint32(r.Size)
	}
	w.Uint32(uint32(len(e.payload.Opcodes)))
	w.Data(e.payload.Opcodes)

	data := buf.Bytes()

	// Store the payload to the database
	id, err := database.Store(ctx, data)
	if err != nil {
		return err
	}

	// Kick the communication handler
	responseR, responseW := io.Pipe()
	comErr := make(chan error)
	go func() {
		err := e.handleReplayCommunication(ctx, connection, id, uint32(len(data)), responseW)
		if err != nil {
			log.W(ctx, "Replay communication failed: %v", err)
			if closeErr := responseW.CloseWithError(err); closeErr != nil {
				log.W(ctx, "Replay execute pipe writer CloseWithError: %v", closeErr)
			}
		} else {
			if closeErr := responseW.Close(); closeErr != nil {
				log.W(ctx, "Replay execute pipe writer Close failed: %v", closeErr)
			}
		}
		comErr <- err
	}()

	// Decode and handle postbacks as they are received
	e.decoder(responseR, nil)

	err = <-comErr
	if closeErr := responseR.Close(); closeErr != nil {
		log.W(ctx, "Replay execute pipe reader Close failed: %v", closeErr)
	}
	if err != nil {
		return log.Err(ctx, err, "Communicating with gapir")
	}
	return nil
}

func (e executor) handleReplayCommunication(ctx context.Context, connection io.ReadWriteCloser, replayID id.ID, replaySize uint32, postbacks io.WriteCloser) error {
	defer connection.Close()
	bw := bufio.NewWriter(connection)
	br := bufio.NewReader(connection)
	w := endian.Writer(bw, e.memoryLayout.GetEndian())
	r := endian.Reader(br, e.memoryLayout.GetEndian())

	w.Uint8(uint8(protocol.ConnectionType_Replay))
	w.String(replayID.String())
	w.Uint32(replaySize)
	if err := w.Error(); err != nil {
		return err
	}
	if err := bw.Flush(); err != nil {
		return err
	}

	for {
		msg := r.Uint8()
		switch {
		case r.Error() == io.EOF:
			return nil
		case r.Error() != nil:
			return r.Error()
		}

		switch protocol.MessageType(msg) {
		case protocol.MessageType_Get:
			if err := e.handleGetData(ctx, r, w); err != nil {
				return fmt.Errorf("Failed to read replay postback data: %v", err)
			}
		case protocol.MessageType_Post:
			if err := e.handleDataResponse(ctx, r, postbacks); err != nil {
				return fmt.Errorf("Failed to send replay resource data: %v", err)
			}
		case protocol.MessageType_Crash:
			log.I(ctx, "Crash from GAPIR incoming")
			if err := e.handleCrash(ctx, r); err != nil {
				return fmt.Errorf("Failed to handle crash sent by gapir: %v", err)
			}
			// replay crashed, will we get an EOF next, or should we return here
		default:
			return fmt.Errorf("Unknown message type: %v", msg)
		}

		if err := bw.Flush(); err != nil {
			return err
		}
	}
}

func (e executor) handleCrash(ctx context.Context, r binary.Reader) error {
	filename := r.String()
	if r.Error() != nil {
		return r.Error()
	}

	n := r.Uint32()
	if r.Error() != nil {
		return r.Error()
	}

	crashData := make([]byte, n)
	r.Data(crashData)
	if r.Error() != nil {
		return r.Error()
	}

	// TODO(baldwinn860): get the actual version from GAPIR in case it ever goes out of sync
	if err := reporting.ReportMinidump(reporting.Reporter{
		AppName:    "GAPIR",
		AppVersion: app.Version.String(),
		OSName:     e.OS.GetName(),
		OSVersion:  fmt.Sprintf("%v %v.%v.%v", e.OS.GetBuild(), e.OS.GetMajor(), e.OS.GetMinor(), e.OS.GetPoint()),
	}, filename, crashData); err != nil {
		return log.Err(ctx, err, "Failed to report crash in GAPIR")
	}
	return nil
}

func (e executor) handleDataResponse(ctx context.Context, r binary.Reader, postbacks io.Writer) error {
	n := r.Uint32()
	if r.Error() != nil {
		return r.Error()
	}

	c, err := io.CopyN(postbacks, r, int64(n))
	if c != int64(n) {
		return err
	}

	return nil
}

func (e executor) handleGetData(ctx context.Context, r binary.Reader, w binary.Writer) error {
	ctx = log.Enter(ctx, "handleGetData")

	resourceCount := r.Uint32()
	if err := r.Error(); err != nil {
		return log.Err(ctx, err, "Failed to decode resource count")
	}

	totalExpectedSize := r.Uint64()
	if err := r.Error(); err != nil {
		return log.Err(ctx, err, "Failed to decode total expected size")
	}

	totalReturnedSize := 0

	response := make([][]byte, 0, resourceCount)
	db := database.Get(ctx)
	for i := uint32(0); i < resourceCount; i++ {
		idString := r.String()
		if r.Error() != nil {
			return r.Error()
		}

		rID, err := id.Parse(idString)
		if err != nil {
			return log.Errf(ctx, err, "Failed to parse resource id: %v", idString)
		}

		obj, err := db.Resolve(ctx, rID)
		if err != nil {
			return log.Errf(ctx, err, "Failed to resolve resource with id: %v", rID)
		}

		data := obj.([]byte)
		response = append(response, data)
		totalReturnedSize += len(data)
	}

	if totalExpectedSize != uint64(totalReturnedSize) {
		return log.Errf(ctx, nil, "Total resources size mismatch. expected: %v, got: %v",
			totalExpectedSize, totalReturnedSize)
	}

	for _, b := range response {
		w.Data(b)
	}
	if err := w.Error(); err != nil {
		return log.Errf(ctx, err, "Failed to send resources")
	}

	return nil
}
