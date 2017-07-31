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

import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.server.Client;
import com.google.gapid.util.Events;

import org.eclipse.swt.widgets.Shell;

import java.util.List;
import java.util.logging.Logger;

/**
 * Model containing the capture resources (textures, shaders, etc.) metadata.
 */
public class Resources
    extends CaptureDependentModel.ForValue<Service.Resources, Resources.Listener> {
  private static final Logger LOG = Logger.getLogger(Resources.class.getName());

  public Resources(Shell shell, Client client, Capture capture) {
    super(LOG, shell, client, Listener.class, capture);
  }

  @Override
  protected Path.Any getPath(Path.Capture capturePath) {
    return Path.Any.newBuilder()
        .setResources(Path.Resources.newBuilder()
            .setCapture(capturePath))
        .build();
  }

  @Override
  protected Service.Resources unbox(Service.Value value) {
    return value.getResources();
  }

  @Override
  protected void fireLoadStartEvent() {
    // Do nothing.
  }

  @Override
  protected void fireLoadedEvent() {
    listeners.fire().onResourcesLoaded();
  }

  public List<Service.ResourcesByType> getResources() {
    return getData().getTypesList();
  }

  public static interface Listener extends Events.Listener {
    /**
     * Event indicating that the resources metadata has been loaded.
     */
    public default void onResourcesLoaded() { /* empty */ }
  }
}
