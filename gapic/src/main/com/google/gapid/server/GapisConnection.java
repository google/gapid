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

import com.google.gapid.proto.log.Log;
import com.google.gapid.proto.service.GapidGrpc;

import java.io.Closeable;
import java.io.IOException;
import java.util.Iterator;
import java.util.logging.Level;
import java.util.logging.Logger;

import com.google.gapid.proto.service.Service;
import io.grpc.Channel;
import io.grpc.ManagedChannel;
import io.grpc.Metadata;
import io.grpc.Status;
import io.grpc.StatusRuntimeException;
import io.grpc.okhttp.OkHttpChannelProvider;

/**
 * A connection to a running Graphics API Server (GAPIS).
 */
public abstract class GapisConnection implements Closeable {
  private static final Logger LOG = Logger.getLogger(GapisConnection.class.getName());

  /**
   * The interface passed to {@link #setLogMonitor} used to listen for GAPIS log messages.
   */
  public interface LogMonitor {
    void onLogMessage(Log.Message message);
  }

  public static final GapisConnection NOT_CONNECTED = new GapisConnection(null) {
    @Override
    public boolean isConnected() {
      return false;
    }

    @Override
    public GapidGrpc.GapidFutureStub createGapidClient() throws IOException {
      throw new IOException("Not connected");
    }

    @Override
    public void close() {
      // Ignored.
    }

    @Override
    public void setLogMonitor(LogMonitor monitor) {
      // no-op.
    }
  };

  private final CloseListener listener;

  public GapisConnection(CloseListener listener) {
    this.listener = listener;
  }

  public static GapisConnection create(String target, String authToken) {
    return create(target, authToken, con -> { /* ignore */ });
  }

  public static GapisConnection create(String target, String authToken, CloseListener listener) {
    return new GRpcGapisConnection(listener, target, authToken);
  }

  @Override
  public void close() {
    listener.onClose(this);
  }

  public abstract boolean isConnected();

  public abstract GapidGrpc.GapidFutureStub createGapidClient() throws IOException;

  /**
   * Begins monitoring GAPIS for log messages. Each log message will be forwarded to the {@link LogMonitor}.
   * Only one {@link LogMonitor} can be bound at any time.
   *
   * @param monitor the {@link LogMonitor} that listens for GAPIS log messages. Pass null to unlisten.
   */
  public abstract void setLogMonitor(LogMonitor monitor);

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
    private LogMonitorThread logMonitorThread;

    public GRpcGapisConnection(CloseListener listener, String target, String authToken) {
      super(listener);

      // Us OkHTTP as netty deadlocks a lot with the go server.
      // TODO: figure out what exactly is causing netty to deadlock.
      baseChannel = new OkHttpChannelProvider().builderForTarget(target)
        .usePlaintext(true)
        .maxMessageSize(128 * 1024 * 1024)
        .build();

      channel = authToken.isEmpty() ? baseChannel :
        intercept(baseChannel, newAttachHeadersInterceptor(getAuthHeader(authToken)));
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
    public GapidGrpc.GapidFutureStub createGapidClient() throws IOException {
      return GapidGrpc.newFutureStub(channel);
    }

    @Override
    public synchronized void setLogMonitor(LogMonitor monitor) {
      if (monitor == logMonitorThread) {
        return; // No change.
      }
      if (logMonitorThread != null) {
        logMonitorThread.interrupt();
        logMonitorThread = null;
      }
      if (monitor == null) {
        return;
      }
      logMonitorThread = new LogMonitorThread(channel, monitor);
      logMonitorThread.start();
    }

    @Override
    public void close() {
      setLogMonitor(null);
      baseChannel.shutdown();
      super.close();
    }

    private static class LogMonitorThread extends Thread {
      private final Channel channel;
      private final LogMonitor monitor;

      LogMonitorThread(Channel channel, LogMonitor monitor) {
        this.channel = channel;
        this.monitor = monitor;
      }

      @Override
      public void run() {
        Service.GetLogStreamRequest request = Service.GetLogStreamRequest.newBuilder().build();
        try {
          Iterator<Log.Message> it = GapidGrpc.newBlockingStub(channel).getLogStream(request);
          while (it.hasNext()) {
            monitor.onLogMessage(it.next());
          }
        } catch(StatusRuntimeException ex) {
          if (!ex.getStatus().getCode().equals(Status.Code.CANCELLED)) {
            LOG.log(Level.WARNING, "getLogStream() threw unexpected exception", ex.getStatus());
          }
        }
      }
    }
  }
}
