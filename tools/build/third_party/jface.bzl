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

_BASE = "https://download.eclipse.org/eclipse/updates/4.24/R-4.24-202206070700/plugins/"
_LIBS = [
    struct(
        name = "org.eclipse.core.commands",
        version = "3.10.200.v20220512-0851",
        sha = "35af2254a0b6d8d5b328853739882ff4baae734b6d19e82c955426e470890a25",
        sha_src = "7cd3d55fbeec09178e635dd22074a22c9c984e1c12f499cc5063ef6252e48432",
    ),
    struct(
        name = "org.eclipse.core.runtime",
        version = "3.25.0.v20220506-1157",
        sha = "a6f61da3c508c618ea805188a84db9d798528ad5f3039c1d594833db7412897e",
        sha_src = "35336312aa5182afa4bd987b5d1fc0c4425ca225f6ed4e0182c0c6a64866650b",
    ),
    struct(
        name = "org.eclipse.equinox.common",
        version = "3.16.100.v20220315-2327",
        sha = "8b9d998c2ed00e05cbca44b6b8464586ca362a6d347790289e3602566c607c7e",
        sha_src = "797f439cd37d76589a85e402ba52f79e5300e094e3030da26dd823bd0a7766f2",
    ),
    struct(
        name = "org.eclipse.jface",
        version = "3.26.0.v20220513-0449",
        sha = "7d13ef252ccf2e12b4afc80b3dd18066aa1b3d9783cc87daeb6d70a6445ec8db",
        sha_src = "8e974a8f4be4b7a8434aef420b3fc402f0c3558cce50abfe54e3f2f742a6d5fa",
    ),
    struct(
        name = "org.eclipse.jface.databinding",
        version = "1.13.0.v20210619-1146",
        sha = "b97d4c975e35fd6e7e785f1c9c42aa6a2540642a9dfcb5145fe2d34758940911",
        sha_src = "97057ac3770e47144f235c6e7c8cc12c1ce2c668f1386d3ed378b920cf05c028",
    ),
    struct(
        name = "org.eclipse.jface.text",
        version = "3.20.100.v20220516-0819",
        sha = "1be756b8f7f2ec4ab3c1c4e1b0887175cf2a57594d54b1287032438026ec0f91",
        sha_src = "089d13957165bbf9a4d922c852c2944433552d426bc6637ed6ed127e00a8d799",
    ),
    struct(
        name = "org.eclipse.osgi",
        version = "3.18.0.v20220516-2155",
        sha = "e4feacfbe8843b67608beeaff8a9513654902767999d6e3de941c6d4b85c9a1e",
        sha_src = "6c17c695faf39039229148f756c50a720c5367c47297ea1722d75f71725948d7",
    ),
    struct(
        name = "org.eclipse.text",
        version = "3.12.100.v20220506-1404",
        sha = "d8f34bda39d3917c06817cc5bb98b382bc8c395777d92f611339f8c01acbaa73",
        sha_src = "834bb6b725cdd047a04ddfd09a3d978cf12a1b4ce7ea0baf00efdaec2fddc973",
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
