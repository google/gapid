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

# Repository rule to download and extract SWT.
_URL_LINUX = "https://download.eclipse.org/eclipse/downloads/drops4/R-4.18-202012021800/swt-4.18-gtk-linux-x86_64.zip"
_SHA_LINUX = "25eb95522f67e68c24965373124776860da29f32b95cec3dc67c666d81f9460f"
_URL_WIN = "https://download.eclipse.org/eclipse/downloads/drops4/R-4.18-202012021800/swt-4.18-win32-win32-x86_64.zip"
_SHA_WIN = "89f7cb23c41c5642c35a3039b58abfdbafd6e01b407cea67fa769847446e0dae"
_URL_OSX = "https://download.eclipse.org/eclipse/downloads/drops4/R-4.18-202012021800/swt-4.18-cocoa-macosx-x86_64.zip"
_SHA_OSX = "ae37d150caded21e94731a0a6662403d97918df57c36da2a5d257eabf10d76a2"

def _swt_impl(repository_ctx):
    url = ""
    sha = ""
    if repository_ctx.os.name.startswith("linux"):
        url = _URL_LINUX
        sha = _SHA_LINUX
    elif repository_ctx.os.name.startswith("windows"):
        url = _URL_WIN
        sha = _SHA_WIN
    elif repository_ctx.os.name.startswith("mac os"):
        url = _URL_OSX
        sha = _SHA_OSX
    else:
        fail("No SWT available for os: " + repository_ctx.os.name)

    repository_ctx.download_and_extract(
        url = url,
        output = ".",
        sha256 = sha,
    )
    repository_ctx.file("BUILD.bazel",
        content = """
java_import(
    name = "swt",
    jars = ["swt.jar"],
    visibility = ["//visibility:public"]
)
""",
    )

swt = repository_rule(
    implementation = _swt_impl,
)
