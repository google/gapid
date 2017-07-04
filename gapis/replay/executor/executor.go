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
	"bytes"
	"context"
	"fmt"
	"io"

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
	connection   io.ReadWriteCloser
	memoryLayout *device.MemoryLayout
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
	memoryLayout *device.MemoryLayout) error {

	return executor{
		payload:      payload,
		decoder:      decoder,
		connection:   connection,
		memoryLayout: memoryLayout,
	}.execute(ctx)
}

func (r executor) execute(ctx context.Context) error {
	// Encode the payload
	// TODO: Make this a proto.
	buf := &bytes.Buffer{}
	w := endian.Writer(buf, r.memoryLayout.GetEndian())
	w.Uint32(r.payload.StackSize)
	w.Uint32(r.payload.VolatileMemorySize)
	w.Uint32(uint32(len(r.payload.Constants)))
	w.Data(r.payload.Constants)
	w.Uint32(uint32(len(r.payload.Resources)))
	for _, r := range r.payload.Resources {
		w.String(r.ID)
		w.Uint32(r.Size)
	}
	w.Uint32(uint32(len(r.payload.Opcodes)))
	w.Data(r.payload.Opcodes)

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
		err := r.handleReplayCommunication(ctx, id, uint32(len(data)), responseW)
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
	r.decoder(responseR, nil)

	err = <-comErr
	if closeErr := responseR.Close(); closeErr != nil {
		log.W(ctx, "Replay execute pipe reader Close failed: %v", closeErr)
	}
	if err != nil {
		return log.Err(ctx, err, "Communicating with gapir")
	}
	return nil
}

func (r executor) handleReplayCommunication(ctx context.Context, replayID id.ID, replaySize uint32, postbacks io.WriteCloser) error {
	connection := r.connection
	defer connection.Close()
	e := endian.Writer(connection, r.memoryLayout.GetEndian())
	d := endian.Reader(connection, r.memoryLayout.GetEndian())

	e.Uint8(uint8(protocol.ConnectionType_Replay))
	e.String(replayID.String())
	e.Uint32(replaySize)
	if e.Error() != nil {
		return e.Error()
	}

	for {
		msg := d.Uint8()
		switch {
		case d.Error() == io.EOF:
			return nil
		case d.Error() != nil:
			return d.Error()
		}

		switch protocol.MessageType(msg) {
		case protocol.MessageType_Get:
			if err := r.handleGetData(ctx); err != nil {
				return fmt.Errorf("Failed to read replay postback data: %v", err)
			}
		case protocol.MessageType_Post:
			if err := r.handleDataResponse(ctx, postbacks); err != nil {
				return fmt.Errorf("Failed to send replay resource data: %v", err)
			}
		default:
			return fmt.Errorf("Unknown message type: %v\n", msg)
		}
	}
}

func (r executor) handleDataResponse(ctx context.Context, postbacks io.Writer) error {
	d := endian.Reader(r.connection, r.memoryLayout.GetEndian())

	n := d.Uint32()
	if d.Error() != nil {
		return d.Error()
	}

	c, err := io.CopyN(postbacks, r.connection, int64(n))
	if c != int64(n) {
		return err
	}

	return nil
}

func (r executor) handleGetData(ctx context.Context) error {
	ctx = log.Enter(ctx, "handleGetData")
	d := endian.Reader(r.connection, r.memoryLayout.GetEndian())

	resourceCount := d.Uint32()
	if err := d.Error(); err != nil {
		return log.Err(ctx, err, "Failed to decode resource count")
	}

	totalExpectedSize := d.Uint64()
	if err := d.Error(); err != nil {
		return log.Err(ctx, err, "Failed to decode total expected size")
	}

	resourceIDs := make([]id.ID, resourceCount)
	for i := range resourceIDs {
		idString := d.String()
		if d.Error() != nil {
			return d.Error()
		}
		var err error
		resourceIDs[i], err = id.Parse(idString)
		if err != nil {
			return log.Errf(ctx, err, "Failed to parse resource id: %v", idString)
		}
	}

	totalReturnedSize := uint64(0)
	for _, rid := range resourceIDs {
		obj, err := database.Resolve(ctx, rid)
		if err != nil {
			return log.Errf(ctx, err, "Failed to resolve resource with id: %v", rid)
		}

		data := obj.([]byte)
		n, err := r.connection.Write(data)
		if err != nil {
			return log.Errf(ctx, err, "Failed to send resource with id: %v", rid)
		}

		totalReturnedSize += uint64(n)
	}

	if totalExpectedSize != totalReturnedSize {
		return log.Errf(ctx, nil, "Total resources size mismatch. expected: %v, got: %v",
			totalExpectedSize, totalReturnedSize)
	}
	return nil
}
