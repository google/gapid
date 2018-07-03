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

package vertex

import (
	"context"

	"github.com/google/gapid/core/stream"
)

type semanticKey struct {
	typ   Semantic_Type
	index uint32
}

func (s *Semantic) key() semanticKey {
	if s == nil {
		return semanticKey{}
	}
	return semanticKey{s.Type, s.Index}
}

// ConvertTo converts the vertex buffer to the requested format.
func (s *Buffer) ConvertTo(ctx context.Context, f *BufferFormat) (*Buffer, error) {
	streams := make(map[semanticKey]*Stream, len(s.Streams))
	for _, s := range s.Streams {
		streams[s.Semantic.key()] = s
	}
	out := &Buffer{Streams: make([]*Stream, 0, len(f.Streams))}
	for _, f := range f.Streams {
		if s, ok := streams[f.Semantic.key()]; ok {
			s, err := s.ConvertTo(ctx, f.Format)
			if err != nil {
				return nil, err
			}
			out.Streams = append(out.Streams, s)
		}
	}
	return out, nil
}

// ConvertTo converts the vertex stream to the requested format.
func (s *Stream) ConvertTo(ctx context.Context, f *stream.Format) (*Stream, error) {
	data, err := stream.Convert(f, s.Format, s.Data)
	if err != nil {
		return nil, err
	}
	out := *s
	out.Format = f
	out.Data = data
	return &out, nil
}
