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
package com.google.gapid.views;

import static com.google.gapid.image.Images.noAlpha;
import static com.google.gapid.models.Thumbnails.THUMB_SIZE;
import static com.google.gapid.util.GeoUtils.left;
import static com.google.gapid.util.GeoUtils.right;
import static com.google.gapid.util.Loadable.MessageType.Error;
import static com.google.gapid.util.Loadable.MessageType.Info;
import static com.google.gapid.widgets.Widgets.redrawIfNotDisposed;

import com.google.common.collect.Lists;
import com.google.gapid.Server.GapisInitException;
import com.google.gapid.models.ApiContext;
import com.google.gapid.models.ApiContext.FilteringContext;
import com.google.gapid.models.AtomStream;
import com.google.gapid.models.AtomStream.AtomIndex;
import com.google.gapid.models.Capture;
import com.google.gapid.models.Models;
import com.google.gapid.models.Thumbnails;
import com.google.gapid.models.Timeline;
import com.google.gapid.proto.service.Service;
import com.google.gapid.util.BigPoint;
import com.google.gapid.util.Messages;
import com.google.gapid.widgets.InfiniteScrolledComposite;
import com.google.gapid.widgets.LoadableImage;
import com.google.gapid.widgets.LoadablePanel;
import com.google.gapid.widgets.LoadingIndicator;
import com.google.gapid.widgets.Widgets;

import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.GC;
import org.eclipse.swt.graphics.Image;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.graphics.Rectangle;
import org.eclipse.swt.layout.FillLayout;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;

import java.math.BigInteger;
import java.util.ArrayList;
import java.util.Collections;
import java.util.Iterator;
import java.util.List;
import java.util.function.Consumer;

/**
 * Scrubber view displaying thumbnails of the frames in the current capture.
 */
public class ThumbnailScrubber extends Composite
    implements Tab, Capture.Listener, AtomStream.Listener, ApiContext.Listener, Timeline.Listener {
  private final Models models;
  private final LoadablePanel<InfiniteScrolledComposite> loading;
  private final InfiniteScrolledComposite scroll;
  private final Carousel carousel;

  public ThumbnailScrubber(Composite parent, Models models, Widgets widgets) {
    super(parent, SWT.NONE);
    this.models = models;

    setLayout(new FillLayout(SWT.VERTICAL));

    carousel = new Carousel(
        this, models.thumbs, widgets, this::redrawScroll, this::resizeScroll, this::scrollTo);
    loading = new LoadablePanel<InfiniteScrolledComposite>(this, widgets,
        panel -> new InfiniteScrolledComposite(panel, SWT.H_SCROLL | SWT.V_SCROLL, carousel));
    scroll = loading.getContents();

    scroll.addContentListener(SWT.MouseDown, e -> {
      Data frame = carousel.selectFrame(scroll.getLocation(e));
      if (frame != null) {
        models.atoms.selectAtoms(frame.range, false);
      }
    });
    scroll.setCursor(getDisplay().getSystemCursor(SWT.CURSOR_HAND));

    models.capture.addListener(this);
    models.contexts.addListener(this);
    models.timeline.addListener(this);
    models.atoms.addListener(this);
    addListener(SWT.Dispose, e -> {
      models.capture.removeListener(this);
      models.contexts.removeListener(this);
      models.timeline.removeListener(this);
      models.atoms.removeListener(this);
      carousel.dispose();
    });
  }

  private void redrawScroll() {
    redrawIfNotDisposed(scroll);
  }

  private void resizeScroll() {
    scroll.updateMinSize();
  }

  private void scrollTo(BigInteger x) {
    scroll.scrollTo(
        x.subtract(BigInteger.valueOf(scroll.getClientArea().width / 2)).max(BigInteger.ZERO),
        BigInteger.ZERO);
  }

  @Override
  public Control getControl() {
    return this;
  }

  @Override
  public void reinitialize() {
    onCaptureLoadingStart(false);
    if (models.capture.isLoaded()) {
      updateScrubber();
    }
  }

  @Override
  public void onCaptureLoadingStart(boolean maintainState) {
    loading.showMessage(Info, Messages.LOADING_CAPTURE);
    carousel.setData(Collections.emptyList());
  }

  @Override
  public void onCaptureLoaded(GapisInitException error) {
    if (error != null) {
      loading.showMessage(Error, Messages.CAPTURE_LOAD_FAILURE);
      carousel.setData(Collections.emptyList());
    } else {
      updateScrubber();
    }
  }

  @Override
  public void onContextsLoaded() {
    if (!models.contexts.isLoaded()) {
      loading.showMessage(Error, Messages.CAPTURE_LOAD_FAILURE);
    } else {
      updateScrubber();
    }
  }

  @Override
  public void onContextSelected(FilteringContext context) {
    updateScrubber();
  }

  @Override
  public void onTimeLineLoadingStart() {
    updateScrubber();
  }

  @Override
  public void onTimeLineLoaded() {
    updateScrubber();
  }

  @Override
  public void onAtomsSelected(AtomIndex range) {
    carousel.selectFrame(range);
  }

  private void updateScrubber() {
    if (models.timeline.isLoaded()) {
      loading.stopLoading();
      List<Data> datas = prepareData(models.timeline.getEndOfFrames());
      if (datas.isEmpty()) {
        loading.showMessage(Info, Messages.NO_FRAMES_IN_CONTEXT);
      } else {
        loading.stopLoading();
        carousel.setData(datas);
        scroll.updateMinSize();

        if (models.atoms.getSelectedAtoms() != null) {
          carousel.selectFrame(models.atoms.getSelectedAtoms());
        }
      }
    } else {
      loading.startLoading();
    }
  }

  private static List<Data> prepareData(Iterator<Service.Event> events) {
    List<Data> generatedList = new ArrayList<>();
    int frameCount = 0;
    while (events.hasNext()) {
      generatedList.add(new Data(AtomIndex.forGroup(events.next().getCommand()), ++frameCount));
    }
    return generatedList;
  }

  /**
   * Metadata about a frame in the scrubber.
   */
  private static class Data {
    public final AtomIndex range;
    public final int frame;

    public LoadableImage image;

    public Data(AtomIndex range, int frame) {
      this.range = range;
      this.frame = frame;
    }

    public void paint(GC gc, Image toDraw, int x, int y, int w, int h, boolean selected) {
      if (selected) {
        gc.setForeground(gc.getDevice().getSystemColor(SWT.COLOR_LIST_SELECTION));
        gc.drawRectangle(x - 2, y - 2, w + 3, h + 3);
        gc.drawRectangle(x - 1, y - 1, w + 1, h + 1);
      } else {
        gc.setForeground(gc.getDevice().getSystemColor(SWT.COLOR_WIDGET_BORDER));
        gc.drawRectangle(x - 1, y - 1, w + 1, h + 1);
      }

      Rectangle size = toDraw.getBounds();
      gc.drawImage(toDraw, 0, 0, size.width, size.height,
          x + (w - size.width) / 2, y + (h - size.height) / 2,
          size.width, size.height);

      String label = String.valueOf(frame);
      Point labelSize = gc.stringExtent(label);
      gc.setForeground(gc.getDevice().getSystemColor(SWT.COLOR_LIST_FOREGROUND));
      gc.setBackground(gc.getDevice().getSystemColor(SWT.COLOR_LIST_BACKGROUND));
      gc.fillRoundRectangle(x + 4, y + 4, labelSize.x + 4, labelSize.y + 4, 6, 6);
      gc.drawRoundRectangle(x + 4, y + 4, labelSize.x + 4, labelSize.y + 4, 6, 6);
      gc.drawString(label, x + 6, y + 6);
    }

    public void dispose() {
      if (image != null) {
        image.dispose();
      }
    }
  }

  /**
   * Renders the frame thumbnails.
   */
  private static class Carousel
      implements InfiniteScrolledComposite.Scrollable, Thumbnails.Listener {
    private static final int MARGIN = 4;
    private static final int MIN_SIZE = 80;

    private final Control parent;
    private final Thumbnails thumbs;
    private final Widgets widgets;
    private final LoadingIndicator.Repaintable repainter;
    private final Runnable updateSize;
    private final Consumer<BigInteger> scrollTo;
    private List<Data> datas = Collections.emptyList();
    private Point imageSize;
    private int selectedIndex = -1;

    public Carousel(Control parent, Thumbnails thumbs, Widgets widgets,
        LoadingIndicator.Repaintable repainter, Runnable updateSize,
        Consumer<BigInteger> scrollTo) {
      this.parent = parent;
      this.thumbs = thumbs;
      this.widgets = widgets;
      this.repainter = repainter;
      this.updateSize = updateSize;
      this.scrollTo = scrollTo;

      thumbs.addListener(this);
    }

    public Data selectFrame(BigPoint point) {
      int frame = point.x.divide(BigInteger.valueOf(getCellWidth())).intValueExact();
      if (frame < 0 || frame >= datas.size()) {
        return null;
      }

      selectedIndex = frame;
      repainter.repaint();
      return datas.get(frame);
    }

    public void selectFrame(AtomIndex range) {
      int index = Collections.<Data>binarySearch(datas, null,
          (x, ignored) -> x.range.compareTo(range));
      if (index < 0) {
        index = -index - 1;
      }
      selectAndScroll(index);
      repainter.repaint();
    }

    public void setData(List<Data> newDatas) {
      dispose();
      datas = Lists.newArrayList(newDatas);
    }

    public void dispose() {
      thumbs.removeListener(this);
      for (Data data : datas) {
        data.dispose();
      }
      datas = Collections.emptyList();
      imageSize = null;
      selectedIndex = -1;
    }

    @Override
    public void onThumbnailsChanged() {
      for (Data data : datas) {
        if (data.image != null) {
          data.image.dispose();
          data.image = null;
        }
      }
      repainter.repaint();
    }

    @Override
    public BigInteger getWidth() {
      return BigInteger.valueOf(datas.size() * (long)getCellWidth());
    }

    private int getCellWidth() {
      return ((imageSize == null) ? THUMB_SIZE : imageSize.x) + MARGIN * 2;
    }

    @Override
    public BigInteger getHeight() {
      return BigInteger.valueOf(((imageSize == null) ? THUMB_SIZE : imageSize.y) + MARGIN);
    }

    @Override
    public void paint(BigInteger xOffset, BigInteger yOffset, GC gc) {
      if (datas.isEmpty()) {
        return;
      }

      Rectangle clip = gc.getClipping();
      Point size = (imageSize == null) ? new Point(THUMB_SIZE, THUMB_SIZE) : imageSize;
      int first = (int)((xOffset.longValueExact() + left(clip)) / (size.x + 2 * MARGIN));
      int last = Math.min(datas.size(),
          (int)((xOffset.longValueExact() + right(clip) + size.x - 1) / size.x));
      int x = (int)(first * ((long)size.x + 2 * MARGIN) - xOffset.longValueExact());

      prepareImages(first, last);
      for (int i = first; i < last; i++, x += size.x + 2 * MARGIN) {
        Data data = datas.get(i);
        Image toDraw;
        if (data.image != null) {
          toDraw = data.image.getImage();
        } else {
          toDraw = widgets.loading.getCurrentFrame();
          widgets.loading.scheduleForRedraw(repainter);
        }
        data.paint(gc, toDraw, x + MARGIN, MARGIN / 2, size.x, size.y, i == selectedIndex);
      }
      updateSize(first, last);
    }

    private void prepareImages(int first, int last) {
      for (int i = first; i < last; i++) {
        Data data = datas.get(i);
        if (data.image == null && thumbs.isReady()) {
          data.image = LoadableImage.newBuilder(widgets.loading)
              .forImageData(noAlpha(thumbs.getThumbnail(data.range.getCommand(), THUMB_SIZE)))
              .onErrorShowErrorIcon(widgets.theme)
              .build(parent, repainter);
        }
      }
    }

    private void updateSize(int first, int last) {
      boolean dirty = false;
      for (int i = first; i < last; i++) {
        Data data = datas.get(i);
        if (data.image != null && data.image.hasFinished()) {
          Rectangle bounds = data.image.getImage().getBounds();
          if (imageSize == null) {
            imageSize =
                new Point(Math.max(MIN_SIZE, bounds.width), Math.max(MIN_SIZE, bounds.height));
            dirty = true;
          } else if (bounds.width > imageSize.x || bounds.height > imageSize.y) {
            imageSize.x = Math.max(bounds.width, imageSize.x);
            imageSize.y = Math.max(bounds.height, imageSize.y);
            dirty = true;
          }
        }
      }

      if (dirty) {
        updateSize.run();
        selectAndScroll(selectedIndex);
        repainter.repaint();
      }
    }

    private void selectAndScroll(int index) {
      selectedIndex = index;
      if (index >= 0) {
        scrollTo.accept(
            BigInteger.valueOf(selectedIndex).multiply(BigInteger.valueOf(getCellWidth())));
      }
    }
  }
}
