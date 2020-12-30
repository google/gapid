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

# Repository rule to download the JFace jars.

_BASE = "https://download.eclipse.org/eclipse/updates/4.18/R-4.18-202012021800/plugins/"
_LIBS = [
    struct(
        name = "org.eclipse.core.commands",
        version = "3.9.800.v20201021-1339",
        sha = "6130eed0dec2fc67b7b01fbcec72064dd6ae7ae955b00076e3105a78855f6e5c",
        sha_src = "af82e080927a405fd0be8d4eff3b6a8fbe256b81c247292e22f05c5d869a8b0d",
    ),
    struct(
        name = "org.eclipse.core.runtime",
        version = "3.20.0.v20201027-1526",
        sha = "78d3cea9487a2e3a361cfd92baae65bf33f22e0106b41748b6c3e6a33ef4f29a",
        sha_src = "61257d6221bedfae7fd33ad029b129fa1c275d6b91d066c3e085972e3e968706",
    ),
    struct(
        name = "org.eclipse.equinox.common",
        version = "3.14.0.v20201102-2053",
        sha = "2e4c3ef5bb5b2a610d03944274d4891169f0728d85cd8842233e57f469f0d3b9",
        sha_src = "0bcd1f2e71cadb05ce3033c0de51d13072345bf3c20b879a2a6f015a4b3774b0",
    ),
    struct(
        name = "org.eclipse.jface",
        version = "3.22.0.v20201106-0834",
        sha = "750496b3e8b4ed840715ee587daa75337f9d69c935c76f3f8f5a7b44fb2c2889",
        sha_src = "5ec32308acfccede45c64821ba80e3f48fe725506d2f01c6048c53643ddee971",
    ),
    struct(
        name = "org.eclipse.jface.databinding",
        version = "1.12.100.v20201014-0742",
        sha = "0f55e83e0784f8262160cd29650502b11b5a8340855c23da58daf8ef9524e221",
        sha_src = "c5ed08a11e9b67afd8135889c29ecc03da5832fcbc1772ae4f11d7e33a35f3c0",
    ),
    struct(
        name = "org.eclipse.jface.text",
        version = "3.16.500.v20201112-1545",
        sha = "29394d51209ce165af0d2069a9eff6608e96be4eb2f7f629b632e5b9dc80db6c",
        sha_src = "ea62f314e1cfb8b11bace8b0f214dfe1a773db1bae92fc02d569d04fabe8f37c",
    ),
    struct(
        name = "org.eclipse.osgi",
        version = "3.16.100.v20201030-1916",
        sha = "b3a99dc58841dc0073c572ecc6ceff926ef127b933dcdd5e8e5105ae0edfec25",
        sha_src = "04b0b0746a0666b6e31e2c246c3d9279b2538a74ff6dcd5feb46dea6faed8bd6",
    ),
    struct(
        name = "org.eclipse.text",
        version = "3.10.400.v20200925-0557",
        sha = "91bf3fa0e1043dd7437eadff0e0a96b980d17d89a02b97e9a5492d1a6eb37c73",
        sha_src = "257d03d0aab4aa535c26fbb90c6f0881bac2a515331e02e81fe0c78bf032ec5b",
    ),
]

def _jface_impl(repository_ctx):
    for lib in _LIBS:
        repository_ctx.download("{}{}_{}.jar".format(_BASE, lib.name, lib.version),
            output = lib.name + ".jar",
            sha256 = lib.sha,
        )
        repository_ctx.download("{}{}.source_{}.jar".format(_BASE, lib.name, lib.version),
            output = lib.name + "-sources.jar",
            sha256 = lib.sha_src,
        )

    repository_ctx.file("BUILD.bazel",
        content = "\n".join([
            "java_import(",
            "  name = \"jface\",",
            "  jars = [",
        ] + [ "\"{}.jar\",".format(lib.name) for lib in _LIBS ] + [
            "  ],",
            "  visibility = [\"//visibility:public\"],",
            ")"
        ])
    )

jface = repository_rule(
    implementation = _jface_impl,
)
