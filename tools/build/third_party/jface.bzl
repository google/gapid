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

_BASE = "http://download.eclipse.org/eclipse/updates/4.9/R-4.9-201809060745/plugins/"
_LIBS = [
    struct(
        name = "org.eclipse.core.commands",
        version = "3.9.200.v20180827-1727",
        sha = "66e3c28060f69bcd97eee01cbbebf1f2b4b5cbcd5c3ee145c7800941ed5b01fa",
        sha_src = "07d03873c7ca80aa59b62893a58a9d488d51526e0a5060aea9a57b43bd3303ae",
    ),
    struct(
        name = "org.eclipse.core.runtime",
        version = "3.15.0.v20180817-1401",
        sha = "50f839bda2b3e83f66c062037b4df6d5727a94c68f622715a6c10d36b2cf743a",
        sha_src = "c1ac4f2d43b11501dc67db923dbca73c1d4b0109fca04e5cb37b351420c8e665",
    ),
    struct(
        name = "org.eclipse.equinox.common",
        version = "3.10.100.v20180827-1235",
        sha = "87ef393c6693a5104ec6132816b25098685703f28b71a5ba36d67fa23a44be74",
        sha_src = "751eda9db7edf814cad685b6f56ac6c8940f4ccc67d52c0816f8f18722677df0",
    ),
    struct(
        name = "org.eclipse.jface",
        version = "3.14.100.v20180828-0836",
        sha = "372e38db82d248af007563578ceab3a7c0251057b0f2e2abc5fe337bf846d808",
        sha_src = "67f69f63260743aa5bde046956a00a350639d66fc25a58eae51fd1302dbe8b9f",
    ),
    struct(
        name = "org.eclipse.jface.databinding",
        version = "1.8.300.v20180828-0836",
        sha = "35ec8cf71b30e7cc6590ee20315d73ff24909c4d0fffc6e6dcd7b6a267f03e3a",
        sha_src = "a70010666241a5068ebb011d42d7b0eeb60492fa6ca4dc14237aba3ac6671a6d",
    ),
    struct(
        name = "org.eclipse.jface.text",
        version = "3.14.0.v20180824-1140",
        sha = "694159ecbfddeff0160b9d5200a13f049109820b48a7ac44093d85781f95d7c6",
        sha_src = "90af1d910ac777451c2d3a03ecf1ad659f7d28081e033c8f70b0520b9255a7cd",
    ),
    struct(
        name = "org.eclipse.osgi",
        version = "3.13.100.v20180827-1536",
        sha = "03e360b9152d270e6dd06822b8a1c2d5bb922ffb508dd634ba13f3b78bb72534",
        sha_src = "977b145de33c00c35b0b9c68ceb5ce10e11085edf062b46f70724201053dd119",
    ),
    struct(
        name = "org.eclipse.text",
        version = "3.7.0.v20180822-1511",
        sha = "c491fcf85ec450d4ddae592e4811bee75297872f7aaec8a728326f2789ce6a28",
        sha_src = "c84d3107fdba3c0059b71590011ad19407178834c407f7ad788b542da73555de",
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
