// Copyright (C) 2018 Google Inc.
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

package bapi_test

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"context"
	"io/ioutil"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/text/parse"
	"github.com/google/gapid/gapil"
	"github.com/google/gapid/gapil/bapi"
	"github.com/google/gapid/gapil/semantic"
)

const toRoot = "../../"

var testAPIs = []string{
	toRoot + "gapis/api/gles/gles.api",
	toRoot + "gapis/api/gvr/gvr.api",
	toRoot + "gapis/api/vulkan/vulkan.api",
}

// Encode the APIs, decode the APIs and check they are identical with the
// pre-encoded.
func TestEncodeDecode(t *testing.T) {
	ctx := log.Testing(t)

	apis := resolveAPIs(ctx)

	data, err := bapi.Encode(apis)
	assert.For(ctx, "err").ThatError(err).Succeeded()

	decodedAPIs, err := bapi.Decode(data)
	assert.For(ctx, "err").ThatError(err).Succeeded()

	if assert.For(ctx, "num apis").That(len(decodedAPIs)).Equals(len(apis)) {
		for i := range apis {
			assert.For(ctx, "apis").That(decodedAPIs[i]).DeepEquals(apis[i])
		}
	}
}

// Run the benchmarks with:
// bazel run //gapil/bapi:go_default_test -- -test.bench=.

// Times how long a decode takes.
func BenchmarkDecode(b *testing.B) {
	ctx := log.Testing(b)

	apis := resolveAPIs(ctx)

	data, err := bapi.Encode(apis)
	check(err)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := bapi.Decode(data)
		check(err)
	}
}

// Times how long each of the following takes:
// * Parsing and resolving from source
// * Encoding the APIs
// * Decoding the APIs
// * Decompressing the encoded data with various compressors.
func BenchmarkApproaches(b *testing.B) {
	ctx := log.Testing(b)

	var apis []*semantic.API

	// Time how long it takes to parse and resolve the APIs from source
	b.Run("Parse & Resolve", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			apis = resolveAPIs(ctx)
		}
	})

	// Time how long it takes to encode
	b.Run("Encode", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := bapi.Encode(apis)
			check(err)
		}
	})

	data, err := bapi.Encode(apis)
	check(err)

	// Time how long it takes to decode
	b.Run("Decode", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := bapi.Decode(data)
			check(err)
		}
	})

	// Compress using different compressors and measure the decompression time
	for _, f := range compressors {
		b.Run(f.name, func(b *testing.B) {
			c := f.compress(data)
			b.Log("Compressed size: ", len(c))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				f.decompress(c)
			}
		})
	}
}

func resolveAPIs(ctx context.Context) []*semantic.API {
	apis := []*semantic.API{}
	processor := gapil.NewProcessor()
	for _, path := range testAPIs {
		api, errs := processor.Resolve(path)
		if !assert.For(ctx, "Resolve").ThatSlice(errs).Equals(parse.ErrorList{}) {
			continue
		}
		apis = append(apis, api)
	}
	return apis
}

type compressor struct {
	name       string
	compress   func([]byte) []byte
	decompress func([]byte) []byte
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

var compressors = []compressor{
	{
		"flate",
		func(data []byte) []byte {
			b := bytes.Buffer{}
			w, err := flate.NewWriter(&b, flate.BestCompression)
			check(err)
			_, err = w.Write(data)
			check(err)
			check(w.Close())
			return b.Bytes()
		},
		func(data []byte) []byte {
			out, err := ioutil.ReadAll(flate.NewReader(bytes.NewReader(data)))
			check(err)
			return out
		},
	},
	{
		"gzip",
		func(data []byte) []byte {
			b := bytes.Buffer{}
			w, err := gzip.NewWriterLevel(&b, gzip.BestCompression)
			check(err)
			_, err = w.Write(data)
			check(err)
			check(w.Close())
			return b.Bytes()
		},
		func(data []byte) []byte {
			r, err := gzip.NewReader(bytes.NewReader(data))
			check(err)
			out, err := ioutil.ReadAll(r)
			check(err)
			return out
		},
	},
	{
		"zlib",
		func(data []byte) []byte {
			b := bytes.Buffer{}
			w, err := zlib.NewWriterLevel(&b, zlib.BestCompression)
			check(err)
			_, err = w.Write(data)
			check(err)
			check(w.Close())
			return b.Bytes()
		},
		func(data []byte) []byte {
			r, err := zlib.NewReader(bytes.NewReader(data))
			check(err)
			out, err := ioutil.ReadAll(r)
			check(err)
			return out
		},
	},
}
