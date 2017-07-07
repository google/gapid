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

package gles

import "github.com/google/gapid/gapis/api"

func isIssueWhitelisted(cmd api.Cmd, e error) bool {
	switch cmd.(type) {
	case *GlActiveTexture:
		// TODO: b/29446056 - Apps break replay by looping over all texture units.
		// Just ignore the replay errors for now. We can do something better later.
		return true
	case *GlInvalidateFramebuffer, *GlDiscardFramebufferEXT:
		if e, ok := e.(ErrUnexpectedDriverTraceError); ok {
			if e.DriverError == GLenum_GL_NONE && e.ExpectedError == GLenum_GL_INVALID_ENUM {
				// Bug in Nexus 6 driver (Adreno 420). See b/29124256 (QCOM) and b/29124194 (DEQP).
				return true
			}
		}
	}
	return false
}
