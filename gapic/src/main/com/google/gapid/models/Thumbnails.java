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

import static com.google.gapid.image.FetchedImage.loadThumbnail;
import static com.google.gapid.util.Paths.thumbnail;

import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.proto.image.Image;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.server.Client;
import com.google.gapid.util.Events;
import com.google.gapid.util.Events.ListenerCollection;

import org.eclipse.swt.graphics.ImageData;
import org.eclipse.swt.internal.DPIUtil;

import java.util.function.Consumer;
import java.util.logging.Logger;

/**
 * Manages the loading of thumbnail previews.
 */
public class Thumbnails {
  protected static final Logger LOG = Logger.getLogger(ApiState.class.getName());

  public static final int THUMB_SIZE = 192;
  private static final int MIN_SIZE = DPIUtil.autoScaleUp(18);
  private static final int THUMB_PIXELS = DPIUtil.autoScaleUp(THUMB_SIZE);

  private final Client client;
  private final Devices devices;
  private final Capture capture;
  private final ListenerCollection<Listener> listeners = Events.listeners(Listener.class);

  public Thumbnails(Client client, Devices devices, Capture capture) {
    this.client = client;
    this.devices = devices;
    this.capture = capture;

    devices.addListener(new Devices.Listener() {
      @Override
      public void onReplayDeviceChanged() {
        update();
      }
    });
  }

  protected void update() {
    if (isReady()) {
      listeners.fire().onThumbnailsChanged();
    }
  }

  public boolean isReady() {
    return devices.hasReplayDevice() && capture.isLoaded();
  }

  public ListenableFuture<ImageData> getThumbnail(
      Path.Command command, int size, Consumer<Image.Info> onInfo) {
    return Futures.transform(loadThumbnail(client, thumbnail(command, THUMB_PIXELS), onInfo),
        image -> processImage(image, size));
  }

  public ListenableFuture<ImageData> getThumbnail(
      Path.CommandTreeNode node, int size, Consumer<Image.Info> onInfo) {
    return Futures.transform(loadThumbnail(client, thumbnail(node, THUMB_PIXELS), onInfo),
        image -> processImage(image, size));
  }

  private static ImageData processImage(ImageData image, int size) {
    size = DPIUtil.autoScaleUp(size);
    if (image.width >= image.height) {
      if (image.width > size) {
        return image.scaledTo(size, (image.height * size) / image.width);
      } else if (image.width < MIN_SIZE) {
        return image.scaledTo(MIN_SIZE, (image.height * MIN_SIZE) / image.width);
      }
    } else {
      if (image.height > size) {
        return image.scaledTo((image.width * size) / image.height, size);
      } else if (image.height < MIN_SIZE) {
        return image.scaledTo((image.width * MIN_SIZE) / image.height, MIN_SIZE);
      }
    }
    return image;
  }

  public void addListener(Listener listener) {
    listeners.addListener(listener);
  }

  public void removeListener(Listener listener) {
    listeners.removeListener(listener);
  }

  public static interface Listener extends Events.Listener {
    /**
     * Event indicating that render settings have changed an thumbnails need to be updated.
     */
    public default void onThumbnailsChanged() { /* empty */ }
  }
}
