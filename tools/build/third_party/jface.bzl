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

_BASE = "https://download.eclipse.org/eclipse/updates/4.21/R-4.21-202109060500/plugins/"
_LIBS = [
    struct(
        name = "org.eclipse.core.commands",
        version = "3.10.100.v20210722-1426",
        sha = "177c605efd78681e28765b869a2ff5284a79b02d133007a6169a64317cee8633",
        sha_src = "767ffe3b4c9e345d82666cc27d0453df487fbe69e3df3b871d2eaa33e395da7a",
    ),
    struct(
        name = "org.eclipse.core.runtime",
        version = "3.23.0.v20210730-2035",
        sha = "c81c0fd5c3cb632c93586c80f31461749f43104e3499f53e0cf33525bd606ce3",
        sha_src = "c90874b9868149603e4370431db9f87565b47aa566a88001854a9a7cbaad012b",
    ),
    struct(
        name = "org.eclipse.equinox.common",
        version = "3.15.0.v20210518-0604",
        sha = "90e1b2a17b6e9256e3bfae0fc12b287cd96bc1481d64d83e0c7c30f4c12b248b",
        sha_src = "25632e7df81e57696223a844d826323c62e46403e0baa21a2c2546c80578a1c3",
    ),
    struct(
        name = "org.eclipse.jface",
        version = "3.23.0.v20210723-1324",
        sha = "14150fd90a0b095ee45051c94764e8f9908f58578398055e40adcb33a4242798",
        sha_src = "6050d1d29f9d76782d3c2afc34895cac77acf23f59e7fc826da4aca0f87516fe",
    ),
    struct(
        name = "org.eclipse.jface.databinding",
        version = "1.13.0.v20210619-1146",
        sha = "b97d4c975e35fd6e7e785f1c9c42aa6a2540642a9dfcb5145fe2d34758940911",
        sha_src = "97057ac3770e47144f235c6e7c8cc12c1ce2c668f1386d3ed378b920cf05c028",
    ),
    struct(
        name = "org.eclipse.jface.text",
        version = "3.18.100.v20210820-1651",
        sha = "bac1aea8eb813eaca2da9212c1079b577611f5010d6f93cd509ae346356bee29",
        sha_src = "c9ebb5b99ce8a023b4bad86cd1f80fe0be8f3210938eb285c9a7d8510f7b55c3",
    ),
    struct(
        name = "org.eclipse.osgi",
        version = "3.17.0.v20210823-1805",
        sha = "b9b5cf6bb057b94be55f3510bdd831cdbaef16f2d1cab6b3770b72452811e538",
        sha_src = "c42096fadeb3367dfdc63fec3ddc1da161310c6f70f16c2e2b30958b9a8e1d30",
    ),
    struct(
        name = "org.eclipse.text",
        version = "3.12.0.v20210512-1644",
        sha = "56d3c997d0c60916012f71cdb7d4b25245fe1eb82775d2d0dc83e432e71f220a",
        sha_src = "49067b4537181a43ab2883944d82ec8c102aff72e1b54f77f75179dd8480269b",
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
