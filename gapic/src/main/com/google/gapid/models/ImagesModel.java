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

import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.image.FetchedImage;
import com.google.gapid.models.CommandStream.CommandIndex;
import com.google.gapid.proto.image.Image;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.api.API;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.server.Client;
import com.google.gapid.util.MoreFutures;
import com.google.gapid.util.Paths;

import org.eclipse.swt.graphics.ImageData;
import org.eclipse.swt.internal.DPIUtil;

import java.util.function.Consumer;
import java.util.logging.Logger;

/**
 * Manages the loading of thumbnail previews and texture and framebuffer images.
 */
public class ImagesModel {
  protected static final Logger LOG = Logger.getLogger(ImagesModel.class.getName());

  public static final int THUMB_SIZE = 192;
  private static final int MIN_SIZE = DPIUtil.autoScaleUp(18);
  private static final int THUMB_PIXELS = DPIUtil.autoScaleUp(THUMB_SIZE);
  private static final Service.UsageHints FB_HINTS = Service.UsageHints.newBuilder()
      .setPrimary(true)
      .build();

  private final Client client;
  private final Devices devices;
  private final Capture capture;
  private final Settings settings;

  public ImagesModel(Client client, Devices devices, Capture capture, Settings settings) {
    this.client = client;
    this.devices = devices;
    this.capture = capture;
    this.settings = settings;
  }

  public boolean isReady() {
    return devices.hasReplayDevice() && capture.isLoaded();
  }

  public ListenableFuture<FetchedImage> getFramebuffer(CommandIndex command,
      API.FramebufferAttachment attachment, Service.RenderSettings renderSettings) {
    return FetchedImage.load(client, getReplayDevice(), client.getFramebufferAttachment(
        getReplayDevice(), command.getCommand(), attachment, renderSettings,
        FB_HINTS, shouldDisableReplayOptimization()));
  }

  public ListenableFuture<FetchedImage> getResource(Path.ResourceData path) {
    return FetchedImage.load(client, getReplayDevice(), path);
  }

  public ListenableFuture<ImageData> getThumbnail(
      Path.Command command, int size, Consumer<Image.Info> onInfo) {
    return MoreFutures.transform(loadThumbnail(client, getReplayDevice(), thumbnail(command), onInfo),
        image -> processImage(image, size));
  }

  public ListenableFuture<ImageData> getThumbnail(
      Path.CommandTreeNode node, int size, Consumer<Image.Info> onInfo) {
    return MoreFutures.transform(loadThumbnail(client, getReplayDevice(), thumbnail(node), onInfo),
        image -> processImage(image, size));
  }

  public ListenableFuture<ImageData> getThumbnail(
      Path.ResourceData resource, int size, Consumer<Image.Info> onInfo) {
    return MoreFutures.transform(loadThumbnail(client, getReplayDevice(), thumbnail(resource), onInfo),
        image -> processImage(image, size));
  }

  private Path.Thumbnail thumbnail(Path.Command command) {
    return Paths.thumbnail(command, THUMB_PIXELS, shouldDisableReplayOptimization());
  }

  private Path.Thumbnail thumbnail(Path.CommandTreeNode node) {
    return Paths.thumbnail(node, THUMB_PIXELS, shouldDisableReplayOptimization());
  }

  private Path.Thumbnail thumbnail(Path.ResourceData resource) {
    return Paths.thumbnail(resource, THUMB_PIXELS, shouldDisableReplayOptimization());
  }

  private Path.Device getReplayDevice() {
    return devices.getReplayDevicePath();
  }

  private boolean shouldDisableReplayOptimization() {
    return settings.preferences().getDisableReplayOptimization();
  }

  private static ImageData processImage(ImageData image, int size) {
    size = DPIUtil.autoScaleUp(size);
    if (image.width >= image.height) {
      if (image.width > size) {
        return image.scaledTo(size, Math.max(1, (image.height * size) / image.width));
      } else if (image.width < MIN_SIZE) {
        return image.scaledTo(MIN_SIZE, Math.max(1, (image.height * MIN_SIZE) / image.width));
      }
    } else {
      if (image.height > size) {
        return image.scaledTo(Math.max(1, (image.width * size) / image.height), size);
      } else if (image.height < MIN_SIZE) {
        return image.scaledTo(Math.max(1, (image.width * MIN_SIZE) / image.height), MIN_SIZE);
      }
    }
    return image;
  }
}
