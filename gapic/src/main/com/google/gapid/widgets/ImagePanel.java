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

import static com.google.gapid.util.Loadable.MessageType.Info;
import static com.google.gapid.widgets.Widgets.centered;
import static com.google.gapid.widgets.Widgets.createBaloonToolItem;
import static com.google.gapid.widgets.Widgets.createCheckbox;
import static com.google.gapid.widgets.Widgets.createComposite;
import static com.google.gapid.widgets.Widgets.createLabel;
import static com.google.gapid.widgets.Widgets.createSeparator;
import static com.google.gapid.widgets.Widgets.createToggleToolItem;
import static com.google.gapid.widgets.Widgets.createToolItem;
import static com.google.gapid.widgets.Widgets.withSpans;
import static org.eclipse.swt.widgets.SwtUtil.disableAutoHideScrollbars;

import com.google.common.collect.Lists;
import com.google.common.collect.Maps;
import com.google.common.collect.Sets;
import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.glviewer.gl.Renderer;
import com.google.gapid.glviewer.gl.Scene;
import com.google.gapid.glviewer.gl.Shader;
import com.google.gapid.glviewer.gl.Texture;
import com.google.gapid.glviewer.gl.VertexBuffer;
import com.google.gapid.glviewer.vec.MatD;
import com.google.gapid.glviewer.vec.VecD;
import com.google.gapid.image.Histogram;
import com.google.gapid.image.Histogram.Range;
import com.google.gapid.image.Image;
import com.google.gapid.image.Image.PixelInfo;
import com.google.gapid.image.Image.PixelValue;
import com.google.gapid.image.MultiLayerAndLevelImage;
import com.google.gapid.proto.stream.Stream.Channel;
import com.google.gapid.rpc.Rpc;
import com.google.gapid.rpc.RpcException;
import com.google.gapid.rpc.SingleInFlight;
import com.google.gapid.rpc.UiErrorCallback;
import com.google.gapid.server.Client.DataUnavailableException;
import com.google.gapid.util.Loadable;
import com.google.gapid.util.Messages;
import com.google.gapid.util.MouseAdapter;

import org.eclipse.swt.SWT;
import org.eclipse.swt.events.MouseEvent;
import org.eclipse.swt.graphics.Color;
import org.eclipse.swt.graphics.GC;
import org.eclipse.swt.graphics.ImageData;
import org.eclipse.swt.graphics.ImageLoader;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.graphics.RGB;
import org.eclipse.swt.graphics.Rectangle;
import org.eclipse.swt.internal.DPIUtil;
import org.eclipse.swt.layout.FillLayout;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.layout.RowData;
import org.eclipse.swt.layout.RowLayout;
import org.eclipse.swt.widgets.Button;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Display;
import org.eclipse.swt.widgets.FileDialog;
import org.eclipse.swt.widgets.Label;
import org.eclipse.swt.widgets.Scale;
import org.eclipse.swt.widgets.ScrollBar;
import org.eclipse.swt.widgets.Shell;
import org.eclipse.swt.widgets.ToolBar;
import org.eclipse.swt.widgets.ToolItem;
import org.lwjgl.opengl.GL11;
import org.lwjgl.opengl.GL12;
import org.lwjgl.opengl.GL13;
import org.lwjgl.opengl.GL30;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.concurrent.ExecutionException;
import java.util.function.Consumer;
import java.util.function.IntConsumer;
import java.util.logging.Logger;

/**
 * Image viewer panel with various image inspection tools.
 */
public class ImagePanel extends Composite {
  protected static final Logger LOG = Logger.getLogger(ImagePanel.class.getName());
  protected static final int ZOOM_AMOUNT = 5;
  protected static final int NUM_HISTOGRAM_BINS = 256;
  protected static final int HISTOGRAM_X = 15;
  protected static final int HISTOGRAM_Y = 15;
  protected static final int HISTOGRAM_H = 50;
  protected static final int HISTOGRAM_ARROW_W = 10;
  protected static final int HISTOGRAM_ARROW_H = 7;

  private static final int CHANNEL_RED = 0, CHANNEL_GREEN = 1, CHANNEL_BLUE = 2, CHANNEL_ALPHA = 3;
  private static final float ALPHA_WARNING_THRESHOLD = 2 / 255f;
  private static final Image[] NO_LAYERS = new Image[] { Image.EMPTY };

  private final Widgets widgets;
  private final SingleInFlight imageRequestController = new SingleInFlight();
  protected final LoadablePanel<ImageComponent> loading;
  private final StatusBar status;
  protected final ImageComponent imageComponent;
  private final BackgroundSelection backgroundSelection;
  private ToolItem zoomFitItem, backgroundItem, saveItem, colorChanelsItem;
  private MultiLayerAndLevelImage image = MultiLayerAndLevelImage.EMPTY;
  private Image[] layers = NO_LAYERS;

  public ImagePanel(Composite parent, Widgets widgets, boolean naturallyFlipped) {
    super(parent, SWT.NONE);
    this.widgets = widgets;
    this.backgroundSelection = new BackgroundSelection(getDisplay());

    setLayout(Widgets.withMargin(new GridLayout(1, false), 5, 2));

    loading = LoadablePanel.create(this, widgets, panel ->
        new ImageComponent(panel, widgets.theme, this::showAlphaWarning, naturallyFlipped));
    status = new StatusBar(this, widgets.theme, this::loadLevel, this::setAlphaEnabled);
    imageComponent = loading.getContents();

    loading.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));
    status.setLayoutData(new GridData(SWT.FILL, SWT.BOTTOM, true, false));

    imageComponent.addMouseListeners(new MouseAdapter() {
      private static final int NO_HANDLE = -1;
      private static final int LOW_HANDLE = 0;
      private static final int HIGH_HANDLE = 1;

      private static final int MODE_VIEW = 0;
      private static final int MODE_PANNING = 1;
      private static final int MODE_MAYBE_DRAGGING_RANGE = 2;
      private static final int MODE_DRAGGING_LOW_HANDLE = 3;
      private static final int MODE_DRAGGING_HIGH_HANDLE = 4;

      private Point last = new Point(0, 0); // In the parent space.
      private Point mouseDownPoint = null;
      private int mode = MODE_VIEW;

      @Override
      public void mouseDown(MouseEvent e) {
        last = getPoint(e);
        mouseDownPoint = last;

        if (inHistogram(e)) {
          int handle = getHistogramRangeHandle(e);
          if (handle != NO_HANDLE) {
            int mode = (handle == LOW_HANDLE) ? MODE_DRAGGING_LOW_HANDLE : MODE_DRAGGING_HIGH_HANDLE;
            setMode(mode, e);
          } else {
            setMode(MODE_MAYBE_DRAGGING_RANGE, e);
          }
        } else if (isPanningButton(e)) {
          setMode(MODE_PANNING, e);
        } else {
          setZoomToFit(true);
        }
      }

      @Override
      public void mouseUp(MouseEvent e) {
        setMode(MODE_VIEW, e);
      }

      @Override
      public void mouseMove(MouseEvent e) {
        Point current = getPoint(e);
        int dx = last.x - current.x, dy = last.y - current.y;
        last = current;

        switch (mode) {
          case MODE_PANNING:
            if (!isPanningButton(e)) {
              setMode(MODE_VIEW, e);
              return;
            }
            imageComponent.scrollBy(dx, dy);
            break;
          case MODE_DRAGGING_LOW_HANDLE:
            imageComponent.data.setHistogramLowX(e.x);
            imageComponent.refresh();
            break;
          case MODE_DRAGGING_HIGH_HANDLE:
            imageComponent.data.setHistogramHighX(e.x);
            imageComponent.refresh();
            break;
          case MODE_MAYBE_DRAGGING_RANGE:
            if (Math.abs(mouseDownPoint.x - e.x) > 5) {
              if (mouseDownPoint.x < e.x) {
                imageComponent.data.setHistogramLowX(mouseDownPoint.x);
                imageComponent.data.setHistogramHighX(e.x);
                setMode(MODE_DRAGGING_HIGH_HANDLE, e);
              } else {
                imageComponent.data.setHistogramLowX(e.x);
                imageComponent.data.setHistogramHighX(mouseDownPoint.x);
                setMode(MODE_DRAGGING_LOW_HANDLE, e);
              }
              setCursor(getDisplay().getSystemCursor(SWT.CURSOR_SIZEWE));
              imageComponent.refresh();
            }
            break;
          default:
            if (inHistogram(e)) {
              int handle = getHistogramRangeHandle(e);
              setCursor(handle != NO_HANDLE ? getDisplay().getSystemCursor(SWT.CURSOR_SIZEWE) : null);
            } else {
              setPreviewPixel(imageComponent.getPixel(getPoint(e)));
              setCursor(null);
            }
            break;
        }
      }

      @Override
      public void mouseScrolled(MouseEvent e) {
        zoom(Math.max(-ZOOM_AMOUNT, Math.min(ZOOM_AMOUNT, -e.count)), getPoint(e));
      }

      @Override
      public void mouseExit(MouseEvent e) {
        setPreviewPixel(Pixel.OUT_OF_BOUNDS);
      }

      private Point getPoint(MouseEvent e) {
        return new Point(e.x, e.y);
      }

      private void setMode(int newMode, MouseEvent e) {
        if (mode == newMode) {
          return;
        }
        mode = newMode;
        switch (mode) {
          case MODE_PANNING:
            setCursor(getDisplay().getSystemCursor(SWT.CURSOR_SIZEALL));
            imageComponent.setPreviewPixel(Pixel.OUT_OF_BOUNDS);
            break;

          case MODE_DRAGGING_LOW_HANDLE:
          case MODE_DRAGGING_HIGH_HANDLE:
          case MODE_MAYBE_DRAGGING_RANGE:
            getDisplay().getSystemCursor(SWT.CURSOR_SIZEWE);
            imageComponent.setPreviewPixel(Pixel.OUT_OF_BOUNDS);
            break;

          default:
            setCursor(null);
            setPreviewPixel(imageComponent.getPixel(getPoint(e)));
        }
      }

      private boolean inHistogram(MouseEvent e) {
        if (!imageComponent.getHistogramVisible()) {
          return false;
        }
        int x = e.x - HISTOGRAM_X, y = e.y - HISTOGRAM_Y;
        int w = imageComponent.getClientArea().width;
        return x >= 0 && x < w - HISTOGRAM_X &&
            y >= 0 && y < HISTOGRAM_H + HISTOGRAM_ARROW_H;
      }

      private boolean isPanningButton(MouseEvent e) {
        // Pan for either the primary mouse button or the mouse wheel.
        if (e.button != 0) {
          return e.button == 1 || e.button == 2;
        } else {
          return (e.stateMask & (SWT.BUTTON1 | SWT.BUTTON2)) != 0;
        }
      }

      /**
       * Returns the histogram range handle for the mouse position.
       */
      private int getHistogramRangeHandle(MouseEvent e) {
        if (!inHistogram(e)) {
          return NO_HANDLE;
        }
        int lowDist = Math.abs(imageComponent.data.getHistogramLowX() - e.x);
        int highDist = Math.abs(imageComponent.data.getHistogramHighX() - e.x);
        if (Math.min(lowDist, highDist) > HISTOGRAM_ARROW_W / 2) {
          return NO_HANDLE;
        }
        return (lowDist < highDist) ? LOW_HANDLE : HIGH_HANDLE;
      }
    });

    addListener(SWT.Dispose, e -> backgroundSelection.dispose());
  }

  protected void setPreviewPixel(Pixel pixel) {
    imageComponent.setPreviewPixel(pixel);
    status.setPixel(pixel);
  }

  private void showAlphaWarning(AlphaWarning message) {
    status.showAlphaWarning(message);
  }

  private void setAlphaEnabled(boolean enabled) {
    imageComponent.autoToggleAlphaChannel(enabled);
    updateColorChannelsIcon();
  }

  public void createToolbar(ToolBar bar, Theme theme) {
    zoomFitItem = createToggleToolItem(bar, theme.zoomFit(),
        e -> setZoomToFit(((ToolItem)e.widget).getSelection()), "Zoom to fit");
    setZoomToFit(true);
    createToolItem(bar, theme.zoomActual(), e -> zoomToActual(), "Original size");
    createToolItem(bar, theme.zoomIn(), e -> zoom(-ZOOM_AMOUNT), "Zoom in");
    createToolItem(bar, theme.zoomOut(), e -> zoom(ZOOM_AMOUNT), "Zoom out");
    createSeparator(bar);
    createToggleToolItem(bar, theme.toggleHistogram(),
        e -> setShowHistogram(((ToolItem)e.widget).getSelection()), "Toggle histogram");
    colorChanelsItem = createBaloonToolItem(bar, theme.colorChannels()[15], shell -> {
      Composite c = createComposite(shell, new RowLayout(SWT.HORIZONTAL), SWT.BORDER);
      final ImageComponent i = imageComponent;
      createCheckbox(c, "Red", i.isChannelEnabled(CHANNEL_RED), e -> {
        i.setChannelEnabled(CHANNEL_RED, ((Button)e.widget).getSelection());
        updateColorChannelsIcon();
      });
      createCheckbox(c, "Green", i.isChannelEnabled(CHANNEL_GREEN), e -> {
        i.setChannelEnabled(CHANNEL_GREEN, ((Button)e.widget).getSelection());
        updateColorChannelsIcon();
      });
      createCheckbox(c, "Blue", i.isChannelEnabled(CHANNEL_BLUE), e -> {
        i.setChannelEnabled(CHANNEL_BLUE, ((Button)e.widget).getSelection());
        updateColorChannelsIcon();
      });
      createCheckbox(c, "Alpha", i.isChannelEnabled(CHANNEL_ALPHA), e -> {
        i.setChannelEnabled(CHANNEL_ALPHA, ((Button)e.widget).getSelection());
        updateColorChannelsIcon();
      });
    }, "Color channel selection");
    backgroundItem = createBaloonToolItem(bar, theme.transparency(),
        shell -> backgroundSelection.createBaloonContents(shell, theme,
            mode -> updateBackgroundMode(mode, theme)), "Choose image background");
    createToggleToolItem(bar, theme.flipVertically(),
        e -> imageComponent.setFlipped(((ToolItem)e.widget).getSelection()), "Flip vertically");
    createSeparator(bar);
    saveItem = createToolItem(bar, theme.save(), e -> save(), "Save image to file");
    saveItem.setEnabled(false);
  }

  private void updateColorChannelsIcon() {
    colorChanelsItem.setImage(widgets.theme.colorChannels()[
        (imageComponent.isChannelEnabled(CHANNEL_RED) ? 1 : 0) |
        (imageComponent.isChannelEnabled(CHANNEL_GREEN) ? 2 : 0) |
        (imageComponent.isChannelEnabled(CHANNEL_BLUE) ? 4 : 0) |
        (imageComponent.isChannelEnabled(CHANNEL_ALPHA) ? 8 : 0)]);
  }

  protected void setZoomToFit(boolean enabled) {
    zoomFitItem.setSelection(enabled);
    imageComponent.setZoomToFit(enabled);
  }

  protected void zoomToActual() {
    setZoomToFit(false);
    imageComponent.zoomToActual();
  }

  protected void zoom(int amount) {
    zoom(amount, null);
  }

  protected void setShowHistogram(boolean show) {
    imageComponent.setHistogramVisible(show);
  }

  protected void zoom(int amount, Point cursor) {
    setZoomToFit(false);
    imageComponent.zoom(amount, cursor);
  }

  protected void updateBackgroundMode(BackgroundMode mode, Theme theme) {
    imageComponent.setBackgroundMode(mode, backgroundSelection.color);
    switch (mode) {
      case Checkerboard:
        backgroundItem.setImage(theme.transparency());
        break;
      case SolidColor:
        backgroundItem.setImage(backgroundSelection.image);
        break;
      default:
        throw new AssertionError();
    }
  }

  protected void save() {
    FileDialog dialog = new FileDialog(getShell(), SWT.SAVE);
    dialog.setText("Save image to...");
    dialog.setFilterNames(new String[] { "PNG Images" });
    dialog.setFilterExtensions(new String[] { "*.png" });
    dialog.setOverwrite(true);
    String path = dialog.open();
    if (path != null) {
      ImageLoader saver = new ImageLoader();
      saver.data = new ImageData[] { layers[0].getImageData() }; // TODO: Save each layer
      saver.save(path, SWT.IMAGE_PNG);
    }
  }

  public Loadable getLoading() {
    return loading;
  }

  public void setImage(MultiLayerAndLevelImage image) {
    if (image == null || image == MultiLayerAndLevelImage.EMPTY) {
      clearImage();
    } else {
      this.image = image;
      this.layers = NO_LAYERS;
      loadLevel(0);
      status.setLevelCount(image.getLevelCount());
    }
  }

  public void clearImage() {
    this.image = MultiLayerAndLevelImage.EMPTY;
    this.layers = NO_LAYERS;
    if (saveItem != null) {
      saveItem.setEnabled(false);
    }
    status.setLevelCount(0);
    imageComponent.setImages(layers);
  }

  private void loadLevel(int level) {
    if (image.getLevelCount() == 0) {
      clearImage();
      loading.showMessage(Info, Messages.NO_IMAGE_DATA);
      if (saveItem != null) {
        saveItem.setEnabled(false);
      }
      return;
    }

    level = Math.min(image.getLevelCount() - 1, level);
    loading.startLoading();

    List<ListenableFuture<Image>> layerFutures = Lists.newArrayList();
    for (int layer = 0; layer < image.getLayerCount(); layer++) {
      layerFutures.add(image.getImage(layer, level));
    }
    ListenableFuture<LevelData> future = Futures.transform(Futures.allAsList(layerFutures), imageList -> {
      Image[] images = imageList.toArray(new Image[imageList.size()]);

      boolean isHDR = false;
      for (Image image : images) {
        if (image.isHDR()) {
          isHDR = true;
        }
      }

      Histogram histogram = new Histogram(images, NUM_HISTOGRAM_BINS, isHDR);
      return new LevelData(images, histogram);
    });

    imageRequestController.start().listen(future,
        new UiErrorCallback<LevelData, LevelData, Loadable.Message>(this, LOG) {
      @Override
      protected ResultOrError<LevelData, Loadable.Message> onRpcThread(Rpc.Result<LevelData> result)
          throws RpcException, ExecutionException {
        try {
          return success(result.get());
        } catch (DataUnavailableException e) {
          return error(Loadable.Message.info(e));
        } catch (RpcException e) {
          return error(Loadable.Message.error(e));
        }
      }

      @Override
      protected void onUiThreadSuccess(LevelData data) {
        updateLayers(data);
      }

      @Override
      protected void onUiThreadError(Loadable.Message message) {
        clearImage();
        loading.showMessage(message);
      }
    });
  }

  protected void updateLayers(LevelData data) {
    boolean valid = data != null && data.layers.length > 0;
    if (valid) {
      layers = data.layers;
      status.setLevelSize(layers[0].getWidth(), layers[0].getHeight());
    } else {
      layers = NO_LAYERS;
    }
    loading.stopLoading();
    if (saveItem != null) {
      saveItem.setEnabled(valid);
    }
    List<Image> images = new ArrayList<>(layers.length);
    for (Image layer : layers) {
      for (int i = 0, c = layer.getDepth(); i < c; i++) {
        images.add(layer.getSlice(i));
      }
    }
    imageComponent.setImages(images.toArray(new Image[images.size()]));
    imageComponent.setHistogram(data.histogram);
  }

  private static final class LevelData {
    public final Image[] layers;
    public final Histogram histogram;
    public LevelData(Image[] layers, Histogram histogram) {
      this.layers = layers;
      this.histogram = histogram;
    }
  }

  private static final class SceneData {
    public Image[] images = {};
    public MatD[] transforms = {};
    public final boolean channels[] = { true, true, true, true };
    public Histogram histogram;
    public Histogram.Range displayRange = Range.IDENTITY;
    public boolean histogramVisible;
    public int histogramX, histogramY;
    public int histogramW, histogramH;
    public int histogramArrowW, histogramArrowH;
    public Color histogramBackgroundDark;
    public Color histogramBackgroundLight;
    public Color histogramCurtainColor;
    public Color histogramArrowColor;
    public Pixel previewPixel = Pixel.OUT_OF_BOUNDS;
    public boolean flipped;
    public int borderWidth;
    public Color borderColor;
    public Color panelColor;
    public Color backgroundColor;
    public BackgroundMode backgroundMode;
    public Color checkerLight;
    public Color checkerDark;
    public int checkerSize;
    public Color cursorLight;
    public Color cursorDark;
    public Map<Channel, Color> channelColors;

    public SceneData() {
    }

    public SceneData copy() {
      SceneData out = new SceneData();
      out.images = images;
      out.transforms = transforms.clone();
      System.arraycopy(channels, 0, out.channels, 0, channels.length);
      out.histogram = histogram;
      out.displayRange = displayRange;
      out.histogramVisible = histogramVisible;
      out.histogramX = histogramX;
      out.histogramY = histogramY;
      out.histogramW = histogramW;
      out.histogramH = histogramH;
      out.histogramArrowW = histogramArrowW;
      out.histogramArrowH = histogramArrowH;
      out.histogramBackgroundLight = histogramBackgroundLight;
      out.histogramBackgroundDark = histogramBackgroundDark;
      out.histogramCurtainColor = histogramCurtainColor;
      out.histogramArrowColor = histogramArrowColor;
      out.previewPixel = previewPixel;
      out.flipped = flipped;
      out.borderWidth = borderWidth;
      out.borderColor = borderColor;
      out.panelColor = panelColor;
      out.backgroundColor = backgroundColor;
      out.backgroundMode = backgroundMode;
      out.checkerLight = checkerLight;
      out.checkerDark = checkerDark;
      out.checkerSize = checkerSize;
      out.cursorLight = cursorLight;
      out.cursorDark = cursorDark;
      out.channelColors = channelColors;
      return out;
    }

    public int getHistogramLowX() {
      return histogramX + (int)(histogramW * histogram.getNormalizedXFromValue(displayRange.min));
    }

    public void setHistogramLowX(int x) {
      displayRange = new Histogram.Range(
          histogram.getValueFromNormalizedX((x - histogramX) / (double)histogramW),
          displayRange.max);
    }

    public int getHistogramHighX() {
      return histogramX + (int)(histogramW * histogram.getNormalizedXFromValue(displayRange.max));
    }

    public void setHistogramHighX(int x) {
      displayRange = new Histogram.Range(
          displayRange.min,
          histogram.getValueFromNormalizedX((x - histogramX) / (double)histogramW));
    }
  }

  /**
   * Component that renders the image using OpenGL.
   */
  private static class ImageComponent extends Composite {
    private static final VecD BORDER_SIZE = new VecD(2, 2, 0);
    private static final double MAX_ZOOM_FACTOR = 8;
    private static final VecD MIN_ZOOM_SIZE = new VecD(100, 100, 0);

    private final Consumer<AlphaWarning> showAlphaWarning;
    private final boolean naturallyFlipped;

    private final ScrollBar scrollbars[];
    private final ScenePanel<SceneData> canvas;
    private final SceneData data;
    private Image[] images = {};

    private double scaleGridToViewMin = 0;
    private double scaleGridToViewMax = Double.POSITIVE_INFINITY;
    private double scaleGridToViewFit = 1;

    private VecD viewSize = VecD.ZERO;
    private VecD viewOffset = VecD.ZERO;
    private VecD viewOffsetMin = VecD.MIN;
    private VecD viewOffsetMax = VecD.MAX;
    private VecD gridSize = VecD.ZERO;
    private VecD tileSize = VecD.ZERO;
    private VecD tileOffsets[] = {};

    private double scaleGridToView = 1.0;
    private boolean zoomToFit;

    private boolean alphaWasAutoDisabled = false;

    public ImageComponent(Composite parent, Theme theme, Consumer<AlphaWarning> showAlphaWarning,
        boolean naturallyFlipped) {
      super(parent, SWT.BORDER | SWT.V_SCROLL | SWT.H_SCROLL | SWT.NO_BACKGROUND);
      setLayout(new FillLayout(SWT.VERTICAL));
      disableAutoHideScrollbars(this);

      this.showAlphaWarning = showAlphaWarning;
      this.naturallyFlipped = naturallyFlipped;

      scrollbars = new ScrollBar[] { getHorizontalBar(), getVerticalBar() };

      data = new SceneData();
      data.flipped = naturallyFlipped;
      data.borderWidth = (int)BORDER_SIZE.x;
      data.borderColor = getDisplay().getSystemColor(SWT.COLOR_WIDGET_NORMAL_SHADOW);
      data.panelColor = getDisplay().getSystemColor(SWT.COLOR_WIDGET_BACKGROUND);
      data.backgroundMode = BackgroundMode.Checkerboard;
      data.checkerDark = theme.imageCheckerDark();
      data.checkerLight = theme.imageCheckerLight();
      data.checkerSize = 30;
      data.cursorLight = theme.imageCursorLight();
      data.cursorDark = theme.imageCursorDark();
      data.histogramBackgroundDark = theme.histogramBackgroundDark();
      data.histogramBackgroundLight = theme.histogramBackgroundLight();
      data.histogramCurtainColor = theme.histogramCurtain();
      data.histogramArrowColor = theme.histogramArrow();
      data.histogramX = HISTOGRAM_X;
      data.histogramY = HISTOGRAM_Y;
      data.histogramArrowW = HISTOGRAM_ARROW_W;
      data.histogramArrowH = HISTOGRAM_ARROW_H;

      Map<Channel, Color> channelColors = Maps.newHashMap();
      channelColors.put(Channel.Red, getDisplay().getSystemColor(SWT.COLOR_DARK_RED));
      channelColors.put(Channel.Green, getDisplay().getSystemColor(SWT.COLOR_DARK_GREEN));
      channelColors.put(Channel.Blue, getDisplay().getSystemColor(SWT.COLOR_DARK_BLUE));
      channelColors.put(Channel.Alpha, getDisplay().getSystemColor(SWT.COLOR_DARK_YELLOW));
      channelColors.put(Channel.Luminance, getDisplay().getSystemColor(SWT.COLOR_DARK_GRAY));
      channelColors.put(Channel.Depth, getDisplay().getSystemColor(SWT.COLOR_DARK_GRAY));
      channelColors.put(Channel.Gray, getDisplay().getSystemColor(SWT.COLOR_DARK_GRAY));
      channelColors.put(Channel.Stencil, getDisplay().getSystemColor(SWT.COLOR_DARK_YELLOW));

      data.channelColors = Maps.immutableEnumMap(channelColors);
      canvas = new ScenePanel<SceneData>(this, new ImageScene());
      canvas.setSceneData(data.copy());

      getHorizontalBar().addListener(SWT.Selection, e -> onScroll());
      getVerticalBar().addListener(SWT.Selection, e -> onScroll());
      canvas.addListener(SWT.Resize, e -> onResize());

      // Prevent the mouse wheel from scrolling the view.
      addListener(SWT.MouseWheel, e -> e.doit = false);
    }

    public void addMouseListeners(MouseAdapter mouseHandler) {
      canvas.addMouseListener(mouseHandler);
      canvas.addMouseMoveListener(mouseHandler);
      canvas.addMouseWheelListener(mouseHandler);
      canvas.addMouseTrackListener(mouseHandler);
    }

    public void setImages(Image[] images) {
      this.images = images;
      this.tileOffsets = new VecD[images.length];

      tileSize = VecD.ZERO;
      for (Image image : images) {
        VecD imageSize = new VecD(image.getWidth(), image.getHeight(), 0);
        tileSize = tileSize.max(imageSize);
      }
      int numColumns = (int)Math.round(Math.sqrt(images.length));
      int numRows = (images.length + numColumns - 1) / numColumns;

      gridSize = tileSize.multiply(numColumns, numRows, 1).add(BORDER_SIZE.multiply(numColumns - 1, numRows - 1, 1));
      VecD center = gridSize.subtract(tileSize).divide(2);

      for (int i = 0; i < images.length; i++) {
        int x = i % numColumns;
        int y = i / numColumns;
        tileOffsets[i] = tileSize.add(BORDER_SIZE)
            .multiply(x, y, 0)
            .subtract(center);
      }

      updateScaleLimits();
      setScale(zoomToFit ? scaleGridToViewFit : scaleGridToView);
      refresh();
    }

    public void setHistogram(Histogram histogram) {
      data.histogram = histogram;
      data.displayRange = histogram.power != 1.0 ?
          calcHistogramRange(data.histogram) : Range.IDENTITY;
      refresh();
    }

    private void refresh() {
      data.images = images;
      data.transforms = calcTransforms();
      canvas.setSceneData(data.copy());

      if (images.length == 0) {
        showAlphaWarning.accept(AlphaWarning.NONE);
      } else if (isChannelEnabled(CHANNEL_ALPHA)) {
        boolean noAlpha = true;
        for (Image image : images) {
          if (image.getInfo().getAlphaMax() > ALPHA_WARNING_THRESHOLD) {
            noAlpha = false;
            break;
          }
        }
        showAlphaWarning.accept(noAlpha ? AlphaWarning.NO_ALPHA : AlphaWarning.NONE);
      } else if (alphaWasAutoDisabled) {
        boolean noAlpha = true;
        for (Image image : images) {
          PixelInfo info = image.getInfo();
          if (info.getAlphaMax() > ALPHA_WARNING_THRESHOLD &&
              info.getAlphaMin() < 1 - ALPHA_WARNING_THRESHOLD) {
            // Consider an image with all alpha values mostly 1.0 as an image without alpha.
            noAlpha = false;
            break;
          }
        }
        showAlphaWarning.accept(noAlpha ? AlphaWarning.NONE : AlphaWarning.ALPHA_DISABLED);
      } else {
        showAlphaWarning.accept(AlphaWarning.NONE);
      }
    }

    private Histogram.Range calcHistogramRange(Histogram histogram) {
      double rangeMin = histogram.getPercentile(1, false, Channel.Alpha);
      double rangeMax = histogram.getPercentile(99, true, Channel.Alpha);

      // Snap the range to the limits if they're close enough.
      final double SNAP_THRESHOLD = 0.1;
      if (histogram.limits.frac(rangeMin) < SNAP_THRESHOLD) {
        rangeMin = histogram.limits.min;
      }
      if (histogram.limits.frac(rangeMax) > 1.0 - SNAP_THRESHOLD) {
        rangeMax = histogram.limits.max;
      }

      return new Histogram.Range(rangeMin, rangeMax);
    }

    public void setPreviewPixel(Pixel previewPixel) {
      data.previewPixel = previewPixel;
      refresh();
    }

    protected void scrollBy(int dx, int dy) {
      viewOffset = viewOffset
          .add(dx / scaleGridToView, dy / scaleGridToView, 0)
          .clamp(viewOffsetMin, viewOffsetMax);
      updateScrollbars();
      refresh();
    }

    protected void setBackgroundMode(BackgroundMode backgroundMode, Color backgroundColor) {
      data.backgroundMode = backgroundMode;
      data.backgroundColor = backgroundColor;
      refresh();
    }

    protected void setFlipped(boolean flipped) {
      data.flipped = flipped ^ naturallyFlipped;
      refresh();
    }

    protected boolean isChannelEnabled(int channel) {
      return data.channels[channel];
    }

    protected void setChannelEnabled(int channel, boolean enabled) {
      data.channels[channel] = enabled;
      if (channel == CHANNEL_ALPHA) {
        alphaWasAutoDisabled = false;
      }
      refresh();
    }

    protected void autoToggleAlphaChannel(boolean enabled) {
      alphaWasAutoDisabled = true;
      data.channels[CHANNEL_ALPHA] = enabled;
      refresh();
    }

    public Pixel getPixel(Point point) {
      VecD ndc = pointToNDC(point);
      for (int i = 0; i < images.length; i++) {
        Image image = images[i];
        VecD imageNormalized = calcInvTransform(i).multiply(ndc);
        VecD imageTexel = imageNormalized.multiply(0.5).add(0.5, 0.5, 0.0).multiply(tileSize);
        float u = (float)imageNormalized.x * 0.5f + 0.5f;
        float v = (float)imageNormalized.y * (data.flipped ? 0.5f : 0.5f) + 0.5f;
        if (u < 0 || v < 0 || u > 1 || v > 1) {
          continue;
        }
        int x = (int)imageTexel.x;
        int y = (int)imageTexel.y;
        int sampleY = data.flipped ? (image.getHeight() - y - 1) : y;
        return new Pixel(i, x, y, u, v, image.getPixel(x, sampleY, 0));
      }
      return Pixel.OUT_OF_BOUNDS;
    }

    public void setZoomToFit(boolean zoomToFit) {
      this.zoomToFit = zoomToFit;
      if (zoomToFit) {
        setScale(scaleGridToViewFit);
        updateScrollbars();
        refresh();
      }
    }

    public void zoomToActual() {
      setScale(DPIUtil.autoScaleDown(1.0f));
      updateScrollbars();
      refresh();
    }

    public void zoom(int amount, Point cursor) {
      double scale = scaleGridToView * (1 - 0.05f * amount);

      if (cursor != null) {
        VecD ndc = pointToNDC(cursor);
        VecD imageNormPreScale = calcInvTransform(0).multiply(ndc);
        setScale(scale);
        VecD imageNormPostScale = calcInvTransform(0).multiply(ndc);
        VecD imageNormDelta = imageNormPostScale.subtract(imageNormPreScale);
        viewOffset = viewOffset
            .subtract(imageNormDelta.multiply(tileSize).multiply(0.5))
            .clamp(viewOffsetMin, viewOffsetMax);
      } else {
        setScale(scale);
      }
      updateScrollbars();
      refresh();
    }

    public void setHistogramVisible(boolean show) {
      data.histogramVisible = show;
      refresh();
    }

    public boolean getHistogramVisible() {
      return data.histogramVisible;
    }

    /**
     * Converts a SWT {@link Point} into normalized-device-coordinates.
     * The shaders flip Y to simplify the calculations, keeping positive x
     * and positive y pointing right and down respectively.
     *
     * NDC coordinates:
     * <code>
     *   [-1,-1] ------ [+1,-1]
     *      |              |
     *      |              |
     *   [-1,+1] ------ [+1,+1]
     * </code>
     */
    private VecD pointToNDC(Point point) {
      return new VecD(point.x, point.y, 0)
          .divide(viewSize.x, viewSize.y, 1)
          .subtract(0.5, 0.5, 0)
          .multiply(2.0);
    }

    private MatD[] calcTransforms() {
      MatD[] out = new MatD[images.length];
      for (int i = 0; i < images.length; i++) {
        out[i] = calcTransform(i);
      }
      return out;
    }

    /**
     * @return a transform that can be used to convert the tile quad [-1, -1] to [1, 1] into
     * NDC coordinates based on the current scale and offset.
     */
    private MatD calcTransform(int index) {
      return MatD.makeScale(new VecD(2, 2, 0).safeDivide(viewSize))
              .scale(scaleGridToView)
              .translate(tileOffsets[index].subtract(viewOffset))
              .scale(tileSize.multiply(0.5));
    }

    /**
     * @return the inverse of the matrix returned from {@link #calcTransform(int)}.
     */
    private MatD calcInvTransform(int index) {
      return MatD.makeScale(new VecD(2, 2, 0).safeDivide(tileSize))
          .translate(viewOffset.subtract(tileOffsets[index]))
          .scale(1.0 / scaleGridToView)
          .scale(viewSize.multiply(0.5));
    }

    private static double clamp(double x, double min, double max) {
      return Math.max(Math.min(x, max), min);
    }

    private void setScale(double scale) {
      scaleGridToView = clamp(scale, scaleGridToViewMin, scaleGridToViewMax);

      VecD viewSizeSubBorder = viewSize.subtract(BORDER_SIZE.multiply(2));

      viewOffsetMax = gridSize
          .subtract(viewSizeSubBorder.safeDivide(scaleGridToView))
          .multiply(0.5)
          .max(VecD.ZERO);
      viewOffsetMin = viewOffsetMax.negate();
      viewOffset = viewOffset.clamp(viewOffsetMin, viewOffsetMax);
    }

    private void onResize() {
      Rectangle area = canvas.getClientArea();
      data.histogramW = area.width - 2 * HISTOGRAM_X;
      data.histogramH = HISTOGRAM_H;

      viewSize = new VecD(area.width, area.height, 0);
      updateScaleLimits();
      if (zoomToFit) {
        setScale(scaleGridToViewFit);
      }
      updateScrollbars();
      refresh();
    }

    private void updateScaleLimits() {
      VecD viewSpace = viewSize.subtract(data.borderWidth).max(VecD.ZERO);
      scaleGridToViewFit = viewSpace.safeDivide(gridSize).minXY();
      scaleGridToViewMax = Math.max(MAX_ZOOM_FACTOR, scaleGridToViewFit);
      // The smallest zoom factor to see the whole image or that causes the larger dimension to be
      // no less than MIN_ZOOM_WIDTH pixels.
      scaleGridToViewMin = Math.min(MIN_ZOOM_SIZE.safeDivide(gridSize).minXY(), scaleGridToViewFit);
    }

    private void updateScrollbars() {
      for (int i = 0; i < scrollbars.length; i++) {
        ScrollBar scrollbar = scrollbars[i];
        int val = (int)(viewOffset.get(i) * scaleGridToView); // offset in view pixels
        int min = (int)(viewOffsetMin.get(i) * scaleGridToView); // min movement in view pixels
        int max = (int)(viewOffsetMax.get(i) * scaleGridToView); // max movement in view pixels
        int rng = max - min;
        if (rng == 0) {
          scrollbar.setEnabled(false);
          scrollbar.setValues(0, 0, 1, 1, 1, 1);
        } else {
          int view = (int)this.viewSize.get(i);
          scrollbar.setEnabled(true);
          scrollbar.setValues(
              val - min,        // selection
              0,                // min
              view + rng,       // max
              view,             // thumb
              (rng + 99) / 100, // increment
              (rng + 9) / 10    // page increment
          );
        }
      }
    }

    private void onScroll() {
      for (int i = 0; i < scrollbars.length; i++) {
        ScrollBar scrollbar = scrollbars[i];
        if (scrollbar.getEnabled()) {
          int min = (int)(viewOffsetMin.get(i) * scaleGridToView); // min movement in view pixels
          int val = min + scrollbar.getSelection();
          viewOffset = viewOffset.set(i, val / scaleGridToView);
        }
      }
      refresh();
    }
  }

  private static class ImageScene implements Scene<SceneData> {
    private static final int PREVIEW_WIDTH = 19; // Should be odd, so center pixel looks nice.
    private static final int PREVIEW_HEIGHT = 11; // Should be odd, so center pixel looks nice.
    private static final int PREVIEW_SIZE = 7;

    private final Map<Image, Texture> imageToTexture = Maps.newHashMap();

    private Shader shader;
    private Texture[] textures;
    private SceneData data;

    private final float[] uChannels = new float[] { 1, 1, 1, 1 };

    public ImageScene() {
    }

    @Override
    public void init(Renderer renderer) {
      GL30.glBindVertexArray(GL30.glGenVertexArrays());
      shader = renderer.loadShader("image");

      GL11.glDisable(GL11.GL_DEPTH_TEST);
      GL11.glDisable(GL11.GL_CULL_FACE);
      GL11.glEnable(GL11.GL_BLEND);
      GL11.glBlendFunc(GL11.GL_SRC_ALPHA, GL11.GL_ONE_MINUS_SRC_ALPHA);
    }

    @Override
    public void update(Renderer renderer, SceneData newData) {
      // Release textures that are no longer in data.
      Set<Image> newSet = Sets.newHashSet(newData.images);
      for (Map.Entry<Image, Texture> entry : imageToTexture.entrySet()) {
        if (!newSet.contains(entry.getKey())) {
          entry.getValue().delete();
        }
      }

      this.textures = new Texture[newData.images.length];
      for (int i = 0; i < newData.images.length; i++) {
        Image image = newData.images[i];
        Texture texture = imageToTexture.get(image);
        if (texture == null) {
          texture = renderer
              .newTexture(GL11.GL_TEXTURE_2D)
              .setMinMagFilter(GL11.GL_LINEAR, GL11.GL_NEAREST)
              .setBorderColor(newData.borderColor);
          image.uploadToTexture(texture);
          imageToTexture.put(image, texture);
        }

        this.textures[i] = texture;
      }

      float rangeMin = (float)newData.displayRange.min;
      float rangeMax = (float)newData.displayRange.max;
      shader.setUniform("uRange", new float[] { rangeMin, rangeMax - rangeMin });
      for (int i = 0; i < 4; i++) {
        uChannels[i] = newData.channels[i] ? 1.0f : 0.0f;
      }
      data = newData;
    }

    @Override
    public void render(Renderer renderer) {
      if (data == null) {
        return;
      }
      Renderer.clear(data.panelColor);
      drawBackground(renderer);
      drawImages(renderer);
      drawPreview(renderer);
      drawHistogram(renderer);
    }

    @Override
    public void resize(Renderer renderer, int width, int height) {
      // Ignore.
    }

    private void drawBackground(Renderer renderer) {
      switch (data.backgroundMode) {
        case Checkerboard:
          for (MatD transform : data.transforms) {
            renderer.drawChecker(transform, data.checkerLight, data.checkerDark, data.checkerSize);
          }
          break;
        case SolidColor:
          for (MatD transform : data.transforms) {
            renderer.drawSolid(transform, data.backgroundColor);
          }
          break;
        default:
          throw new AssertionError();
      }
      for (MatD transform : data.transforms) {
        renderer.drawBorder(transform, data.borderColor, data.borderWidth);
      }
    }

    private void drawImages(Renderer renderer) {
      shader.setUniform("uPixelSize", VecD.ONE.safeDivide(renderer.getViewSize()));
      shader.setUniform("uTextureSize", new float[] { 1, 1 });
      shader.setUniform("uTextureOffset", new float[] { 0, 0 });
      shader.setUniform("uChannels", uChannels);
      shader.setUniform("uFlipped", data.flipped ? 1 : 0);
      for (int i = 0; i < textures.length; i++) {
        textures[i].setWrapMode(GL12.GL_CLAMP_TO_EDGE, GL12.GL_CLAMP_TO_EDGE);
        shader.setUniform("uTexture", textures[i]);
        renderer.drawQuad(data.transforms[i], shader);
      }
    }

    private void drawPreview(Renderer renderer) {
      if (data.previewPixel == Pixel.OUT_OF_BOUNDS) {
        return;
      }

      int imageIndex = data.previewPixel.imageIndex;
      Image image = data.images[imageIndex];
      int width = PREVIEW_WIDTH * PREVIEW_SIZE;
      int height = PREVIEW_HEIGHT * PREVIEW_SIZE;
      int x = data.borderWidth;
      int y = renderer.getViewHeight() - height - data.borderWidth;

      renderer.drawBorder(x, y, width, height, data.borderColor, data.borderWidth);

      float[] texScale = new float[] {
          (float)PREVIEW_WIDTH / image.getWidth(),
          (float)PREVIEW_HEIGHT / image.getHeight()
      };
      float[] texOffset = new float[] {
          (float)(data.previewPixel.x - PREVIEW_WIDTH / 2) / image.getWidth(),
          (float)(data.previewPixel.y - PREVIEW_HEIGHT / 2) / image.getHeight()
      };

      Texture texture = textures[imageIndex];
      texture.setWrapMode(GL13.GL_CLAMP_TO_BORDER, GL13.GL_CLAMP_TO_BORDER);
      shader.setUniform("uTexture", texture);
      shader.setUniform("uTextureSize", texScale);
      shader.setUniform("uTextureOffset", texOffset);
      shader.setUniform("uChannels", new float[] { 1, 1, 1, 0 });
      shader.setUniform("uFlipped", data.flipped ? 1 : 0);
      renderer.drawQuad(x, y, width, height, shader);

      renderer.drawBorder(
          x + (width-PREVIEW_SIZE)/2, y + (height-PREVIEW_SIZE)/2,
          PREVIEW_SIZE, PREVIEW_SIZE,
          data.previewPixel.value.isDark() ? data.cursorLight : data.cursorDark,
          2);
    }

    private void drawHistogram(Renderer renderer) {
      if (data.histogram == null || !data.histogramVisible) {
        return;
      }

      int x = data.histogramX;
      int y = data.histogramY;
      int w = data.histogramW;
      int h = data.histogramH;

      MatD transform = renderer.rectTransform(x, y, w, h);

      // Draw the background.
      GL11.glBlendFunc(GL11.GL_SRC_ALPHA, GL11.GL_ONE_MINUS_SRC_ALPHA);
      if (data.histogram.power != 1.0) {
        // Non-linear image. Draw log-scale.
        int gridLines = w / 10;
        int lastX = x;
        for (int i = 0; i < gridLines; i++) {
          int lineX = x + (int)(w * Math.pow(i / (gridLines - 1.0), data.histogram.power));
          if (i > 0) {
            renderer.drawSolid(lastX, y, lineX - lastX, h, ((i & 1) == 0) ? data.histogramBackgroundLight : data.histogramBackgroundDark);
          }
          lastX = lineX;
        }
      } else {
        // Linear (typically non-HDR image) is a solid color.
        renderer.drawSolid(x, y, w, h, data.histogramBackgroundDark);
      }

      // Draw the histogram content.
      for (Channel channel : data.histogram.getChannels()) {
        if (channel == Channel.Alpha) {
          continue;
        }

        Color color = data.channelColors.getOrDefault(channel, data.cursorLight);
        int numBins = data.histogram.getNumBins();

        List<Float> triangles = Lists.newArrayList();
        List<Float> lines = Lists.newArrayList();

        lines.add(-1.0f);
        lines.add(-1.0f);

        for (int bin = 0; bin < numBins; bin++) {
          float value = data.histogram.get(channel, bin);
          value = (float)Math.pow(value, 0.3); // Exaggerate small values.
          float y0 = 1.0f;
          float y1 = -2.0f * value + 1.0f;
          float x0 = 2.0f * (bin + 0.5f) / numBins - 1.0f;

          triangles.add(x0);
          triangles.add(y0);

          triangles.add(x0);
          triangles.add(y1);

          lines.add(x0);
          lines.add(y1);
        }

        lines.add(1.0f);
        lines.add(-1.0f);

        VertexBuffer trianglesVB = renderer.newVertexBuffer(triangles, 2);
        GL11.glBlendFunc(GL11.GL_ONE, GL11.GL_ONE);
        renderer.drawSolid(transform, color, trianglesVB, GL11.GL_TRIANGLE_STRIP);
        trianglesVB.delete();

        VertexBuffer linesVB = renderer.newVertexBuffer(lines, 2);
        GL11.glBlendFunc(GL11.GL_ONE, GL11.GL_ZERO);
        renderer.drawSolid(transform, color, linesVB, GL11.GL_LINE_LOOP);
        linesVB.delete();
      }

      // Draw the 'curtain' - the dark bits either side of the window.
      int windowLeftW = (int)(w * data.histogram.getNormalizedXFromValue(data.displayRange.min));
      int windowRightW = (int)(w * (1.0 - data.histogram.getNormalizedXFromValue(data.displayRange.max)));

      GL11.glBlendFunc(GL11.GL_SRC_ALPHA, GL11.GL_ONE_MINUS_SRC_ALPHA);
      renderer.drawSolid(x, y, windowLeftW, h, data.histogramCurtainColor);
      renderer.drawSolid(x + w - windowRightW, y, windowRightW, h, data.histogramCurtainColor);

      // Draw the border.
      renderer.drawBorder(transform, data.borderColor, 1);

      // Draw the handle arrows.
      float arrowW = data.histogramArrowW / (float)w;
      float arrowH = data.histogramArrowH / (float)h;
      float handleLowX = 2.0f * windowLeftW / (float)w - 1.0f;
      float handleHighX = -2.0f * windowRightW / (float)w + 1.0f;

      float[] arrowTriangles = {
          handleLowX, 1.0f,
          handleLowX - arrowW, 1.0f + 2.0f * arrowH,
          handleLowX + arrowW, 1.0f + 2.0f * arrowH,

          handleHighX, 1.0f,
          handleHighX - arrowW, 1.0f + 2.0f * arrowH,
          handleHighX + arrowW, 1.0f + 2.0f * arrowH,
      };
      VertexBuffer arrowTrianglesVB = renderer.newVertexBuffer(arrowTriangles, 2);
      renderer.drawSolid(transform, data.histogramArrowColor, arrowTrianglesVB, GL11.GL_TRIANGLES);
      arrowTrianglesVB.delete();

      float[] arrowLines = {
          handleLowX, -1.0f,
          handleLowX, 1.0f,
          handleLowX, 1.0f,
          handleLowX - arrowW, 1.0f + 2.0f * arrowH,
          handleLowX - arrowW, 1.0f + 2.0f * arrowH,
          handleLowX + arrowW, 1.0f + 2.0f * arrowH,
          handleLowX + arrowW, 1.0f + 2.0f * arrowH,
          handleLowX, 1.0f,

          handleHighX, -1.0f,
          handleHighX, 1.0f,
          handleHighX, 1.0f,
          handleHighX - arrowW, 1.0f + 2.0f * arrowH,
          handleHighX - arrowW, 1.0f + 2.0f * arrowH,
          handleHighX + arrowW, 1.0f + 2.0f * arrowH,
          handleHighX + arrowW, 1.0f + 2.0f * arrowH,
          handleHighX, 1.0f,
      };
      VertexBuffer arrowLinesVB = renderer.newVertexBuffer(arrowLines, 2);
      renderer.drawSolid(transform, data.borderColor, arrowLinesVB, GL11.GL_LINES);
      arrowLinesVB.delete();
    }
  }

  /**
   * Information regarding the currently hovered pixel.
   */
  private static class Pixel {
    public static final Pixel OUT_OF_BOUNDS = new Pixel(-1, -1, -1, -1, -1, PixelValue.NULL_PIXEL) {
      @Override
      public void formatTo(Label label) {
        label.setText(" ");
      }
    };

    public final int imageIndex;
    public final int x, y;
    public final float u, v;
    public final PixelValue value;

    public Pixel(int imageIndex, int x, int y, float u, float v, PixelValue value) {
      this.imageIndex = imageIndex;
      this.x = x;
      this.y = y;
      this.u = u;
      this.v = v;
      this.value = value;
    }

    public void formatTo(Label label) {
      label.setText(String.format("X: %d, Y: %d, U: %05f, V: %05f, %s", x, y, u, v, value));
    }
  }

  /**
   * Background rendring mode.
   */
  private static enum BackgroundMode {
    Checkerboard, SolidColor;
  }

  /**
   * UI components to allow the user to select how to render the background behind the image.
   */
  private static class BackgroundSelection {
    public final org.eclipse.swt.graphics.Image image;
    public Color color;

    public BackgroundSelection(Display display) {
      image = new org.eclipse.swt.graphics.Image(display, 24, 24);
      updateImage(new RGB(0, 0, 0), display);
    }

    public void createBaloonContents(Shell shell, Theme theme, Listener listener) {
      Composite container = createComposite(shell, new RowLayout(SWT.HORIZONTAL), SWT.BORDER);
      ToolBar bar = new ToolBar(container, SWT.HORIZONTAL);
      ToolItem transparency, backgroundColor;
      Widgets.exclusiveSelection(
          transparency = createToggleToolItem(bar, theme.transparency(),
              e -> listener.onBackgroundSelectionChanged(BackgroundMode.Checkerboard),
              "Show checkerboard background"),
          backgroundColor = createToggleToolItem(bar, image,
              e -> listener.onBackgroundSelectionChanged(BackgroundMode.SolidColor),
              "Show solid color background"));
      new QuickColorPiker(container, 128, newColor -> {
        updateImage(newColor, shell.getDisplay());
        transparency.setSelection(false);
        backgroundColor.setSelection(true);
        backgroundColor.setImage(image);
        listener.onBackgroundSelectionChanged(BackgroundMode.SolidColor);
      });
    }

    protected void updateImage(RGB newColor, Display display) {
      if (color != null) {
        color.dispose();
      }

      GC gc = new GC(image);
      gc.setBackground(display.getSystemColor(SWT.COLOR_WIDGET_BACKGROUND));
      gc.fillRectangle(image.getBounds());
      color = new Color(image.getDevice(), newColor);
      gc.setBackground(color);
      gc.fillRoundRectangle(2, 2, 20, 20, 4, 4);
      gc.dispose();
    }

    public void dispose() {
      image.dispose();
      color.dispose();
    }

    public static interface Listener {
      /**
       * Event that indicates the background mode has changed.
       */
      public void onBackgroundSelectionChanged(BackgroundMode mode);
    }
  }

  /**
   * UI status bar component below the image that shows information about the currently hovered
   * pixel and a level selection slider for mipmaps.
   */
  private static class StatusBar extends Composite {
    private final Composite levelComposite;
    private final Scale levelScale;
    private final Label levelValue;
    private final Label levelSize;
    private final Label pixelLabel;
    private final Label warning;
    private int lastSelection = 0;
    private AlphaWarning lastWarning = AlphaWarning.NONE;

    public StatusBar(
        Composite parent, Theme theme, IntConsumer levelListener, Consumer<Boolean> enableAlpha) {
      super(parent, SWT.NONE);
      setLayout(new GridLayout(3, false));

      levelComposite = createComposite(this, centered(new RowLayout(SWT.HORIZONTAL)));
      createLabel(levelComposite, "Level:");
      levelValue = createLabel(levelComposite, "");
      levelScale = createScale(levelComposite);
      levelSize = createLabel(this, "");
      pixelLabel = createLabel(this, "");
      warning = createLabel(this, "");
      warning.setForeground(theme.imageWarning());
      warning.setCursor(getDisplay().getSystemCursor(SWT.CURSOR_HAND));

      levelComposite.setLayoutData(new GridData(SWT.LEFT, SWT.CENTER, false, true));
      levelSize.setLayoutData(new GridData(SWT.LEFT, SWT.CENTER, false, true));
      pixelLabel.setLayoutData(new GridData(SWT.FILL, SWT.CENTER, true, true));
      warning.setLayoutData(withSpans(new GridData(SWT.CENTER, SWT.CENTER, true, false), 3, 1));
      showAlphaWarning(AlphaWarning.NONE);

      levelScale.addListener(SWT.Selection, e -> {
        int selection = levelScale.getSelection();
        if (selection != lastSelection) {
          lastSelection = selection;
          levelValue.setText(String.valueOf(selection));
          levelComposite.requestLayout();
          levelListener.accept(selection);
        }
      });
      warning.addListener(SWT.MouseUp,
          e -> enableAlpha.accept(lastWarning == AlphaWarning.ALPHA_DISABLED));

      setLevelCount(0);
    }

    public void setLevelCount(int count) {
      if (count <= 1) {
        ((GridData)levelComposite.getLayoutData()).exclude = true;
        levelComposite.setVisible(false);
      } else {
        ((GridData)levelComposite.getLayoutData()).exclude = false;
        levelComposite.setVisible(true);
        levelScale.setMaximum(count - 1);
        levelScale.setSelection(0);
        levelValue.setText("0");
        lastSelection = 0;
      }
      levelSize.setText("");
      levelComposite.requestLayout();
    }

    public void setLevelSize(int width, int height) {
      levelSize.setText("W: " + width + " H: " + height);
      levelSize.requestLayout();
    }

    public void setPixel(Pixel pixel) {
      pixel.formatTo(pixelLabel);
      requestLayout();
    }

    public void showAlphaWarning(AlphaWarning message) {
      if (lastWarning == message) {
        return;
      }

      lastWarning = message;
      ((GridData)warning.getLayoutData()).exclude = message == AlphaWarning.NONE;
      warning.setVisible(message != AlphaWarning.NONE);
      warning.setText(message.warning);
      requestLayout();
    }

    private static Scale createScale(Composite parent) {
      Scale scale = new Scale(parent, SWT.HORIZONTAL);
      scale.setMinimum(0);
      scale.setMaximum(10);
      scale.setIncrement(1);
      scale.setPageIncrement(1);
      scale.setLayoutData(new RowData(150, SWT.DEFAULT));
      return scale;
    }
  }

  private static enum AlphaWarning {
    NONE(""),
    NO_ALPHA("The alpha channels appears to be empty. Click to disable the alpha channel."),
    ALPHA_DISABLED("Image contains an alpha channel. Click to re-enable alpha channel.");

    public final String warning;

    private AlphaWarning(String warning) {
      this.warning = warning;
    }
  }
}
