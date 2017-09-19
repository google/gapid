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

import static com.google.common.util.concurrent.Futures.immediateFuture;
import static com.google.gapid.util.ProtoDebugTextFormat.shortDebugString;
import static java.util.logging.Level.FINE;

import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.common.util.concurrent.SettableFuture;
import com.google.gapid.models.Strings;
import com.google.gapid.proto.log.Log;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.Service.CheckForUpdatesRequest;
import com.google.gapid.proto.service.Service.ExportCaptureRequest;
import com.google.gapid.proto.service.Service.FollowRequest;
import com.google.gapid.proto.service.Service.GetAvailableStringTablesRequest;
import com.google.gapid.proto.service.Service.GetDevicesForReplayRequest;
import com.google.gapid.proto.service.Service.GetDevicesRequest;
import com.google.gapid.proto.service.Service.GetFramebufferAttachmentRequest;
import com.google.gapid.proto.service.Service.GetRequest;
import com.google.gapid.proto.service.Service.GetServerInfoRequest;
import com.google.gapid.proto.service.Service.GetStringTableRequest;
import com.google.gapid.proto.service.Service.ImportCaptureRequest;
import com.google.gapid.proto.service.Service.LoadCaptureRequest;
import com.google.gapid.proto.service.Service.Release;
import com.google.gapid.proto.service.Service.ServerInfo;
import com.google.gapid.proto.service.Service.SetRequest;
import com.google.gapid.proto.service.Service.Value;
import com.google.gapid.proto.service.api.API;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.proto.stringtable.Stringtable;
import com.google.gapid.rpc.RpcException;
import com.google.gapid.util.Paths;
import com.google.gapid.util.Scheduler;
import com.google.protobuf.ByteString;

import java.util.List;
import java.util.function.Consumer;
import java.util.function.Function;
import java.util.function.Supplier;
import java.util.logging.Logger;

/**
 * Client interface to the RPC server.
 */
public class Client {
  private static final Logger LOG = Logger.getLogger(Client.class.getName());

  private final GapidClient client;

  public Client(GapidClient client) {
    this.client = client;
  }

  public ListenableFuture<ServerInfo> getSeverInfo() {
    return call(() -> "RPC->getServerInfo()",
        stack -> Futures.transformAsync(
            client.getServerInfo(GetServerInfoRequest.getDefaultInstance()),
            in -> immediateFuture(throwIfError(in.getInfo(), in.getError(), stack))));
  }

  public ListenableFuture<Release> checkForUpdates(boolean includePrereleases) {
    return call(() -> String.format("RPC->checkForUpdates(%b)", includePrereleases),
        stack -> Futures.transformAsync(
            client.checkForUpdates(CheckForUpdatesRequest.newBuilder()
                .setIncludePrereleases(includePrereleases)
                .build()),
            in -> immediateFuture(throwIfError(in.getRelease(), in.getError(), stack))));
  }

  public ListenableFuture<Value> get(Path.Any path) {
    return call(() -> String.format("RPC->get(%s)", shortDebugString(path)),
        stack -> Futures.transformAsync(
            client.get(GetRequest.newBuilder()
                .setPath(path)
                .build()),
            in -> immediateFuture(throwIfError(in.getValue(), in.getError(), stack))));
  }

  public ListenableFuture<Path.Any> set(Path.Any path, Service.Value value) {
    return call(
        () -> String.format("RPC->set(%s, %s)", shortDebugString(path), shortDebugString(value)),
        stack -> Futures.transformAsync(
            client.set(SetRequest.newBuilder()
                .setPath(path)
                .setValue(value)
                .build()),
            in -> immediateFuture(throwIfError(in.getPath(), in.getError(), stack))));
  }

  public ListenableFuture<Path.Any> follow(Path.Any path) {
    return call(() -> String.format("RPC->follow(%s)", shortDebugString(path)),
        stack -> Futures.transformAsync(
            client.follow(FollowRequest.newBuilder()
                .setPath(path)
                .build()),
            in -> immediateFuture(throwIfError(in.getPath(), in.getError(), stack))));
  }

  public ListenableFuture<List<Stringtable.Info>> getAvailableStringTables() {
    return call(() -> "RPC->getAvailableStringTables()",
        stack -> Futures.transformAsync(
          client.getAvailableStringTables(GetAvailableStringTablesRequest.getDefaultInstance()),
          in -> immediateFuture(throwIfError(in.getTables(), in.getError(), stack).getListList())));
  }

  public ListenableFuture<Stringtable.StringTable> getStringTable(Stringtable.Info info) {
    return call(() -> String.format("RPC->getStringTable(%s)", shortDebugString(info)),
        stack -> Futures.transformAsync(
            client.getStringTable(GetStringTableRequest.newBuilder()
                .setTable(info)
                .build()),
            in -> immediateFuture(throwIfError(in.getTable(), in.getError(), stack))));
  }

  public ListenableFuture<Path.Capture> importCapture(byte[] data) {
    return call(() -> String.format("RPC->importCapture(<%d bytes>)", data.length),
        stack -> Futures.transformAsync(client.importCapture(
            ImportCaptureRequest.newBuilder()
                .setData(ByteString.copyFrom(data))
                .build()),
            in -> immediateFuture(throwIfError(in.getCapture(), in.getError(), stack))));
  }

  public ListenableFuture<Path.Capture> loadCapture(String path) {
    return call(() -> String.format("RPC->loadCapture(%s)", path),
        stack ->Futures.transformAsync(
            client.loadCapture(LoadCaptureRequest.newBuilder()
                .setPath(path)
                .build()),
            in -> Futures.immediateFuture(throwIfError(in.getCapture(), in.getError(), stack))));
  }

  public ListenableFuture<byte[]> exportCapture(Path.Capture path) {
    return call(() -> String.format("RPC->exportCapture(%s)", shortDebugString(path)),
        stack -> Futures.transformAsync(
            client.exportCapture(ExportCaptureRequest.newBuilder()
                .setCapture(path)
                .build()),
            in -> immediateFuture(throwIfError(in.getData().toByteArray(), in.getError(), stack))));
  }

  public ListenableFuture<List<Path.Device>> getDevices() {
    return call(() -> "RPC->getDevices()",
        stack -> Futures.transformAsync(
            client.getDevices(GetDevicesRequest.getDefaultInstance()),
            in -> immediateFuture(throwIfError(in.getDevices(), in.getError(), stack)
                .getListList())));
  }

  public ListenableFuture<List<Path.Device>> getDevicesForReplay(Path.Capture capture) {
    return call(() -> String.format("RPC->getDevicesForReplay(%s)", shortDebugString(capture)),
        stack -> Futures.transformAsync(
            client.getDevicesForReplay(GetDevicesForReplayRequest.newBuilder()
              .setCapture(capture)
              .build()),
            in -> immediateFuture(throwIfError(in.getDevices(), in.getError(), stack)
                .getListList())));
  }

  public ListenableFuture<Path.ImageInfo> getFramebufferAttachment(Path.Device device,
      Path.Command after, API.FramebufferAttachment attachment,
      Service.RenderSettings settings, Service.UsageHints hints) {
    return call(
        () -> String.format("RPC->getFramebufferAttachment(%s, %s, %s, %s, %s)",
            shortDebugString(device), shortDebugString(after), attachment,
            shortDebugString(settings), shortDebugString(hints)),
        stack -> Futures.transformAsync(
            client.getFramebufferAttachment(GetFramebufferAttachmentRequest.newBuilder()
                .setDevice(device)
                .setAfter(after)
                .setAttachment(attachment)
                .setSettings(settings)
                .setHints(hints)
                .build()),
            in -> immediateFuture(throwIfError(in.getImage(), in.getError(), stack))));
  }

  public ListenableFuture<Void> streamLog(Consumer<Log.Message> onLogMessage) {
    LOG.log(FINE, "RPC->getLogStream()");
    return client.streamLog(onLogMessage);
  }

  public ListenableFuture<Void> streamSearch(
      Service.FindRequest request, Consumer<Service.FindResponse> onResult) {
    LOG.log(FINE, "RPC->find({0})", request);
    return client.streamSearch(request, onResult);
  }

  private static <V> ListenableFuture<V> call(
      Supplier<String> stackMessage, Function<Stack, ListenableFuture<V>> call) {
    SettableFuture<V> result = SettableFuture.create();
    Stack stack = new Stack(stackMessage);
    Scheduler.EXECUTOR.execute(() -> {
      if (LOG.isLoggable(FINE)) {
        LOG.log(FINE, stackMessage.get());
      }
      result.setFuture(call.apply(stack));
    });
    return result;
  }

  private static <V> V throwIfError(V value, Service.Error err, Stack stack) throws RpcException {
    switch (err.getErrCase()) {
      case ERR_NOT_SET:
        return value;
      case ERR_INTERNAL: {
        Service.ErrInternal e = err.getErrInternal();
        throw new InternalServerErrorException(e.getMessage(), stack);
      }
      case ERR_INVALID_ARGUMENT: {
        Service.ErrInvalidArgument e = err.getErrInvalidArgument();
        throw new InvalidArgumentException(e.getReason(), stack);
      }
      case ERR_INVALID_PATH: {
        Service.ErrInvalidPath e = err.getErrInvalidPath();
        throw new InvalidPathException(e.getReason(), e.getPath(), stack);
      }
      case ERR_DATA_UNAVAILABLE: {
        Service.ErrDataUnavailable e = err.getErrDataUnavailable();
        throw new DataUnavailableException(e.getReason(), stack);
      }
      case ERR_PATH_NOT_FOLLOWABLE: {
        Service.ErrPathNotFollowable e = err.getErrPathNotFollowable();
        throw new PathNotFollowableException(e.getPath(), stack);
      }
      case ERR_UNSUPPORTED_VERSION: {
        Service.ErrUnsupportedVersion e = err.getErrUnsupportedVersion();
        throw new UnsupportedVersionException(e.getReason()/*, e.getSuggestUpdate()*/, stack);
      }
      default:
        throw new RuntimeException("Unknown error: " + err.getErrCase(), stack);
    }
  }

  public static class InternalServerErrorException extends RpcException {
    public InternalServerErrorException (String message, Stack stack) {
      super(message, stack);
    }
  }

  public static class DataUnavailableException extends RpcException {
    public DataUnavailableException(Stringtable.Msg reason, Stack stack) {
      super(Strings.getMessage(reason), stack);
    }
  }

  public static class InvalidArgumentException extends RpcException {
    public InvalidArgumentException(Stringtable.Msg reason, Stack stack) {
      super(Strings.getMessage(reason), stack);
    }
  }

  public static class InvalidPathException extends RpcException {
    public final Path.Any path;

    public InvalidPathException(Stringtable.Msg reason, Path.Any path, Stack stack) {
      super(Strings.getMessage(reason), stack);
      this.path = path;
    }

    @Override
    public String toString() {
      return super.toString() + " Path: " + Paths.toString(path);
    }
  }

  public static class PathNotFollowableException extends RpcException {
    public PathNotFollowableException(Path.Any path, Stack stack) {
      super("Path " + Paths.toString(path) + " not followable.", stack);
    }
  }

  public static class UnsupportedVersionException extends RpcException {
    public UnsupportedVersionException(Stringtable.Msg reason, Stack stack) {
      super(Strings.getMessage(reason), stack);
    }
  }

  public static class Stack extends Exception {
    private final Supplier<String> requestString;

    public Stack(Supplier<String> requestString) {
      this.requestString = requestString;
    }

    @Override
    public String getMessage() {
      return "For request: " + requestString.get();
    }
  }
}
