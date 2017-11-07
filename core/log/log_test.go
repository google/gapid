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

package log_test

import (
	"context"
	"time"

	"github.com/google/gapid/core/log"
)

var testClock log.Clock

func init() {
	t, err := time.Parse("Mon Jan _2 15:04:05.999 2006", "Mon Jan 22 12:34:56.789 2000")
	if err != nil {
		panic(err)
	}
	testClock = log.FixedClock(t)
}

type testMessage struct {
	msg      string
	args     []interface{}
	values   log.V
	severity log.Severity
	tag      string
	process  string

	raw      string
	brief    string
	normal   string
	detailed string
}

func (m testMessage) send(h log.Handler) {
	ctx := context.Background()
	ctx = log.PutHandler(ctx, h)
	ctx = log.PutTag(ctx, m.tag)
	ctx = log.PutClock(ctx, testClock)
	ctx = m.values.Bind(ctx)
	log.From(ctx).Logf(m.severity, false, m.msg, m.args...)
}

var testMessages = []testMessage{
	{
		msg:      "plain warning",
		severity: log.Warning,

		raw:      "plain warning",
		brief:    "W: plain warning",
		normal:   "12:34:56.789 W: plain warning",
		detailed: "12:34:56.789 Warning: plain warning",
	}, {
		msg:      "info with values",
		severity: log.Info,
		values:   log.V{"cat": "meow", "dog": "woof"},

		raw:      "info with values",
		brief:    "I: info with values",
		normal:   "12:34:56.789 I: info with values",
		detailed: "12:34:56.789 Info: info with values \n  cat: meow\n  dog: woof",
	},
}
