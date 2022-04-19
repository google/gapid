// Copyright (C) 2022 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

#include <lib/async-loop/cpp/loop.h>
#include <lib/async-loop/default.h>
#include <lib/fdio/directory.h>
#include <lib/fdio/io.h>
#include <lib/vfs/cpp/pseudo_dir.h>
#include <lib/vfs/cpp/remote_dir.h>
#include <zircon/processargs.h>

// Serve /pkg as /pkg in the outgoing directory.
int main(int argc, const char* const* argv) {
  async::Loop loop(&kAsyncLoopConfigAttachToCurrentThread);
  zx::channel client_end, server_end;
  zx_status_t status = zx::channel::create(0, &client_end, &server_end);
  if (status != ZX_OK) {
    fprintf(stderr, "Couldn't create channel, %d\n", status);
    return -1;
  }
  status = fdio_open("/pkg",
                     static_cast<uint32_t>(fuchsia::io::OPEN_RIGHT_READABLE |
                                           fuchsia::io::OPEN_RIGHT_EXECUTABLE),
                     server_end.release());
  if (status != ZX_OK) {
    fprintf(stderr, "Failed to open /pkg");
    return -1;
  }

  vfs::PseudoDir root_dir;
  root_dir.AddEntry("pkg", std::make_unique<vfs::RemoteDir>(std::move(client_end)));

  status = root_dir.Serve(
      fuchsia::io::OPEN_RIGHT_READABLE | fuchsia::io::OPEN_RIGHT_EXECUTABLE,
      zx::channel(zx_take_startup_handle(PA_DIRECTORY_REQUEST)));

  if (status != ZX_OK) {
    fprintf(stderr, "Failed to serve outgoing.");
    return -1;
  }

  loop.Run();
  return 0;
}
