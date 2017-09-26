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

package com.google.android.gapid;

import android.net.LocalServerSocket;
import android.net.LocalSocket;
import java.io.DataOutputStream;
import java.util.concurrent.Future;

class SocketWriter {
    /**
     * connectAndWrite waits for the incoming connection to the specified local-abstract socket.
     * When a connection is made, data is written to the accepted connection, then both sockets
     * are closed. Data elements are written using an ObjectOutputStream.
     *
     * The method takes a {@link Future} because we want the socket to be ready and listening
     * as early as possible. The remote tries the socket and then waits in increments of a fixed
     * 1s interval if it fails to connect, which could mean an extra 1s to wait if the socket
     * happens to get created immediately after a failed attempt.
     *
     * @param socketName The name of the local-abstract socket to listen on.
     * @param data The data to send to the first accepted socket.
     */
    static void connectAndWrite(String socketName, Future<byte[]> data) throws Exception {
        LocalServerSocket server = new LocalServerSocket(socketName);
        try {
            LocalSocket socket = server.accept();
            try {
                DataOutputStream dos = new DataOutputStream(socket.getOutputStream());
                byte[] bytes;
                try (Counter.Scope t = Counter.time("data.get()")) {
                    bytes = data.get();
                }
                try (Counter.Scope t = Counter.time("transmit")) {
                    dos.write(bytes);
                    dos.flush();
                }
            } finally {
                socket.close();
            }
        } finally {
            server.close();
        }
    }
}