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

#include "socket_connection.h"

#include "core/cc/log.h"
#include "core/cc/target.h"

#include <stdint.h>
#include <stdlib.h>
#include <string.h>

#if TARGET_OS == GAPID_OS_WINDOWS

#define _WSPIAPI_EMIT_LEGACY
#include <winsock2.h>
#include <ws2tcpip.h>

#else  // TARGET_OS == GAPID_OS_WINDOWS

#include <errno.h>
#include <netdb.h>
#include <netinet/in.h>
#include <sys/socket.h>
#include <sys/types.h>
#include <sys/un.h>
#include <unistd.h>

#endif  // TARGET_OS == GAPID_OS_WINDOWS

namespace core {
namespace {

static const int ACCEPT_ERROR = -1;
static const int ACCEPT_TIMEOUT = -2;

#if TARGET_OS == GAPID_OS_WINDOWS
int gWinsockUsageCount = 0;
#endif  // TARGET_OS == GAPID_OS_WINDOWS

template <typename T>
size_t clamp_size_t(T val) {
  return static_cast<size_t>(val > 0 ? val : 0);
}

int error() {
#if TARGET_OS == GAPID_OS_WINDOWS
  return ::WSAGetLastError();
#else   // TARGET_OS == GAPID_OS_WINDOWS
  return errno;
#endif  // TARGET_OS == GAPID_OS_WINDOWS
}

void close(int fd) {
#if TARGET_OS == GAPID_OS_WINDOWS
  ::closesocket(fd);
#else   // TARGET_OS == GAPID_OS_WINDOWS
  ::close(fd);
#endif  // TARGET_OS == GAPID_OS_WINDOWS
}

size_t recv(int sockfd, void* buf, size_t len, int flags) {
#if TARGET_OS == GAPID_OS_WINDOWS
  return clamp_size_t(
      ::recv(sockfd, static_cast<char*>(buf), static_cast<int>(len), flags));
#else   // TARGET_OS == GAPID_OS_WINDOWS
  return clamp_size_t(::recv(sockfd, buf, len, flags));
#endif  // TARGET_OS == GAPID_OS_WINDOWS
}

size_t send(int sockfd, const void* buf, size_t len, int flags) {
#if TARGET_OS == GAPID_OS_WINDOWS
  return clamp_size_t(::send(sockfd, static_cast<const char*>(buf),
                             static_cast<int>(len), flags));
#else   // TARGET_OS == GAPID_OS_WINDOWS
  const size_t result = len;
  while (len > 0u) {
    const ssize_t n = ::send(sockfd, buf, len, flags);
    if (n != -1) {
      // A signal after some data was transmitted can result in partial send.
      buf = reinterpret_cast<const char*>(buf) + n;
      len -= n;
    } else if (errno == EINTR) {
      // A signal occurred before any data was transmitted - retry.
      continue;
    } else {
      // Other error.
      return 0u;
    }
  }
  return result;
#endif  // TARGET_OS == GAPID_OS_WINDOWS
}

int accept(int sockfd, int timeoutMs) {
  if (timeoutMs != Connection::NO_TIMEOUT) {
    fd_set set;
    FD_ZERO(&set);
    FD_SET(sockfd, &set);

    timeval timeout;
    timeout.tv_sec = timeoutMs / 1000;
    timeout.tv_usec = (timeoutMs - timeout.tv_sec * 1000) * 1000;

    switch (select(sockfd + 1, &set, NULL, NULL, &timeout)) {
      case -1:
        GAPID_WARNING("Error from select(): %s", strerror(error()));
        return ACCEPT_ERROR;
      case 0:
        return ACCEPT_TIMEOUT;
      default:
        break;
    }
  }

  return static_cast<int>(::accept(sockfd, nullptr, nullptr));
}

int getaddrinfo(const char* node, const char* service,
                const struct addrinfo* hints, struct addrinfo** res) {
  return ::getaddrinfo(node, service, hints, res);
}

void freeaddrinfo(struct addrinfo* res) { ::freeaddrinfo(res); }

int setsockopt(int sockfd, int level, int optname, const void* optval,
               size_t optlen) {
#if TARGET_OS == GAPID_OS_WINDOWS
  return ::setsockopt(sockfd, level, optname, static_cast<const char*>(optval),
                      static_cast<int>(optlen));
#else   // TARGET_OS == GAPID_OS_WINDOWS
  return ::setsockopt(sockfd, level, optname, optval, optlen);
#endif  // TARGET_OS == GAPID_OS_WINDOWS
}

int socket(int domain, int type, int protocol) {
  return static_cast<int>(::socket(domain, type, protocol));
}

int listen(int sockfd, int backlog) { return ::listen(sockfd, backlog); }

int bind(int sockfd, const struct sockaddr* addr, size_t addrlen) {
#if TARGET_OS == GAPID_OS_WINDOWS
  return ::bind(sockfd, addr, static_cast<int>(addrlen));
#else   // TARGET_OS == GAPID_OS_WINDOWS
  return ::bind(sockfd, addr, addrlen);
#endif  // TARGET_OS == GAPID_OS_WINDOWS
}

int getsockname(int sockfd, struct sockaddr* addr, socklen_t* addrlen) {
#if TARGET_OS == GAPID_OS_WINDOWS
  return ::getsockname(sockfd, addr, static_cast<int*>(addrlen));
#else   // TARGET_OS == GAPID_OS_WINDOWS
  return ::getsockname(sockfd, addr, addrlen);
#endif  // TARGET_OS == GAPID_OS_WINDOWS
}

}  // anonymous namespace

SocketConnection::SocketConnection(int socket) : mSocket(socket) {}

SocketConnection::~SocketConnection() { core::close(mSocket); }

size_t SocketConnection::send(const void* data, size_t size) {
  return core::send(mSocket, data, size, 0);
}

size_t SocketConnection::recv(void* data, size_t size) {
  return core::recv(mSocket, data, size, MSG_WAITALL);
}

const char* SocketConnection::error() { return strerror(core::error()); }

void SocketConnection::close() {
  core::close(mSocket);
  mSocket = -1;
}

std::unique_ptr<Connection> SocketConnection::accept(
    int timeoutMs /* = NO_TIMEOUT */) {
  int clientSocket = core::accept(mSocket, timeoutMs);
  switch (clientSocket) {
    case ACCEPT_ERROR:
      GAPID_WARNING("Failed to accept incoming connection: %s",
                    strerror(core::error()));
      return nullptr;
    case ACCEPT_TIMEOUT:
      GAPID_INFO("Timeout accepting incoming connection");
      return nullptr;
    default:
      return std::unique_ptr<Connection>(new SocketConnection(clientSocket));
  }
}

std::unique_ptr<Connection> SocketConnection::createSocket(const char* hostname,
                                                           const char* port) {
  // Network initializer to ensure that the network driver is initialized during
  // the lifetime of the create function. If the connection created successfully
  // then the new connection will hold a reference to a networkInitializer
  // struct to ensure that the network is initialized
  NetworkInitializer networkInitializer;

  struct addrinfo* addr;
  struct addrinfo hints {};
  hints.ai_family = AF_INET;
  hints.ai_socktype = SOCK_STREAM;

  const int getaddrinfoRes = core::getaddrinfo(hostname, port, &hints, &addr);
  if (0 != getaddrinfoRes) {
    GAPID_WARNING("getaddrinfo() failed: %d - %s.", getaddrinfoRes,
                  strerror(core::error()));
    return nullptr;
  }
  auto addrDeleter = [](struct addrinfo* ptr) {
    core::freeaddrinfo(ptr);
  };  // deferred.
  std::unique_ptr<struct addrinfo, decltype(addrDeleter)> addrScopeGuard(
      addr, addrDeleter);

  const int sock =
      core::socket(addr->ai_family, addr->ai_socktype, addr->ai_protocol);
  if (-1 == sock) {
    GAPID_WARNING("socket() failed: %s.", strerror(core::error()));
    return nullptr;
  }
  auto socketCloser = [](const int* ptr) { core::close(*ptr); };  // deferred.
  std::unique_ptr<const int, decltype(socketCloser)> sockScopeGuard(
      &sock, socketCloser);

  const int one = 1;
  if (-1 == core::setsockopt(sock, SOL_SOCKET, SO_REUSEADDR, (const char*)&one,
                             sizeof(int))) {
    GAPID_WARNING("setsockopt() failed: %s", strerror(core::error()));
    return nullptr;
  }

  if (-1 == core::bind(sock, addr->ai_addr, addr->ai_addrlen)) {
    GAPID_WARNING("bind() failed: %s.", strerror(core::error()));
    return nullptr;
  }
  struct sockaddr_in sin;
  socklen_t len = sizeof(sin);
  if (-1 == core::getsockname(sock, (struct sockaddr*)&sin, &len)) {
    GAPID_WARNING("getsockname() failed: %s.", strerror(core::error()));
    return nullptr;
  }

  if (-1 == core::listen(sock, 10)) {
    GAPID_WARNING("listen() failed: %s.", strerror(core::error()));
    return nullptr;
  }

  // The following message is parsed by launchers to detect the selected port.
  // DO NOT CHANGE!
  printf("Bound on port '%d'\n", ntohs(sin.sin_port));
  fflush(stdout);  // Force the message for piped readers

  const char* portFile = getenv("GAPII_PORT_FILE");
  if (portFile != nullptr) {
    FILE* f = fopen(portFile, "w");
    if (f != nullptr) {
      fprintf(f, "Bound on port '%d'", ntohs(sin.sin_port));
      fclose(f);
    }
  }

  sockScopeGuard.release();
  return std::unique_ptr<Connection>(new SocketConnection(sock));
}

uint32_t SocketConnection::getFreePort(const char* hostname) {
  // Network initializer to ensure that the network driver is initialized during
  // the lifetime of the create function. If the connection created successfully
  // then the new connection will hold a reference to a networkInitializer
  // struct to ensure that the network is initialized
  NetworkInitializer networkInitializer;

  struct addrinfo* addr;
  struct addrinfo hints {};
  hints.ai_family = AF_INET;
  hints.ai_socktype = SOCK_STREAM;

  const int getaddrinfoRes = core::getaddrinfo(hostname, "0", &hints, &addr);
  if (0 != getaddrinfoRes) {
    GAPID_WARNING("getaddrinfo() failed: %d - %s.", getaddrinfoRes,
                  strerror(core::error()));
    return 0;
  }
  auto addrDeleter = [](struct addrinfo* ptr) {
    core::freeaddrinfo(ptr);
  };  // deferred.
  std::unique_ptr<struct addrinfo, decltype(addrDeleter)> addrScopeGuard(
      addr, addrDeleter);

  const int sock =
      core::socket(addr->ai_family, addr->ai_socktype, addr->ai_protocol);
  if (-1 == sock) {
    GAPID_WARNING("socket() failed: %s.", strerror(core::error()));
    return 0;
  }
  auto socketCloser = [](const int* ptr) { core::close(*ptr); };  // deferred.
  std::unique_ptr<const int, decltype(socketCloser)> sockScopeGuard(
      &sock, socketCloser);

  const int one = 1;
  if (-1 == core::setsockopt(sock, SOL_SOCKET, SO_REUSEADDR, (const char*)&one,
                             sizeof(int))) {
    GAPID_WARNING("setsockopt() failed: %s", strerror(core::error()));
    return 0;
  }

  if (-1 == core::bind(sock, addr->ai_addr, addr->ai_addrlen)) {
    GAPID_WARNING("bind() failed: %s.", strerror(core::error()));
    return 0;
  }
  struct sockaddr_in sin;
  socklen_t len = sizeof(sin);
  if (-1 == core::getsockname(sock, (struct sockaddr*)&sin, &len)) {
    GAPID_WARNING("getsockname() failed: %s.", strerror(core::error()));
    return 0;
  }

  if (-1 == core::listen(sock, 10)) {
    GAPID_WARNING("listen() failed: %s.", strerror(core::error()));
    return 0;
  }
  return ntohs(sin.sin_port);
}

std::unique_ptr<Connection> SocketConnection::createPipe(const char* pipename,
                                                         bool abstract) {
#if TARGET_OS == GAPID_OS_WINDOWS
  // AF_UNIX is not supported on Windows.
  return nullptr;
#else  // TARGET_OS == GAPID_OS_WINDOWS
  const int sock = core::socket(AF_UNIX, SOCK_STREAM, 0);
  if (-1 == sock) {
    GAPID_WARNING("socket() failed: %s.", strerror(core::error()));
    return nullptr;
  }
  auto socketCloser = [](const int* ptr) { core::close(*ptr); };  // deferred.
  std::unique_ptr<const int, decltype(socketCloser)> sockScopeGuard(
      &sock, socketCloser);

  // In order to create a socket in the abstract namespace (no filesystem link),
  // sun_path must start with a null byte followed by the abstract socket name.
  // Abstract sockets/pipes are a non-portable Linux extension available on
  // Android, described in Linux's man unix(7).

#if (TARGET_OS != GAPID_OS_LINUX) && (TARGET_OS != GAPID_OS_ANDROID)
  if (abstract) {
    GAPID_WARNING(
        "Abstract pipe '%s' creation unsupported for this platform. "
        "Falling back to non-abstract.",
        pipename);
    abstract = false;
  }
#endif

  // Rely on zero-initialization here to assume pipe.sun_path is zeroed.
  struct sockaddr_un pipe {};
  pipe.sun_family = AF_UNIX;
  strncpy(pipe.sun_path + (abstract ? 1 : 0), pipename,
          sizeof(pipe.sun_path) - 2);
  const size_t pipelen =
      sizeof(pipe.sun_family) + strlen(pipename) + (abstract ? 1 : 0);

  if (-1 == core::bind(sock, (struct sockaddr*)&pipe, pipelen)) {
    GAPID_WARNING("bind() failed: %s.", strerror(core::error()));
    return nullptr;
  }

  if (-1 == core::listen(sock, 10)) {
    GAPID_WARNING("listen() failed: %s.", strerror(core::error()));
    return nullptr;
  }

  sockScopeGuard.release();
  return std::unique_ptr<Connection>(new SocketConnection(sock));
#endif  // TARGET_OS == GAPID_OS_WINDOWS
}

SocketConnection::NetworkInitializer::NetworkInitializer() {
#if TARGET_OS == GAPID_OS_WINDOWS
  if (gWinsockUsageCount++ == 0) {
    WSADATA wsaData;
    int wsaInitRes = ::WSAStartup(MAKEWORD(2, 2), &wsaData);
    if (wsaInitRes != 0) {
      GAPID_FATAL("WSAStartup failed with error code: %d", wsaInitRes);
    }
  }
#endif  // TARGET_OS == GAPID_OS_WINDOWS
}

SocketConnection::NetworkInitializer::~NetworkInitializer() {
#if TARGET_OS == GAPID_OS_WINDOWS
  if (--gWinsockUsageCount == 0) {
    ::WSACleanup();
  }
#endif  // TARGET_OS == GAPID_OS_WINDOWS
}

}  // namespace core
