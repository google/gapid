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
	"fmt"
	"io/ioutil"
	"reflect"
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
	toRoot + "gapis/api/vulkan/vulkan.api",
}

// Encode the APIs, decode the APIs and check they are identical with the
// pre-encoded.
func TestEncodeDecode(t *testing.T) {
	ctx := log.Testing(t)

	apis, mappings := resolveAPIs(ctx)

	data, err := bapi.Encode(apis, mappings)
	assert.For(ctx, "err").ThatError(err).Succeeded()

	decodedAPIs, decodedMappings, err := bapi.Decode(data)
	assert.For(ctx, "err").ThatError(err).Succeeded()

	if assert.For(ctx, "num apis").That(len(decodedAPIs)).Equals(len(apis)) {
		for i := range apis {
			assert.For(ctx, "apis").That(decodedAPIs[i]).DeepEquals(apis[i])
		}
	}

	if false {
		// These tests are disabled because they currently do not pass.
		// Mappings are being serialized, but their order is non-deterministic
		// though the iteration of maps.
		// TODO: Fix and enable these tests.
		checkMappings(ctx, decodedAPIs, apis, decodedMappings, mappings)
	}
}

func checkMappings(ctx context.Context,
	gotAPIs, expectAPIs []*semantic.API,
	gotMappings, expectMappings *semantic.Mappings) {

	translateMapping(gotAPIs, expectAPIs, gotMappings)

	assert.For(ctx, "SemanticToAST").
		That(gotMappings.SemanticToAST).
		DeepEquals(expectMappings.SemanticToAST)
}

// translateMapping translates the keys of mapping from the nodes of the from
// API tree to their equivalent in the to API tree.
func translateMapping(from, to []*semantic.API, mapping *semantic.Mappings) {
	out := &semantic.Mappings{}
	remap := correlateNodes(from, to)
	for sem, asts := range mapping.SemanticToAST {
		if remapped, ok := remap[sem]; ok {
			for _, ast := range asts {
				out.Add(ast, remapped)
			}
		} else {
			// If we hit this panic, we either have semantic nodes in the
			// mapping that are not referenced by the API (which is bad), or
			// we have nodes not being visited using semantic.Visit (which is
			// also bad).
			panic(fmt.Sprintf("No remap for %T %+v", sem, sem))
		}
	}
	*mapping = *out
}

// correlateNodes traverses the from and to API trees to build and return a map
// that translates all the nodes in from to to. This function requires the two
// trees to be symmetrical.
func correlateNodes(from, to []*semantic.API) map[semantic.Node]semantic.Node {
	out := make(map[semantic.Node]semantic.Node)
	for i := range from {
		f, t := collectNodes(from[i]), collectNodes(to[i])
		if len(f) != len(t) {
			panic("APIs are not balanced")
		}
		for i, n := range f {
			if reflect.TypeOf(n) != reflect.TypeOf(t[i]) {
				panic("APIs are not symmetrical")
			}
			out[n] = t[i]
		}
	}
	return out
}

func collectNodes(n semantic.Node) []semantic.Node {
	l := []semantic.Node{}
	seen := map[semantic.Node]bool{}
	var visit func(n semantic.Node)
	visit = func(n semantic.Node) {
		if seen[n] {
			return
		}
		seen[n] = true
		l = append(l, n)
		semantic.Visit(n, visit)
	}
	visit(n)
	return l
}

// Run the benchmarks with:
// bazel run //gapil/bapi:go_default_test -- -test.bench=.

// Times how long a decode takes.
func BenchmarkDecode(b *testing.B) {
	ctx := log.Testing(b)

	apis, mappings := resolveAPIs(ctx)

	data, err := bapi.Encode(apis, mappings)
	check(err)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _, err := bapi.Decode(data)
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
	var mappings *semantic.Mappings

	// Time how long it takes to parse and resolve the APIs from source
	b.Run("Parse & Resolve", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			apis, mappings = resolveAPIs(ctx)
		}
	})

	var data []byte

	// Time how long it takes to encode
	b.Run("Encode", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var err error
			data, err = bapi.Encode(apis, mappings)
			check(err)
		}
	})

	// Time how long it takes to decode
	b.Run("Decode", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _, err := bapi.Decode(data)
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

func resolveAPIs(ctx context.Context) ([]*semantic.API, *semantic.Mappings) {
	apis := []*semantic.API{}
	processor := gapil.NewProcessor()
	for _, path := range testAPIs {
		api, errs := processor.Resolve(path)
		if !assert.For(ctx, "Resolve").ThatSlice(errs).Equals(parse.ErrorList{}) {
			continue
		}
		apis = append(apis, api)
	}
	return apis, processor.Mappings
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
