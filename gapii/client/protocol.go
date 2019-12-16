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

// This protocol is mirrored in gapii/cc/protocol.h

package client

import (
	"context"
	"io"
	"net"
	"sync/atomic"
	"time"
)

const messageHeaderSize uint = 6
const messageDataBytes uint = 5

type messageType byte

const (
	messageData       messageType = 0x00
	messageStartTrace messageType = 0x01
	messageEndTrace   messageType = 0x02
	messageError      messageType = 0x03
	messageInvalid    messageType = 0xff
)

var startTraceMessage [messageHeaderSize]byte = [messageHeaderSize]byte{byte(messageStartTrace)}
var endTraceMessage [messageHeaderSize]byte = [messageHeaderSize]byte{byte(messageEndTrace)}

func readHeader(conn net.Conn) (msgType messageType, dataSize uint64, err error) {
	msgType = messageInvalid
	now := time.Now()
	conn.SetReadDeadline(now.Add(time.Millisecond * 500)) // Allow for stop event and UI refreshes.
	buf := make([]byte, messageHeaderSize)
	if _, err = io.ReadFull(conn, buf); err == nil {
		// first header byte contains the message type
		msgType = messageType(buf[0])
		// next messageDataBytes contain the data size as little-endian unsigned integer
		for i := uint(0); i < messageDataBytes; i++ {
			dataSize += uint64(buf[i+1]) << (i * 8)
		}
	}
	return
}

func readData(ctx context.Context, conn net.Conn, dataSize uint64, w io.Writer, written *int64) (read siSize, err error) {
	const maxBufSize siSize = 64 * 1024
	for read < siSize(dataSize) {
		copyCount := siSize(dataSize) - read
		if copyCount > maxBufSize {
			copyCount = maxBufSize
		}
		now := time.Now()
		conn.SetReadDeadline(now.Add(time.Millisecond * 500)) // Allow for stop event and UI refreshes.
		writeCount, copyErr := io.CopyN(w, conn, int64(copyCount))
		read += siSize(writeCount)
		atomic.AddInt64(written, writeCount)
		if abort, abortErr := handleCommError(ctx, copyErr, true); abort {
			err = abortErr
			return
		}
	}
	return
}

func readError(conn net.Conn, dataSize uint64) (errorMsg string, err error) {
	now := time.Now()
	conn.SetReadDeadline(now.Add(time.Millisecond * 500)) // Allow for stop event and UI refreshes.
	buf := make([]byte, dataSize)
	if _, err = io.ReadFull(conn, buf); err == nil {
		errorMsg = string(buf)
	}
	return
}

func writeStartTrace(conn net.Conn) error {
	_, err := conn.Write(startTraceMessage[:])
	return err
}

func writeEndTrace(conn net.Conn) error {
	_, err := conn.Write(endTraceMessage[:])
	return err
}
