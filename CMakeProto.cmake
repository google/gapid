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

#The set of auto generated protoc rules

protoc_go("github.com/google/gapid/core/data/pack" "core/data/pack" "pack.proto")
protoc_cc("core/data/pack" "core/data/pack" "pack.proto")
protoc_go("github.com/google/gapid/core/data/pod" "core/data/pod" "pod.proto")
protoc_java("core/data/pod" "pod.proto" "com/google/gapid/proto/core/pod/Pod")
protoc_go("github.com/google/gapid/core/data/protoutil/testprotos" "core/data/protoutil/testprotos" "testprotos.proto")
protoc_go("github.com/google/gapid/core/image" "core/image" "image.proto")
protoc_java("core/image" "image.proto" "com/google/gapid/proto/image/Image")
protoc_go("github.com/google/gapid/core/log/log_pb" "core/log/log_pb" "log.proto")
protoc_java("core/log/log_pb" "log.proto" "com/google/gapid/proto/log/Log")
protoc_go("github.com/google/gapid/core/os/android/apk" "core/os/android/apk" "apk.proto")
protoc_go("github.com/google/gapid/core/os/android" "core/os/android" "keycodes.proto")
protoc_go("github.com/google/gapid/core/os/device/bind" "core/os/device/bind" "bind.proto")
protoc_go("github.com/google/gapid/core/os/device" "core/os/device" "device.proto")
protoc_java("core/os/device" "device.proto" "com/google/gapid/proto/device/Device")
protoc_cc("core/os/device" "core/os/device" "device.proto")
protoc_go("github.com/google/gapid/core/stream" "core/stream" "stream.proto")
protoc_java("core/stream" "stream.proto" "com/google/gapid/proto/stream/Stream")
protoc_go("github.com/google/gapid/gapidapk/pkginfo" "gapidapk/pkginfo" "pkginfo.proto")
protoc_java("gapidapk/pkginfo" "pkginfo.proto" "com/google/gapid/proto/pkginfo/PkgInfo")
protoc_go("github.com/google/gapid/gapis/capture" "gapis/capture" "capture.proto")
protoc_cc("gapis/capture" "gapis/capture" "capture.proto")
protoc_go("github.com/google/gapid/gapis/api" "gapis/api" "gfxtrace.proto;service.proto")
protoc_java("gapis/api" "service.proto" "com/google/gapid/proto/service/api/API")
protoc_cc("gapis/api" "gapis/api" "gfxtrace.proto")

protoc_go("github.com/google/gapid/gapis/api/gles/gles_pb" "gapis/api/gles/gles_pb" "api.proto;extras.proto")
protoc_cc("gapis/api/gles/gles_pb" "gapis/api/gles/gles_pb" "api.proto;extras.proto")
protoc_go("github.com/google/gapid/gapis/api/gles" "gapis/api/gles" "resolvables.proto")

protoc_go("github.com/google/gapid/gapis/api/gvr/gvr_pb" "gapis/api/gvr/gvr_pb" "api.proto")
protoc_cc("gapis/api/gvr/gvr_pb" "gapis/api/gvr/gvr_pb" "api.proto")
protoc_go("github.com/google/gapid/gapis/api/gvr" "gapis/api/gvr" "resolvables.proto")

protoc_go("github.com/google/gapid/gapis/api/vulkan/vulkan_pb" "gapis/api/vulkan/vulkan_pb" "api.proto")
protoc_cc("gapis/api/vulkan/vulkan_pb" "gapis/api/vulkan/vulkan_pb" "api.proto")
protoc_go("github.com/google/gapid/gapis/api/vulkan" "gapis/api/vulkan" "resolvables.proto")

protoc_go("github.com/google/gapid/gapis/api/test/test_pb" "gapis/api/test/test_pb" "api.proto")
protoc_cc("gapis/api/test/test_pb" "gapis/api/test/test_pb" "api.proto")

protoc_go("github.com/google/gapid/gapis/api/testcmd/test_pb" "gapis/api/testcmd/test_pb" "test.proto")

protoc_go("github.com/google/gapid/gapis/memory" "gapis/memory" "memory.proto")
protoc_java("gapis/memory" "memory.proto" "com/google/gapid/proto/service/memory/Memory")
protoc_cc("gapis/memory" "gapis/memory" "memory.proto")
protoc_go("github.com/google/gapid/gapis/memory/memory_pb" "gapis/memory/memory_pb" "memory.proto")
protoc_cc("gapis/memory/memory_pb" "gapis/memory/memory_pb" "memory.proto")
protoc_go("github.com/google/gapid/gapis/replay/protocol" "gapis/replay/protocol" "replay_protocol.proto")
protoc_go("github.com/google/gapid/gapis/replay" "gapis/replay" "replay.proto")
protoc_go("github.com/google/gapid/gapis/resolve" "gapis/resolve" "resolvables.proto")
protoc_go("github.com/google/gapid/gapis/resolve/dependencygraph" "gapis/resolve/dependencygraph" "resolvables.proto")
protoc_go("github.com/google/gapid/gapis/service/path" "gapis/service/path" "path.proto")
protoc_java("gapis/service/path" "path.proto" "com/google/gapid/proto/service/path/Path")
protoc_go("github.com/google/gapid/gapis/service/box" "gapis/service/box" "box.proto")
protoc_java("gapis/service/box" "box.proto" "com/google/gapid/proto/service/box/Box")
protoc_go("github.com/google/gapid/gapis/service" "gapis/service" "service.proto")
protoc_java("gapis/service" "service.proto" "com/google/gapid/proto/service/Service;com/google/gapid/proto/service/GapidGrpc")
protoc_go("github.com/google/gapid/gapis/stringtable" "gapis/stringtable" "stringtable.proto")
protoc_java("gapis/stringtable" "stringtable.proto" "com/google/gapid/proto/stringtable/Stringtable")
protoc_go("github.com/google/gapid/gapis/vertex" "gapis/vertex" "vertex.proto")
protoc_java("gapis/vertex" "vertex.proto" "com/google/gapid/proto/service/vertex/Vertex")
protoc_go("github.com/google/gapid/test/robot/build" "test/robot/build" "build.proto")
protoc_go("github.com/google/gapid/test/robot/job" "test/robot/job" "job.proto")
protoc_go("github.com/google/gapid/test/robot/job/worker" "test/robot/job/worker" "worker.proto")
protoc_go("github.com/google/gapid/test/robot/master" "test/robot/master" "master.proto")
protoc_go("github.com/google/gapid/test/robot/record" "test/robot/record" "record.proto")
protoc_go("github.com/google/gapid/test/robot/replay" "test/robot/replay" "replay.proto")
protoc_go("github.com/google/gapid/test/robot/report" "test/robot/report" "report.proto")
protoc_go("github.com/google/gapid/test/robot/search" "test/robot/search" "search.proto")
protoc_go("github.com/google/gapid/test/robot/stash" "test/robot/stash" "stash.proto")
protoc_go("github.com/google/gapid/test/robot/stash/grpc" "test/robot/stash/grpc" "stash.proto")
protoc_go("github.com/google/gapid/test/robot/subject" "test/robot/subject" "subject.proto")
protoc_go("github.com/google/gapid/test/robot/trace" "test/robot/trace" "trace.proto")
