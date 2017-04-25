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
import com.google.gapid.proto.service.GapidGrpc;
import com.google.gapid.proto.service.GapidGrpc.GapidFutureStub;
import com.google.gapid.proto.service.Service;

/**
 * A {@link GapidClient} based on a gRPC service.
 */
public class GapidClientGrpc implements GapidClient {
  private final GapidGrpc.GapidFutureStub client;

  public GapidClientGrpc(GapidFutureStub client) {
    this.client = client;
  }

  @Override
  public ListenableFuture<Service.GetServerInfoResponse> getServerInfo(
      Service.GetServerInfoRequest request) {
    return client.getServerInfo(request);
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
  public ListenableFuture<Service.GetSchemaResponse> getSchema(Service.GetSchemaRequest request) {
    return client.getSchema(request);
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
}
