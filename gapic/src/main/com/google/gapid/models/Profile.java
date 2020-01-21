/*
 * Copyright (C) 2019 Google Inc.
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
package com.google.gapid.models;
import static com.google.gapid.rpc.UiErrorCallback.error;
import static com.google.gapid.rpc.UiErrorCallback.success;
import static com.google.gapid.util.Logging.throttleLogRpcError;
import static com.google.gapid.util.MoreFutures.transform;
import static java.util.logging.Level.WARNING;

import com.google.common.base.Objects;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.rpc.Rpc;
import com.google.gapid.rpc.RpcException;
import com.google.gapid.rpc.UiErrorCallback.ResultOrError;
import com.google.gapid.server.Client;
import com.google.gapid.util.Events;
import com.google.gapid.util.Loadable;

import org.eclipse.swt.widgets.Shell;

import java.util.concurrent.ExecutionException;
import java.util.logging.Logger;

public class Profile
    extends CaptureDependentModel<Profile.Data, Profile.Source, Loadable.Message, Profile.Listener> {
  private static final Logger LOG = Logger.getLogger(Profile.class.getName());

  private final Capture capture;

  public Profile(
      Shell shell, Analytics analytics, Client client, Capture capture, Devices devices) {
    super(LOG, shell, analytics, client, Listener.class, capture, devices);
    this.capture = capture;
  }

  @Override
  protected Source getSource(Capture.Data data) {
    return new Source(data.path);
  }

  @Override
  protected boolean shouldLoad(Capture.Data data) {
    return data.isGraphics();
  }

  @Override
  protected ListenableFuture<Data> doLoad(Source source, Path.Device device) {
    return transform(client.profile(capture.getData().path, device), r -> new Data(device, r));
  }

  @Override
  protected ResultOrError<Data, Loadable.Message> processResult(Rpc.Result<Data> result) {
    try {
      return success(result.get());
    } catch (RpcException e) {
      LOG.log(WARNING, "Failed to load the GPU profile", e);
      return error(Loadable.Message.error(e));
    } catch (ExecutionException e) {
      if (!shell.isDisposed()) {
        throttleLogRpcError(LOG, "Failed to load the GPU profile", e);
      }
      return error(Loadable.Message.error("Failed to load the GPU profile"));
    }
  }

  @Override
  protected void updateError(Loadable.Message error) {
    listeners.fire().onProfileLoaded(error);
  }

  @Override
  protected void fireLoadStartEvent() {
    listeners.fire().onProfileLoadingStart();
  }

  @Override
  protected void fireLoadedEvent() {
    listeners.fire().onProfileLoaded(null);
  }

  public static class Source {
    public final Path.Capture capture;

    public Source(Path.Capture capture) {
      this.capture = capture;
    }

    @Override
    public boolean equals(Object obj) {
      if (obj == this) {
        return true;
      } else if (!(obj instanceof Source)) {
        return false;
      }
      return Objects.equal(capture, ((Source)obj).capture);
    }

    @Override
    public int hashCode() {
      return (capture == null) ? 0 : capture.hashCode();
    }
  }

  public static class Data extends DeviceDependentModel.Data {
    public final Service.ProfilingData profile;

    public Data(Path.Device device, Service.ProfilingData profile) {
      super(device);
      this.profile = profile;
    }

    public boolean hasSlices() {
      return profile.getSlices().getSlicesCount() > 0 &&
          profile.getSlices().getTracksCount() > 0;
    }

    public Service.ProfilingData.GpuSlices getSlices() {
      return profile.getSlices();
    }

    public TimeSpan getSlicesTimeSpan() {
      if (!hasSlices()) {
        return TimeSpan.ZERO;
      }

      long start = Long.MAX_VALUE, end = 0;
      for (Service.ProfilingData.GpuSlices.Slice slice : profile.getSlices().getSlicesList()) {
        start = Math.min(slice.getTs(), start);
        end = Math.max(slice.getTs() + slice.getDur(), end);
      }
      return new TimeSpan(start, end);
    }
  }

  public static interface Listener extends Events.Listener {
    /**
     * Event indicating that profiling data is being loaded.
     */
    public default void onProfileLoadingStart() { /* empty */ }

    /**
     * Event indicating that the profiling data has finished loading.
     *
     * @param error the loading error or {@code null} if loading was successful.
     */
    public default void onProfileLoaded(Loadable.Message error) { /* empty */ }
  }
}
