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

// Package param holds Google Analytics parameter names.
package param

// Parameter represents a Google Analytics parameter name.
type Parameter string

const (
	ApplicationName      Parameter = "an"
	ApplicationVersion   Parameter = "av"
	ClientID             Parameter = "cid"
	DataSource           Parameter = "ds"
	EventAction          Parameter = "ea"
	EventCategory        Parameter = "ec"
	EventLabel           Parameter = "el"
	EventValue           Parameter = "ev"
	ExceptionDescription Parameter = "exd"
	ExceptionFatal       Parameter = "exf"
	HitType              Parameter = "t"
	ProtocolVersion      Parameter = "v"
	SessionControl       Parameter = "sc"
	TimingCategory       Parameter = "utc"
	TimingDuration       Parameter = "utt"
	TimingLabel          Parameter = "utl"
	TimingName           Parameter = "utv"
	TrackingID           Parameter = "tid"

	// Custom dimensions

	HostOS    Parameter = "cd1"
	HostGPU   Parameter = "cd2"
	TargetOS  Parameter = "cd3"
	TargetGPU Parameter = "cd4"

	// Custom metrics

	Size  Parameter = "cm1"
	Count Parameter = "cm2"
)
