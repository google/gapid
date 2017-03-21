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

#include "gapir/cc/gles_gfx_api.h"
#include "gles_renderer.h"

#include "server_connection.h"
#include "server_listener.h"

#include "core/cc/connection.h"
#include "core/cc/log.h"
#include "core/cc/supported_abis.h"

#include <string.h>

#include <memory>
#include <sstream>
#include <string>

namespace {

const uint32_t kProtocolVersion = 1;
const char kAuthTokenHeader[] = { 'A', 'U', 'T', 'H' };

}  // anonymous namespace

namespace gapir {

ServerListener::ServerListener(std::unique_ptr<core::Connection> conn, uint64_t maxMemorySize) :
        mConn(std::move(conn)),
        mMaxMemorySize(maxMemorySize) {
}

std::unique_ptr<ServerConnection> ServerListener::acceptConnection(int idleTimeoutMs, const char* authToken) {
    while (true) {
        GAPID_DEBUG("Waiting for new connection...");
        std::unique_ptr<core::Connection> client = mConn->accept(idleTimeoutMs);
        if (client == nullptr) {
            return nullptr;
        }

        if (authToken != nullptr) {
            GAPID_DEBUG("Checking auth-token...");
            char header[sizeof(kAuthTokenHeader)];
            if (client->recv(&header, sizeof(header)) != sizeof(header)) {
                GAPID_WARNING("Failed to read auth-token header");
                continue;
            }
            if (memcmp(header, kAuthTokenHeader, sizeof(kAuthTokenHeader)) != 0) {
                GAPID_WARNING("Invalid auth-token header");
                continue;
            }
            std::string gotAuthToken;
            if (!client->readString(&gotAuthToken) || gotAuthToken != authToken) {
                GAPID_WARNING("Invalid auth-token");
                continue;
            }
        }

        uint8_t connectionType;
        if (client->recv(&connectionType, sizeof(connectionType)) != sizeof(connectionType)) {
            GAPID_WARNING("Failed to read connection type");
            continue;
        }

        switch (connectionType) {
            case REPLAY_REQUEST: {
                GAPID_INFO("Replay requested");
                std::unique_ptr<ServerConnection> conn = ServerConnection::create(std::move(client));
                if (conn != nullptr) {
                    return conn;
                } else {
                    GAPID_WARNING("Loading ServerConnection failed!");
                }
                break;
            }
            case SHUTDOWN_REQUEST: {
                GAPID_INFO("Shutdown request received!");
                return nullptr;
            }
            case PING: {
                client->sendString("PONG");
                break;
            }
            default: {
                GAPID_WARNING("Unknown connection type %d ignored", connectionType);
            }
        }
    }
}

}  // namespace gapir
