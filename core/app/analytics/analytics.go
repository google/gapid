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

// +build analytics

// Package analytics implements methods for sending GAPID usage data to
// Google Analytics.
package analytics

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/gapid/core/net"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/device/host"
)

const (
	trackingID = "UA-106883190-3"
)

var (
	mutex sync.RWMutex
	send  sender
)

type sender interface {
	send(Payload)
	flush()
}

// Send queues the Payload p to be sent to the Google Analytics server if
// analytics is enabled with a prior call to Enable().
func Send(p Payload) {
	mutex.RLock()
	defer mutex.RUnlock()
	if send != nil {
		send.send(p)
	}
}

// Flush flushes any pending Payloads to be sent to the Google Analytics server.
func Flush() {
	mutex.RLock()
	defer mutex.RUnlock()
	if send != nil {
		send.flush()
	}
}

// AppVersion holds information about the currently running application and its
// version.
type AppVersion struct {
	Name, Build         string
	Major, Minor, Point int
}

// Enable turns on Google Analytics tracking functionality using the given
// clientID and version.
func Enable(ctx context.Context, clientID string, version AppVersion) {
	ua, hostOS, hostGPU := getUserAgentOSAndGPU(host.Instance(ctx).GetConfiguration(), version)
	endpoint := newBatchEndpoint(ua)
	encoder := newEncoder(clientID, hostOS, hostGPU, version)
	mutex.Lock()
	defer mutex.Unlock()
	send = newBatcher(endpoint, encoder)
}

// Disable turns off Google Analytics tracking.
func Disable() {
	mutex.Lock()
	send = nil
	mutex.Unlock()
}

func getOSAndGPU(cfg *device.Configuration) (os string, gpu string) {
	os, gpu = "<unknown>", "<unknown>"
	if o := cfg.GetOS(); o != nil {
		os = fmt.Sprintf("%v %v.%v.%v", o.Name, o.MajorVersion, o.MinorVersion, o.PointVersion)
	}
	if g := cfg.GetHardware().GetGPU().GetName(); g != "" {
		gpu = g
	}
	return os, gpu
}

func getUserAgentOSAndGPU(cfg *device.Configuration, version AppVersion) (string, string, string) {
	os, gpu := getOSAndGPU(cfg)
	ua := net.UserAgent(cfg, net.ApplicationInfo{
		Name:         version.Name,
		VersionMajor: version.Major,
		VersionMinor: version.Minor,
		VersionPoint: version.Point,
	})
	return ua, os, gpu
}
