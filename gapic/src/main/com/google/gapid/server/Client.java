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
import com.google.protobuf.ByteString;

import java.util.List;
import java.util.function.Consumer;
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
    LOG.log(FINE, "RPC->getServerInfo()");
    Stack stack = new Stack(() -> "RPC->getServerInfo()");
    return Futures.transformAsync(
        client.getServerInfo(GetServerInfoRequest.newBuilder().build()),
        in -> immediateFuture(throwIfError(in.getInfo(), in.getError(), stack))
    );
  }

  public ListenableFuture<Release> checkForUpdates(boolean includePrereleases) {
    LOG.log(FINE, "RPC->checkForUpdates({0})", includePrereleases);
    Stack stack = new Stack(() -> String.format("RPC->checkForUpdates(%b)", includePrereleases));
    return Futures.transformAsync(
        client.checkForUpdates(CheckForUpdatesRequest.newBuilder()
            .setIncludePrereleases(includePrereleases).build()),
        in -> immediateFuture(throwIfError(in.getRelease(), in.getError(), stack))
    );
  }

  public ListenableFuture<Value> get(Path.Any path) {
    LOG.log(FINE, "RPC->get({0})", path);
    Stack stack = new Stack(() -> String.format("RPC->get(%s)", shortDebugString(path)));
    return Futures.transformAsync(
        client.get(GetRequest.newBuilder().setPath(path).build()),
        in -> immediateFuture(throwIfError(in.getValue(), in.getError(), stack))
    );
  }

  public ListenableFuture<Path.Any> set(Path.Any path, Service.Value value) {
    LOG.log(FINE, "RPC->set({0}, {1})", new Object[] { path, value });
    Stack stack = new Stack(
        () -> String.format("RPC->set(%s, %s)", shortDebugString(path), shortDebugString(value)));
    return Futures.transformAsync(
        client.set(SetRequest.newBuilder().setPath(path).setValue(value).build()),
        in -> immediateFuture(throwIfError(in.getPath(), in.getError(), stack))
    );
  }

  public ListenableFuture<Path.Any> follow(Path.Any path) {
    LOG.log(FINE, "RPC->follow({0})", path);
    Stack stack = new Stack(() -> String.format("RPC->follow(%s)", shortDebugString(path)));
    return Futures.transformAsync(
        client.follow(FollowRequest.newBuilder().setPath(path).setPath(path).build()),
        in -> immediateFuture(throwIfError(in.getPath(), in.getError(), stack))
    );
  }

  public ListenableFuture<List<Stringtable.Info>> getAvailableStringTables() {
    LOG.log(FINE, "RPC->getAvailableStringTables()");
    Stack stack = new Stack(() -> "RPC->getAvailableStringTables()");
    return Futures.transformAsync(
        client.getAvailableStringTables(GetAvailableStringTablesRequest.newBuilder().build()),
        in -> immediateFuture(throwIfError(in.getTables(), in.getError(), stack).getListList())
    );
  }

  public ListenableFuture<Stringtable.StringTable> getStringTable(Stringtable.Info info) {
    LOG.log(FINE, "RPC->getStringTable({0})", info);
    Stack stack = new Stack(() -> String.format("RPC->getStringTable(%s)", shortDebugString(info)));
    return Futures.transformAsync(
        client.getStringTable(GetStringTableRequest.newBuilder().setTable(info).build()),
        in -> immediateFuture(throwIfError(in.getTable(), in.getError(), stack))
    );
  }
  public ListenableFuture<Path.Capture> importCapture(byte[] data) {
    LOG.log(FINE, "RPC->importCapture(<{0} bytes>)", data.length);
    Stack stack = new Stack(() -> String.format("RPC->importCapture(<%d bytes>)", data.length));
    return Futures.transformAsync(client.importCapture(
        ImportCaptureRequest.newBuilder().setData(ByteString.copyFrom(data)).build()),
        in -> immediateFuture(throwIfError(in.getCapture(), in.getError(), stack))
    );
  }

  public ListenableFuture<Path.Capture> loadCapture(String path) {
    LOG.log(FINE, "RPC->loadCapture({0})", path);
    Stack stack = new Stack(() -> String.format("RPC->loadCapture(%s)", path));
    return Futures.transformAsync(
        client.loadCapture(LoadCaptureRequest.newBuilder().setPath(path).build()),
        in -> Futures.immediateFuture(throwIfError(in.getCapture(), in.getError(), stack))
    );
  }

  public ListenableFuture<byte[]> exportCapture(Path.Capture path) {
    LOG.log(FINE, "RPC->exportCapture({0})", path);
    Stack stack = new Stack(() -> String.format("RPC->exportCapture(%s)", shortDebugString(path)));
    return Futures.transformAsync(
        client.exportCapture(ExportCaptureRequest.newBuilder().setCapture(path).build()),
        in -> immediateFuture(throwIfError(in.getData().toByteArray(), in.getError(), stack))
    );
  }

  public ListenableFuture<List<Path.Device>> getDevices() {
    LOG.log(FINE, "RPC->getDevices()");
    Stack stack = new Stack(() -> "RPC->getDevices()");
    return Futures.transformAsync(
        client.getDevices(GetDevicesRequest.newBuilder().build()),
        in -> immediateFuture(throwIfError(in.getDevices(), in.getError(), stack).getListList())
    );
  }

  public ListenableFuture<List<Path.Device>> getDevicesForReplay(Path.Capture capture) {
    LOG.log(FINE, "RPC->getDevicesForReplay({0})", capture);
    Stack stack = new Stack(
        () -> String.format("RPC->getDevicesForReplay(%s)", shortDebugString(capture)));
    return Futures.transformAsync(client.getDevicesForReplay(
        GetDevicesForReplayRequest.newBuilder().setCapture(capture).build()),
        in -> immediateFuture(throwIfError(in.getDevices(), in.getError(), stack).getListList())
    );
  }

  public ListenableFuture<Path.ImageInfo> getFramebufferAttachment(Path.Device device,
      Path.Command after, API.FramebufferAttachment attachment,
      Service.RenderSettings settings, Service.UsageHints hints) {
    LOG.log(FINE, "RPC->getFramebufferAttachment({0}, {1}, {2}, {3}, {4})",
        new Object[] { device, after, attachment, settings, hints });
    Stack stack = new Stack(() ->
      String.format("RPC->getFramebufferAttachment(%s, %s, %s, %s, %s)",
          shortDebugString(device), shortDebugString(after), attachment, shortDebugString(settings),
          shortDebugString(hints)));
    return Futures.transformAsync(
        client.getFramebufferAttachment(GetFramebufferAttachmentRequest.newBuilder()
            .setDevice(device)
            .setAfter(after)
            .setAttachment(attachment)
            .setSettings(settings)
            .setHints(hints)
            .build()),
        in -> immediateFuture(throwIfError(in.getImage(), in.getError(), stack))
    );
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
