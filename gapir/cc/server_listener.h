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

#ifndef GAPIR_GAZER_LISTENER_H
#define GAPIR_GAZER_LISTENER_H

#include <memory>

namespace core {

class Connection;

}  // namespace core


namespace gapir {

class ServerConnection;

// Class for listening to incoming connections from the server.
class ServerListener {
public:
    // Construct a ServerListener using the specified connection.
    // maxMemorySize is the maximum memory size that can be reported as
    // supported by this device.
    explicit ServerListener(core::Connection* conn, uint64_t maxMemorySize);

    // Accept a new incoming connection on the underlying socket and create a ServerConnection over
    // the newly created socket object. idleTimeoutMs is the timeout in milliseconds to wait for
    // activity before returning a null pointer. Pass core::Connection::NO_TIMEOUT to disable the
    // timeout.
    std::unique_ptr<ServerConnection> acceptConnection(int idleTimeoutMs, const char* authToken);

    enum ConnectionType {
        REPLAY_REQUEST   = 0,
        SHUTDOWN_REQUEST = 1,
        PING             = 2,
    };

private:
    // The underlying server socket for the listener
    core::Connection* mConn;
    // The maximum memory size that can be reported as supported by this device.
    uint64_t mMaxMemorySize;
};

}  // namespace gapir

#endif  // GAPIR_SOCKET_LISTENER_H
