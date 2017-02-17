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

// Package log provides a logging system that works well with context
// It tries to make sure that disabled logging statements are cheap enough that they can be left in the code, and
// enabled at run time.
// It does this by reducing the need for memory allocations (to nothing for the common level filter) and by using a
// messaging formatting system that delays all costs until the message is actually being logged.
// It stores all parameters to the logging line into the context, and the message system allows extraction of any
// context value into the message.
//
// Basic usage is
// ctx.Info().WithValue("myName", "George").Log("does lots of logging");
// |------------| this gets a Logger, and filters based on the severity
//              |-----------------------------| this can be repeated, it adds a new value to the context
//                                            |-----------------------| sends a record to the handler
// To control the filtering, use log.SetFilter, to control the destination log.SetHandler, and to control the format
// log.SetStyle.
package log

// binary: java.source = service
// binary: java.package = com.google.gapid.service.log
// binary: java.indent = "  "
// binary: java.member_prefix = my
