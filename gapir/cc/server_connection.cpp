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

#include "server_connection.h"

#include "core/cc/connection.h"
#include "core/cc/log.h"

#include <memory>
#include <string>
#include <vector>

namespace gapir {

std::unique_ptr<ServerConnection> ServerConnection::create(
        std::unique_ptr<core::Connection> conn) {
    std::string replayId;
    if (!conn->readString(&replayId)) {
        GAPID_WARNING("Failed to read replay id. Error: %s", conn->error());
        return nullptr;
    }

    uint32_t replayLen;
    if (conn->recv(&replayLen, sizeof(replayLen)) != sizeof(replayLen)) {
        GAPID_WARNING("Failed to read replay length. Error: %s", conn->error());
        return nullptr;
    }

    return std::unique_ptr<ServerConnection>(
            new ServerConnection(std::move(conn), replayId, replayLen));
}

ServerConnection::ServerConnection(std::unique_ptr<core::Connection> conn,
        const std::string& replayId, uint32_t replayLen) :
        mConn(std::move(conn)),
        mReplayLen(replayLen),
        mReplayId(replayId) {
}

ServerConnection::~ServerConnection() {
}

const std::string& ServerConnection::replayId() const {
    return mReplayId;
}

uint32_t ServerConnection::replayLength() const {
    return mReplayLen;
}

bool ServerConnection::getResources(const ResourceId* resourceIds, size_t count, void* target,
                          size_t size) const {
    uint32_t c = static_cast<uint32_t>(count);

    GAPID_DEBUG("GET resources (count: %lu, size: %d, target: %p)", c, size, target);

    MessageType type = MESSAGE_TYPE_GET;
    if (mConn->send(&type, sizeof(type)) != sizeof(type)) {
        GAPID_WARNING("Failed to send GET messageType to the server. Error: %s", mConn->error());
        return false;
    }

    if (mConn->send(&c, sizeof(c)) != sizeof(c)) {
        GAPID_WARNING("Failed to send GET count to the server. Error: %s", mConn->error());
        return false;
    }

    uint64_t size64 = static_cast<uint64_t>(size);
    if (mConn->send(&size64, sizeof(size64)) != sizeof(size64)) {
        GAPID_WARNING("Failed to send GET size to the server. Error: %s", mConn->error());
        return false;
    }

    for (size_t i = 0; i < count; i++) {
        const ResourceId& id = resourceIds[i];
        if (!mConn->sendString(id)) {
            GAPID_WARNING("Failed to send GET resource id to the server. Error: %s",
                mConn->error());
            return false;
        }
    }

    size_t received = mConn->recv(target, size);
    if (received != size) {
        GAPID_WARNING("GET %lu resources returned unexpected size. "
            "Expected: 0x%x, Got: 0x%x. Error: %s\n",
            c, int(size), int(received), mConn->error());
        return false;
    }

    return true;
}

bool ServerConnection::post(const void* postData, uint32_t postSize) const {
    GAPID_DEBUG("POST: %p (%d)", postData, postSize);

    MessageType type = MESSAGE_TYPE_POST;
    if (mConn->send(&type, sizeof(type)) != sizeof(type)) {
        GAPID_WARNING("Failed to send POST messageType to the server. Error: %s", mConn->error());
        return false;
    }

    if (mConn->send(&postSize, sizeof(postSize)) != sizeof(postSize)) {
        GAPID_WARNING("Failed to send POST length to the server. Error: %s", mConn->error());
        return false;
    }

    if (mConn->send(postData, postSize) != postSize) {
        GAPID_WARNING("Failed to send POST content to the server. Error: %s", mConn->error());
        return false;
    }

    return true;
}

bool ServerConnection::crash(const std::string& filename, const void* crashData, uint32_t crashSize) const {
    GAPID_DEBUG("CRASH: [%s] %p (%d)", filename.c_str(), crashData, crashSize);

    MessageType type = MESSAGE_TYPE_CRASH;
    if (mConn->send(&type, sizeof(type)) != sizeof(type)) {
        GAPID_WARNING("Failed to send CRASH messageType to the server. Error: %s", mConn->error());
        return false;
    }

    if (!mConn->sendString(filename)) {
        GAPID_WARNING("Failed to send CRASH filename to the server. Error: %s", mConn->error());
        return false;
    }

    if (mConn->send(&crashSize, sizeof(crashSize)) != sizeof(crashSize)) {
        GAPID_WARNING("Failed to send CRASH length to the server. Error: %s", mConn->error());
        return false;
    }

    if (mConn->send(crashData, crashSize) != crashSize) {
        GAPID_WARNING("Failed to send CRASH content to the server. Error: %s", mConn->error());
        return false;
    }

    return true;
}

}  // namespace gapir
