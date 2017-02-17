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

// Package all is used to import all known gfxapi APIs for their side effects.
package all

import (
	"github.com/google/gapid/framework/binary/registry"
	"github.com/google/gapid/framework/binary/schema"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/gfxapi/core"
	"github.com/google/gapid/gapis/gfxapi/gles"
	"github.com/google/gapid/gapis/gfxapi/vulkan"
)

var GraphicsNamespace = registry.NewNamespace()

func init() {
	GraphicsNamespace.Add((*atom.FramebufferObservation)(nil).Class())
	GraphicsNamespace.AddFallbacks(core.Namespace)
	GraphicsNamespace.AddFallbacks(gles.Namespace)
	GraphicsNamespace.AddFallbacks(vulkan.Namespace)

	// core types that were moved from gles.api to core.api:
	registry.Global.AddAlias(
		"core.Architecture{[]?,Uint32,Uint32,Uint32,Bool}",
		"gles.Architecture{[]?,Uint32,Uint32,Uint32,Bool}")
	registry.Global.AddAlias(
		"core.SwitchThread{[]?,Uint64}",
		"gles.SwitchThread{[]?,Uint64}")
}

func VisitConstantSets(visitor func(schema.ConstantSet)) {
	for _, c := range core.ConstantValues {
		visitor(c)
	}
	for _, c := range gles.ConstantValues {
		visitor(c)
	}
	for _, c := range vulkan.ConstantValues {
		visitor(c)
	}
}
