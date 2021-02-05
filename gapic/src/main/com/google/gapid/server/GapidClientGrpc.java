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

import static com.google.common.util.concurrent.MoreExecutors.directExecutor;

import com.google.common.util.concurrent.ListenableFuture;
import com.google.common.util.concurrent.SettableFuture;
import com.google.gapid.proto.log.Log;
import com.google.gapid.proto.service.GapidGrpc;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.Service.ClientEventRequest;
import com.google.gapid.proto.service.Service.ClientEventResponse;
import com.google.gapid.proto.service.Service.GpuProfileRequest;
import com.google.gapid.proto.service.Service.GpuProfileResponse;
import com.google.gapid.proto.service.Service.InstallAppRequest;
import com.google.gapid.proto.service.Service.InstallAppResponse;
import com.google.gapid.proto.service.Service.PerfettoQueryRequest;
import com.google.gapid.proto.service.Service.PerfettoQueryResponse;
import com.google.gapid.proto.service.Service.PingRequest;
import com.google.gapid.proto.service.Service.TraceTargetTreeNodeRequest;
import com.google.gapid.proto.service.Service.TraceTargetTreeNodeResponse;
import com.google.gapid.proto.service.Service.UpdateSettingsRequest;
import com.google.gapid.proto.service.Service.UpdateSettingsResponse;
import com.google.gapid.proto.service.Service.ValidateDeviceRequest;
import com.google.gapid.proto.service.Service.ValidateDeviceResponse;
import com.google.gapid.util.MoreFutures;

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
    return MoreFutures.transform(client.ping(PingRequest.getDefaultInstance()), ignored -> null);
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
  public ListenableFuture<Service.SaveCaptureResponse> saveCapture(
      Service.SaveCaptureRequest request) {
    return client.saveCapture(request);
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
  public ListenableFuture<ClientEventResponse> postClientEvent(ClientEventRequest request) {
    return client.clientEvent(request);
  }

  @Override
  public ListenableFuture<TraceTargetTreeNodeResponse> getTraceTargetTreeNode(
      TraceTargetTreeNodeRequest request) {
    return client.traceTargetTreeNode(request);
  }

  @Override
  public ListenableFuture<UpdateSettingsResponse> updateSettings(UpdateSettingsRequest request) {
    return client.updateSettings(request);
  }

  @Override
  public ListenableFuture<PerfettoQueryResponse> perfettoQuery(PerfettoQueryRequest request) {
    return client.perfettoQuery(request);
  }

  @Override
  public ListenableFuture<GpuProfileResponse> profile(GpuProfileRequest request) {
    return client.gpuProfile(request);
  }

  @Override
  public ListenableFuture<ValidateDeviceResponse> validateDevice(ValidateDeviceRequest request) {
    return client.validateDevice(request);
  }

  @Override
  public ListenableFuture<InstallAppResponse> installApp(InstallAppRequest request) {
    return client.installApp(request);
  }

  @Override
  public ListenableFuture<Void> streamLog(Consumer<Log.Message> onLogMessage) {
    StreamHandler<Log.Message> handler = StreamHandler.wrap(onLogMessage);
    stub.getLogStream(Service.GetLogStreamRequest.getDefaultInstance(), handler);
    return handler.future;
  }

  @Override
  public ListenableFuture<Void> streamStatus(
      Service.ServerStatusRequest request, Consumer<Service.ServerStatusResponse> onStatus) {
    StreamHandler<Service.ServerStatusResponse> handler = StreamHandler.wrap(onStatus);
    stub.status(request, handler);
    return handler.future;
  }


  @Override
  public ListenableFuture<Void> streamSearch(
      Service.FindRequest request, Consumer<Service.FindResponse> onResult) {
    StreamHandler<Service.FindResponse> handler = StreamHandler.wrap(onResult);
    stub.find(request, handler);
    return handler.future;
  }

  @Override
  public GapidClient.StreamSender<Service.TraceRequest> streamTrace(
      StreamConsumer<Service.TraceResponse> onTraceResponse) {
    StreamHandler<Service.TraceResponse> handler = StreamHandler.wrap(onTraceResponse);
    return Sender.wrap(handler.future, stub.trace(handler));
  }

  private static class StreamHandler<T> implements StreamObserver<T> {
    public final SettableFuture<Void> future = SettableFuture.create();
    private final GapidClient.StreamConsumer<T> consumer;

    private StreamHandler(GapidClient.StreamConsumer<T> consumer) {
      this.consumer = consumer;
    }

    public static <T> StreamHandler<T> wrap(Consumer<T> consumer) {
      return new StreamHandler<T>(r -> {
        consumer.accept(r);
        return GapidClient.Result.CONTINUE;
      });
    }

    public static <T> StreamHandler<T> wrap(GapidClient.StreamConsumer<T> consumer) {
      return new StreamHandler<T>(consumer);
    }

    @Override
    public void onNext(T value) {
      GapidClient.Result result = consumer.consume(value);
      if (result.error != null) {
        onError(result.error);
      } else if (result.close) {
        onCompleted();
      }
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

  private static class Sender<T> implements GapidClient.StreamSender<T> {
    private final ListenableFuture<Void> future;
    private final StreamObserver<T> observer;

    public Sender(ListenableFuture<Void> future, StreamObserver<T> observer) {
      this.future = future;
      this.observer = observer;

      future.addListener(this::finish, directExecutor());
    }

    public static <T> Sender<T> wrap(ListenableFuture<Void> future, StreamObserver<T> observer) {
      return new Sender<T>(future, observer);
    }

    @Override
    public void send(T value) {
      observer.onNext(value);
    }

    @Override
    public void finish() {
      observer.onCompleted();
    }

    @Override
    public ListenableFuture<Void> getFuture() {
      return future;
    }
  }
}
