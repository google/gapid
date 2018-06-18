/*
 * Copyright (C) 2017 Google Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

#include "connection_stream.h"

#include "core/cc/log.h"
#include "core/cc/socket_connection.h"

namespace gapii {

std::shared_ptr<ConnectionStream> ConnectionStream::listenSocket(
    const char* hostname, const char* port) {
  auto c = core::SocketConnection::createSocket(hostname, port);
  GAPID_INFO("GAPII awaiting connection on socket %s:%s", hostname, port);
  return std::shared_ptr<ConnectionStream>(new ConnectionStream(c->accept()));
}

std::shared_ptr<ConnectionStream> ConnectionStream::listenPipe(
    const char* pipename, bool abstract) {
  auto c = core::SocketConnection::createPipe(pipename, abstract);
  GAPID_INFO("GAPII awaiting connection on pipe %s%s", pipename,
             (abstract ? " (abstract)" : ""));
  return std::shared_ptr<ConnectionStream>(new ConnectionStream(c->accept()));
}

ConnectionStream::ConnectionStream(std::unique_ptr<core::Connection> connection)
    : mConnection(std::move(connection)) {}

uint64_t ConnectionStream::read(void* data, uint64_t max_size) {
  return mConnection->recv(data, max_size);
}

uint64_t ConnectionStream::write(const void* data, uint64_t size) {
  return mConnection->send(data, size);
}

void ConnectionStream::close() { mConnection->close(); }

}  // namespace gapii
