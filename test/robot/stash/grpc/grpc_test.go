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

package grpc_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"testing"
	"time"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/fault"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/net/grpcutil"
	"github.com/google/gapid/test/robot/stash"
	stashgrpc "github.com/google/gapid/test/robot/stash/grpc"
	"github.com/google/gapid/test/robot/stash/local"
	"google.golang.org/grpc"
)

type data struct {
	id    string
	bytes []byte
}

func checkReadFromStash(ctx context.Context, assert assert.Manager, rnd *rand.Rand, s *stash.Client, exp []data) {
	for _, expected := range exp {
		e, err := s.Lookup(ctx, expected.id)
		assert.For("lookup err").ThatError(err).Succeeded()
		assert.For("lookup e").That(e).IsNotNil()
		assert.For("lookup length").That(e.Length).Equals(int64(len(expected.bytes)))

		rs, err := s.Open(ctx, expected.id)
		assert.For("open").ThatError(err).Succeeded()

		data, err := ioutil.ReadAll(rs)
		assert.For("readall").ThatError(err).Succeeded()
		assert.For("readall").ThatSlice(data).Equals(expected.bytes)

		data, err = readInRandomOrder(rnd, rs, 60, 3)
		assert.For("read random").ThatError(err).Succeeded()
		assert.For("read random").ThatSlice(data).Equals(expected.bytes)

		data, err = s.Read(ctx, expected.id)
		assert.For("stash.Read").ThatError(err).Succeeded()
		assert.For("stash.Read").ThatSlice(data).Equals(expected.bytes)
	}
}

func uploadRandomDataToStash(ctx context.Context, assert assert.Manager, r *rand.Rand, s *stash.Client, lens []int) []data {
	out := make([]data, len(lens))
	for i, ln := range lens {
		bytes := randomBytes(r, ln)
		id, err := s.UploadBytes(ctx, stash.Upload{Name: []string{fmt.Sprintf("%d.bin", i)}, Type: []string{"application/binary"}}, bytes)
		assert.For("upload").ThatError(err).Succeeded()
		out[i] = data{id, bytes}
	}
	return out
}

func TestMemoryAndGrpcStash(t *testing.T) {
	assert := assert.Context(t)
	ctx := log.Testing(t)
	r := rand.New(rand.NewSource(12345))

	memStash := local.NewMemoryService()
	testData := uploadRandomDataToStash(ctx, assert, r, memStash, []int{10, 100, 100, 1, 100000, 100000, 0})
	assert.For("test data length").ThatInteger(len(testData)).IsAtLeast(1)
	checkReadFromStash(ctx, assert, r, memStash, testData)

	var srv *grpc.Server
	go grpcutil.ServeWithListener(
		ctx,
		grpcutil.NewPipeListener("pipe:stashgrpc"),
		func(ctx context.Context, listener net.Listener, server *grpc.Server) error {
			srv = server
			return stashgrpc.Serve(ctx, server, memStash)
		},
	)

	ok := fault.Const("ok")
	err := grpcutil.Client(ctx, "pipe:stashgrpc", func(ctx context.Context, cc *grpc.ClientConn) error {
		sc := stashgrpc.MustConnect(ctx, cc)
		// Make sure the in-memory stash is correctly exposed.
		checkReadFromStash(ctx, assert, r, sc, testData)
		// Make sure upload works through GRPC as well.
		testData = uploadRandomDataToStash(ctx, assert, r, sc, []int{17})
		checkReadFromStash(ctx, assert, r, sc, testData)
		return ok
	}, grpc.WithDialer(grpcutil.GetDialer(ctx)), grpc.WithTimeout(1*time.Second), grpc.WithInsecure())
	assert.For("").ThatError(err).Equals(ok)

	srv.GracefulStop()
}

func randomBytes(r *rand.Rand, length int) []byte {
	p := make([]byte, length)
	r.Read(p)
	return p
}

func TestReadInRandomOrder(t *testing.T) {
	assert := assert.Context(t)
	r := rand.New(rand.NewSource(12345))

	for _, c := range []struct {
		length         int
		sections       int
		maxConsecutive int
	}{
		{1000, 40, 5},
		{1000, 1, 1},
		{1000, 1, 30},
		{1000, 40, 1},
		{1, 1, 1},
	} {
		data := randomBytes(r, c.length)
		got, err := readInRandomOrder(r, bytes.NewReader(data), c.sections, c.maxConsecutive)
		assert.For("read random").ThatError(err).Succeeded()
		assert.For("read random").ThatSlice(got).Equals(data)
	}
}

// readInRandom order exercises an io.ReadSeeker by splitting it up in sections which are
// read in random order. Within a section, several consecutive reads may be performed.
// All types of seeking are used.
func readInRandomOrder(r *rand.Rand, rs io.ReadSeeker, sections int, maxConsecutiveReads int) ([]byte, error) {
	length, err := rs.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, err
	}
	var pos int64 = length

	data := make([]byte, length)

	for _, s := range randomSections(r, int(length), sections, maxConsecutiveReads) {
		switch r.Intn(3) {
		case 0:
			pos, err = rs.Seek(s.offset, io.SeekStart)
		case 1:
			pos, err = rs.Seek(s.offset-length, io.SeekEnd)
		case 2:
			pos, err = rs.Seek(s.offset-pos, io.SeekCurrent)
		}
		if err != nil {
			return nil, err
		}
		if pos != s.offset {
			return nil, errors.New("Incorrect offset returned")
		}
		for _, rd := range s.reads {
			n, err := io.ReadAtLeast(rs, data[int(pos):int(pos)+rd], rd)
			if err != nil {
				return nil, err
			}
			if n != rd {
				return nil, fmt.Errorf("Expected %d bytes to be read, got %d", rd, n)
			}
			pos += int64(n)
		}
	}
	return data, nil
}

type readSection struct {
	offset int64 // absolute section offset
	reads  []int // lengths of consecutive reads within section
}

func randomSections(r *rand.Rand, length int, n int, maxReadsPerSection int) []readSection {
	if length == 0 {
		return []readSection{{0, []int{0}}}
	}
	min := func(a, b int) int {
		if a < b {
			return a
		}
		return b
	}
	sections := make([]readSection, n)
	p := r.Perm(n)
	n = min(n, length)
	ch := randomChunks(r, length, n)
	offset := 0
	for i := 0; i < n; i++ {
		sections[p[i]] = readSection{
			offset: int64(offset),
			reads:  randomChunks(r, ch[i], min(ch[i], 1+r.Intn(maxReadsPerSection))),
		}
		offset += ch[i]
	}
	return sections
}

func randomChunks(r *rand.Rand, length int, n int) []int {
	chunks := make([]int, n)

	sum := 0
	for i := 0; i < n; i++ {
		sz := r.Intn(length / n)
		chunks[i] = sz
		sum += sz
	}
	delta := (length - sum) / n
	for i := 0; i < len(chunks); i++ {
		chunks[i] += delta
		sum += delta
	}
	chunks[r.Intn(len(chunks))] += (length - sum)
	return chunks
}
