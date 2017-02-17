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

#ifndef GAPIR_SERVER_CONNECTION_H
#define GAPIR_SERVER_CONNECTION_H

#include "resource.h"

#include <memory>
#include <string>
#include <vector>

namespace core {

class Connection;

}  // namespace core


namespace gapir {

// Class for managing the communication between the replay daemon and the server (gazer)
class ServerConnection {
public:
    // Creates a gazer connection using the given connection
    static std::unique_ptr<ServerConnection> create(std::unique_ptr<core::Connection> conn);

    ~ServerConnection();

    // Returns the resource id of the replay data
    const std::string& replayId() const;

    // Returns the length of the replay data
    uint32_t replayLength() const;

    // Fetch the specified resources to the specified target address from the server. The resources
    // are loaded into the memory address continuously in the order they are specified in the id
    // list. Size have to specify the sum size of the requested resources. The function returns true
    // if fetching of the resources was successful false otherwise.
    bool getResources(const ResourceId* ids, size_t count, void* target, size_t size) const;

    // Post a blob of data from the given address with the given size to the server. Returns true if
    // the posting was successful false otherwise.
    bool post(const void* postData, uint32_t postSize) const;

    // Type of the message sent to the server. It have to be consistent with the values expected by
    // the server
    enum MessageType : uint8_t {
        MESSAGE_TYPE_GET  = 0,
        MESSAGE_TYPE_POST = 1,
    };

private:
    // Initialize the member variables of the ServerConnection object
    ServerConnection(std::unique_ptr<core::Connection> conn, const std::string& replayId,
                    uint32_t replayLen);

    // The connection used for sending and receiving data to and from the server.
    std::unique_ptr<core::Connection> mConn;

    // The length of the replay this connection belongs to.
    uint32_t mReplayLen;

    // The resource id of the replay this request belongs to.
    std::string mReplayId;
};

}  // namespace gapir

#endif  // GAPIR_SERVER_CONNECTION_H
