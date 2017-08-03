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
  public ListenableFuture<Service.BeginCPUProfileResponse> beginCPUProfile(
      Service.BeginCPUProfileRequest request);
  public ListenableFuture<Service.EndCPUProfileResponse> endCPUProfile(
      Service.EndCPUProfileRequest request);
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
  public ListenableFuture<Service.ExportCaptureResponse> exportCapture(
      Service.ExportCaptureRequest request);
  public ListenableFuture<Service.GetDevicesResponse> getDevices(Service.GetDevicesRequest request);
  public ListenableFuture<Service.GetDevicesForReplayResponse> getDevicesForReplay(
      Service.GetDevicesForReplayRequest request);
  public ListenableFuture<Service.GetFramebufferAttachmentResponse> getFramebufferAttachment(
      Service.GetFramebufferAttachmentRequest request);

  public ListenableFuture<Void> streamLog(Consumer<Log.Message> onLogMessage);
  public ListenableFuture<Void> streamSearch(
      Service.FindRequest request, Consumer<Service.FindResponse> onResult);
}
