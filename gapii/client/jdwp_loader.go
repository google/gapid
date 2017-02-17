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

package client

import (
	"fmt"
	"io"
	"net"
	"reflect"
	"time"

	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/fault/cause"
	"github.com/google/gapid/core/java/jdbg"
	"github.com/google/gapid/core/java/jdwp"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/android/adb"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapidapk"
)

func expect(r io.Reader, expected []byte) error {
	got := make([]byte, len(expected))
	if _, err := io.ReadFull(r, got); err != nil {
		return err
	}
	if !reflect.DeepEqual(expected, got) {
		return fmt.Errorf("Expected %v, got %v", expected, got)
	}
	return nil
}

// waitForOnCreate waits for android.app.Application.onCreate to be called, and
// then suspends the thread.
func waitForOnCreate(ctx log.Context, conn *jdwp.Connection, wakeup jdwp.ThreadID) (*jdwp.EventMethodEntry, error) {
	app, err := conn.GetClassBySignature("Landroid/app/Application;")
	if err != nil {
		return nil, err
	}

	onCreate, err := conn.GetClassMethod(app.ClassID(), "onCreate", "()V")
	if err != nil {
		return nil, err
	}

	return conn.WaitForMethodEntry(ctx, app.ClassID(), onCreate.ID, wakeup)
}

// waitForVulkanLoad for android.app.ApplicationLoaders.getClassLoader to be called,
// and then suspends the thread.
// This function is what is used to tell the vulkan loader where to search for
// layers.
func waitForVulkanLoad(ctx log.Context, conn *jdwp.Connection) (*jdwp.EventMethodEntry, error) {
	loaders, err := conn.GetClassBySignature("Landroid/app/ApplicationLoaders;")
	if err != nil {
		return nil, err
	}

	getClassLoader, err := conn.GetClassMethod(loaders.ClassID(), "getClassLoader",
		"(Ljava/lang/String;IZLjava/lang/String;Ljava/lang/String;Ljava/lang/ClassLoader;)Ljava/lang/ClassLoader;")
	if err != nil {
		return nil, err
	}

	return conn.WaitForMethodEntry(ctx, loaders.ClassID(), getClassLoader.ID, 0)
}

// loadLibrariesViaJDWP connects to the application waiting for a JDWP
// connection with the specified process id, sends a number of JDWP commands to
// load the list of libraries.
func loadLibrariesViaJDWP(ctx log.Context, gapidAPK *gapidapk.APK, pid int, d adb.Device) error {
	const (
		reconnectAttempts = 10
		reconnectDelay    = time.Second
	)

	ctx, stop := task.WithCancel(ctx)
	defer stop()

	jdwpPort, err := adb.LocalFreeTCPPort()
	if err != nil {
		return cause.Explain(ctx, err, "Finding free port")
	}
	ctx = ctx.I("jdwpPort", int(jdwpPort))

	ctx.Print("Forwarding JDWP port")
	if err := d.Forward(ctx, adb.TCPPort(jdwpPort), adb.Jdwp(pid)); err != nil {
		return cause.Explain(ctx, err, "Setting up JDWP port forwarding")
	}
	defer d.RemoveForward(ctx, adb.TCPPort(jdwpPort))

	ctx.Print("Connecting to JDWP")

	// Create a JDWP connection with the application.
	var sock net.Conn
	var conn *jdwp.Connection
	for i := 0; i < reconnectAttempts; i++ {
		if sock, err = net.Dial("tcp", fmt.Sprintf("localhost:%v", jdwpPort)); err == nil {
			if conn, err = jdwp.Open(ctx, sock); err == nil {
				break
			}
			sock.Close()
		}
		ctx.Printf("Failed to connect: %v", err)
		time.Sleep(reconnectDelay)
	}
	if err != nil {
		return cause.Explain(ctx, err, "Connecting to JDWP")
	}
	defer func() {
		stop()
		sock.Close()
	}()

	classLoaderThread := jdwp.ThreadID(0)

	processABI := func(j *jdbg.JDbg) (*device.ABI, error) {
		abiName := j.Class("android.os.Build").Field("CPU_ABI").Get().(string)
		abi := device.ABIByName(abiName)
		if abi == nil {
			return nil, fmt.Errorf("Unknown ABI %v", abiName)
		}
		return abi, nil
	}

	loadGAPII := func(j *jdbg.JDbg) error {
		abi, err := processABI(j)
		if err != nil {
			return err
		}
		interceptorPath := gapidAPK.LibInterceptorPath(abi)
		gapiiPath := gapidAPK.LibGAPIIPath(abi)
		ctx := ctx.S("gapii.so", gapiiPath).V("process abi", abi.Name)

		// Load the library.
		ctx.Debug().Log("Loading GAPII library...")
		// Work around for loading libraries in the N previews. See b/29441142.
		j.Class("java.lang.Runtime").Call("getRuntime").Call("doLoad", interceptorPath, nil)
		j.Class("java.lang.Runtime").Call("getRuntime").Call("doLoad", gapiiPath, nil)
		ctx.Debug().Log("Library loaded")
		return nil
	}

	ctx.Print("Waiting for ApplicationLoaders.getClassLoader()")
	if getClassLoader, err := waitForVulkanLoad(ctx, conn); err == nil {
		// If err != nil that means we could not find or break in getClassLoader
		// so we have no vulkan support.
		classLoaderThread = getClassLoader.Thread
		err = jdbg.Do(conn, getClassLoader.Thread, func(j *jdbg.JDbg) error {
			abi, err := processABI(j)
			if err != nil {
				return err
			}
			libsPath := gapidAPK.LibsPath(abi)
			newLibraryPath := j.String(":" + libsPath)
			obj := j.GetStackObject("librarySearchPath").Call("concat", newLibraryPath)
			j.SetStackObject("librarySearchPath", obj)
			// If successfully loaded vulkan support, then we should be good to go
			// load libgapii and friends here.
			return loadGAPII(j)
		})
		if err != nil {
			return cause.Explain(ctx, err, "JDWP failure")
		}
	}

	// If we did not have vulkan support, then we should try to load with
	// Application.onCreate().
	if classLoaderThread == jdwp.ThreadID(0) {
		// Wait for Application.onCreate to be called.
		ctx.Print("Waiting for Application.onCreate()")
		onCreate, err := waitForOnCreate(ctx, conn, classLoaderThread)
		if err != nil {
			return cause.Explain(ctx, err, "Waiting for Application.OnCreate")
		}

		// Create a JDbg session to install and load the libraries.
		ctx.Print("Installing interceptor libraries")
		if err := jdbg.Do(conn, onCreate.Thread, loadGAPII); err != nil {
			return cause.Explain(ctx, err, "JDWP failure")
		}
	}

	return nil
}
