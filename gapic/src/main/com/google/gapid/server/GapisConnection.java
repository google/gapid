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
package com.google.gapid.server;

import static io.grpc.ClientInterceptors.intercept;
import static io.grpc.stub.MetadataUtils.newAttachHeadersInterceptor;

import com.google.gapid.proto.service.GapidGrpc;

import java.io.Closeable;
import java.io.IOException;
import java.util.concurrent.ExecutionException;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.TimeoutException;

import io.grpc.Channel;
import io.grpc.ManagedChannel;
import io.grpc.Metadata;
import io.grpc.okhttp.OkHttpChannelProvider;

/**
 * A connection to a running Graphics API Server (GAPIS).
 */
public abstract class GapisConnection implements Closeable {
  public static final GapisConnection NOT_CONNECTED = new GapisConnection(null) {
    @Override
    public boolean isConnected() {
      return false;
    }

    @Override
    public GapidClient createGapidClient(boolean caching) throws IOException {
      throw new IOException("Not connected");
    }

    @Override
    public void close() {
      // Ignored.
    }
  };

  private final CloseListener listener;

  public GapisConnection(CloseListener listener) {
    this.listener = listener;
  }

  public static GapisConnection create(
      String target, String authToken, int heartbeatRateMS, CloseListener listener) {
    return new GRpcGapisConnection(listener, target, authToken, heartbeatRateMS);
  }

  @Override
  public void close() {
    listener.onClose(this);
  }

  public abstract boolean isConnected();

  public abstract GapidClient createGapidClient(boolean caching) throws IOException;

  public static interface CloseListener {
    public void onClose(GapisConnection connection);
  }

  /**
   * {@link GapisConnection} to a gRPC GAPIS server.
   */
  private static class GRpcGapisConnection extends GapisConnection {
    protected static final Metadata.Key<String> AUTH_HEADER =
        Metadata.Key.of("auth_token", Metadata.ASCII_STRING_MARSHALLER);

    private final ManagedChannel baseChannel;
    private final Channel channel;
    private final int heartbeatRateMS;

    public GRpcGapisConnection(
        CloseListener listener, String target, String authToken, int heartbeatRateMS) {
      super(listener);

      // Us OkHTTP as netty deadlocks a lot with the go server.
      // TODO: figure out what exactly is causing netty to deadlock.
      baseChannel = new OkHttpChannelProvider().builderForTarget(target)
        .usePlaintext(true)
        .maxMessageSize(2 * 1000 * 1000 * 1000) // Do not overflow int32
        .build();

      channel = authToken.isEmpty() ? baseChannel :
        intercept(baseChannel, newAttachHeadersInterceptor(getAuthHeader(authToken)));

      this.heartbeatRateMS = heartbeatRateMS;
    }

    private static Metadata getAuthHeader(String authToken) {
      Metadata md = new Metadata();
      md.put(AUTH_HEADER, authToken);
      return md;
    }

    @Override
    public boolean isConnected() {
      return !baseChannel.isShutdown();
    }

    @Override
    public GapidClient createGapidClient(boolean caching) throws IOException {
      GapidGrpc.GapidFutureStub futureStub = GapidGrpc.newFutureStub(channel);
      GapidGrpc.GapidStub stub = GapidGrpc.newStub(channel);
      GapidClient client = caching ? new GapidClientCache(futureStub, stub) :
          new GapidClientGrpc(futureStub, stub);
      if (heartbeatRateMS > 0) {
        new Heartbeat(client, heartbeatRateMS).start();
      }
      return client;
    }

    @Override
    public void close() {
      baseChannel.shutdown();
      super.close();
    }

    /**
     * Heartbeat is a thread that calls {@link GapidClient#ping()} at regular intervals to prevent
     * the server from exiting due to the --idle-timeout.
     */
    protected static class Heartbeat extends Thread {
      private final GapidClient client;
      private final int rateMS;

      Heartbeat(GapidClient client, int rateMS) {
        this.client = client;
        this.rateMS = rateMS;
      }

      @Override
      public void run() {
        while (true) {
          try {
            client.ping().get(rateMS, TimeUnit.MILLISECONDS);
            Thread.sleep(rateMS);
          } catch (InterruptedException | ExecutionException | TimeoutException e) {
            return; // If the connection failed, the error will appear on another thread.
          }
        }
      }
    }
  }
}
