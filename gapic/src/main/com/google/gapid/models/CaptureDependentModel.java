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
package com.google.gapid.models;

import static com.google.gapid.models.DeviceDependentModel.Source.withSource;
import static com.google.gapid.util.MoreFutures.transform;

import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.server.Client;
import com.google.gapid.util.Events;
import com.google.gapid.util.Loadable;

import org.eclipse.swt.widgets.Shell;

import java.util.logging.Logger;

/**
 * Base class for models that depend on a capture. I.e. models that will trigger a load whenever
 * the capture changes and require a capture to be loaded.
 */
abstract class CaptureDependentModel
    <T extends DeviceDependentModel.Data, S, E, L extends Events.Listener>
    extends DeviceDependentModel<T, S, E, L> {

  public CaptureDependentModel(Logger log, Shell shell, Analytics analytics, Client client,
      Class<L> listenerClass, Capture capture, Devices devices) {
    super(log, shell, analytics, client, listenerClass, devices);

    capture.addListener(new Capture.Listener() {
      @Override
      public void onCaptureLoadingStart(boolean maintainState) {
        reset(maintainState);
      }

      @Override
      public void onCaptureLoaded(Loadable.Message error) {
        if (error == null && shouldLoad(capture.getData())) {
          load(withSource(getSource(), getSource(capture.getData())), false);
        } else {
          reset(false);
        }
      }
    });
  }

  /**
   * Whether this model should be loaded for the given capture.
   */
  protected abstract boolean shouldLoad(Capture.Data capture);

  protected abstract S getSource(Capture.Data capture);

  /**
   * @param maintainState whether the model should attempt to maintain its state.
   */
  protected void reset(boolean maintainState) {
    reset();
  }

  public abstract static class ForPath<T extends Data, L extends Events.Listener>
      extends CaptureDependentModel<T, Path.Any, Void, L> {

    public ForPath(Logger log, Shell shell, Analytics analytics, Client client,
        Class<L> listenerClass, Capture capture, Devices devices) {
      super(log, shell, analytics, client, listenerClass, capture, devices);
    }

    public void load(Path.Any path, boolean force) {
      load(withSource(getSource(), path), force);
    }
  }

  public abstract static class ForValue<T extends Data, L extends Events.Listener>
      extends CaptureDependentModel.ForPath<T, L> {
    public ForValue(Logger log, Shell shell, Analytics analytics, Client client,
        Class<L> listenerClass, Capture capture, Devices devices) {
      super(log, shell, analytics, client, listenerClass, capture, devices);
    }

    @Override
    protected ListenableFuture<T> doLoad(Path.Any path, Path.Device device) {
      return transform(client.get(path, device), val -> unbox(val, device));
    }

    protected abstract T unbox(Service.Value value, Path.Device device);
  }
}
