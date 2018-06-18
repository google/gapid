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

#ifndef CORE_CONNECTION_H
#define CORE_CONNECTION_H

#include <cstddef>
#include <memory>
#include <string>

namespace core {

// Abstract base class for representing a connection to a remote with simple
// helper methods for sending specific data types.
class Connection {
 public:
  // Constant that can be passed to accept() to indicate that the call should
  // block with no timeouts until a connection is made, or the socket closes.
  static const int NO_TIMEOUT = -1;

  virtual ~Connection() {}

  // Try to send size bytes of data from the specified buffer using the
  // underlying connection. Will block if the connection is not ready. Returns
  // the number of bytes successfully sent (possibly less then size if an error
  // occurred).
  virtual size_t send(const void* data, size_t size) = 0;

  // Tries to read size bytes to the buffer specified by data on the underlying
  // connection and blocks until the data is available. Returns the number of
  // bytes successfully retrieved (possibly less then size if an error
  // occurred).
  virtual size_t recv(void* data, size_t size) = 0;

  // Returns the last error message raised by the connection.
  virtual const char* error() = 0;

  // Accept an incoming connection request on the underlying connection and
  // returns the new connection corresponding to it. The returned connection is
  // ready to use. If timeoutMS is NO_TIMEOUT, then the connection will block
  // forever.
  virtual std::unique_ptr<Connection> accept(int timeoutMS = NO_TIMEOUT) = 0;

  // Helper methods for sending and receiving strings
  bool sendString(const std::string& s);
  bool sendString(const char* s);
  bool readString(std::string* s);

  // Helper method for sending plain-old-data values.
  // Returns true if the send was successful, otherwise false.
  template <typename T>
  inline bool send(const T& data);

  // Closes the connection for read/write but leaves the object around.
  virtual void close() = 0;
};

template <typename T>
inline bool Connection::send(const T& data) {
  return send(&data, sizeof(T)) == sizeof(T);
}

}  // namespace core

#endif  // CORE_CONNECTION_H
