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

package resolve

import (
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

func errPathOOB(val uint64, name string, min, max uint64, p path.Node) error {
	return &service.ErrInvalidPath{
		Reason: messages.ErrValueOutOfBounds(val, name, min, max),
		Path:   p.Path(),
	}
}

func errPathSliceOOB(start, end, length uint64, p path.Node) error {
	return &service.ErrInvalidPath{
		Reason: messages.ErrSliceOutOfBounds(start, end, "Start", "End", uint64(0), length-1),
		Path:   p.Path(),
	}
}

func errPathNoCapture(p path.Node) error {
	return &service.ErrInvalidPath{
		Reason: messages.ErrPathWithoutCapture(),
		Path:   p.Path(),
	}
}
