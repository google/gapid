# Copyright (C) 2021 Google Inc.
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

cc_library(
    name = "etc2codec_deps",
    srcs = [
        "EtcLib/Etc/EtcConfig.h",
        "EtcLib/Etc/EtcMath.cpp",
        "EtcLib/Etc/EtcMath.h",
    ],
    hdrs = [
        "EtcLib/Etc/EtcColor.h",
        "EtcLib/Etc/EtcColorFloatRGBA.h",
        "EtcLib/Etc/EtcConfig.h",
        "EtcLib/Etc/EtcImage.h",
        "EtcLib/Etc/EtcMath.h",
    ],
    copts = [],  # keep
    strip_include_prefix = "EtcLib/Etc",
    visibility = ["//visibility:private"],
)

cc_library(
    name = "etc2codec",
    srcs = [
        "EtcLib/EtcCodec/EtcBlock4x4.cpp",
        "EtcLib/EtcCodec/EtcBlock4x4.h",
        "EtcLib/EtcCodec/EtcBlock4x4Encoding.cpp",
        "EtcLib/EtcCodec/EtcBlock4x4Encoding.h",
        "EtcLib/EtcCodec/EtcBlock4x4EncodingBits.h",
        "EtcLib/EtcCodec/EtcBlock4x4Encoding_ETC1.cpp",
        "EtcLib/EtcCodec/EtcBlock4x4Encoding_ETC1.h",
        "EtcLib/EtcCodec/EtcBlock4x4Encoding_R11.cpp",
        "EtcLib/EtcCodec/EtcBlock4x4Encoding_R11.h",
        "EtcLib/EtcCodec/EtcBlock4x4Encoding_RG11.cpp",
        "EtcLib/EtcCodec/EtcBlock4x4Encoding_RG11.h",
        "EtcLib/EtcCodec/EtcBlock4x4Encoding_RGB8.cpp",
        "EtcLib/EtcCodec/EtcBlock4x4Encoding_RGB8.h",
        "EtcLib/EtcCodec/EtcBlock4x4Encoding_RGB8A1.cpp",
        "EtcLib/EtcCodec/EtcBlock4x4Encoding_RGB8A1.h",
        "EtcLib/EtcCodec/EtcBlock4x4Encoding_RGBA8.cpp",
        "EtcLib/EtcCodec/EtcBlock4x4Encoding_RGBA8.h",
        "EtcLib/EtcCodec/EtcDifferentialTrys.cpp",
        "EtcLib/EtcCodec/EtcDifferentialTrys.h",
        "EtcLib/EtcCodec/EtcErrorMetric.h",
        "EtcLib/EtcCodec/EtcIndividualTrys.cpp",
        "EtcLib/EtcCodec/EtcIndividualTrys.h",
        "EtcLib/EtcCodec/EtcSortedBlockList.cpp",
        "EtcLib/EtcCodec/EtcSortedBlockList.h",
    ],
    hdrs = [
        "EtcLib/EtcCodec/EtcBlock4x4.h",
        "EtcLib/EtcCodec/EtcBlock4x4Encoding.h",
        "EtcLib/EtcCodec/EtcBlock4x4EncodingBits.h",
        "EtcLib/EtcCodec/EtcErrorMetric.h",
        "EtcLib/EtcCodec/EtcSortedBlockList.h",
    ],
    copts = [],  # keep
    strip_include_prefix = "EtcLib/EtcCodec",
    visibility = ["//visibility:private"],
    deps = ["etc2codec_deps"],
)

cc_library(
    name = "etc2comp",
    srcs = [
        "EtcLib/Etc/Etc.cpp",
        "EtcLib/Etc/Etc.h",
        "EtcLib/Etc/EtcColor.h",
        "EtcLib/Etc/EtcColorFloatRGBA.h",
        "EtcLib/Etc/EtcConfig.h",
        "EtcLib/Etc/EtcFilter.h",
        "EtcLib/Etc/EtcImage.cpp",
        "EtcLib/Etc/EtcImage.h",
    ],
    hdrs = [
        "EtcLib/Etc/Etc.h",
        "EtcLib/Etc/EtcColor.h",
        "EtcLib/Etc/EtcColorFloatRGBA.h",
        "EtcLib/Etc/EtcConfig.h",
        "EtcLib/Etc/EtcImage.h",
    ],
    copts = [],  # keep
    include_prefix = "third_party/etc2comp",
    visibility = ["//visibility:public"],
    deps = ["etc2codec"],
)
