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

import android.app.IntentService;
import android.content.Intent;
import android.util.Log;

import java.util.concurrent.Callable;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

/**
 * An {@link IntentService} subclass for providing device information to the host.
 * <p/>
 * When the service is sent the {@link #ACTION_SEND_DEVICE_INFO} action, the service will begin
 * listening on the supplied local-abstract socket provided in the {@link #EXTRA_SOCKET_NAME} extra,
 * or if the extra is absent, {@link #DEFAULT_SOCKET_NAME}. When an incoming connection to this
 * socket is made, the service will send the installed package information on the accepted
 * connection, then close the accepted connection and the listening socket.
 */
public class DeviceInfoService extends IntentService {
    private static final String TAG = "gapid-pkginfo";

    /**
     * Action used to start waiting for an incoming connection on the local-abstract port
     * {@link #EXTRA_SOCKET_NAME}. When a connection is made, the package information is send to the
     * connected socket, the socket is closed and the service stops listening on
     * {@link #EXTRA_SOCKET_NAME}.
     */
    private static final String ACTION_SEND_DEVICE_INFO = "com.google.android.gapid.action.SEND_DEV_INFO";

    /**
     * Optional parameter for {@link #ACTION_SEND_DEVICE_INFO} that changes the local-abstract port
     * used to listen for incoming connections. The default value is {@link #DEFAULT_SOCKET_NAME}.
     */
    private static final String EXTRA_SOCKET_NAME = "com.google.android.gapid.extra.SOCKET_NAME";

    /**
     * The default socket name when {@link #EXTRA_SOCKET_NAME} is not provided.
     */
    private static final String DEFAULT_SOCKET_NAME = "gapid-devinfo";

    public DeviceInfoService() {
        super("DeviceInfoService");
    }

    @Override
    protected void onHandleIntent(Intent intent) {
        if (intent != null) {
            final String action = intent.getAction();
            if (ACTION_SEND_DEVICE_INFO.equals(action)) {
                String socketName = intent.getStringExtra(EXTRA_SOCKET_NAME);
                if (socketName == null) {
                    socketName = DEFAULT_SOCKET_NAME;
                }
                handleSendDeviceInfo(socketName);
            }
        }
    }

    /**
     * Handler for the {@link #ACTION_SEND_DEVICE_INFO} intent.
     */
    private void handleSendDeviceInfo(String socketName) {
        Callable<byte[]> deviceInfoCallable = new Callable<byte[]>() {
            @Override
            public byte[] call() throws Exception {
                System.loadLibrary("deviceinfo");
                return getDeviceInfo();
            }
        };

        final ExecutorService executor = Executors.newCachedThreadPool();
        try {
            SocketWriter.connectAndWrite(socketName, executor.submit(deviceInfoCallable));
        } catch (Exception ex) {
            Log.e(TAG, "Error occurred", ex);
        } finally {
            executor.shutdown();
        }
    }

    private native byte[] getDeviceInfo();
}
