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

import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.common.util.concurrent.SettableFuture;
import com.google.gapid.proto.log.Log;
import com.google.gapid.proto.service.GapidGrpc;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.Service.PingRequest;

import java.util.function.Consumer;

import io.grpc.stub.StreamObserver;

/**
 * A {@link GapidClient} based on a gRPC service.
 */
public class GapidClientGrpc implements GapidClient {
  private final GapidGrpc.GapidFutureStub client;
  private final GapidGrpc.GapidStub stub;

  public GapidClientGrpc(GapidGrpc.GapidFutureStub client, GapidGrpc.GapidStub stub) {
    this.client = client;
    this.stub = stub;
  }

  @Override
  public ListenableFuture<Void> ping() {
    return Futures.transform(client.ping(PingRequest.getDefaultInstance()), ignored -> null);
  }

  @Override
  public ListenableFuture<Service.GetServerInfoResponse> getServerInfo(
      Service.GetServerInfoRequest request) {
    return client.getServerInfo(request);
  }

  @Override
  public ListenableFuture<Service.CheckForUpdatesResponse> checkForUpdates(
      Service.CheckForUpdatesRequest request) {
    return client.checkForUpdates(request);
  }

  @Override
  public ListenableFuture<Service.GetResponse> get(Service.GetRequest request) {
    return client.get(request);
  }

  @Override
  public ListenableFuture<Service.SetResponse> set(Service.SetRequest request) {
    return client.set(request);
  }

  @Override
  public ListenableFuture<Service.FollowResponse> follow(Service.FollowRequest request) {
    return client.follow(request);
  }

  @Override
  public ListenableFuture<Service.BeginCPUProfileResponse> beginCPUProfile(
      Service.BeginCPUProfileRequest request) {
    return client.beginCPUProfile(request);
  }

  @Override
  public ListenableFuture<Service.EndCPUProfileResponse> endCPUProfile(
      Service.EndCPUProfileRequest request) {
    return client.endCPUProfile(request);
  }

  @Override
  public ListenableFuture<Service.GetPerformanceCountersResponse> getPerformanceCounters(
      Service.GetPerformanceCountersRequest request) {
    return client.getPerformanceCounters(request);
  }

  @Override
  public ListenableFuture<Service.GetProfileResponse> getProfile(
      Service.GetProfileRequest request) {
    return client.getProfile(request);
  }

  @Override
  public ListenableFuture<Service.GetAvailableStringTablesResponse> getAvailableStringTables(
      Service.GetAvailableStringTablesRequest request) {
    return client.getAvailableStringTables(request);
  }

  @Override
  public ListenableFuture<Service.GetStringTableResponse> getStringTable(
      Service.GetStringTableRequest request) {
    return client.getStringTable(request);
  }

  @Override
  public ListenableFuture<Service.ImportCaptureResponse> importCapture(
      Service.ImportCaptureRequest request) {
    return client.importCapture(request);
  }

  @Override
  public ListenableFuture<Service.ExportCaptureResponse> exportCapture(
      Service.ExportCaptureRequest request) {
    return client.exportCapture(request);
  }

  @Override
  public ListenableFuture<Service.LoadCaptureResponse> loadCapture(
      Service.LoadCaptureRequest request) {
    return client.loadCapture(request);
  }

  @Override
  public ListenableFuture<Service.GetDevicesResponse> getDevices(
      Service.GetDevicesRequest request) {
    return client.getDevices(request);
  }

  @Override
  public ListenableFuture<Service.GetDevicesForReplayResponse> getDevicesForReplay(
      Service.GetDevicesForReplayRequest request) {
    return client.getDevicesForReplay(request);
  }

  @Override
  public ListenableFuture<Service.GetFramebufferAttachmentResponse> getFramebufferAttachment(
      Service.GetFramebufferAttachmentRequest request) {
    return client.getFramebufferAttachment(request);
  }

  @Override
  public ListenableFuture<Void> streamLog(Consumer<Log.Message> onLogMessage) {
    StreamHandler<Log.Message> handler = StreamHandler.wrap(onLogMessage);
    stub.getLogStream(Service.GetLogStreamRequest.getDefaultInstance(), handler);
    return handler.future;
  }

  @Override
  public ListenableFuture<Void> streamSearch(
      Service.FindRequest request, Consumer<Service.FindResponse> onResult) {
    StreamHandler<Service.FindResponse> handler= StreamHandler.wrap(onResult);
    stub.find(request, handler);
    return handler.future;
  }

  private static class StreamHandler<T> implements StreamObserver<T> {
    public final SettableFuture<Void> future = SettableFuture.create();
    private final Consumer<T> consumer;

    private StreamHandler(Consumer<T> consumer) {
      this.consumer = consumer;
    }

    public static <T> StreamHandler<T> wrap(Consumer<T> consumer) {
      return new StreamHandler<T>(consumer);
    }

    @Override
    public void onNext(T value) {
      consumer.accept(value);
    }

    @Override
    public void onCompleted() {
      future.set(null);
    }

    @Override
    public void onError(Throwable t) {
      future.setException(t);
    }
  }
}
