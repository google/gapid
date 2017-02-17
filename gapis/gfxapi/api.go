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

package gfxapi

import (
	"fmt"

	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/image"
)

// API is the common interface to a graphics programming api.
type API interface {
	// Name returns the official name of the api.
	Name() string

	// Index returns the API index.
	Index() uint8

	// ID returns the unique API identifier.
	ID() ID

	// GetFramebufferAttachmentInfo returns the width, height and format of the specified framebuffer attachment.
	GetFramebufferAttachmentInfo(state *State, attachment FramebufferAttachment) (width, height uint32, format *image.Format, err error)

	// Context returns the active context for the given state.
	Context(*State) Context
}

// ID is an API identifier
type ID id.ID

// APIObject is the interface implemented by types that belong to an API.
type APIObject interface {
	// API returns the API identifier that this type belongs to.
	API() API
}

// Char is the possibly signed but maybe unsigned C/C++ char.
type Char uint8

// CharToBytes changes the gfxapi.Char values to their byte[] representation.
func CharToBytes(ϟchars []Char) []byte {
	bytes := make([]byte, len(ϟchars))
	for i := range ϟchars {
		bytes[i] = byte(ϟchars[i])
	}
	return bytes
}

// IsValid returns true if the id is not the default zero value.
func (i ID) IsValid() bool { return id.ID(i).IsValid() }

var apis = map[ID]API{}
var indices = map[uint8]bool{}

// Register adds an api to the understood set.
// It is illegal to register the same name twice.
func Register(api API) {
	id := api.ID()
	if _, present := apis[id]; present {
		panic(fmt.Errorf("API %s registered more than once", id))
	}
	apis[id] = api

	index := api.Index()
	if _, present := indices[index]; present {
		panic(fmt.Errorf("API %s used an occupied index %d", id, index))
	}
	indices[index] = true
}

// Find looks up a graphics API by identifier.
// If the id has not been registered, it returns nil.
func Find(id ID) API {
	return apis[id]
}
