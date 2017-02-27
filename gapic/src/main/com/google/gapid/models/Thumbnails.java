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
import static java.util.logging.Level.FINE;

import com.google.common.collect.Lists;
import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.common.util.concurrent.SettableFuture;
import com.google.gapid.Server.GapisInitException;
import com.google.gapid.image.FetchedImage;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.gfxapi.GfxAPI;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.server.Client;
import com.google.gapid.util.Events;
import com.google.gapid.util.Events.ListenerCollection;

import org.eclipse.swt.graphics.ImageData;
import org.eclipse.swt.internal.DPIUtil;

import java.util.List;
import java.util.PriorityQueue;
import java.util.Queue;
import java.util.concurrent.ExecutionException;
import java.util.logging.Logger;
import java.util.stream.Collectors;

/**
 * Manages the loading of thumbnail previews. In order to keep the UI responsive and give other
 * requests to the server a chance to complete, a single thread is used to request thumbnail
 * previews, which require a replay, from the server. In essence, this prevents request starvation
 * to the server by de-prioritizing requests for thumbnails. Thumbnail requests are batched to take
 * advantage of the batching optimization the server does for replay requests.
 */
public class Thumbnails {
  protected static final Logger LOG = Logger.getLogger(ApiState.class.getName());

  public static final int THUMB_SIZE = 192;

  private static final Service.RenderSettings RENDER_SETTINGS = Service.RenderSettings.newBuilder()
      .setMaxWidth(DPIUtil.autoScaleUp(THUMB_SIZE))
      .setMaxHeight(DPIUtil.autoScaleUp(THUMB_SIZE))
      .setWireframeMode(Service.WireframeMode.None)
      .build();

  private final Client client;
  private final Devices devices;
  private final AtomStream atoms;
  private final ListenerCollection<Listener> listeners = Events.listeners(Listener.class);
  private final Queue<Thumbnail> queuedThumbnails = new PriorityQueue<>();
  private final Thread processorThread;

  public Thumbnails(Client client, Devices devices, Capture capture, AtomStream atoms) {
    this.client = client;
    this.devices = devices;
    this.atoms = atoms;
    this.processorThread = new ProcessorThread(client, queuedThumbnails);

    devices.addListener(new Devices.Listener() {
      @Override
      public void onReplayDeviceChanged() {
        update();
      }
    });
    capture.addListener(new Capture.Listener() {
      @Override
      public void onCaptureLoaded(GapisInitException error) {
        if (error == null) {
          update();
        }
      }
    });
    atoms.addListener(new AtomStream.Listener() {
      @Override
      public void onAtomsLoaded() {
        update();
      }
    });

    processorThread.start();
  }

  protected void update() {
    if (isReady()) {
      listeners.fire().onThumnailsChanged();
    }
  }

  public boolean isReady() {
    return devices.hasReplayDevice() && atoms.isLoaded();
  }

  public ListenableFuture<ImageData> getThumbnail(long atomId, int size) {
    Thumbnail result = new Thumbnail(atomId, getPath(atomId));
    synchronized (queuedThumbnails) {
      queuedThumbnails.add(result);
      queuedThumbnails.notify();
    }
    return Futures.transform(result.result, image -> processImage(image, size));
  }

  public void dispose() {
    synchronized (queuedThumbnails) {
      while (!queuedThumbnails.isEmpty()) {
        Thumbnail thumbnail = queuedThumbnails.remove();
        thumbnail.pathFuture.cancel(true);
        thumbnail.result.cancel(true);
      }
    }
    processorThread.interrupt();
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
        command(atoms.getPath(), atomId), GfxAPI.FramebufferAttachment.Color0, RENDER_SETTINGS);
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
    public default void onThumnailsChanged() { /* empty */ }
  }

  /**
   * A queued thumbnail request and its result.
   */
  private static class Thumbnail implements Comparable<Thumbnail> {
    public final long atomId;
    public final ListenableFuture<Path.ImageInfo> pathFuture;
    public final SettableFuture<ImageData> result;

    public Thumbnail(long atomId, ListenableFuture<Path.ImageInfo> pathFuture) {
      this.atomId = atomId;
      this.pathFuture = pathFuture;
      this.result = SettableFuture.create();
    }

    @Override
    public int compareTo(Thumbnail o) {
      return Long.compare(atomId, o.atomId);
    }
  }

  /**
   * Thread that requests and processes all thumbnail previews..
   */
  private static class ProcessorThread extends Thread {
    private static final long BATCH_TIMEOUT_MS = 10;
    private static final int BATCH_SIZE = 25;

    private final Client client;
    private final Queue<Thumbnail> queue;

    public ProcessorThread(Client client, Queue<Thumbnail> queue) {
      super(ProcessorThread.class.getName());
      this.client = client;
      this.queue = queue;
      setDaemon(true);
    }

    @Override
    public void run() {
      while (true) {
        List<Thumbnail> batch = Lists.newArrayList();
        synchronized (queue) {
          while (queue.isEmpty()) {
            if (!await()) {
              return;
            }
          }

          for (long end = System.currentTimeMillis() + BATCH_TIMEOUT_MS;
              batch.size() < BATCH_SIZE && end > System.currentTimeMillis(); ) {
            while (!queue.isEmpty() && batch.size() < BATCH_SIZE) {
              Thumbnail thumbnail = queue.remove();
              if (!thumbnail.result.isCancelled()) {
                batch.add(thumbnail);
              }
            }
            if (batch.size() < BATCH_SIZE && !await()) {
              return;
            }
          }
        }
        if (!process(batch)) {
          return;
        }
      }
    }

    private boolean process(List<Thumbnail> batch) {
      LOG.log(FINE, "Processing a batch of " + batch.size() + " thumbnails.");
      long start = System.currentTimeMillis();
      try {
        Futures.whenAllComplete(batch.stream().map(thumbnail -> {
          thumbnail.result.setFuture(
              FetchedImage.loadLevel(FetchedImage.load(client, thumbnail.pathFuture), 0));
          return thumbnail.result;
        }).collect(Collectors.toList())).call(() -> null).get();
      } catch (ExecutionException e) {
        throw new AssertionError(); // "() -> null" above should not throw an exception.
      } catch (InterruptedException e) {
        LOG.log(FINE, "Aborting thumbnail fetching in process() due to interrupt");
        return false;
      }
      long end = System.currentTimeMillis();
      LOG.log(FINE,
          "Took " + (end - start) + "ms to process a batch of " + batch.size() + " thumbnails.");
      return true;
    }

    private boolean await() {
      try {
        queue.wait(BATCH_TIMEOUT_MS);
        return true;
      } catch (InterruptedException e) {
        LOG.log(FINE, "Aborting thumbnail fetching in await() due to interrupt");
        return false;
      }
    }
  }
}
