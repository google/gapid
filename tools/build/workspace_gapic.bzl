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
            artifact = "io.grpc:grpc-api:1.40.0",
            sha256 = "e8996c17a0ff6665c3463f6800259a3755aa3d4863c5d51737b93b11e818a0bd",
            sha256_src = "16e6764b3f631bc19313d68eee0f2185d16893bfc3eed77783460be22467560a",
        )

        maybe_repository(
            maven_jar,
            name = "io_grpc_context",
            locals = locals,
            artifact = "io.grpc:grpc-context:1.40.0",
            sha256 = "31882abfcecc8d09ca87a4f514414c3abe0d8cd2a62b379249eb56d63edb9974",
            sha256_src = "9fe71a310ff57b980c8d54c7c5669b65481211a5cc8e89a4b73a50851d672273",
        )

        maybe_repository(
            maven_jar,
            name = "io_grpc_core",
            locals = locals,
            artifact = "io.grpc:grpc-core:1.40.0",
            sha256 = "8d712597726a0478ed0a5e05cc5662e1a6b7b9efbe2d585d43c947ec94275b8b",
            sha256_src = "a3ba9faa0317c5c49fad7ce3f29fea906c2a4d28c67648916cfffc4d176802c5",
        )

        maybe_repository(
            maven_jar,
            name = "io_grpc_okhttp",
            locals = locals,
            artifact = "io.grpc:grpc-okhttp:1.40.0",
            sha256 = "0c60bc57ba811696283f3c7d72f41967b6bc359d49e1d3fde091d9a6c3d5191d",
            sha256_src = "891fbb30337a71d83c429637769751cf9b39a11210c4a5001117eb65d49aec3d",
        )

        maybe_repository(
            maven_jar,
            name = "io_grpc_protobuf",
            locals = locals,
            artifact = "io.grpc:grpc-protobuf:1.40.0",
            sha256 = "f6598354276a1511320e452a18483732632c9a73a2372b9ec0a66c9a8248f298",
            sha256_src = "da5c81d0e7f60ae0d8314d9e77a6fbf96fa9a48915a96b8bcbdf6e60b7e4410e",
        )

        maybe_repository(
            maven_jar,
            name = "io_grpc_protobuf_lite",
            locals = locals,
            artifact = "io.grpc:grpc-protobuf-lite:1.40.0",
            sha256 = "8bfc88d763eab03e7278ee3679e5c6ac0e8263c74eeaec3925dd1125a2bddade",
            sha256_src = "d062c9070c3a5d6fe97b09438217b8493db6d828fe69316b32864f393ab1e29d",
        )

        maybe_repository(
            maven_jar,
            name = "io_grpc_stub",
            locals = locals,
            artifact = "io.grpc:grpc-stub:1.40.0",
            sha256 = "fbb5cede6583efc9c3b74ba934f49fbb82c9f0e5f9dab45bcfb2f1835c0545cb",
            sha256_src = "9b484fdf5170158be45e130d40c9fe7f0c9dee22bd1792a361989cb385ce942e",
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
            artifact = "io.opencensus:opencensus-api:0.28.3",
            sha256 = "16a67b0996c327482e539c5194f246d7aa3cfd80a9d0e2440014db3d4a089246",
            sha256_src = "f53f9b719016d4b976e219106c37a48ecf35a2b4b2ff059e8bbe6940c623ad3c",
        )

        maybe_repository(
            maven_jar,
            name = "io_opencensus_contrib_grpc_metrics",
            locals = locals,
            artifact = "io.opencensus:opencensus-contrib-grpc-metrics:0.28.3",
            sha256 = "965d3d2f82f7fe54f26428c419fea466f0de037c92d03060c04a471e28ee6834",
            sha256_src = "c3b4eb41ec20aa6f25238a1f35c7577dfc223d9fa1122ea29770d0601bd21bcd",
        )

        # Perfmark used by gRPC.
        maybe_repository(
            maven_jar,
            name = "io_perfmark_api",
            locals = locals,
            artifact = "io.perfmark:perfmark-api:0.24.0",
            sha256 = "9b4d1d63ad9eae90192d706c80f6242509a8c677395f46149b208599f8a7b1a7",
            sha256_src = "4cebc85a53db1f7dabc2fe7d34f429fd1220331200b3be50b6cc3979ec041f52",
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
            artifact = "com.google.guava:guava:30.1-jre",
            sha256 = "e6dd072f9d3fe02a4600688380bd422bdac184caf6fe2418cfdd0934f09432aa",
            sha256_src = "b17d4974b591e7e45d982d04ce400c424fa95288cbddce17394b65f65bfdec0f",
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
