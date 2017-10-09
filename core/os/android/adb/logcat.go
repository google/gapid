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

package adb

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/os/android"
)

// "[ MM-DD HH:MM:SS.FFF  PID: TID P/TAG ]"
var logcatMsgRegex = regexp.MustCompile(`\[\s*([0-9]*)-([0-9]*)\s*([0-9]*):([0-9]*):([0-9]*).([0-9]*)\s*([0-9]*):\s*([0-9]*)\s*([VDIWEF])\/([^\s]*)\s*\]`)

func parseLogcatMsg(s string) (android.LogcatMessage, bool) {
	parts := logcatMsgRegex.FindStringSubmatch(s)
	if parts == nil {
		return android.LogcatMessage{}, false
	}
	month, _ := strconv.Atoi(parts[1])
	day, _ := strconv.Atoi(parts[2])
	hour, _ := strconv.Atoi(parts[3])
	minute, _ := strconv.Atoi(parts[4])
	second, _ := strconv.Atoi(parts[5])
	microseconds, _ := strconv.Atoi(parts[6])
	pid, _ := strconv.Atoi(parts[7])
	tid, _ := strconv.Atoi(parts[8])
	priority := parts[9][0]
	tag := parts[10]

	return android.LogcatMessage{
		Timestamp: time.Date(time.Now().Year(), time.Month(month), day, hour, minute, second, microseconds*1e6, time.Local),
		ProcessID: pid,
		ThreadID:  tid,
		Priority:  parseLogcatPriority(priority),
		Tag:       tag,
	}, true
}

func parseLogcatPriority(r byte) android.LogcatPriority {
	switch r {
	case 'V':
		return android.Verbose
	case 'D':
		return android.Debug
	case 'I':
		return android.Info
	case 'W':
		return android.Warning
	case 'E':
		return android.Error
	case 'F':
		return android.Fatal
	default:
		panic(fmt.Errorf("Invalid priority code '%v'", rune(r)))
	}
}

// Logcat writes all logcat messages reported by the device to the chan msgs,
// blocking until the context is stopped.
func (b *binding) Logcat(ctx context.Context, msgs chan<- android.LogcatMessage) error {
	reader, stdout := io.Pipe()
	buf := bufio.NewReader(reader)
	err := make(chan error, 1)
	// Start a go-routine for parsing logcat lines
	crash.Go(func() {
		msg, lines := (*android.LogcatMessage)(nil), []string{}

		// flush writes the current pending message to msgs, and clears the lines.
		flush := func() {
			if count := len(lines); msg != nil && count > 0 {
				if lines[count-1] == "" { // Messages are usually separated with a blank line
					lines = lines[:count-1]
				}
				msg.Message = strings.Join(lines, "\n")
				msgs <- *msg
				lines = lines[:0]
			}
		}

		// ensure any pending output is flushed on exit, and msgs is closes.
		defer func() {
			flush()
			close(msgs)
		}()

		for {
			line, e := buf.ReadString('\n')
			switch e {
			default:
				err <- e // Unexpected stream error
				return
			case io.EOF:
				err <- nil // ADB stopped
				return
			case nil:
				if m, ok := parseLogcatMsg(line); ok {
					flush()
					msg = &m
				} else if msg != nil {
					lines = append(lines, strings.TrimSuffix(line, "\n"))
				}
			}
		}
	})

	if err := b.Command("logcat", "-v", "long", "-T", "0").Capture(stdout, nil).Run(ctx); err != nil {
		stdout.Close()
		return err
	}

	stdout.Close()
	return <-err
}
