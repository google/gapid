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
	"context"
	"fmt"
	"io"
	"net"
	"reflect"
	"time"

	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/context/keys"
	"github.com/google/gapid/core/event/task"
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
func waitForOnCreate(ctx context.Context, conn *jdwp.Connection, wakeup jdwp.ThreadID) (*jdwp.EventMethodEntry, error) {
	app, err := conn.GetClassBySignature("Landroid/app/Application;")
	if err != nil {
		return nil, err
	}

	constructor, err := conn.GetClassMethod(app.ClassID(), "<init>", "()V")
	if err != nil {
		return nil, err
	}

	log.I(ctx, "   Waiting for Application.<init>()")
	initEntry, err := conn.WaitForMethodEntry(ctx, app.ClassID(), constructor.ID, wakeup)
	if err != nil {
		return nil, err
	}

	var entry *jdwp.EventMethodEntry
	err = jdbg.Do(conn, initEntry.Thread, func(j *jdbg.JDbg) error {
		class := j.This().Call("getClass").AsType().(*jdbg.Class)
		var onCreate jdwp.Method
		for class != nil {
			var err error
			onCreate, err = conn.GetClassMethod(class.ID(), "onCreate", "()V")
			if err == nil {
				break
			}
			class = class.Super()
		}
		if class == nil {
			return fmt.Errorf("Couldn't find Application.onCreate")
		}
		log.I(ctx, "   Waiting for %v.onCreate()", class.String())
		out, err := conn.WaitForMethodEntry(ctx, class.ID(), onCreate.ID, initEntry.Thread)
		if err != nil {
			return err
		}
		entry = out
		return nil
	})
	if err != nil {
		return nil, err
	}

	return entry, nil
}

// waitForVulkanLoad for android.app.ApplicationLoaders.getClassLoader to be called,
// and then suspends the thread.
// This function is what is used to tell the vulkan loader where to search for
// layers.
func waitForVulkanLoad(ctx context.Context, conn *jdwp.Connection) (*jdwp.EventMethodEntry, error) {
	loaders, err := conn.GetClassBySignature("Landroid/app/ApplicationLoaders;")
	if err != nil {
		return nil, err
	}

	getClassLoader, err := conn.GetClassMethod(loaders.ClassID(), "getClassLoader",
		"(Ljava/lang/String;IZLjava/lang/String;Ljava/lang/String;Ljava/lang/ClassLoader;)Ljava/lang/ClassLoader;")
	if err != nil {
		getClassLoader, err = conn.GetClassMethod(loaders.ClassID(), "getClassLoader",
			"(Ljava/lang/String;IZLjava/lang/String;Ljava/lang/String;Ljava/lang/ClassLoader;Ljava/lang/String;)Ljava/lang/ClassLoader;")
		if err != nil {
			return nil, err
		}
	}

	return conn.WaitForMethodEntry(ctx, loaders.ClassID(), getClassLoader.ID, 0)
}

// loadAndConnectViaJDWP connects to the application waiting for a JDWP
// connection with the specified process id, sends a number of JDWP commands to
// load the list of libraries.
func (p *Process) loadAndConnectViaJDWP(
	ctx context.Context,
	gapidAPK *gapidapk.APK,
	pid int,
	d adb.Device) error {

	const (
		reconnectAttempts = 10
		reconnectDelay    = time.Second
	)

	jdwpPort, err := adb.LocalFreeTCPPort()
	if err != nil {
		return log.Err(ctx, err, "Finding free port")
	}
	ctx = log.V{"jdwpPort": jdwpPort}.Bind(ctx)

	log.I(ctx, "Forwarding TCP port %v -> JDWP pid %v", jdwpPort, pid)
	if err := d.Forward(ctx, adb.TCPPort(jdwpPort), adb.Jdwp(pid)); err != nil {
		return log.Err(ctx, err, "Setting up JDWP port forwarding")
	}
	defer func() {
		// Clone context to ignore cancellation.
		ctx := keys.Clone(context.Background(), ctx)
		d.RemoveForward(ctx, adb.TCPPort(jdwpPort))
	}()

	ctx, stop := task.WithCancel(ctx)
	defer stop()

	log.I(ctx, "Connecting to JDWP")

	// Create a JDWP connection with the application.
	var sock net.Conn
	var conn *jdwp.Connection
	err = task.Retry(ctx, reconnectAttempts, reconnectDelay, func(ctx context.Context) (bool, error) {
		if sock, err = net.Dial("tcp", fmt.Sprintf("localhost:%v", jdwpPort)); err != nil {
			return false, err
		}
		if conn, err = jdwp.Open(ctx, sock); err != nil {
			sock.Close()
			log.I(ctx, "Failed to connect to the application: %v. Retrying...", err)
			return false, err
		}
		return true, nil
	})
	if err != nil {
		if err == io.EOF {
			return fmt.Errorf("Unable to connect to the application.\n\n" +
				"This can happen when another debugger or IDE is running " +
				"in the background, such as Android Studio.\n" +
				"Please close any running Android debuggers and try again.\n\n" +
				"See https://github.com/google/gapid/issues/911 for more " +
				"information")
		}
		return log.Err(ctx, err, "Connecting to JDWP")
	}
	defer sock.Close()

	processABI := func(j *jdbg.JDbg) (*device.ABI, error) {
		abiName := j.Class("android.os.Build").Field("CPU_ABI").Get().(string)
		abi := device.ABIByName(abiName)
		if abi == nil {
			return nil, fmt.Errorf("Unknown ABI %v", abiName)
		}

		// For NativeBridge emulated devices opt for the native ABI of the
		// emulator.
		abi = d.NativeBridgeABI(ctx, abi)

		return abi, nil
	}

	classLoaderThread := jdwp.ThreadID(0)

	log.I(ctx, "Waiting for ApplicationLoaders.getClassLoader()")
	getClassLoader, err := waitForVulkanLoad(ctx, conn)
	if err == nil {
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
			return nil
		})
		if err != nil {
			return log.Err(ctx, err, "JDWP failure")
		}
	} else {
		log.W(ctx, "Couldn't break in ApplicationLoaders.getClassLoader. Vulkan will not be supported.")
	}

	// Wait for Application.onCreate to be called.
	log.I(ctx, "Waiting for Application Creation")
	onCreate, err := waitForOnCreate(ctx, conn, classLoaderThread)
	if err != nil {
		return log.Err(ctx, err, "Waiting for Application Creation")
	}

	// Attempt to get the GVR library handle.
	// Will throw an exception for non-GVR apps.
	var gvrHandle uint64
	log.I(ctx, "Installing interceptor libraries")
	loadNativeGvrLibrary, vrCoreLibraryLoader := "loadNativeGvrLibrary", "com/google/vr/cardboard/VrCoreLibraryLoader"
	gvrMajor, gvrMinor, gvrPoint := 1, 8, 1

	getGVRHandle := func(j *jdbg.JDbg, libLoader jdbg.Type) error {
		// loadNativeGvrLibrary has a couple of different signatures depending
		// on GVR release.
		for _, f := range []func() error{
			// loadNativeGvrLibrary(Context, int major, int minor, int point)
			func() error {
				gvrHandle = (uint64)(libLoader.Call(loadNativeGvrLibrary, j.This(), gvrMajor, gvrMinor, gvrPoint).Get().(int64))
				return nil
			},
			// loadNativeGvrLibrary(Context)
			func() error {
				gvrHandle = (uint64)(libLoader.Call(loadNativeGvrLibrary, j.This()).Get().(int64))
				return nil
			},
		} {
			if jdbg.Try(f) == nil {
				return nil
			}
		}
		return fmt.Errorf("Couldn't call loadNativeGvrLibrary")
	}
	for _, f := range []func(j *jdbg.JDbg) error{
		func(j *jdbg.JDbg) error {
			libLoader := j.Class(vrCoreLibraryLoader)
			getGVRHandle(j, libLoader)
			return nil
		},
		func(j *jdbg.JDbg) error {
			classLoader := j.This().Call("getClassLoader")
			libLoader := classLoader.Call("findClass", vrCoreLibraryLoader).AsType()
			getGVRHandle(j, libLoader)
			return nil
		},
	} {
		if err := jdbg.Do(conn, onCreate.Thread, f); err == nil {
			break
		}
	}
	if gvrHandle == 0 {
		log.I(ctx, "GVR library not found")
	} else {
		log.I(ctx, "GVR library found")
	}

	// Connect to GAPII.
	// This has to be done on a separate go-routine as the call to load gapii
	// will block until a connection is made.
	connErr := make(chan error)

	// Load GAPII library.
	err = jdbg.Do(conn, onCreate.Thread, func(j *jdbg.JDbg) error {
		abi, err := processABI(j)
		if err != nil {
			return err
		}

		interceptorPath := gapidAPK.LibInterceptorPath(abi)
		crash.Go(func() { connErr <- p.connect(ctx, gvrHandle, interceptorPath) })

		gapiiPath := gapidAPK.LibGAPIIPath(abi)
		ctx = log.V{"gapii.so": gapiiPath, "process abi": abi.Name}.Bind(ctx)

		// Load the library.
		log.D(ctx, "Loading GAPII library...")
		// Work around for loading libraries in the N previews. See b/29441142.
		j.Class("java.lang.Runtime").Call("getRuntime").Call("doLoad", gapiiPath, nil)
		log.D(ctx, "Library loaded")
		return nil
	})
	if err != nil {
		return log.Err(ctx, err, "loadGAPII")
	}

	return <-connErr
}
