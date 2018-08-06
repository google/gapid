/*
 * Copyright (C) 2018 Google Inc.
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

import com.google.common.base.Objects;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.proto.device.Device;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.server.Client;
import com.google.gapid.util.Events;

import org.eclipse.swt.widgets.Shell;

import java.util.logging.Logger;

/**
 * Base class for models that require the replay device.
 */
public abstract class
    DeviceDependentModel <T extends DeviceDependentModel.Data, S, E, L extends Events.Listener>
    extends ModelBase<T, DeviceDependentModel.Source<S>, E, L> {

  public DeviceDependentModel(Logger log, Shell shell, Analytics analytics, Client client,
      Class<L> listenerClass, Devices devices) {
    super(log, shell, analytics, client, listenerClass);

    devices.addListener(new Devices.Listener() {
      @Override
      public void onReplayDevicesLoaded() {
        onReplayDeviceChanged(devices.getSelectedReplayDevice());
      }

      @Override
      public void onReplayDeviceChanged(Device.Instance dev) {
        load(Source.withDevice(getSource(), devices.getReplayDevicePath()), false);
      }
    });
  }

  @Override
  protected boolean isSourceComplete(Source<S> source) {
    return source.device != null && source.source != null;
  }

  @Override
  protected ListenableFuture<T> doLoad(Source<S> source) {
    return doLoad(source.source, source.device);
  }

  protected abstract ListenableFuture<T> doLoad(S source, Path.Device device);

  public static class Source<S> {
    public final Path.Device device;
    public final S source;

    public Source(Path.Device device, S source) {
      this.device = device;
      this.source = source;
    }

    @Override
    public boolean equals(Object obj) {
      if (obj == this) {
        return true;
      } else if (!(obj instanceof Source)) {
        return false;
      }
      Source<?> s = (Source<?>)obj;
      return Objects.equal(device, s.device) && Objects.equal(source, s.source);
    }

    @Override
    public int hashCode() {
      return ((device == null) ? 0 : 31 * device.hashCode()) +
          ((source == null) ? 0 : source.hashCode());
    }

    public static <S> Source<S> withDevice(Source<S> source, Path.Device device) {
      return new Source<S>(device, (source == null) ? null : source.source);
    }

    public static <S> Source<S> withSource(Source<S> source, S val) {
      return new Source<S>((source == null) ? null : source.device, val);
    }
  }

  public static class Data {
    public final Path.Device device;

    public Data(Path.Device device) {
      this.device = device;
    }
  }

  /**
   * A {@link DeviceDependentModel} that uses a path as the source.
   */
  public abstract static class ForPath<T extends Data, E, L extends Events.Listener>
    extends DeviceDependentModel<T, Path.Any, E, L> {

    public ForPath(Logger log, Shell shell, Analytics analytics, Client client,
        Class<L> listenerClass, Devices devices) {
      super(log, shell, analytics, client, listenerClass, devices);
    }

    protected void load(Path.Any source, boolean force) {
      load(Source.withSource(getSource(), source), force);
    }
  }
}
