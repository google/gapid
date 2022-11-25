/*
 * Copyright (C) 2022 Google Inc.
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

#include "common.h"
#define WIN32_LEAN_AND_MEAN

#include <windows.h>
#include <winsock2.h>
#include <ws2tcpip.h>

#include <chrono>
#include <format>
#include <sstream>

#include "json.hpp"
#pragma comment(lib, "Ws2_32.lib")
#pragma comment(lib, "Mswsock.lib")
#pragma comment(lib, "AdvApi32.lib")

namespace {
class stream_messenger : public messenger {
 public:
  void send(const std::string& string) override {
    std::cout << string << std::endl;
  }
  nlohmann::json recv() {
    return nlohmann::json();
  }
};

class socket_messenger : public messenger {
 public:
  socket_messenger(SOCKET _s) : s(_s) {
  }
  ~socket_messenger() {
    closesocket(s);
    WSACleanup();
  }

  void send(const std::string& string) override {
    int total_sent = 0;
    int sent = 0;
    while (sent = ::send(s, string.c_str() + sent, string.size() - sent, 0)) {
      total_sent += sent;
      if (total_sent == string.size()) {
        break;
      }
    }

    if (total_sent != string.size()) {
      std::cerr << "Error could not send properly" << std::endl;
    }
  }

  struct recv_container {
    char* buffer;
    size_t& buffer_size;
    size_t& buffer_offset;
    size_t& buffer_filled;
    SOCKET s;

    void advance() {
      if (buffer_filled - buffer_offset != 0) {
        buffer_offset += 1;
        return;
      } else {
        int recvd = ::recv(s, buffer, buffer_size, 0);
        if (recvd < 0) {
          auto last_error = WSAGetLastError();
          std::cerr << "WsaError " << last_error << std::endl;
          return;
        }
        if (recvd <= 0) {
          std::cerr << "Could not receive" << std::endl;
          buffer = nullptr;
          return;
        }
        buffer_filled = recvd;
        buffer_offset = 0;
      }
    }

    const char& get_current() {
      if (buffer_filled == 0 && buffer_offset == 0) {
        advance();
      }
      return buffer[buffer_offset];
    }
  };

  struct recv_iterator {
    using difference_type = std::ptrdiff_t;
    using value_type = char;
    using pointer = const char*;
    using reference = const char&;
    using iterator_category = std::input_iterator_tag;
    recv_container* container;

    recv_iterator& operator++() {
      container->advance();
      return *this;
    }

    bool operator!=(const recv_iterator& rhs) const {
      return rhs.container != container;
    }

    reference operator*() const {
      return container->get_current();
    }
  };

  recv_iterator begin(recv_container& tgt) {
    return recv_iterator{&tgt};
  }

  recv_iterator end(const recv_container&) {
    return {};
  }

  nlohmann::json
  recv() override {
    recv_container c{&recvbuf[0], buffer_size, buffer_offset, buffer_filled, s};

    nlohmann::json result;
    nlohmann::detail::json_sax_dom_parser<nlohmann::json> sdp(result, true);
    auto parse_result = nlohmann::json::sax_parse(begin(c), end(c), &sdp, nlohmann::detail::input_format_t::json, false);

    if (parse_result) {
      return std::move(result);
    }
    return nlohmann::json();
  }
  SOCKET s;
  char recvbuf[4096];
  size_t buffer_size = 4096;
  size_t buffer_offset = 0;
  size_t buffer_filled = 0;
};

}  // namespace

std::unique_ptr<messenger>& get_messenger() {
  static std::unique_ptr<messenger> f;
  return f;
}

void send(const std::string& str) {
  get_messenger()->send(str);
}

void connect_std_streams() {
  get_messenger() = std::make_unique<stream_messenger>();
}

bool connect_socket(const std::string& addr, const std::string& port) {
  WSADATA wsaData;
  WSAStartup(MAKEWORD(2, 2), &wsaData);
  SOCKET s = INVALID_SOCKET;
  struct addrinfo *result = NULL,
                  *ptr = NULL,
                  hints;
  int iResult;
  ZeroMemory(&hints, sizeof(hints));
  hints.ai_family = AF_UNSPEC;
  hints.ai_socktype = SOCK_STREAM;
  hints.ai_protocol = IPPROTO_TCP;

  iResult = getaddrinfo(addr.c_str(), port.c_str(), &hints, &result);
  if (iResult != 0) {
    std::cerr << "Could not get addrinfo for " << addr << std::endl;
    WSACleanup();
    return false;
  }

  // Attempt to connect to an address until one succeeds
  for (ptr = result; ptr != NULL; ptr = ptr->ai_next) {
    // Create a SOCKET for connecting to server
    s = socket(ptr->ai_family, ptr->ai_socktype,
               ptr->ai_protocol);
    if (s == INVALID_SOCKET) {
      std::cerr << "socket failed with error: " << WSAGetLastError() << std::endl;
      WSACleanup();
      return false;
    }

    // Connect to server.
    iResult = connect(s, ptr->ai_addr, (int)ptr->ai_addrlen);
    if (iResult == SOCKET_ERROR) {
      closesocket(s);
      s = INVALID_SOCKET;
      continue;
    }
    break;
  }
  freeaddrinfo(result);
  if (s == INVALID_SOCKET) {
    std::cerr << "Could not connect to " << addr << ":" << port << std::endl;
    WSACleanup();
    return false;
  }

  get_messenger() = std::make_unique<socket_messenger>(s);
  return true;
}

static auto begin = std::chrono::high_resolution_clock::now();

float get_time() {
  return std::chrono::duration<float>(std::chrono::high_resolution_clock::now() - begin).count();
}

void output_message(message_type type, const std::string& str, uint32_t layer_index) {
  std::string t;
  switch (type) {
    case message_type::error:
      t = "Error";
      break;
    case message_type::info:
      t = "Info";
      break;
    case message_type::critical:
      t = "Critical";
      break;
    case message_type::debug:
      t = "Debug";
      break;
  }

  std::stringstream ss;
  ss << "{ \"";
  ss << "Message\":\"";
  ss << t;
  ss << "\"";
  ss << ",\"Time\":";
  ss << get_time();
  if (layer_index != static_cast<uint32_t>(-1)) {
    ss << ", \"LayerIndex\" : " << layer_index;
  }
  ss << ", \"Content\": ";
  auto j = nlohmann::json::basic_json(str);
  ss << j.dump();
  ss << " }";
  send(ss.str());
}

void send_layer_data(const char* str, size_t length, uint64_t layer_index) {
  std::stringstream ss;
  ss << "{ \"";
  ss << "Message\":\"Object\"";
  ss << ",\"Time\":";
  ss << get_time();
  if (layer_index != static_cast<uint32_t>(-1)) {
    ss << ", \"LayerIndex\" : " << layer_index;
  }
  ss << ", \"Content\": ";
  ss.write(str, length);
  ss << " }";

  send(ss.str());
}

void send_layer_log(message_type type, const char* str, size_t length, uint64_t layer_index) {
  output_message(type, std::string(str, length), layer_index);
}

nlohmann::json receive_message() {
  return get_messenger()->recv();
}