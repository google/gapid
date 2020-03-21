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
        commit = "3dbd250eae2c5e4f4e5e7046c6573805cc0dcc29",  # 1.28.0
        sha256 = "988cde6fa4cbbbdad4b13c646be17d079ae961a23f4479f6520aaed3baab019b"
    )

    if not no_maven:
        # gRPC and it's dependencies.
        ########################################################################
        maybe_repository(
            maven_jar,
            name = "io_grpc_api",
            locals = locals,
            artifact = "io.grpc:grpc-api:1.28.0",
            sha256 = "10db0e02a85601d38da1b77bfcd7ae08f56b719a5e22aae9894a19c64b0fa8ce",
            sha256_src = "a1ecf073671930e4883525cfa11850f04ba78b73f1e8434b81a0b2bf9b2f5927",
        )

        maybe_repository(
            maven_jar,
            name = "io_grpc_context",
            locals = locals,
            artifact = "io.grpc:grpc-context:1.28.0",
            sha256 = "cc57df006555be067af2a6ae9c6510bd7ed40a2dc1af278ceb4e491ce7f184de",
            sha256_src = "d0f932244bee0f4c497646b5d94baa13877f4eddc4623ec6007dd5698253b421",
        )

        maybe_repository(
            maven_jar,
            name = "io_grpc_core",
            locals = locals,
            artifact = "io.grpc:grpc-core:1.28.0",
            sha256 = "be7754fd1bcc58d25009e2f8aff5d5bb243ca0b8acf969b77b2ee606c2a1fcc3",
            sha256_src = "6943ae4fbef30cd9192213fd220a62a60f751048ee11c78cce277f95d3a36101",
        )

        maybe_repository(
            maven_jar,
            name = "io_grpc_okhttp",
            locals = locals,
            artifact = "io.grpc:grpc-okhttp:1.28.0",
            sha256 = "6e7a080c064b2f9b3639b576d0bdfb4c5180616ce88df7d4211cbf952691e28c",
            sha256_src = "c37f1317dbc93092e38d5ad6627f80fa595be3daed4484d9c8c71de0d6dce800",
        )

        maybe_repository(
            maven_jar,
            name = "io_grpc_protobuf",
            locals = locals,
            artifact = "io.grpc:grpc-protobuf:1.28.0",
            sha256 = "a48ef62c55e2bd92325ce0924b60363cfb00d274ba1ab281dc8d9c568fd48fd8",
            sha256_src = "52148a963d712418ed8c8378635863998c33db90e89fcbdfb75009068916f0f7",
        )

        maybe_repository(
            maven_jar,
            name = "io_grpc_protobuf_lite",
            locals = locals,
            artifact = "io.grpc:grpc-protobuf-lite:1.28.0",
            sha256 = "5dbcf11cec631fa1b99b3aa338b8f306dbf660f127126f29efc4218166c44857",
            sha256_src = "844585c241a3a021a5f2e9f75881d8da118f842672f03a654d5abb7d1c24cf9f",
        )

        maybe_repository(
            maven_jar,
            name = "io_grpc_stub",
            locals = locals,
            artifact = "io.grpc:grpc-stub:1.28.0",
            sha256 = "f10b2f46cb5142f18135dcfb163e4db7b12aab47504746a00c4a2800a791dc01",
            sha256_src = "eb0ca640f9147ea9c3d94626c55d8a73696401d6e9f37cda7182a2300e8be214",
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
            artifact = "com.squareup.okio:okio:1.13.0",
            sha256 = "734269c3ebc5090e3b23566db558f421f0b4027277c79ad5d176b8ec168bb850",
            sha256_src = "b0305445ad4001639413951837a2248c5e9ca28386fbae2e31e1556f7710c5a8",
        )

        # Opencensus used by gRPC.
        maybe_repository(
            maven_jar,
            name = "io_opencensus_api",
            locals = locals,
            artifact = "io.opencensus:opencensus-api:0.24.0",
            sha256 = "f561b1cc2673844288e596ddf5bb6596868a8472fd2cb8993953fc5c034b2352",
            sha256_src = "01693c455b3748a494813ae612e1766c9e804d56561b759d8399270865427bf6",
        )

        maybe_repository(
            maven_jar,
            name = "io_opencensus_contrib_grpc_metrics",
            locals = locals,
            artifact = "io.opencensus:opencensus-contrib-grpc-metrics:0.24.0",
            sha256 = "875582e093f11950ad3f4a50b5fee33a008023f7d1e47820a1bef05d23b9ed42",
            sha256_src = "48c84a321af149c35a95b0d433a49c78e21674e10568fbc529675de0ee46fa96",
        )

        # Perfmark used by gRPC.
        maybe_repository(
            maven_jar,
            name = "io_perfmark_api",
            locals = locals,
            artifact = "io.perfmark:perfmark-api:0.19.0",
            sha256 = "b734ba2149712409a44eabdb799f64768578fee0defe1418bb108fe32ea43e1a",
            sha256_src = "05cfbdd34e6fc1f10181c755cec67cf1ee517dfee615e25d1007a8aabd569dba",
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
            artifact = "org.lwjgl:lwjgl:3.2.3",
            sha256 = "f9928c3b4b540643a1bbd59286d3c7175e470849261a0c29a81389f52265ad8b",
            sha256_src = "97b9c693337f76a596b86b07db26a0a8022e3a4e0a0360edb9bb87bc9b172cda",
            sha256_linux = "002810129fc6ac4cdfcdf190e18a643a5021b6300f489c1026bbc5d00140ca2e",
            sha256_windows = "bdf519b9aa90f799954113a15dfa84b273ee4781876b3ecdebf192ce4f88a26c",
            sha256_macos = "5c520c465a84034b8bc23e1d7ecd621bb99c437cd254ea46b53197448d1b8128",
        )

        maybe_repository(
            maven_jar,
            name = "org_lwjgl_opengl",
            locals = locals,
            artifact = "org.lwjgl:lwjgl-opengl:3.2.3",
            sha256 = "10bcc37506e01d1477d65f1fcf0aa672c95eb785265b28b7f321c8381093eda2",
            sha256_src = "6082a81f350dfc0e390a9ceb4347fa2a28cd07dfd54dc757fb05fa6f3350314e",
            sha256_linux = "466e8bae1818c4c584771ee093c8a735e26f56fb25a81dde5675160aaa2fa045",
            sha256_windows = "c08e3de31632163ac5f746fa945f1924142e08520bd9c81b7dd1b5dbd1b0b8bb",
            sha256_macos = "e4b4d0cd9138d52271c1d5c18e43c9ac5d36d1a727c47e5ee4031cb45ce730ca",
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
