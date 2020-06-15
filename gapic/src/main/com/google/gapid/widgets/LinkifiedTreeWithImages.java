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
package com.google.gapid.widgets;

import static com.google.gapid.models.ImagesModel.THUMB_SIZE;
import static com.google.gapid.util.GeoUtils.vertCenter;

import com.google.common.collect.Maps;
import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.proto.service.path.Path.Any;
import com.google.gapid.util.Scheduler;

import org.eclipse.swt.graphics.GC;
import org.eclipse.swt.graphics.Image;
import org.eclipse.swt.graphics.ImageData;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.graphics.Rectangle;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Event;
import org.eclipse.swt.widgets.Shell;
import org.eclipse.swt.widgets.TreeItem;

import java.util.Map;
import java.util.concurrent.Future;
import java.util.concurrent.TimeUnit;

/**
 * A {@link LinkifiedTree} that also shows an optional image to the left of the label.
 */
public abstract class LinkifiedTreeWithImages<T, F> extends LinkifiedTree<T, F> {
  private static final int IMAGE_SIZE = 18, IMAGE_PADDING = 2;
  private static final int TEXT_START = IMAGE_SIZE + IMAGE_PADDING;

  protected final ImageProvider imageProvider;

  public LinkifiedTreeWithImages(Composite parent, int treeStyle, Widgets widgets) {
    super(parent, treeStyle, widgets);
    this.imageProvider = new ImageProvider(widgets.loading);
  }

  @Override
  protected LabelProvider createLabelProvider(Theme theme) {
    return new ImageLabelProvider(theme);
  }

  @Override
  public void setInput(T root) {
    imageProvider.reset();
    super.setInput(root);
  }

  public void refreshImages() {
    imageProvider.reset();
    refresher.refresh();
  }

  @Override
  protected void reset() {
    super.reset();
    imageProvider.reset();
  }

  protected abstract boolean shouldShowImage(T node);
  protected abstract ListenableFuture<ImageData> loadImage(T node, int size);
  protected abstract void createImagePopupContents(Shell shell, T node);

  private class ImageProvider implements LoadingIndicator.Repaintable {
    private final LoadingIndicator loading;
    private final Map<T, LoadableImage> images = Maps.newIdentityHashMap();

    public ImageProvider(LoadingIndicator loading) {
      this.loading = loading;
    }

    public void load(T node) {
      LoadableImage image = getLoadableImage(node);
      if (image != null) {
        image.load();
      }
    }

    public void unload(T node) {
      LoadableImage image = images.get(node);
      if (image != null) {
        image.unload();
      }
    }

    public Image getImage(T node) {
      LoadableImage image = getLoadableImage(node);
      return (image == null) ? null : image.getImage();
    }

    private LoadableImage getLoadableImage(T node) {
      LoadableImage image = images.get(node);
      if (image == null) {
        if (!shouldShowImage(node)) {
          return null;
        }

        image = LoadableImage.newBuilder(loading)
            .small()
            .forImageData(() -> loadImage(node, IMAGE_SIZE))
            .onErrorReturnNull()
            .build(getControl(), this);
        images.put(node, image);
      }
      return image;
    }

    @Override
    public void repaint() {
      refresher.refresh();
    }

    public void reset() {
      for (LoadableImage image : images.values()) {
        image.dispose();
      }
      images.clear();
    }
  }

  private class ImageLabelProvider extends LabelProvider {
    private static final int PREVIEW_HOVER_DELAY_MS = 500;

    private TreeItem lastHoveredImage;
    private Future<?> lastScheduledFuture = Futures.immediateFuture(null);
    private Balloon lastShownBalloon;


    public ImageLabelProvider(Theme theme) {
      super(theme);
      //imageProvider.getImage(cast(element))
    }

    @Override
    public void onShow(TreeItem item) {
      super.onShow(item);
      imageProvider.load(getElement(item));
    }

    @Override
    public void onHide(TreeItem item) {
      super.onHide(item);
      imageProvider.unload(getElement(item));
    }

    @Override
    protected void measure(Event event, Object element) {
      super.measure(event, element);
      event.width += TEXT_START;
      event.height = Math.max(event.height, IMAGE_SIZE);
    }

    @Override
    protected void drawText(T node, GC gc, Rectangle bounds, Label label, boolean ignoreColors) {
      Image image = imageProvider.getImage(node);
      if (image != null) {
        Rectangle size = image.getBounds();
        gc.drawImage(image,
            bounds.x + (IMAGE_SIZE - size.width) / 2, bounds.y + (bounds.height - size.height) / 2);
      }

      bounds.x += TEXT_START;
      bounds.width -= TEXT_START;
      super.drawText(node, gc, bounds, label, ignoreColors);
    }

    @Override
    public boolean hoverItem(TreeItem item, Point location) {
      boolean result = super.hoverItem(item, location);
      if (item != null && isWithinImageBounds(item, location)) {
        hoverImage(item);
        result = false;
      } else {
        hoverImage(null);
      }
      return result;
    }

    private void hoverImage(TreeItem item) {
      if (item != lastHoveredImage) {
        lastScheduledFuture.cancel(true);
        lastHoveredImage = item;
        if (item != null) {
          lastScheduledFuture = Scheduler.EXECUTOR.schedule(() ->
              Widgets.scheduleIfNotDisposed(item, () -> showBalloon(item)),
              PREVIEW_HOVER_DELAY_MS, TimeUnit.MILLISECONDS);
        }
        if (lastShownBalloon != null) {
          lastShownBalloon.close();
        }
      }
    }

    private void showBalloon(TreeItem item) {
      if (lastShownBalloon != null) {
        lastShownBalloon.close();
      }
      Rectangle bounds = item.getTextBounds(0);
      lastShownBalloon = Balloon.createAndShow(item.getParent(), shell -> {
        createImagePopupContents(shell, getElement(item));
      }, new Point(bounds.x + TEXT_START, vertCenter(bounds) - THUMB_SIZE / 2));

    }

    private boolean isWithinImageBounds(TreeItem item, Point location) {
      if (item == null || imageProvider.getImage(getElement(item)) == null) {
        return false;
      }
      Rectangle bounds = item.getTextBounds(0);
      bounds.width = TEXT_START;
      return bounds.contains(location);
    }

    @Override
    public Any getFollow(TreeItem item, Point location) {
      location.x -= TEXT_START;
      return super.getFollow(item, location);
    }
  }
}
