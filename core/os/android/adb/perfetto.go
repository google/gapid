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
	"context"
	"encoding/base64"
	"io"
	"strings"

	perfetto_pb "perfetto/config"

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
)

const (
	perfettoProducerLaunchersDirectory = "/data/local/tmp/perfetto_producer_launchers"
)

func (b *binding) preparePerfettoProducerLauncherFromApk(ctx context.Context, packageName string, launcherBinary string) (string, error) {
	if b.Instance().GetConfiguration().GetOS().GetAPIVersion() < 29 {
		return "", log.Errf(ctx, nil, "Querying perfetto capability requires Android API >= 29")
	}
	launcherPath := perfettoProducerLaunchersDirectory + "/" + launcherBinary
	if _, err := b.Shell("rm", "-f", launcherPath).Call(ctx); err != nil {
		return "", log.Errf(ctx, err, "Can't clean up existing perfetto producer launcher %v", launcherBinary)
	}

	// Attempt to create the directory
	b.Shell("mkdir", "-p", perfettoProducerLaunchersDirectory).Call(ctx)
	res, err := b.Shell("pm", "path", packageName).Call(ctx)
	if err != nil {
		return "", log.Errf(ctx, err, "Failed to query path to apk %v", packageName)
	}
	packagePath := strings.Split(res, ":")[1]
	if _, err := b.Shell("unzip", "-o", packagePath, "assets/"+launcherBinary, "-d", perfettoProducerLaunchersDirectory).Call(ctx); err != nil {
		return "", log.Errf(ctx, err, "Failed to unzip %v from %v", launcherBinary, packageName)
	}

	// unzip also creates the directory structure, clean it up.
	b.Shell("mv", perfettoProducerLaunchersDirectory+"/assets/"+launcherBinary, perfettoProducerLaunchersDirectory).Call(ctx)
	b.Shell("rm", "-rf", perfettoProducerLaunchersDirectory+"/assets").Call(ctx)

	// Finally, make sure the binary is executable
	b.Shell("chmod", "a+x", launcherPath).Call(ctx)
	return launcherPath, nil
}

func (b *binding) LaunchPerfettoProducerFromApk(ctx context.Context, packageName string, launcherBinary string, started chan int) error {
	// Firstly, extract the producer launcher from Apk.
	binaryPath, err := b.preparePerfettoProducerLauncherFromApk(ctx, packageName, launcherBinary)
	if err != nil {
		return err
	}

	// Construct IO pipe, shell command outputs to stdout, GAPID reads from
	// reader for logging purpose.
	reader, stdout := io.Pipe()
	fail := make(chan error, 1)
	crash.Go(func() {
		buf := bufio.NewReader(reader)
		for {
			line, e := buf.ReadString('\n')
			// As long as there's output, consider the binary starting running.
			started <- 1
			switch e {
			default:
				log.E(ctx, "[launch producer] Read error %v", e)
				fail <- e
				return
			case io.EOF:
				fail <- nil
				return
			case nil:
				log.E(ctx, "[launch producer] %s", strings.TrimSuffix(line, "\n"))
			}
		}
	})

	// Start the shell command to launch producer
	process, err := b.Shell(binaryPath).Capture(stdout, stdout).Start(ctx)
	if err != nil {
		stdout.Close()
		return err
	}
	wait := make(chan error, 1)
	crash.Go(func() {
		wait <- process.Wait(ctx)
	})

	// Wait until either an error or EOF is read, or shell command exits.
	select {
	case err = <-fail:
		return err
	case err = <-wait:
		// Do nothing.
	}
	stdout.Close()
	if err != nil {
		return err
	}
	return <-fail
}

// StartPerfettoTrace starts a perfetto trace on this device.
func (b *binding) StartPerfettoTrace(ctx context.Context, config *perfetto_pb.TraceConfig, out string, stop task.Signal) error {
	reader, stdout := io.Pipe()
	data, err := proto.Marshal(config)
	if err != nil {
		return err
	}

	fail := make(chan error, 1)
	crash.Go(func() {
		buf := bufio.NewReader(reader)
		for {
			line, e := buf.ReadString('\n')
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
		// Do nothing.
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
