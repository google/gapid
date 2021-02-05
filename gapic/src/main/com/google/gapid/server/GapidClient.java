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

import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.proto.log.Log;
import com.google.gapid.proto.service.Service;

import java.util.function.Consumer;

/**
 * The public API to communicate with the server.
 */
public interface GapidClient {
  public ListenableFuture<Void> ping();
  public ListenableFuture<Service.GetServerInfoResponse> getServerInfo(
      Service.GetServerInfoRequest request);
  public ListenableFuture<Service.CheckForUpdatesResponse> checkForUpdates(
      Service.CheckForUpdatesRequest request);
  public ListenableFuture<Service.GetResponse> get(Service.GetRequest request);
  public ListenableFuture<Service.SetResponse> set(Service.SetRequest request);
  public ListenableFuture<Service.FollowResponse> follow(Service.FollowRequest request);
  public ListenableFuture<Service.GetPerformanceCountersResponse> getPerformanceCounters(
      Service.GetPerformanceCountersRequest request);
  public ListenableFuture<Service.GetProfileResponse> getProfile(Service.GetProfileRequest request);
  public ListenableFuture<Service.GetAvailableStringTablesResponse> getAvailableStringTables(
      Service.GetAvailableStringTablesRequest request);
  public ListenableFuture<Service.GetStringTableResponse> getStringTable(
      Service.GetStringTableRequest request);
  public ListenableFuture<Service.ImportCaptureResponse> importCapture(
      Service.ImportCaptureRequest request);
  public ListenableFuture<Service.LoadCaptureResponse> loadCapture(
      Service.LoadCaptureRequest request);
  public ListenableFuture<Service.SaveCaptureResponse> saveCapture(
      Service.SaveCaptureRequest request);
  public ListenableFuture<Service.GetDevicesResponse> getDevices(Service.GetDevicesRequest request);
  public ListenableFuture<Service.GetDevicesForReplayResponse> getDevicesForReplay(
      Service.GetDevicesForReplayRequest request);
  public ListenableFuture<Service.ClientEventResponse> postClientEvent(
      Service.ClientEventRequest request);
  public ListenableFuture<Service.TraceTargetTreeNodeResponse> getTraceTargetTreeNode(
      Service.TraceTargetTreeNodeRequest request);
  public ListenableFuture<Service.UpdateSettingsResponse> updateSettings(
      Service.UpdateSettingsRequest request);
  public ListenableFuture<Service.PerfettoQueryResponse> perfettoQuery(
      Service.PerfettoQueryRequest request);
  public ListenableFuture<Service.GpuProfileResponse> profile(Service.GpuProfileRequest request);
  public ListenableFuture<Service.ValidateDeviceResponse> validateDevice(
      Service.ValidateDeviceRequest request);
  public ListenableFuture<Service.InstallAppResponse> installApp(Service.InstallAppRequest request);

  public ListenableFuture<Void> streamLog(Consumer<Log.Message> onLogMessage);
  public ListenableFuture<Void> streamStatus(
      Service.ServerStatusRequest request, Consumer<Service.ServerStatusResponse> onStatus);
  public ListenableFuture<Void> streamSearch(
      Service.FindRequest request, Consumer<Service.FindResponse> onResult);
  public StreamSender<Service.TraceRequest> streamTrace(
      StreamConsumer<Service.TraceResponse> onTraceResponse);

  public static interface StreamSender<T> {
    public ListenableFuture<Void> getFuture();
    public void send(T value);
    public void finish();
  }

  public static interface StreamConsumer<T> {
    public Result consume(T value);
  }

  public static class Result {
    public static final Result DONE = new Result(true, null);
    public static final Result CONTINUE = new Result(false, null);

    public final boolean close;
    public final Throwable error;

    private Result(boolean close, Throwable error) {
      this.close = close;
      this.error = error;
    }

    public static Result error(Throwable error) {
      return new Result(true, error);
    }
  }
}
