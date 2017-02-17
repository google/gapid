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

// Package monitor is a helper for keeping a local in memory representation
// of the key data from some of the robot services.
// It's purpose is to provice the main data source for packages like the web
// server and the scheduler.
// Most of the types are opaque about their contents to allow for lazy aquisition
// construction or lookup of their members.
// For instance, id's are normally resolved to objects for you, but that resolve may
// not happen until the member is asked for.
package monitor
