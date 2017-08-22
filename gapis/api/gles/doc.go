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

// Package gles implementes the API interface for the OpenGL ES graphics library.
package gles

// The following are the imports that generated source files pull in when present
// Having these here helps out tools that can't cope with missing dependancies
import (
	_ "context"
	_ "fmt"
	_ "reflect"
	_ "sort"
	_ "strings"

	_ "github.com/golang/protobuf/proto"
	_ "github.com/google/gapid/core/data"
	_ "github.com/google/gapid/core/data/binary"
	_ "github.com/google/gapid/core/data/id"
	_ "github.com/google/gapid/core/data/protoconv"
	_ "github.com/google/gapid/core/event/task"
	_ "github.com/google/gapid/core/math/u64"
	_ "github.com/google/gapid/core/os/device"
	_ "github.com/google/gapid/gapil/constset"
	_ "github.com/google/gapid/gapis/api"
	_ "github.com/google/gapid/gapis/capture"
	_ "github.com/google/gapid/gapis/memory"
	_ "github.com/google/gapid/gapis/memory/memory_pb"
	_ "github.com/google/gapid/gapis/messages"
	_ "github.com/google/gapid/gapis/replay"
	_ "github.com/google/gapid/gapis/replay/builder"
	_ "github.com/google/gapid/gapis/replay/protocol"
	_ "github.com/google/gapid/gapis/replay/value"
	_ "github.com/google/gapid/gapis/service/path"
	_ "github.com/google/gapid/gapis/stringtable"
	_ "github.com/pkg/errors"
)
