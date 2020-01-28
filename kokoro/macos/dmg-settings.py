# Copyright (C) 2017 Google Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# DMG configuration for dmgbuild.

format = 'UDZO'
files = [ 'AGI.app' ]
symlinks = { 'Applications': '/Applications' }
badge_icon = 'AGI.app/Contents/Resources/AGI.icns'
icon_locations = {
	'AGI.app': (120, 172),
	'Applications': (360, 172),
	'.background.tiff': (1000, 1000), # hide
	'.VolumeIcon.icns': (1000, 1000), # hide
}
background = 'background.png'
window_rect = ((100, 600), (480, 360))
icon_size = 64
