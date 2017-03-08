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
import static com.google.gapid.util.Loadable.MessageType.Error;
import static com.google.gapid.util.Loadable.MessageType.Info;
import static com.google.gapid.util.Ranges.commands;
import static com.google.gapid.util.Ranges.end;
import static com.google.gapid.util.Ranges.first;
import static com.google.gapid.widgets.Widgets.redrawIfNotDisposed;

import com.google.common.collect.Lists;
import com.google.gapid.Server.GapisInitException;
import com.google.gapid.models.ApiContext;
import com.google.gapid.models.ApiContext.FilteringContext;
import com.google.gapid.models.AtomStream;
import com.google.gapid.models.Capture;
import com.google.gapid.models.Models;
import com.google.gapid.models.Thumbnails;
import com.google.gapid.proto.service.Service.CommandRange;
import com.google.gapid.service.atom.AtomList;
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
import java.util.List;

public class ThumbnailScrubber extends Composite
    implements Capture.Listener, AtomStream.Listener, ApiContext.Listener {
  private final Models models;
  private final LoadablePanel<InfiniteScrolledComposite> loading;
  private final InfiniteScrolledComposite scroll;
  private final Carousel carousel;

  public ThumbnailScrubber(Composite parent, Models models, Widgets widgets) {
    super(parent, SWT.NONE);
    this.models = models;

    setLayout(new FillLayout(SWT.VERTICAL));

    carousel = new Carousel(
        this, models.thumbs, widgets.loading, this::redrawScroll, this::resizeScroll);
    loading = new LoadablePanel<>(this, widgets,
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
    models.atoms.addListener(this);
    models.contexts.addListener(this);
  }

  private void redrawScroll() {
    redrawIfNotDisposed(scroll);
  }

  private void resizeScroll() {
    scroll.updateMinSize();
  }

  @Override
  public void dispose() {
    carousel.dispose();
    super.dispose();
  }

  @Override
  public void onCaptureLoadingStart() {
    loading.showMessage(Info, Messages.LOADING_CAPTURE);
    carousel.setData(Collections.emptyList());
  }

  @Override
  public void onCaptureLoaded(GapisInitException error) {
    if (error != null) {
      loading.showMessage(Error, Messages.CAPTURE_LOAD_FAILURE);
      carousel.setData(Collections.emptyList());
    }
  }

  @Override
  public void onAtomsLoaded() {
    if (!models.atoms.isLoaded()) {
      loading.showMessage(Error, Messages.CAPTURE_LOAD_FAILURE);
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

  private void updateScrubber() {
    if (models.atoms.isLoaded() && models.contexts.isLoaded()) {
      List<Data> datas = prepareData(models.atoms.getData(), models.contexts.getSelectedContext());
      if (datas.isEmpty()) {
        loading.showMessage(Info, Messages.NO_FRAMES_IN_CONTEXT);
      } else {
        loading.stopLoading();
        carousel.setData(datas);
        scroll.updateMinSize();
      }
    }
  }

  private static List<Data> prepareData(AtomList atoms, FilteringContext context) {
    List<Data> generatedList = new ArrayList<>();
    int frameCount = 0;
    long frameStart = -1, drawCall = -1;
    for (CommandRange contextRange : context.getRanges(atoms)) {
      for (long index = first(contextRange); index < end(contextRange); index++) {
        if (frameStart < 0) {
          frameStart = index;
        }
        if (atoms.get(index).isDrawCall()) {
          drawCall = index;
        }
        if (atoms.get(index).isEndOfFrame()) {
          CommandRange frameRange = commands(frameStart, index - frameStart + 1);
          Data frameData = new Data(frameRange, (drawCall < 0) ? index : drawCall, ++frameCount);
          generatedList.add(frameData);
          frameStart = drawCall = -1;
        }
      }
    }
    return generatedList;
  }

  private static class Data {
    public final CommandRange range;
    public final long previewAtomIndex;
    public final int frame;

    public LoadableImage image;

    public Data(CommandRange range, long previewAtomIndex, int frame) {
      this.range = range;
      this.previewAtomIndex = previewAtomIndex;
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

  private static class Carousel
      implements InfiniteScrolledComposite.Scrollable, Thumbnails.Listener {
    private static final int MARGIN = 4;
    private static final int MIN_SIZE = 80;

    private final Control parent;
    private final Thumbnails thumbs;
    private final LoadingIndicator loading;
    private final LoadingIndicator.Repaintable repainter;
    private final Runnable updateSize;
    private List<Data> datas = Collections.emptyList();
    private Point imageSize;
    private int selectedIndex = -1;

    public Carousel(Control parent, Thumbnails thumbs, LoadingIndicator loading,
        LoadingIndicator.Repaintable repainter, Runnable updateSize) {
      this.parent = parent;
      this.thumbs = thumbs;
      this.loading = loading;
      this.repainter = repainter;
      this.updateSize = updateSize;

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

    public void setData(List<Data> newDatas) {
      dispose();
      datas = Lists.newArrayList(newDatas);
    }

    public void dispose() {
      for (Data data : datas) {
        data.dispose();
      }
      datas = Collections.emptyList();
      imageSize = null;
      selectedIndex = -1;
    }

    @Override
    public void onThumnailsChanged() {
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
      int first = (int)((xOffset.longValueExact() + clip.x) / (size.x + 2 * MARGIN));
      int last = (int)((xOffset.longValueExact() + clip.x + clip.width + size.x - 1) / size.x);
      int x = (int)(first * ((long)size.x + 2 * MARGIN) - xOffset.longValueExact());

      for (int i = first; i < last && i < datas.size(); i++, x += size.x + 2 * MARGIN) {
        Data data = datas.get(i);
        if (data.image == null && thumbs.isReady()) {
          data.image = LoadableImage.forImageData(parent,
              noAlpha(thumbs.getThumbnail(data.previewAtomIndex, THUMB_SIZE)), loading, repainter);
        }
        Image toDraw;
        if (data.image != null) {
          toDraw = data.image.getImage();
          if (data.image.hasFinished()) {
            Rectangle bounds = toDraw.getBounds();
            if (imageSize == null) {
              imageSize =
                  new Point(Math.max(MIN_SIZE, bounds.width), Math.max(MIN_SIZE, bounds.height));
              updateSize.run();
            } else if (bounds.width > imageSize.x || bounds.height > imageSize.y) {
              imageSize.x = Math.max(bounds.width, imageSize.x);
              imageSize.y = Math.max(bounds.height, imageSize.y);
              updateSize.run();
            }
          }
        } else {
          toDraw = loading.getCurrentFrame();
          loading.scheduleForRedraw(repainter);
        }
        data.paint(gc, toDraw, x + MARGIN, MARGIN / 2, size.x, size.y, i == selectedIndex);
      }
    }
  }
}
