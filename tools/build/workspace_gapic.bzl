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

load("@gapid//tools/build/rules:repository.bzl", "github_repository", "maybe_repository")
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
    )

    if not no_maven:
        # gRPC and it's dependencies.
        ########################################################################
        maybe_repository(
            native.maven_jar,
            name = "io_grpc_context",
            locals = locals,
            artifact = "io.grpc:grpc-context:1.16.1",
            sha1 = "4adb6d55045b21cb384bc4498d4a7593f6cab8d7",
            sha1_src = "2a74b951a4393b6b3328416e1f397af1e519d0c9",
        )

        maybe_repository(
            native.maven_jar,
            name = "io_grpc_core",
            locals = locals,
            artifact = "io.grpc:grpc-core:1.16.1",
            sha1 = "8a938ece0ad8d8bf77d790c502ba51ebec114aa9",
            sha1_src = "1df482c20f3b58fc7d0affb35584e610a27d1b15",
        )

        maybe_repository(
            native.maven_jar,
            name = "io_grpc_okhttp",
            locals = locals,
            artifact = "io.grpc:grpc-okhttp:1.16.1",
            sha1 = "ae34ca46a3366cdb6d1836e7540162ee6c3627d1",
            sha1_src = "12abd56a0d7cbed07c4419b41967b7dcb1e00560",
        )

        maybe_repository(
            native.maven_jar,
            name = "io_grpc_protobuf",
            locals = locals,
            artifact = "io.grpc:grpc-protobuf:1.16.1",
            sha1 = "1f8ac89924b5de4a94058ae26c9de28f8eff49dd",
            sha1_src = "6e36a77b2b1f3ea4d272d5d6c69e83746c3b9117",
        )

        maybe_repository(
            native.maven_jar,
            name = "io_grpc_protobuf_lite",
            locals = locals,
            artifact = "io.grpc:grpc-protobuf-lite:1.16.1",
            sha1 = "3d03ee1e5e292f2312d7ca99c00ddcf9d0544c35",
            sha1_src = "18943e40bed01a48de91babffa247be28cd1ae9e",
        )

        maybe_repository(
            native.maven_jar,
            name = "io_grpc_stub",
            locals = locals,
            artifact = "io.grpc:grpc-stub:1.16.1",
            sha1 = "f3c30248564608791407bf43b1d4db52a80e6c36",
            sha1_src = "6324f9fdbdd224addab073495c54c109b325c4a6",
        )

        # OKHttp used by gRPC.
        maybe_repository(
            native.maven_jar,
            name = "com_squareup_okhttp",
            locals = locals,
            artifact = "com.squareup.okhttp:okhttp:2.5.0",
            sha1 = "4de2b4ed3445c37ec1720a7d214712e845a24636",
            sha1_src = "cd4ddf1fb4ad84ea5d67ee3b386aea25f02ea096",
        )

        maybe_repository(
            native.maven_jar,
            name = "com_squareup_okio",
            locals = locals,
            artifact = "com.squareup.okio:okio:1.6.0",
            sha1 = "98476622f10715998eacf9240d6b479f12c66143",
            sha1_src = "fb6ec0fbaa0229088b0d3dfe3b1f9d24add3e775",
        )

        # Opencensus used by gRPC.
        maybe_repository(
            native.maven_jar,
            name = "io_opencensus_api",
            locals = locals,
            artifact = "io.opencensus:opencensus-api:0.12.3",
            sha1 = "743f074095f29aa985517299545e72cc99c87de0",
            sha1_src = "09c2dad7aff8b6d139723b9181ba5da3f689213b",
        )

        maybe_repository(
            native.maven_jar,
            name = "io_opencensus_contrib_grpc_metrics",
            locals = locals,
            artifact = "io.opencensus:opencensus-contrib-grpc-metrics:0.12.3",
            sha1 = "a4c7ff238a91b901c8b459889b6d0d7a9d889b4d",
            sha1_src = "9a7d004b774700837eeebff61230b8662d0e30d1",
        )

        # LWJGL.
        ############################################################################
        maybe_repository(
            _maven_jar_with_natives,
            name = "org_lwjgl_core",
            locals = locals,
            artifact = "org.lwjgl:lwjgl:3.2.0",
            sha1s = {
                "base": "7723544dc3fc740f0ee59cce9a3a0cecc8681747",
                "base-src": "7b6c54e5beb9ef0824ca0c31726a623c36d88c56",
                "linux": "4c23e3f9ae657a52bddfa1c92d1b0ba770259eed",
                "windows": "86c90ce2abe6129bfd5052a8b82f3dc2394c8dd1",
                "macos": "84bf26af17298d47b0ff9765a426279aaa133cad",
            },
        )

        maybe_repository(
            _maven_jar_with_natives,
            name = "org_lwjgl_opengl",
            locals = locals,
            artifact = "org.lwjgl:lwjgl-opengl:3.2.0",
            sha1s = {
                "base": "1c64c692473a70af297651d369debc93efa2e49f",
                "base-src": "1c3a04979231835d20c46c8daf1c9c4020d64568",
                "linux": "953ac48def909b1c67fc54299f5c403479ef8ac7",
                "windows": "b1f27bce30f8e40b03502a5d86687b30d844ba35",
                "macos": "4b2015f5d180dc707ac47d000acd35d49b5d7463",
            },
        )

        # Other dependencies.
        ############################################################################
        maybe_repository(
            native.maven_jar,
            name = "com_google_guava",
            locals = locals,
            artifact = "com.google.guava:guava:27.0-jre",
            sha1 = "c6ad87d2575af8ac8ec38e28e75aefa882cc3a1f",
            sha1_src = "d6484e2ee11ad928ccf61cf3e4ce9cedc2eead7e",
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
    "io_grpc_context": "@io_grpc_context//jar",
    "io_grpc_core": "@io_grpc_core//jar",
    "io_grpc_okhttp": "@io_grpc_okhttp//jar",
    "io_grpc_protobuf": "@io_grpc_protobuf//jar",
    "io_grpc_protobuf_lite": "@io_grpc_protobuf_lite//jar",
    "io_grpc_stub": "@io_grpc_stub//jar",
    "com_squareup_okhttp": "@com_squareup_okhttp//jar",
    "com_squareup_okio": "@com_squareup_okio//jar",
    "io_opencensus_api": "@io_opencensus_api//jar",
    "io_opencensus_contrib_grpc_metrics": "@io_opencensus_contrib_grpc_metrics//jar",
    # LWJGL
    "org_lwjgl_core": "@org_lwjgl_core//jar",
    "org_lwjgl_core_natives_linux": "@org_lwjgl_core_natives_linux//jar",
    "org_lwjgl_core_natives_windows": "@org_lwjgl_core_natives_windows//jar",
    "org_lwjgl_core_natives_macos": "@org_lwjgl_core_natives_macos//jar",
    "org_lwjgl_opengl": "@org_lwjgl_opengl//jar",
    "org_lwjgl_opengl_natives_linux": "@org_lwjgl_opengl_natives_linux//jar",
    "org_lwjgl_opengl_natives_windows": "@org_lwjgl_opengl_natives_windows//jar",
    "org_lwjgl_opengl_natives_macos": "@org_lwjgl_opengl_natives_macos//jar",
    # Others
    "com_google_guava": "@com_google_guava//jar",
    "jface": "@jface",
    "swt": "@swt",
}

def gapic_third_party(mappings = DEFAULT_MAPPINGS):
    _gapic_third_party(
        name = "gapic_third_party",
        mappings = mappings,
    )

def _maven_jar_with_natives(name, artifact, sha1s = {}):
    native.maven_jar(
        name = name,
        artifact = artifact,
        sha1 = sha1s["base"],
        sha1_src = sha1s["base-src"]
    )

    toks = artifact.split(":")
    toks.insert(len(toks) - 1, "jar")

    toks.insert(len(toks) - 1, "natives-linux")
    native.maven_jar(
        name = name + "_natives_linux",
        artifact = ":".join(toks),
        sha1 = sha1s["linux"],
    )

    toks[len(toks) - 2] = "natives-windows"
    native.maven_jar(
        name = name + "_natives_windows",
        artifact = ":".join(toks),
        sha1 = sha1s["windows"],
    )

    toks[len(toks) - 2] = "natives-macos"
    native.maven_jar(
        name = name + "_natives_macos",
        artifact = ":".join(toks),
        sha1 = sha1s["macos"],
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
