// Copyright (C) 2019 Google Inc.
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

package adb

import (
	"bufio"
	"container/ring"
	"context"
	"encoding/base64"
	"errors"
	"io"
	"regexp"
	"strings"
	"time"

	perfetto_pb "protos/perfetto/config"

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
)

var (
	AnsiRegex = regexp.MustCompile("\u001b\\[\\d+m")
)

// StartPerfettoTrace starts a perfetto trace on this device.
func (b *binding) StartPerfettoTrace(ctx context.Context, config *perfetto_pb.TraceConfig, out string, stop task.Signal, ready task.Task) error {
	// Silently attempt to delete the old trace in case the user has changed
	// If the user has changed from shell => root, this is fine. For root => shell,
	// the root user will have still have to manually delete the old trace if it exists
	b.RemoveFile(ctx, out)

	reader, stdout := io.Pipe()
	logRing := ring.New(10)
	// TODO(apbodnar) Find a way to reliably know when Perfetto/producers are ready (b/147388497)
	readyOnce := task.Once(task.Delay(ready, 1000*time.Millisecond))
	data, err := proto.Marshal(config)
	if err != nil {
		return err
	}

	fail := make(chan error, 1)
	crash.Go(func() {
		buf := bufio.NewReader(reader)
		for {
			line, e := buf.ReadString('\n')
			// Remove ANSI color codes from perfetto output
			line = AnsiRegex.ReplaceAllString(line, "")
			readyOnce(ctx)
			logRing.Value = line
			logRing = logRing.Next()
			switch e {
			default:
				log.E(ctx, "[perfetto] Read error %v", e)
				fail <- e
				return
			case io.EOF:
				fail <- nil
				return
			case nil:
				log.I(ctx, "[perfetto] %s", strings.TrimSuffix(line, "\n"))
			}
		}
	})

	process, err := b.Shell("base64", "-d", "|", "perfetto", "-c", "-", "-o", out).
		Read(strings.NewReader(base64.StdEncoding.EncodeToString(data))).
		Capture(stdout, stdout).
		Start(ctx)
	if err != nil {
		stdout.Close()
		return err
	}

	wait := make(chan error, 1)
	crash.Go(func() {
		wait <- process.Wait(ctx)
	})

	select {
	case err = <-fail:
		return err
	case err = <-wait:
		// Use perfetto's error messaging instead of the default
		if err != nil {
			msg := ""
			logRing.Do(func(p interface{}) {
				if s, ok := p.(string); ok && len(s) > 0 {
					msg += "\n" + s
				}
			})

			if msg != "" {
				err = errors.New(err.Error() + "\nPerfetto Output:" + msg)
			}
		}
	case <-stop:
		// TODO: figure out why "killall -2 perfetto" doesn't work.
		var pid string
		if pid, err = b.Shell("pidof perfetto").Call(ctx); err != nil {
			break
		}
		if err = b.Shell("kill -2 " + pid).Run(ctx); err != nil {
			break
		}
		err = <-wait
	}

	stdout.Close()
	if err != nil {
		return err
	}
	return <-fail
}
