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

import static com.google.gapid.util.Loadable.MessageType.Error;
import static com.google.gapid.util.Loadable.MessageType.Info;
import static com.google.gapid.widgets.Widgets.centered;
import static com.google.gapid.widgets.Widgets.createBaloonToolItem;
import static com.google.gapid.widgets.Widgets.createCheckbox;
import static com.google.gapid.widgets.Widgets.createComposite;
import static com.google.gapid.widgets.Widgets.createLabel;
import static com.google.gapid.widgets.Widgets.createSeparator;
import static com.google.gapid.widgets.Widgets.createToggleToolItem;
import static com.google.gapid.widgets.Widgets.createToolItem;

import com.google.gapid.glviewer.Constants;
import com.google.gapid.glviewer.ShaderSource;
import com.google.gapid.glviewer.gl.Buffer;
import com.google.gapid.glviewer.gl.Shader;
import com.google.gapid.glviewer.gl.Texture;
import com.google.gapid.image.Image;
import com.google.gapid.image.Image.PixelInfo;
import com.google.gapid.image.Image.PixelValue;
import com.google.gapid.image.MultiLevelImage;
import com.google.gapid.rpclib.futures.FutureController;
import com.google.gapid.rpclib.futures.SingleInFlight;
import com.google.gapid.rpclib.rpccore.Rpc;
import com.google.gapid.rpclib.rpccore.RpcException;
import com.google.gapid.server.Client.DataUnavailableException;
import com.google.gapid.util.Loadable;
import com.google.gapid.util.Messages;
import com.google.gapid.util.MouseAdapter;
import com.google.gapid.util.UiErrorCallback;

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
import org.lwjgl.opengl.GL15;
import org.lwjgl.opengl.GL30;

import java.util.concurrent.ExecutionException;
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.function.IntConsumer;
import java.util.logging.Logger;

/**
 * Image viewer panel with various image inspection tools.
 */
public class ImagePanel extends Composite {
  protected static final Logger LOG = Logger.getLogger(ImagePanel.class.getName());
  protected static final int ZOOM_AMOUNT = 5;
  private static final int CHANNEL_RED = 0, CHANNEL_GREEN = 1, CHANNEL_BLUE = 2, CHANNEL_ALPHA = 3;

  private final FutureController imageRequestController = new SingleInFlight();
  protected final LoadablePanel<ImageComponent> loading;
  private final StatusBar status;
  protected final ImageComponent imageComponent;
  private final BackgroundSelection backgroundSelection;
  private ToolItem backgroundItem, saveItem;
  private MultiLevelImage image = MultiLevelImage.EMPTY;
  private Image level = Image.EMPTY;

  public ImagePanel(Composite parent, Widgets widgets) {
    super(parent, SWT.NONE);
    backgroundSelection = new BackgroundSelection(getDisplay());

    setLayout(Widgets.withMargin(new GridLayout(1, false), 5, 2));

    loading = LoadablePanel.create(this, widgets, panel -> new ImageComponent(panel));
    status = new StatusBar(this, this::loadLevel);
    imageComponent = loading.getContents();

    loading.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));
    status.setLayoutData(new GridData(SWT.FILL, SWT.BOTTOM, true, false));

    imageComponent.addMouseListeners(new MouseAdapter() {
      private Point last = new Point(0, 0); // In the parent space.

      @Override
      public void mouseDown(MouseEvent e) {
        last = getPoint(e);

        if (isPanningButton(e)) {
          setCursor(getDisplay().getSystemCursor(SWT.CURSOR_SIZEALL));
          imageComponent.setPreviewPixel(Pixel.OUT_OF_BOUNDS);
        } else {
          imageComponent.zoomToFit();
        }
      }

      @Override
      public void mouseUp(MouseEvent e) {
        setCursor(null);
        setPreviewPixel(imageComponent.getPixel(getPoint(e)));
      }

      @Override
      public void mouseMove(MouseEvent e) {
        Point current = getPoint(e);
        int dx = last.x - current.x, dy = last.y - current.y;
        last = current;

        if (isPanningButton(e)) {
          imageComponent.scrollBy(dx, dy);
        } else {
          setPreviewPixel(imageComponent.getPixel(getPoint(e)));
        }
      }

      @Override
      public void mouseScrolled(MouseEvent e) {
        imageComponent.zoom(Math.max(-ZOOM_AMOUNT, Math.min(ZOOM_AMOUNT, -e.count)), getPoint(e));
      }

      @Override
      public void mouseExit(MouseEvent e) {
        setPreviewPixel(Pixel.OUT_OF_BOUNDS);
      }

      private Point getPoint(MouseEvent e) {
        return new Point(e.x, e.y);
      }

      private boolean isPanningButton(MouseEvent e) {
        // Pan for either the primary mouse button or the mouse wheel.
        if (e.button != 0) {
          return e.button == 1 || e.button == 2;
        } else {
          return (e.stateMask & (SWT.BUTTON1 | SWT.BUTTON2)) != 0;
        }
      }
    });
  }

  protected void setPreviewPixel(Pixel pixel) {
    imageComponent.setPreviewPixel(pixel);
    status.setPixel(pixel);
  }

  public void createToolbar(ToolBar bar, Theme theme, boolean enableVerticalFlip) {
    createToolItem(bar, theme.zoomFit(), e -> imageComponent.zoomToFit(), "Zoom to fit");
    createToolItem(bar, theme.zoomActual(), e -> imageComponent.zoomToActual(), "Original size");
    createToolItem(bar, theme.zoomIn(), e -> imageComponent.zoom(-ZOOM_AMOUNT, null), "Zoom in");
    createToolItem(bar, theme.zoomOut(), e -> imageComponent.zoom(ZOOM_AMOUNT, null), "Zoom out");
    createSeparator(bar);
    saveItem = createToolItem(bar, theme.save(), e -> save(), "Save image to file");
    saveItem.setEnabled(false);
    createBaloonToolItem(bar, theme.colorChannels(), shell -> {
      Composite c = createComposite(shell, new RowLayout(SWT.HORIZONTAL), SWT.BORDER);
      final ImageComponent i = imageComponent;
      createCheckbox(c, "Red", i.isChannelEnabled(CHANNEL_RED),
          e -> i.setChannelEnabled(CHANNEL_RED, ((Button)e.widget).getSelection()));
      createCheckbox(c, "Green", i.isChannelEnabled(CHANNEL_GREEN),
          e -> i.setChannelEnabled(CHANNEL_GREEN, ((Button)e.widget).getSelection()));
      createCheckbox(c, "Blue", i.isChannelEnabled(CHANNEL_BLUE),
          e -> i.setChannelEnabled(CHANNEL_BLUE, ((Button)e.widget).getSelection()));
      createCheckbox(c, "Alpha", i.isChannelEnabled(CHANNEL_ALPHA),
          e -> i.setChannelEnabled(CHANNEL_ALPHA, ((Button)e.widget).getSelection()));
    }, "Color channel selection");
    backgroundItem = createBaloonToolItem(bar, theme.transparency(),
        shell -> backgroundSelection.createBaloonContents(shell, theme,
            mode -> updateBackgroundMode(mode, theme)), "Choose image background");
    if (enableVerticalFlip) {
      createToggleToolItem(bar, theme.flipVertically(),
          e -> imageComponent.setFlipped(((ToolItem)e.widget).getSelection()), "Flip vertically");
    }
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
    dialog.setFilterNames(new String[] { "PNG Images" });
    dialog.setFilterExtensions(new String[] { "*.png" });
    dialog.setOverwrite(true);
    String path = dialog.open();
    if (path != null) {
      ImageLoader saver = new ImageLoader();
      saver.data = new ImageData[] { level.getData().getImageData() };
      saver.save(path, SWT.IMAGE_PNG);
    }
  }

  public Loadable getLoading() {
    return loading;
  }

  public void setImage(MultiLevelImage image) {
    if (image == null || image == MultiLevelImage.EMPTY) {
      clearImage();
    } else {
      if (this.image == MultiLevelImage.EMPTY) {
        // Ignore any zoom actions that might have happened before the first real image was shown.
        imageComponent.zoomToFit();
      }
      this.image = image;
      this.level = Image.EMPTY;
      loadLevel(0);
      status.setLevelCount(image.getLevelCount());
    }
  }

  public void clearImage() {
    this.image = MultiLevelImage.EMPTY;
    this.level = Image.EMPTY;
    if (saveItem != null) {
      saveItem.setEnabled(false);
    }
    status.setLevelCount(0);
    imageComponent.setImageData(level);
  }

  @Override
  public void dispose() {
    backgroundSelection.dispose();
    super.dispose();
  }

  private void loadLevel(int index) {
    if (image.getLevelCount() == 0) {
      clearImage();
      loading.showMessage(Info, Messages.NO_IMAGE_DATA);
      if (saveItem != null) {
        saveItem.setEnabled(false);
      }
      return;
    }

    index = Math.min(image.getLevelCount() - 1, index);
    loading.startLoading();
    Rpc.listen(image.getLevel(index), imageRequestController,
        new UiErrorCallback<Image, Image, String>(this, LOG) {
      @Override
      protected ResultOrError<Image, String> onRpcThread(Rpc.Result<Image> result)
          throws RpcException, ExecutionException {
        try {
          return success(result.get());
        } catch (DataUnavailableException e) {
          return error(e.getMessage());
        }
      }

      @Override
      protected void onUiThreadSuccess(Image newLevel) {
        updateLevel(newLevel);
      }

      @Override
      protected void onUiThreadError(String message) {
        clearImage();
        loading.showMessage(Error, message);
      }
    });
  }

  protected void updateLevel(Image newLevel) {
    if (level == Image.EMPTY) {
      // Ignore any zoom actions that might have happened before the first real image was shown.
      imageComponent.zoomToFit();
    }
    level = (newLevel == null) ? Image.EMPTY : newLevel;
    status.setLevelSize(level.getWidth(), level.getHeight());
    loading.stopLoading();
    if (saveItem != null) {
      saveItem.setEnabled(newLevel != null);
    }
    imageComponent.setImageData(level);
  }

  /**
   * Component that renders the image using OpenGL.
   */
  private static class ImageComponent extends Composite implements GlComposite.Listener {
    private static final double ZOOM_FIT = Double.POSITIVE_INFINITY;
    private static final int BORDER_SIZE = 2;
    private static final double MAX_ZOOM_FACTOR = 8;
    private static final double MIN_ZOOM_WIDTH = 100.0;
    private static final int PREVIEW_WIDTH = 19; // Should be odd, so center pixel looks nice.
    private static final int PREVIEW_HEIGHT = 11; // Should be odd, so center pixel looks nice.
    private static final int PREVIEW_SIZE = 7;
    private static final int MODE_TEXTURE = 0, MODE_CHECKER = 1, MODE_SOLID = 2;
    private static final float[] CURSOR_DARK = new float[] { 0, 0, 0 };
    private static final float[] CURSOR_LIGHT = new float[] { 1, 1, 1 };

    private final GlComposite canvas;
    private Image imageData = Image.EMPTY;
    private final AtomicBoolean newImageData = new AtomicBoolean(false);
    private double zoom = ZOOM_FIT;
    private boolean flipped = false;
    private float[] channels = new float[] { 1f, 1f, 1f, 1f };
    private BackgroundMode backgroundMode = BackgroundMode.Checkerboard;
    private Color backgroundColor;
    private Pixel previewPixel = Pixel.OUT_OF_BOUNDS;

    private Shader shader;
    private Buffer buffer;
    private Texture texture;
    private int[] uSize = new int[2], uOffset = new int[2];
    private float[] uRange = new float[] { 0, 1 };

    public ImageComponent(Composite parent) {
      super(parent, SWT.BORDER | SWT.V_SCROLL | SWT.H_SCROLL);
      setLayout(new FillLayout(SWT.VERTICAL));

      canvas = new GlComposite(this);
      canvas.addListener(this);

      getHorizontalBar().addListener(SWT.Selection, e -> canvas.paint());
      getVerticalBar().addListener(SWT.Selection, e -> canvas.paint());

      // Prevent the mouse wheel from scrolling the view.
      addListener(SWT.MouseWheel, e -> e.doit = false);
    }

    public void addMouseListeners(MouseAdapter mouseHandler) {
      canvas.getControl().addMouseListener(mouseHandler);
      canvas.getControl().addMouseMoveListener(mouseHandler);
      canvas.getControl().addMouseWheelListener(mouseHandler);
      canvas.getControl().addMouseTrackListener(mouseHandler);
    }

    public void setImageData(Image data) {
      imageData = data;
      newImageData.set(true);
      PixelInfo info = data.getData().getInfo();
      uRange[0] = info.getMin();
      uRange[1] = info.getMax() - info.getMin();
      canvas.paint();
    }

    public Pixel getPixel(Point point) {
      if (imageData == Image.EMPTY) {
        return Pixel.OUT_OF_BOUNDS;
      }

      Rectangle size = getClientArea();
      double scale = getScale(size);
      int w = uSize[0], h = uSize[1];
      int x = point.x - Math.max(0, size.width - w) / 2 + getHorizontalBar().getSelection();
      int y = point.y - Math.max(0, size.height - h) / 2 + getVerticalBar().getSelection();

      if (x < 0 || x >= w || y < 0 || y >= h) {
        return Pixel.OUT_OF_BOUNDS;
      }

      // Use OpenGL coordinates: origin at bottom left. While XY will be as shown
      // (possibly flipped), UV stays constant to the origin of the image used as a texture.
      int pixelX = (int)(x / scale), pixelY = (int)(y / scale);
      float u = (x - 0.5f) / w, v = (y - 0.5f) / h; // This is actually flipped v.
      int lookupY = flipped ? pixelY : imageData.getHeight() - pixelY - 1;

      return new Pixel(pixelX, imageData.getHeight() - pixelY - 1, u, flipped ? v : 1 - v,
          imageData.getData().getPixel(pixelX, lookupY));
    }

    public void setPreviewPixel(Pixel previewPixel) {
      this.previewPixel = previewPixel;
      canvas.paint();
    }

    protected void scrollBy(int dx, int dy) {
      scroll(getHorizontalBar(), dx);
      scroll(getVerticalBar(), dy);
      canvas.paint();
    }

    private static void scroll(ScrollBar bar, int delta) {
      if (delta != 0) {
        bar.setSelection(bar.getSelection() + delta);
      }
    }

    protected void zoom(int amount, Point cursor) {
      Rectangle size = getClientArea();
      double scale = getScale(size);
      zoom = Math.min(getMaxZoom(size), Math.max(getMinZoom(size), scale * (1 - 0.05f * amount)));

      if (zoom != scale) {
        updateSize(size);
        ScrollBar hBar = getHorizontalBar(), vBar = getVerticalBar();
        if (cursor == null) {
          cursor = new Point(size.width / 2, size.height / 2);
        }
        hBar.setSelection((int)((hBar.getSelection() + cursor.x) * zoom / scale - cursor.x));
        vBar.setSelection((int)((vBar.getSelection() + cursor.y) * zoom / scale - cursor.y));

        canvas.paint();
      }
    }

    protected void zoomToFit() {
      zoom = ZOOM_FIT;
      canvas.paint();
    }

    protected void zoomToActual() {
      zoom = 1;
      canvas.paint();
    }

    protected boolean isChannelEnabled(int channel) {
      return channels[channel] == 1;
    }

    protected void setChannelEnabled(int channel, boolean enabled) {
      channels[channel] = enabled ? 1 : 0;
      canvas.paint();
    }

    protected void setFlipped(boolean flipped) {
      this.flipped = flipped;
      canvas.paint();
    }

    protected void setBackgroundMode(BackgroundMode backgroundMode, Color backgroundColor) {
      this.backgroundMode = backgroundMode;
      this.backgroundColor = backgroundColor;
      canvas.paint();
    }

    @Override
    public void init() {
      GL30.glBindVertexArray(GL30.glGenVertexArrays());
      shader = ShaderSource.loadShader("image");
      buffer = new Buffer(GL15.GL_ARRAY_BUFFER);
      buffer.bind();
      buffer.loadData(new float[] { -1f, -1f, 1f, -1f, 1f,  1f, -1f,  1f });

      GL11.glDisable(GL11.GL_DEPTH_TEST);
      GL11.glDisable(GL11.GL_CULL_FACE);
      GL11.glEnable(GL11.GL_BLEND);
      GL11.glBlendFunc(GL11.GL_SRC_ALPHA, GL11.GL_ONE_MINUS_SRC_ALPHA);

      shader.bind();
      shader.bindAttribute(Constants.POSITION_ATTRIBUTE, 2, GL11.GL_FLOAT, 0, 0);
      Texture.activate(0);
      shader.setUniform("uTexture", 0);
      shader.setUniform("uRange", uRange);
    }

    @Override
    public void reshape(int x, int y, int w, int h) {
      GL11.glViewport(x, y, w, h);
    }

    @Override
    public void display() {
      Rectangle size = getClientArea();
      updateSize(size);

      if (newImageData.getAndSet(false)) {
        if (texture != null) {
          texture.delete();
        }
        texture = new Texture(GL11.GL_TEXTURE_2D);
        texture.bind()
            .setMinMagFilter(GL11.GL_LINEAR, GL11.GL_NEAREST)
            .setWrapMode(GL12.GL_CLAMP_TO_EDGE, GL12.GL_CLAMP_TO_EDGE);
        imageData.getData().uploadToTexture(texture);
        shader.setUniform("uRange", uRange);
      }

      clearWith(getDisplay().getSystemColor(SWT.COLOR_WIDGET_BACKGROUND));
      drawBackground(size);

      shader.setUniform("uMode", MODE_TEXTURE);
      shader.setUniform("uPixelSize", new float[] { 1f / size.width, 1f / size.height });
      shader.setUniform("uSize", uSize);
      shader.setUniform("uOffset", uOffset);
      shader.setUniform("uTextureSize", new float[] { 1, 1 });
      shader.setUniform("uTextureOffset", new float[] { 0, 0 });
      shader.setUniform("uChannels", channels);
      shader.setUniform("uFlipped", flipped ? 1 : 0);
      GL11.glDrawArrays(GL11.GL_TRIANGLE_FAN, 0, 4);

      drawPreview(size);
    }

    @Override
    public void dispose() {
      if (texture != null) {
        texture.delete();
      }
      buffer.delete();
      shader.delete();
    }

    private void drawBackground(Rectangle size) {
      int x = (size.width - uSize[0]) / 2, y = (size.height - uSize[1]) / 2;
      drawBorderAround(x, y, uSize[0], uSize[1]);

      switch (backgroundMode) {
        case Checkerboard:
          shader.setUniform("uMode", MODE_CHECKER);
          shader.setUniform("uSize", uSize);
          shader.setUniform("uOffset", uOffset);
          GL11.glDrawArrays(GL11.GL_TRIANGLE_FAN, 0, 4);
          break;
        case SolidColor:
          withSciscor(x, y, uSize[0], uSize[1], () -> clearWith(backgroundColor));
          break;
        default:
          throw new AssertionError();
      }
    }

    private void drawPreview(Rectangle size) {
      if (previewPixel == Pixel.OUT_OF_BOUNDS) {
        return;
      }

      drawBorderAround(
          BORDER_SIZE, BORDER_SIZE, PREVIEW_WIDTH * PREVIEW_SIZE, PREVIEW_HEIGHT * PREVIEW_SIZE);

      int[] scale = new int[] { PREVIEW_WIDTH * PREVIEW_SIZE, PREVIEW_HEIGHT * PREVIEW_SIZE };
      int[] offset = new int[] {
          (-size.width + PREVIEW_WIDTH * PREVIEW_SIZE) / 2 + BORDER_SIZE,
          (-size.height + PREVIEW_HEIGHT * PREVIEW_SIZE) / 2 + BORDER_SIZE
      };
      float[] texScale = new float[] {
          (float)PREVIEW_WIDTH / imageData.getWidth(),
          (float)PREVIEW_HEIGHT / imageData.getHeight()
      };
      float[] texOffset = new float[] {
          previewPixel.u - ((PREVIEW_WIDTH - 0.5f) / imageData.getWidth() / 2),
          previewPixel.v - ((PREVIEW_HEIGHT - 0.5f) / imageData.getHeight() / 2),
      };

      shader.setUniform("uMode", MODE_TEXTURE);
      shader.setUniform("uSize", scale);
      shader.setUniform("uOffset", offset);
      shader.setUniform("uTextureSize", texScale);
      shader.setUniform("uTextureOffset", texOffset);
      shader.setUniform("uChannels", new float[] { 1, 1, 1, 0 });
      shader.setUniform("uFlipped", 0);
      GL11.glDrawArrays(GL11.GL_TRIANGLE_FAN, 0, 4);

      // Render cursor "cross-hair"
      scale = new int[] { PREVIEW_SIZE, PREVIEW_SIZE };
      offset = new int[] {
          (-size.width + PREVIEW_WIDTH * PREVIEW_SIZE) / 2 + BORDER_SIZE,
          (-size.height + PREVIEW_HEIGHT * PREVIEW_SIZE) / 2 + BORDER_SIZE
      };

      shader.setUniform("uMode", MODE_SOLID);
      shader.setUniform("uSize", scale);
      shader.setUniform("uOffset", offset);
      shader.setUniform("uColor", previewPixel.value.isDark() ? CURSOR_LIGHT : CURSOR_DARK);
      GL11.glDrawArrays(GL11.GL_LINE_LOOP, 0, 4);
    }

    // TODO: Maybe this is not quite the best way?
    private void drawBorderAround(int x, int y, int w, int h) {
      withSciscor(x - BORDER_SIZE, y - BORDER_SIZE, w + 2 * BORDER_SIZE, h + 2 * BORDER_SIZE,
          () -> clearWith(getDisplay().getSystemColor(SWT.COLOR_WIDGET_NORMAL_SHADOW)));
    }

    private static void clearWith(Color c) {
      GL11.glClearColor(c.getRed() / 255f, c.getGreen() / 255f, c.getBlue() / 255f, 1f);
      GL11.glClear(GL11.GL_COLOR_BUFFER_BIT);
    }

    private static void withSciscor(int x, int y, int w, int h, Runnable run) {
      if (DPIUtil.getDeviceZoom() != 100) {
        // Translate SWT points to GL pixels.
        Rectangle scaled = DPIUtil.autoScaleUp(new Rectangle(x, y, w, h));
        x = scaled.x; y = scaled.y; w = scaled.width; h = scaled.height;
      }

      GL11.glScissor(x, y, w, h);
      GL11.glEnable(GL11.GL_SCISSOR_TEST);
      run.run();
      GL11.glDisable(GL11.GL_SCISSOR_TEST);
    }

    private void updateSize(Rectangle size) {
      double scale = getScale(size);
      uSize[0] = roundAndSameParity(scale, imageData.getWidth(), size.width);
      uSize[1] = roundAndSameParity(scale, imageData.getHeight(), size.height);

      uOffset[0] = -updateScrollbar(getHorizontalBar(), uSize[0], size.width);
      uOffset[1] = updateScrollbar(getVerticalBar(), uSize[1], size.height);
    }

    // Ensures the result has the same parity as the screen, so we don't draw any half pixels.
    // I.e. (screenSize - result) is even, and so borders can be easily split.
    private static int roundAndSameParity(double scale, int imageSize, int screenSize) {
      int result = (int)Math.round(scale * imageSize);
      return ((result & 1) == (screenSize & 1)) ? result : result - 1;
    }

    private static int updateScrollbar(ScrollBar bar, int imageSize, int screenSize) {
      if (imageSize > screenSize) {
        int selection = bar.getSelection();
        bar.setValues(selection, 0, imageSize, screenSize, imageSize / 100, imageSize / 10);
        bar.setEnabled(true);
        return selection - (imageSize - screenSize) / 2;
      } else {
        bar.setEnabled(false);
        bar.setValues(0, 0, imageSize, imageSize, imageSize / 100, imageSize / 10);
        return 0;
      }
    }

    private double getFitRatio(Rectangle size) {
      return Math.min((double)(size.width - 2 * BORDER_SIZE) / imageData.getWidth(),
          (double)(size.height- 2 * BORDER_SIZE) / imageData.getHeight());
    }

    private double getMinZoom(Rectangle size) {
      // The smallest zoom factor to see the whole image or that causes the larger dimension to be
      // no less than MIN_ZOOM_WIDTH pixels.
      return Math.min(1, Math.min(getFitRatio(size),
          Math.min(MIN_ZOOM_WIDTH / imageData.getWidth(), MIN_ZOOM_WIDTH / imageData.getHeight())));
    }

    private double getMaxZoom(Rectangle size) {
      return Math.max(MAX_ZOOM_FACTOR, getFitRatio(size));
    }

    private double getScale(Rectangle size) {
      return (zoom == ZOOM_FIT) ? getFitRatio(size) : zoom;
    }
  }

  /**
   * Information regarding the currently hovered pixel.
   */
  private static class Pixel {
    public static final Pixel OUT_OF_BOUNDS = new Pixel(-1, -1, -1, -1, PixelValue.NULL_PIXEL) {
      @Override
      public void formatTo(Label label) {
        label.setText(" ");
      }
    };

    public final int x, y;
    public final float u, v;
    public final PixelValue value;

    public Pixel(int x, int y, float u, float v, PixelValue value) {
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
      image = new org.eclipse.swt.graphics.Image(display, 16, 16);
      updateImage(new RGB(0, 0, 0));
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
        updateImage(newColor);
        transparency.setSelection(false);
        backgroundColor.setSelection(true);
        backgroundColor.setImage(image);
        listener.onBackgroundSelectionChanged(BackgroundMode.SolidColor);
      });
    }

    protected void updateImage(RGB newColor) {
      if (color != null) {
        color.dispose();
      }

      GC gc = new GC(image);
      color = new Color(image.getDevice(), newColor);
      gc.setBackground(color);
      gc.fillRectangle(image.getBounds());
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
    private final Label pixelLabel;
    private int lastSelection = 0;

    public StatusBar(Composite parent, IntConsumer levelListener) {
      super(parent, SWT.NONE);
      setLayout(new GridLayout(2, false));

      levelComposite = createComposite(this, centered(new RowLayout(SWT.HORIZONTAL)));
      createLabel(levelComposite, "Level:");
      levelScale = createScale(levelComposite);
      levelValue = createLabel(levelComposite, "");
      pixelLabel = createLabel(this, "");

      levelComposite.setLayoutData(new GridData(SWT.LEFT, SWT.CENTER, false, true));
      pixelLabel.setLayoutData(new GridData(SWT.FILL, SWT.CENTER, true, true));

      levelScale.addListener(SWT.Selection, e -> {
        int selection = levelScale.getSelection();
        if (selection != lastSelection) {
          lastSelection = selection;
          levelValue.setText(String.valueOf(selection));
          levelComposite.requestLayout();
          levelListener.accept(selection);
        }
      });
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
      levelComposite.requestLayout();
    }

    public void setLevelSize(int width, int height) {
      levelValue.setText(levelScale.getSelection() + ": " + width + "x" + height);
      levelComposite.requestLayout();
    }

    public void setPixel(Pixel pixel) {
      pixel.formatTo(pixelLabel);
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
}
