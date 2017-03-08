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

package log

import (
	"context"
	"time"

	"github.com/google/gapid/core/context/keys"
)

type clockKeyTy string

const clockKey clockKeyTy = "log.clockKey"

// Clock is the interface implemented by types that tell the time.
type Clock interface {
	Time() time.Time
}

// PutClock returns a new context with the Clock assigned to w.
func PutClock(ctx context.Context, c Clock) context.Context {
	return keys.WithValue(ctx, clockKey, c)
}

// GetClock returns the Clock assigned to ctx.
func GetClock(ctx context.Context) Clock {
	out, _ := ctx.Value(clockKey).(Clock)
	return out
}

// FixedClock is a Clock that returns a fixed time.
type FixedClock time.Time

// Time returns the fixed clock time.
func (c FixedClock) Time() time.Time { return time.Time(c) }

var (
	// NoClock is a Clock that disables printing of the time.
	NoClock = FixedClock(time.Time{})
)
