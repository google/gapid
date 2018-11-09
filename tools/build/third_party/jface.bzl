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
_VERSIONS = {
    "org.eclipse.core.commands": "3.9.200.v20180827-1727",
    "org.eclipse.core.runtime": "3.15.0.v20180817-1401",
    "org.eclipse.equinox.common": "3.10.100.v20180827-1235",
    "org.eclipse.jface": "3.14.100.v20180828-0836",
    "org.eclipse.jface.databinding": "1.8.300.v20180828-0836",
    "org.eclipse.jface.text": "3.14.0.v20180824-1140",
    "org.eclipse.osgi": "3.13.100.v20180827-1536",
    "org.eclipse.text": "3.7.0.v20180822-1511",
}

def _jface_impl(repository_ctx):
    for lib in _VERSIONS:
        repository_ctx.download("{}{}_{}.jar".format(_BASE, lib, _VERSIONS[lib]),
            output = lib + ".jar",
        )
        repository_ctx.download("{}{}.source_{}.jar".format(_BASE, lib, _VERSIONS[lib]),
            output = lib + "-sources.jar",
        )

    repository_ctx.file("BUILD.bazel",
        content = "\n".join([
            "java_import(",
            "  name = \"jface\",",
            "  jars = [",
        ] + [ "\"{}.jar\",".format(lib) for lib in _VERSIONS ] + [
            "  ],",
            "  visibility = [\"//visibility:public\"],",
            ")"
        ])
    )

jface = repository_rule(
    implementation = _jface_impl,
)
