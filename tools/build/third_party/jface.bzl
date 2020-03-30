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

_BASE = "https://download.eclipse.org/eclipse/updates/4.15/R-4.15-202003050155/plugins/"
_LIBS = [
    struct(
        name = "org.eclipse.core.commands",
        version = "3.9.700.v20191217-1850",
        sha = "8b7f98682b827fb5e1ba22d95e701e41ea6857552ada691db4c472566a246d7b",
        sha_src = "00ec3c2d0dbb24216b6055232f6ca2b0cccf9cb750e4f0af11dca053aad7e050",
    ),
    struct(
        name = "org.eclipse.core.runtime",
        version = "3.17.100.v20200203-0917",
        sha = "b07d2b9a5a87b340266fd8d36da9b6ac35993b15f22a13e9e767c0e0b4284e19",
        sha_src = "de7555cc1f3d5546ab65012d3adff17e637562dc96da827269802a79fd450460",
    ),
    struct(
        name = "org.eclipse.equinox.common",
        version = "3.11.0.v20200206-0817",
        sha = "0aec4833fa9773547ea39529e2955c722d79622f11314ba95a7b28aaf75c7364",
        sha_src = "09e2577bdc6448ffdc6312fd4918b539e066ec52fa5201481d90993fd8ac83a5",
    ),
    struct(
        name = "org.eclipse.jface",
        version = "3.19.0.v20200218-1607",
        sha = "37446fc4af699c57a80aab9f301c0397a56b556214c74b6102f262dd4387c1c4",
        sha_src = "317047abc92ad7a3c3627d8cd6ebbd4bf326686813ca4d24e9de2ff4e723fdd5",
    ),
    struct(
        name = "org.eclipse.jface.databinding",
        version = "1.11.0.v20200205-2119",
        sha = "5f360fe957d3e2915580a1e528794573133db68d41ac475209bd66654f030619",
        sha_src = "45bbe2e3b3727bb08790c56c632141f1121ebd883ce73b22f76b6f038e34fec8",
    ),
    struct(
        name = "org.eclipse.jface.text",
        version = "3.16.200.v20200218-0828",
        sha = "d0de8da9f5e82e85c6b3e6f83ef1ac4b74461da3b960a06b315d236058eccc7b",
        sha_src = "ed852d4d131c3ae5c24a6067ff9dd9314d3a4959a54d902563c7c9701ae33558",
    ),
    struct(
        name = "org.eclipse.osgi",
        version = "3.15.200.v20200214-1600",
        sha = "c8fad365d0eb989bc8be991bcb071abc9f9ad101c890795d284f2e4621fcee1d",
        sha_src = "2f004645e7da73499f1018d468ee612d68cc808c3bee38e29dfed659816ecdfc",
    ),
    struct(
        name = "org.eclipse.text",
        version = "3.10.100.v20200217-1239",
        sha = "b693e39e13bcc7483758daea5d67d91f69900b89e9f778578bb8cf6442d2075d",
        sha_src = "ef7fdd60ad4d8c9de9d133e7b91fd6ef4804330781797a20f1837b3a4abf08fc",
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
