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

// Package client provides helper methods and types for starting GAPIR
// instances and communicating with them. The actual interaction is
// achieved via the gRPC service defined in:
// gapir/replay_service/service.proto
// The client package abstracts the RPC handling details and offers a
// higher-level interface, in particular it can manage several live
// GAPIR instances at the same time.
package client
