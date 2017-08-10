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

package pack

import (
	"context"

	"github.com/golang/protobuf/proto"
)

// Events describes the events used to construct groups and objects that are
// stored in a proto-pack stream.
type Events interface {
	// BeginGroup is called to start a new root group with the given identifier.
	BeginGroup(ctx context.Context, msg proto.Message, id uint64) error

	// BeginChildGroup is called to start a new group with the given identifier
	// as a child of the group with the parent identifier.
	BeginChildGroup(ctx context.Context, msg proto.Message, id, parentID uint64) error

	// EndGroup finalizes the group with the given identifier. It is illegal to
	// attempt to add children to the group after this is called.
	// EndGroup should be closed immediately after the last child has been added
	// to the group.
	EndGroup(ctx context.Context, id uint64) error

	// Object is called to declare an object outside of any group.
	Object(ctx context.Context, msg proto.Message) error

	// ChildObject is called to declare an object in the group with the given
	// identifier.
	ChildObject(ctx context.Context, msg proto.Message, parentID uint64) error
}
