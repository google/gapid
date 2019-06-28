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

#ifndef CORE_SOCKET_CONNECTION_H
#define CORE_SOCKET_CONNECTION_H

#include <stdint.h>

#include <memory>

#include "connection.h"

namespace core {

// Connection object using a native socket
class SocketConnection : public Connection {
 public:
  ~SocketConnection();

  // Creates a new socket connection listening on the specified hostname and
  // port. Returns a connection object on successful open or a nullptr if
  // opening the connection is unsuccessful
  static std::unique_ptr<Connection> createSocket(const char* hostname,
                                                  const char* port);

  // Returns a free port to use with the given hostname. In case of error,
  // returns 0
  static uint32_t getFreePort(const char* hostname);

  // Creates a new pipe connection listening on the specified UNIX pipename,
  // without pipe creation on the local file system if abstract is true. Returns
  // a connection object on successful open or a nullptr if opening the
  // connection is unsuccessful
  static std::unique_ptr<Connection> createPipe(const char* pipename,
                                                bool abstract);

  // Implementation of the Connection interface
  size_t send(const void* data, size_t size) override;
  size_t recv(void* data, size_t size) override;
  const char* error() override;
  std::unique_ptr<Connection> accept(int timeoutMs = NO_TIMEOUT) override;

  void close() override;

 private:
  // Private constructor used only by the static create function
  explicit SocketConnection(int socket);

  // Network initializer class to handle the initialization of the network
  // driver. A socket function can be called only if there is at least one
  // active network initializer is in the system
  struct NetworkInitializer {
    NetworkInitializer();
    ~NetworkInitializer();
  };

  // The underlying socket for the connection
  int mSocket;

  // Network initializer instance lasts for the lifetime of the connection
  NetworkInitializer mNetworkInitializer;
};

}  // namespace core

#endif
