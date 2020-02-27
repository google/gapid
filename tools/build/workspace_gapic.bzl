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
        commit = "009c51f2f793aabf516db90a14a52da2b613aa21",
        build_file = "@gapid//tools/build/third_party:grpc_java.BUILD",
        sha256 = "ffb06532376cfc78742d2ac5cbf244deb2885d0464ac8ab51de0dfdf408ec517"
    )

    if not no_maven:
        # gRPC and it's dependencies.
        ########################################################################
        maybe_repository(
            maven_jar,
            name = "io_grpc_context",
            locals = locals,
            artifact = "io.grpc:grpc-context:1.16.1",
            sha256 = "3a8d6548308bd100c61e1c399a1a32f601f81b4162d30f04872c05a2a5b824b9",
            sha256_src = "027e241d4fd675392c957cbb4df368e4babdad52a7bef9d13c70d3e2fbe406a1",
        )

        maybe_repository(
            maven_jar,
            name = "io_grpc_core",
            locals = locals,
            artifact = "io.grpc:grpc-core:1.16.1",
            sha256 = "4b20fb3bd4b07e284ac639ce7372483f83050fd67962fa628d353c762571e964",
            sha256_src = "12a6508ea698786860f9a0849caad4df85139d9c7a484eaf9bed259419f93977",
        )

        maybe_repository(
            maven_jar,
            name = "io_grpc_okhttp",
            locals = locals,
            artifact = "io.grpc:grpc-okhttp:1.16.1",
            sha256 = "c6f804d5bdf33a19c414b2e5743d647f3daabc261813abb0f9013c60ad6ace94",
            sha256_src = "9ddc86ed5a1612ee8e3cd11a5d4601a770fd231591c88270db3c8a8b59a6c39c",
        )

        maybe_repository(
            maven_jar,
            name = "io_grpc_protobuf",
            locals = locals,
            artifact = "io.grpc:grpc-protobuf:1.16.1",
            sha256 = "c46cd81341f002995c178687226c6174094635f13a95b7e8389c7c1d84290d82",
            sha256_src = "fd723b55711d40d7e61edd92c205e708531f5fbcbf464e616ff580aef54ac5a5",
        )

        maybe_repository(
            maven_jar,
            name = "io_grpc_protobuf_lite",
            locals = locals,
            artifact = "io.grpc:grpc-protobuf-lite:1.16.1",
            sha256 = "4059becc5c8a25f5907c998632cd7187df6523e1317390d43a3ef7909e99956a",
            sha256_src = "412dbd13c2b61c6f2f2f72f8f1426dcf4c4d4a3080a8c1b8ec282ed2e709fafd",
        )

        maybe_repository(
            maven_jar,
            name = "io_grpc_stub",
            locals = locals,
            artifact = "io.grpc:grpc-stub:1.16.1",
            sha256 = "4a6a33253dc68805f005a945686c567e9d73a618b237caf308c48d7a54d771fc",
            sha256_src = "d8a87b55ac7284e1fdafcd64df3b258e4ed78d764996620c9a70b8d1445c06b3",
        )

        # OKHttp used by gRPC.
        maybe_repository(
            maven_jar,
            name = "com_squareup_okhttp",
            locals = locals,
            artifact = "com.squareup.okhttp:okhttp:2.5.0",
            sha256 = "1cc716e29539adcda677949508162796daffedb4794cbf947a6f65e696f0381c",
            sha256_src = "1bf6850f38f34036f096a9deb5cb714f3f41b529c80de9c79b33f11adcedcc1a",
        )

        maybe_repository(
            maven_jar,
            name = "com_squareup_okio",
            locals = locals,
            artifact = "com.squareup.okio:okio:1.6.0",
            sha256 = "114bdc1f47338a68bcbc95abf2f5cdc72beeec91812f2fcd7b521c1937876266",
            sha256_src = "cf31dcd63db43c48c62ef41560006a25bbe3e207b170ecbd7bfe0b675880a0ac",
        )

        # Opencensus used by gRPC.
        maybe_repository(
            maven_jar,
            name = "io_opencensus_api",
            locals = locals,
            artifact = "io.opencensus:opencensus-api:0.12.3",
            sha256 = "8c1de62cbdaf74b01b969d1ed46c110bca1a5dd147c50a8ab8c5112f42ced802",
            sha256_src = "67e8b2120737c7dcfc61eef33f75319b1c4e5a2806d3c1a74cab810650ac7a19",
        )

        maybe_repository(
            maven_jar,
            name = "io_opencensus_contrib_grpc_metrics",
            locals = locals,
            artifact = "io.opencensus:opencensus-contrib-grpc-metrics:0.12.3",
            sha256 = "632c1e1463db471b580d35bc4be868facbfbf0a19aa6db4057215d4a68471746",
            sha256_src = "d54f6611f75432ca0ab13636a613392ae8b7136ba67eb1588fccdb8481f4d665",
        )

        maybe_repository(
            maven_jar,
            name = "javax_annotation_api",
            locals = locals,
            artifact = "javax.annotation:javax.annotation-api:1.2",
            sha256 = "5909b396ca3a2be10d0eea32c74ef78d816e1b4ead21de1d78de1f890d033e04",
            sha256_src = "8bd08333ac2c195e224cc4063a72f4aab3c980cf5e9fb694130fad41689689d0",
        )

        # LWJGL.
        ############################################################################
        maybe_repository(
            maven_jar,
            name = "org_lwjgl_core",
            locals = locals,
            artifact = "org.lwjgl:lwjgl:3.2.0",
            sha256 = "97af9a688081bbdf0cf208b93f02b58fe2db0504ee7333e54780c4b6f70694f8",
            sha256_src = "1919899fbea2dcf392d0bba6161da058f7f8c4da0877a7e6258ee57305c398e7",
            sha256_linux = "08ca6d394ef7ac97002bc939642dc285c7f61c94a69070fa9a916921fee637ab",
            sha256_windows = "ee93b31388356835fe8fbc6155dc83c73ceec7422b777aa2e7e3187e9689b2cc",
            sha256_macos = "6db0910dea5323a3b61c8b16a28e5f84ee780f2affc2cd06da34b9fe09295051",
        )

        maybe_repository(
            maven_jar,
            name = "org_lwjgl_opengl",
            locals = locals,
            artifact = "org.lwjgl:lwjgl-opengl:3.2.0",
            sha256 = "4cc168087708653bdbc1d700daf3fb4b8c1fc89d23d4cf6ee834c3b1208c85a6",
            sha256_src = "3693081b41f4259be2df7e37c36a5a2b9ce3f8a451b7acc9d609749a1c6e7974",
            sha256_linux = "1f04e87ab78cb9616447f2abbf3d8b0c3cf25c73aea7f40b2580caba2a1269f6",
            sha256_windows = "4e515b2c596a7a0794f0fe855637aafe1427a1f4e331d6b5be83c03add04e0eb",
            sha256_macos = "a9cc1b2de4be574261c9923027076d5ef5c9565dc2c98c074857d4455ed14848",
        )

        # Other dependencies.
        ############################################################################
        maybe_repository(
            maven_jar,
            name = "com_google_guava",
            locals = locals,
            artifact = "com.google.guava:guava:27.0-jre",
            sha256 = "63b09db6861011e7fb2481be7790c7fd4b03f0bb884b3de2ecba8823ad19bf3f",
            sha256_src = "170dbf09858d1cffdaaa53d4d6ab15e6253c845318b0cc3bf21f8dffa9d433ab",
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
