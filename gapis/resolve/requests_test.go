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

package resolve

import "github.com/google/gapid/gapis/database"

// Interface compliance tests
var _ = []database.Resolvable{
	(*CommandTreeResolvable)(nil),
	(*ContextListResolvable)(nil),
	(*FollowResolvable)(nil),
	(*FramebufferAttachmentBytesResolvable)(nil),
	(*FramebufferAttachmentResolvable)(nil),
	(*FramebufferChangesResolvable)(nil),
	(*GetResolvable)(nil),
	(*GlobalStateResolvable)(nil),
	(*IndexLimitsResolvable)(nil),
	(*ReportResolvable)(nil),
	(*ResourceDataResolvable)(nil),
	(*ResourceMetaResolvable)(nil),
	(*ResourcesResolvable)(nil),
	(*SetResolvable)(nil),
	(*StateResolvable)(nil),
}
