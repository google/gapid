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

// +build integration

package shadertools

// TODO(b/33192755): This is a workaround for the fact that simply running
// 'go test' doesn't correctly set library paths, resulting in cgo errors.
// A possibly better alternative would be to have a cmake target that runs
// integration tests, setting  up the environment beforehand. We also need
// to make sure that integration tests are  at least _compiled_ during the
// normal build process. As  we don't run them automatically, they tend to
// break arbitrarily during refactoring.

func ConvertGlsl(source string, option *Option) CodeWithDebugInfo { panic("not implemented") }
func DisassembleSpirvBinary(words []uint32) string                { panic("not implemented") }
func AssembleSpirvText(chars string) []uint32                     { panic("not implemented") }
