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

#ifndef GAPII_CONNECTION_STREAM_H
#define GAPII_CONNECTION_STREAM_H

#include <memory>

#include "core/cc/stream_reader.h"
#include "core/cc/stream_writer.h"

namespace core {

class Connection;

}  // namespace core

namespace gapii {

// ConnectionStream is an implementation of the StreamReader and StreamWriter
// interfaces that reads and writes to an incoming TCP connection.
class ConnectionStream : public core::StreamWriter, public core::StreamReader {
 public:
  // listenSocket blocks and waits for a TCP connection to be made on the
  // specified host and port, returning a ConnectionStream once a connection is
  // established.
  static std::shared_ptr<ConnectionStream> listenSocket(const char* hostname,
                                                        const char* port);

  // listenPipe blocks and waits for a UNIX connection to be made on the
  // specified pipe name, optionally abstract, returning a ConnectionStream once
  // a connection is established.
  static std::shared_ptr<ConnectionStream> listenPipe(const char* pipename,
                                                      bool abstract);

  // core::StreamReader compliance
  virtual uint64_t read(void* data, uint64_t max_size) override;

  // core::StreamWriter compliance
  virtual uint64_t write(const void* data, uint64_t size) override;

  // Closes the connection stream
  virtual void close();

 private:
  ConnectionStream(std::unique_ptr<core::Connection>);

  std::unique_ptr<core::Connection> mConnection;
};

}  // namespace gapii

#endif  // GAPII_CONNECTION_STREAM_H
