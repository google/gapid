# Copyright (C) 2018 Google Inc.
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

# Defines macros to be called from a WORKSPACE file to setup the GAPID
# Java client dependencies.

load("@gapid//tools/build/rules:repository.bzl", "github_repository", "maybe_repository", "maven_jar")
load("@gapid//tools/build/third_party:jface.bzl", "jface")
load("@gapid//tools/build/third_party:swt.bzl", "swt")

# Defines the repositories for GAPID's Java client's dependencies.
#  no_maven - if true, none of the maven managed dependencies are initialized.
#  no_swt - if true, the SWT repository is not initialized.
#  no_jface - if true, the JFace repository is not initialized.
#  locals - can be used to provide local path overrides for repos:
#     {"foo": "/path/to/foo"} would cause @foo to be a local repo based on /path/to/foo.
def gapic_dependencies(no_maven = False, no_swt = False, no_jface = False, locals = {}):

    maybe_repository(
        github_repository,
        name = "com_github_grpc_java",
        locals = locals,
        organization = "grpc",
        project = "grpc-java",
        commit = "6e2e18bb728793df32b2ba195a954ad380e546de",  # 1.48.1
        sha256 = "b4d9b7827470e1cb9b2ce2dced0cfa6caed3d0a76cfb6f32eb41f2e54c88805f"
    )

    if not no_maven:
        # gRPC and it's dependencies.
        ########################################################################
        maybe_repository(
            maven_jar,
            name = "io_grpc_api",
            locals = locals,
            artifact = "io.grpc:grpc-api:1.48.1",
            sha256 = "aeb8d7a1361aa3d8f5a191580fa7f8cbc5ceb53137a4a698590f612f791e2c45",
            sha256_src = "f6c8ee8aea763e2b4c9d3e392e4d05438e0d8d641230667a4e551b4e4f4ce959",
        )

        maybe_repository(
            maven_jar,
            name = "io_grpc_context",
            locals = locals,
            artifact = "io.grpc:grpc-context:1.48.1",
            sha256 = "2fb9007e12f768e9c968f9db292be4ea9cba2ef40fb8d179f3f8746ebdc73c1b",
            sha256_src = "c6e63958d0d8050ff8c2669ba19516f4bbe8f9a8cf78c9da0acaf71c8d71e908",
        )

        maybe_repository(
            maven_jar,
            name = "io_grpc_core",
            locals = locals,
            artifact = "io.grpc:grpc-core:1.48.1",
            sha256 = "6d472ee6d2b60ef3f3e6801e7cd4dbec5fbbef81e883a0de1fbc55e6defe1cb7",
            sha256_src = "00a76915e3bcab4bfa5332b2087e2b591b5dc1892305ca9463da2e02a6e0ad38",
        )

        maybe_repository(
            maven_jar,
            name = "io_grpc_okhttp",
            locals = locals,
            artifact = "io.grpc:grpc-okhttp:1.48.1",
            sha256 = "2b771a645967ddcac4950a9c068c98dbfcf3678fddb5c7719f79f6c3cc1323df",
            sha256_src = "2efc0d12a55b88ba148f3152ae3b175b55dc1ba34b197ed2704ffc911b8b4312",
        )

        maybe_repository(
            maven_jar,
            name = "io_grpc_protobuf",
            locals = locals,
            artifact = "io.grpc:grpc-protobuf:1.48.1",
            sha256 = "6ab68b0a3bb3834af44208df058be4631425b56ef95f9b9412aa21df3311e8d3",
            sha256_src = "7707101da1ff2a88814ab73c2c8b63e99d447da507a1b78906d5eb5a244b46b2",
        )

        maybe_repository(
            maven_jar,
            name = "io_grpc_protobuf_lite",
            locals = locals,
            artifact = "io.grpc:grpc-protobuf-lite:1.48.1",
            sha256 = "0a4c735bb80e342d418c0ef7d2add7793aaf72b91c449bde2769ea81f1869737",
            sha256_src = "d93a6cc1089cf65d059a46bf986ce326edc47273e462ee89825aceb209735f2f",
        )

        maybe_repository(
            maven_jar,
            name = "io_grpc_stub",
            locals = locals,
            artifact = "io.grpc:grpc-stub:1.48.1",
            sha256 = "6436f19cef264fd949fb7a41e11424e373aa3b1096cad0b7e518f1c81aa60f23",
            sha256_src = "d2ab4864568f63059942a914802808a7c9aa99f4d1071267e9817c4c63687057",
        )

        # OKHttp used by gRPC.
        maybe_repository(
            maven_jar,
            name = "com_squareup_okhttp",
            locals = locals,
            artifact = "com.squareup.okhttp:okhttp:2.7.4",
            sha256 = "c88be9af1509d5aeec9394a818c0fa08e26fad9d64ba134e6f977e0bb20cb114",
            sha256_src = "57c3b223fb40568eabb97e2be989625746af99120a8112bbcfa49d7d9ab3c746",
        )

        maybe_repository(
            maven_jar,
            name = "com_squareup_okio",
            locals = locals,
            artifact = "com.squareup.okio:okio:1.17.5",
            sha256 = "19a7ff48d86d3cf4497f7f250fbf295f430c13a528dd5b7b203f821802b886ad",
            sha256_src = "537b41075d390d888aec040d0798211b1702d34f558efc09364b5f7d388ec496",
        )

        # Opencensus used by gRPC.
        maybe_repository(
            maven_jar,
            name = "io_opencensus_api",
            locals = locals,
            artifact = "io.opencensus:opencensus-api:0.31.1",
            sha256 = "f1474d47f4b6b001558ad27b952e35eda5cc7146788877fc52938c6eba24b382",
            sha256_src = "6748d57aaae81995514ad3e2fb11a95aa88e158b3f93450288018eaccf31e86b",
        )

        maybe_repository(
            maven_jar,
            name = "io_opencensus_contrib_grpc_metrics",
            locals = locals,
            artifact = "io.opencensus:opencensus-contrib-grpc-metrics:0.31.1",
            sha256 = "c862a1d783652405512e26443f6139e6586f335086e5e1f1dca2b0c4e338a174",
            sha256_src = "c2b4d7c9928b7bf40c65008c4966f9fe00b4a0fe9150f21f43d6e4e85c7f3767",
        )

        # Perfmark used by gRPC.
        maybe_repository(
            maven_jar,
            name = "io_perfmark_api",
            locals = locals,
            artifact = "io.perfmark:perfmark-api:0.25.0",
            sha256 = "2044542933fcdf40ad18441bec37646d150c491871157f288847e29cb81de4cb",
            sha256_src = "007b6b6beaba11fabb025d79b8774b6a7583596a8ec0a28157570304642b0e72",
        )

        maybe_repository(
            maven_jar,
            name = "javax_annotation_api",
            locals = locals,
            artifact = "javax.annotation:javax.annotation-api:1.3.2",
            sha256 = "e04ba5195bcd555dc95650f7cc614d151e4bcd52d29a10b8aa2197f3ab89ab9b",
            sha256_src = "128971e52e0d84a66e3b6e049dab8ad7b2c58b7e1ad37fa2debd3d40c2947b95",
        )

        # LWJGL.
        ############################################################################
        maybe_repository(
            maven_jar,
            name = "org_lwjgl_core",
            locals = locals,
            artifact = "org.lwjgl:lwjgl:3.3.1",
            sha256 = "cf83f90e32fb973ff5edfca4ef35f55ca51bb70a579b6a1f290744f552e8e484",
            sha256_src = "30cb8190660bcbe9ce5761690a5abbf45735c9af4d58f61bd568203025fd3a36",
            sha256_linux = "22ef2afa31a1740a337ec9c6806c6b8d97e931a63e2c43270cbaf14fb3f6fc4e",
            sha256_windows = "093d13d62a6434bc656bf10a3b37e2530fd9af3b0bc20f8e9545be58659d1443",
            sha256_macos = "f48e610a981dca515b9bbe7345cc76e3450c13404b052d8ac357ef0af8e90abf",
        )

        maybe_repository(
            maven_jar,
            name = "org_lwjgl_opengl",
            locals = locals,
            artifact = "org.lwjgl:lwjgl-opengl:3.3.1",
            sha256 = "e436d2144f3a36fff772fd64233316809b795f390defd4d55660fc686e7c2834",
            sha256_src = "42434482dffbcdbc5716c53b33eb0475c13e2c34106e0e1d0766cae20386e092",
            sha256_linux = "bcfcd9f8dfd229488ad9ec353d48c7b02de5ca21badd4092f1902239a1ed0690",
            sha256_windows = "851b9b593cac21e3af51dd4c303e8df3fe6a344a3f81d6b9bcc186e7e540bfda",
            sha256_macos = "711f29972888509262a957f1b3138bfa9b505519d11df6e5763d1c9a3f8b3c5d",
        )

        # Other dependencies.
        ############################################################################
        maybe_repository(
            maven_jar,
            name = "com_google_guava",
            locals = locals,
            artifact = "com.google.guava:guava:31.1-jre",
            sha256 = "a42edc9cab792e39fe39bb94f3fca655ed157ff87a8af78e1d6ba5b07c4a00ab",
            sha256_src = "8ab1853cdaf936ec88be80c17302b7c20abafbd4f54d4fb54d7011c529e3a44a",
        )

        maybe_repository(
            maven_jar,
            name = "com_google_guava-failureaccess",
            locals = locals,
            artifact = "com.google.guava:failureaccess:1.0.1",
            sha256 = "a171ee4c734dd2da837e4b16be9df4661afab72a41adaf31eb84dfdaf936ca26",
            sha256_src = "092346eebbb1657b51aa7485a246bf602bb464cc0b0e2e1c7e7201fadce1e98f",
        )

    if not no_swt:
        maybe_repository(
            swt,
            name = "swt",
            locals = locals,
        )

    if not no_jface:
        maybe_repository(
            jface,
            name = "jface",
            locals = locals,
        )

DEFAULT_MAPPINGS = {
    # gRPC
    "io_grpc_api": "@io_grpc_api//:jar",
    "io_grpc_context": "@io_grpc_context//:jar",
    "io_grpc_core": "@io_grpc_core//:jar",
    "io_grpc_okhttp": "@io_grpc_okhttp//:jar",
    "io_grpc_protobuf": "@io_grpc_protobuf//:jar",
    "io_grpc_protobuf_lite": "@io_grpc_protobuf_lite//:jar",
    "io_grpc_stub": "@io_grpc_stub//:jar",
    "com_squareup_okhttp": "@com_squareup_okhttp//:jar",
    "com_squareup_okio": "@com_squareup_okio//:jar",
    "io_opencensus_api": "@io_opencensus_api//:jar",
    "io_opencensus_contrib_grpc_metrics": "@io_opencensus_contrib_grpc_metrics//:jar",
    "io_perfmark_api": "@io_perfmark_api//:jar",
    "javax_annotation_api": "@javax_annotation_api//:jar",
    # LWJGL
    "org_lwjgl_core": "@org_lwjgl_core//:jar",
    "org_lwjgl_core_natives_linux": "@org_lwjgl_core//:jar-natives-linux",
    "org_lwjgl_core_natives_windows": "@org_lwjgl_core//:jar-natives-windows",
    "org_lwjgl_core_natives_macos": "@org_lwjgl_core//:jar-natives-macos",
    "org_lwjgl_opengl": "@org_lwjgl_opengl//:jar",
    "org_lwjgl_opengl_natives_linux": "@org_lwjgl_opengl//:jar-natives-linux",
    "org_lwjgl_opengl_natives_windows": "@org_lwjgl_opengl//:jar-natives-windows",
    "org_lwjgl_opengl_natives_macos": "@org_lwjgl_opengl//:jar-natives-macos",
    # Others
    "com_google_guava": "@com_google_guava//:jar",
    "com_google_guava-failureaccess": "@com_google_guava-failureaccess//:jar",
    "jface": "@jface",
    "swt": "@swt",
}

def gapic_third_party(mappings = DEFAULT_MAPPINGS):
    _gapic_third_party(
        name = "gapic_third_party",
        mappings = mappings,
    )

def _gapic_third_party_impl(ctx):
    ctx.template(
        ctx.path("BUILD.bazel"),
        Label("@gapid//tools/build/third_party:gapic_third_party.BUILD"),
        substitutions = {
            k.join(["{{", "}}"]): ctx.attr.mappings[k] for k in ctx.attr.mappings
        },
        executable = False,
    )

_gapic_third_party = repository_rule(
    implementation = _gapic_third_party_impl,
    attrs = {
        "mappings": attr.string_dict(
            allow_empty = False,
            mandatory = True,
        ),
    },
)
