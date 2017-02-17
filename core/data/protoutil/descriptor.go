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

package protoutil

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
)

type (
	// Described is the interface to something that self describes with a compressed
	// FileDescriptorProto.
	Described interface {
		Descriptor() ([]byte, []int)
	}
)

var (
	fileCache = map[*byte]*descriptor.FileDescriptorProto{}
	cacheLock = sync.Mutex{}
)

func GetFileDescriptor(data []byte) (*descriptor.FileDescriptorProto, error) {
	if len(data) == 0 {
		return nil, nil
	}
	cacheLock.Lock()
	defer cacheLock.Unlock()

	if d, found := fileCache[&data[0]]; found {
		return d, nil
	}
	d, err := decodeFileDescriptor(data)
	if err != nil {
		return nil, err
	}
	fileCache[&data[0]] = d
	return d, nil
}

func decodeFileDescriptor(data []byte) (*descriptor.FileDescriptorProto, error) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	fd := &descriptor.FileDescriptorProto{}
	if err := proto.Unmarshal(b, fd); err != nil {
		return nil, err
	}
	return fd, nil
}

// DescriptorOf returns the descriptor for a given proto message.
func DescriptorOf(msg Described) (*descriptor.DescriptorProto, error) {
	data, path := msg.Descriptor()
	fileDescriptor, err := GetFileDescriptor(data)
	if err != nil {
		return nil, err
	}
	messageDescriptor := fileDescriptor.MessageType[path[0]]
	for _, i := range path[1:] {
		messageDescriptor = messageDescriptor.NestedType[i]
	}
	return messageDescriptor, nil
}
