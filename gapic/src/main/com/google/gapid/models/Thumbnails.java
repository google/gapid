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

import static com.google.gapid.util.Paths.command;

import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.image.FetchedImage;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.gfxapi.GfxAPI;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.server.Client;

import org.eclipse.swt.graphics.ImageData;
import org.eclipse.swt.internal.DPIUtil;

import java.util.logging.Logger;

/**
 * Manages the loading of thumbnail previews.
 */
public class Thumbnails {
  protected static final Logger LOG = Logger.getLogger(ApiState.class.getName());

  public static final int THUMB_SIZE = 192;

  private static final Service.RenderSettings RENDER_SETTINGS = Service.RenderSettings.newBuilder()
      .setMaxWidth(DPIUtil.autoScaleUp(THUMB_SIZE))
      .setMaxHeight(DPIUtil.autoScaleUp(THUMB_SIZE))
      .setWireframeMode(Service.WireframeMode.None)
      .build();
  private static final Service.UsageHints HINTS = Service.UsageHints.newBuilder()
      .setPreview(true)
      .build();

  private final Client client;
  private final Devices devices;
  private final AtomStream atoms;

  public Thumbnails(Client client, Devices devices, AtomStream atoms) {
    this.client = client;
    this.devices = devices;
    this.atoms = atoms;
  }

  public boolean isReady() {
    return devices.hasReplayDevice() && atoms.isLoaded();
  }

  public ListenableFuture<ImageData> getThumbnail(long atomId, int size) {
    ListenableFuture<ImageData> future = FetchedImage.loadLevel(FetchedImage.load(client, getPath(atomId)), 0);
    return Futures.transform(future, image -> processImage(image, size));
  }

  private static ImageData processImage(ImageData image, int size) {
    size = DPIUtil.autoScaleUp(size);
    if (image.width > image.height && image.width > size) {
      return image.scaledTo(size, (image.height * size) / image.width);
    } else if (image.height > size) {
      return image.scaledTo((image.width * size) / image.height, size);
    } else {
      return image;
    }
  }

  private ListenableFuture<Path.ImageInfo> getPath(long atomId) {
    return client.getFramebufferAttachment(devices.getReplayDevice(),
        command(atoms.getPath(), atomId), GfxAPI.FramebufferAttachment.Color0, RENDER_SETTINGS, HINTS);
  }

}
