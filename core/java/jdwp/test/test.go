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

package test

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/google/gapid/core/context/jot"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/java/jdwp"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/shell"
)

// findFreePort returns a currently free TCP port on the localhost.
// There are two potential issues with this function:
// * There is the potential for the port to be taken between the function
//   returning and the port actually being used.
// * The system _may_ hold on to the socket after it has been told to close.
// Because of these issues, there is a potential for flakiness.
func findFreePort() (int, error) {
	dummy, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer dummy.Close()
	return dummy.Addr().(*net.TCPAddr).Port, nil
}

// JavaSource is a map of file name 'foo.java' to the source code.
type JavaSource map[string]string

func runJavaServer(ctx log.Context, sources JavaSource, port int) task.Signal {
	signal, done := task.NewSignal()
	go func() {
		defer done(ctx)
		tmp, err := ioutil.TempDir("", "jdwp")
		if err != nil {
			jot.Fatal(ctx, err, "Couldn't get temporary directory")
			return
		}
		defer os.RemoveAll(tmp)
		sourceNames := make([]string, 0, len(sources))
		for name, source := range sources {
			if err := ioutil.WriteFile(filepath.Join(tmp, name), []byte(source), 0777); err != nil {
				jot.Fatal(ctx, err, "Couldn't write to temporary source file")
				return
			}
			sourceNames = append(sourceNames, name)
		}
		if err := shell.
			Command("javac", sourceNames...).
			In(tmp).
			Capture(nil, ctx.Error().Writer()).
			Run(ctx); err != nil {

			jot.Fatal(ctx, err, "Couldn't compile java source")
			return
		}
		agentlib := fmt.Sprintf("-agentlib:jdwp=transport=dt_socket,suspend=y,server=y,address=%v", port)
		if err := shell.
			Command("java", agentlib, "main").
			In(tmp).
			Capture(ctx.Info().Writer(), ctx.Error().Writer()).
			Run(ctx); err != nil {

			if !task.Stopped(ctx) {
				jot.Fatal(ctx, err, "Couldn't start java server")
			}
			return
		}
	}()
	return signal
}

// BuildRunAndConnect builds all the source files with javac, executes them with
// java, attaches to it via JDWP, then calls onConnect with the connection.
func BuildRunAndConnect(source JavaSource, onConnect func(ctx log.Context, conn *jdwp.Connection) int) int {
	ctx := log.Testing(nil)
	ctx, stop := task.WithTimeout(ctx, time.Second*30)

	port, err := findFreePort()
	if err != nil {
		jot.Fatal(ctx, nil, "Failed to find a free port")
		return -1
	}

	done := runJavaServer(ctx, source, port)
	defer func() {
		stop()
		<-done
	}()

	var socket io.ReadWriteCloser
	for i := 0; i < 5; i++ {
		var err error
		if socket, err = net.Dial("tcp", fmt.Sprintf("localhost:%v", port)); err == nil {
			break
		}
		time.Sleep(time.Second)
	}

	if socket == nil {
		jot.Fatal(ctx, nil, "Failed to connect to the socket")
		return -1
	}

	conn, err := jdwp.Open(ctx, socket)
	if err != nil {
		jot.Fatal(ctx, err, "Failed to open the JDWP connection")
		return -1
	}

	return onConnect(ctx, conn)
}
