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
import static com.google.gapid.models.ImagesModel.THUMB_SIZE;
import static com.google.gapid.util.Loadable.MessageType.Info;
import static com.google.gapid.widgets.Widgets.scheduleIfNotDisposed;

import com.google.common.collect.Lists;
import com.google.gapid.models.Capture;
import com.google.gapid.models.CommandStream;
import com.google.gapid.models.CommandStream.CommandIndex;
import com.google.gapid.models.ImagesModel;
import com.google.gapid.models.Models;
import com.google.gapid.models.Timeline;
import com.google.gapid.proto.service.Service;
import com.google.gapid.util.Loadable;
import com.google.gapid.util.Messages;
import com.google.gapid.widgets.HorizontalList;
import com.google.gapid.widgets.LoadableImage;
import com.google.gapid.widgets.LoadablePanel;
import com.google.gapid.widgets.LoadingIndicator;
import com.google.gapid.widgets.Widgets;

import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.GC;
import org.eclipse.swt.graphics.Image;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.graphics.Rectangle;
import org.eclipse.swt.internal.DPIUtil;
import org.eclipse.swt.layout.FillLayout;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;

import java.util.Collections;
import java.util.Iterator;
import java.util.List;

/**
 * Scrubber view displaying thumbnails of the frames in the current capture.
 */
public class ThumbnailScrubber extends Composite
    implements Tab, Capture.Listener, CommandStream.Listener, Timeline.Listener {
  private final Models models;
  private final LoadablePanel<Carousel> loading;
  private final Carousel carousel;

  public ThumbnailScrubber(Composite parent, Models models, Widgets widgets) {
    super(parent, SWT.NONE);
    this.models = models;

    setLayout(new FillLayout(SWT.VERTICAL));

    loading = new LoadablePanel<Carousel>(this, widgets,
        panel -> new Carousel(panel, models.images, widgets));
    carousel = loading.getContents();

    carousel.addContentListener(SWT.MouseDown, e -> {
      Data frame = carousel.selectFrame(carousel.getItemAt(e.x));
      if (frame != null) {
        models.commands.selectCommands(frame.range, false);
      }
    });
    carousel.setCursor(getDisplay().getSystemCursor(SWT.CURSOR_HAND));

    models.capture.addListener(this);
    models.timeline.addListener(this);
    models.commands.addListener(this);
    addListener(SWT.Dispose, e -> {
      models.capture.removeListener(this);
      models.timeline.removeListener(this);
      models.commands.removeListener(this);
      carousel.reset();
    });
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
  public void onCaptureLoaded(Loadable.Message error) {
    if (error != null) {
      loading.showMessage(error);
      carousel.setData(Collections.emptyList());
    } else {
      updateScrubber();
    }
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
  public void onCommandsSelected(CommandIndex range) {
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

        if (models.commands.getSelectedCommands() != null) {
          carousel.selectFrame(models.commands.getSelectedCommands());
        }
      }
    } else {
      loading.startLoading();
    }
  }

  private static List<Data> prepareData(Iterator<Service.Event> events) {
    List<Data> generatedList = Lists.newArrayList();
    int frameCount = 0;
    while (events.hasNext()) {
      generatedList.add(new Data(CommandIndex.forGroup(events.next().getCommand()), ++frameCount));
    }
    return generatedList;
  }

  /**
   * Metadata about a frame in the scrubber.
   */
  private static class Data {
    public final CommandIndex range;
    public final int frame;

    public LoadableImage image;

    public Data(CommandIndex range, int frame) {
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
  private static class Carousel extends HorizontalList implements LoadingIndicator.Repaintable {
    private static final int MIN_SIZE = 80;

    private final ImagesModel thumbs;
    private final Widgets widgets;
    private List<Data> datas = Collections.emptyList();
    private int selectedIndex = -1;

    public Carousel(Composite parent, ImagesModel thumbs, Widgets widgets) {
      super(parent);
      this.thumbs = thumbs;
      this.widgets = widgets;
    }

    @Override
    protected void paint(GC gc, int index, int x, int y, int w, int h) {
      Data data = datas.get(index);
      if (data.image == null && thumbs.isReady()) {
        load(data, index);
      }

      Image toDraw;
      if (data.image != null) {
        toDraw = data.image.getImage();
      } else {
        toDraw = widgets.loading.getCurrentFrame();
        widgets.loading.scheduleForRedraw(this);
      }
      data.paint(gc, toDraw, x, y, w, h, index == selectedIndex);
    }

    private void load(Data data, int index) {
      data.image = LoadableImage.newBuilder(widgets.loading)
          .forImageData(noAlpha(thumbs.getThumbnail(data.range.getCommand(), THUMB_SIZE,
              info -> scheduleIfNotDisposed(this, () -> setItemSize(index,
                  Math.max(MIN_SIZE, DPIUtil.autoScaleDown(info.getWidth())),
                  Math.max(MIN_SIZE, DPIUtil.autoScaleDown(info.getHeight())))))))
          .onErrorShowErrorIcon(widgets.theme)
          .build(this, this);
      data.image.addListener(new LoadableImage.Listener() {
        @Override
        public void onLoaded(boolean success) {
          Rectangle bounds = data.image.getImage().getBounds();
          setItemSize(index, Math.max(MIN_SIZE, bounds.width), Math.max(MIN_SIZE, bounds.height));
        }
      });
    }

    public Data selectFrame(int frame) {
      if (frame < 0 || frame >= datas.size()) {
        return null;
      }

      selectedIndex = frame;
      repaint();
      return datas.get(frame);
    }

    public void selectFrame(CommandIndex range) {
      int index = Collections.<Data>binarySearch(datas, null,
          (x, ignored) -> x.range.compareTo(range));
      if (index < 0) {
        index = -index - 1;
      }
      selectAndScroll(index);
      repaint();
    }

    public void setData(List<Data> newDatas) {
      reset();
      datas = Lists.newArrayList(newDatas);
      setItemCount(datas.size(), THUMB_SIZE, THUMB_SIZE);
    }

    public void reset() {
      for (Data data : datas) {
        data.dispose();
      }
      datas = Collections.emptyList();
      selectedIndex = -1;
      setItemCount(0, THUMB_SIZE, THUMB_SIZE);
    }

    private void selectAndScroll(int index) {
      selectedIndex = index;
      if (index >= 0) {
        scrollIntoView(index);
      }
    }
  }
}
