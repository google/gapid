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

import com.google.gapid.glviewer.gl.Renderer;
import com.google.gapid.glviewer.gl.Scene;
import com.google.gapid.glviewer.gl.Shader;
import com.google.gapid.glviewer.gl.Texture;
import com.google.gapid.glviewer.vec.MatD;
import com.google.gapid.glviewer.vec.VecD;
import com.google.gapid.image.Image;
import com.google.gapid.image.Image.PixelInfo;
import com.google.gapid.image.Image.PixelValue;
import com.google.gapid.image.MultiLevelImage;
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

import java.util.concurrent.ExecutionException;
import java.util.function.IntConsumer;
import java.util.logging.Logger;

/**
 * Image viewer panel with various image inspection tools.
 */
public class ImagePanel extends Composite {
  protected static final Logger LOG = Logger.getLogger(ImagePanel.class.getName());
  protected static final int ZOOM_AMOUNT = 5;
  private static final int CHANNEL_RED = 0, CHANNEL_GREEN = 1, CHANNEL_BLUE = 2, CHANNEL_ALPHA = 3;

  private final SingleInFlight imageRequestController = new SingleInFlight();
  protected final LoadablePanel<ImageComponent> loading;
  private final StatusBar status;
  protected final ImageComponent imageComponent;
  private final BackgroundSelection backgroundSelection;
  private ToolItem zoomFitItem, backgroundItem, saveItem;
  private MultiLevelImage image = MultiLevelImage.EMPTY;
  private Image level = Image.EMPTY;

  public ImagePanel(Composite parent, Widgets widgets, boolean naturallyFlipped) {
    super(parent, SWT.NONE);
    backgroundSelection = new BackgroundSelection(getDisplay());

    setLayout(Widgets.withMargin(new GridLayout(1, false), 5, 2));

    loading = LoadablePanel.create(this, widgets, panel ->
        new ImageComponent(panel, widgets.theme, naturallyFlipped));
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
          setZoomToFit(true);
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
        zoom(Math.max(-ZOOM_AMOUNT, Math.min(ZOOM_AMOUNT, -e.count)), getPoint(e));
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

    addListener(SWT.Dispose, e -> backgroundSelection.dispose());
  }

  protected void setPreviewPixel(Pixel pixel) {
    imageComponent.setPreviewPixel(pixel);
    status.setPixel(pixel);
  }

  public void createToolbar(ToolBar bar, Theme theme) {
    zoomFitItem = createToggleToolItem(bar, theme.zoomFit(),
        e -> setZoomToFit(((ToolItem)e.widget).getSelection()), "Zoom to fit");
    setZoomToFit(true);
    createToolItem(bar, theme.zoomActual(), e -> zoomToActual(), "Original size");
    createToolItem(bar, theme.zoomIn(), e -> zoom(-ZOOM_AMOUNT), "Zoom in");
    createToolItem(bar, theme.zoomOut(), e -> zoom(ZOOM_AMOUNT), "Zoom out");
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
    createToggleToolItem(bar, theme.flipVertically(),
        e -> imageComponent.setFlipped(((ToolItem)e.widget).getSelection()), "Flip vertically");
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
    imageComponent.setImage(level);
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
    imageRequestController.start().listen(image.getLevel(index),
        new UiErrorCallback<Image, Image, Loadable.Message>(this, LOG) {
      @Override
      protected ResultOrError<Image, Loadable.Message> onRpcThread(Rpc.Result<Image> result)
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
      protected void onUiThreadSuccess(Image newLevel) {
        updateLevel(newLevel);
      }

      @Override
      protected void onUiThreadError(Loadable.Message message) {
        clearImage();
        loading.showMessage(message);
      }
    });
  }

  protected void updateLevel(Image newLevel) {
    level = (newLevel == null) ? Image.EMPTY : newLevel;
    status.setLevelSize(level.getWidth(), level.getHeight());
    loading.stopLoading();
    if (saveItem != null) {
      saveItem.setEnabled(newLevel != null);
    }
    imageComponent.setImage(level);
  }

  private static class SceneData {
    public Image image;
    public MatD transform = MatD.IDENTITY;
    public final boolean channels[] = { true, true, true, true };
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

    public SceneData() {
    }

    public SceneData copy() {
      SceneData out = new SceneData();
      out.image = image;
      out.transform = MatD.copyOf(transform);
      System.arraycopy(channels, 0, out.channels, 0, channels.length);
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
      return out;
    }
  }

  /**
   * Component that renders the image using OpenGL.
   */
  private static class ImageComponent extends Composite {
    private static final int BORDER_SIZE = 2;
    private static final double MAX_ZOOM_FACTOR = 8;
    private static final VecD MIN_ZOOM_SIZE = new VecD(100, 100, 0);

    private final boolean naturallyFlipped;

    private final ScrollBar scrollbars[];
    private final ScenePanel<SceneData> canvas;
    private final SceneData settings;
    private Image image;

    private double scaleImageToViewMin = 0;
    private double scaleImageToViewMax = Double.POSITIVE_INFINITY;
    private double scaleImageToViewFit = 1;

    private VecD viewSize = VecD.ZERO;
    private VecD imageSize = VecD.ZERO;
    private VecD imageOffset = VecD.ZERO;
    private VecD imageOffsetMin = VecD.MIN;
    private VecD imageOffsetMax = VecD.MAX;

    private double scaleImageToView = 1.0;
    private boolean zoomToFit;

    public ImageComponent(Composite parent, Theme theme, boolean naturallyFlipped) {
      super(parent, SWT.BORDER | SWT.V_SCROLL | SWT.H_SCROLL | SWT.NO_BACKGROUND);
      setLayout(new FillLayout(SWT.VERTICAL));

      this.naturallyFlipped = naturallyFlipped;

      scrollbars = new ScrollBar[] { getHorizontalBar(), getVerticalBar() };

      settings = new SceneData();
      settings.flipped = naturallyFlipped;
      settings.borderWidth = BORDER_SIZE;
      settings.borderColor = getDisplay().getSystemColor(SWT.COLOR_WIDGET_NORMAL_SHADOW);
      settings.panelColor = getDisplay().getSystemColor(SWT.COLOR_WIDGET_BACKGROUND);
      settings.backgroundMode = BackgroundMode.Checkerboard;
      settings.checkerDark = theme.imageCheckerDark();
      settings.checkerLight = theme.imageCheckerLight();
      settings.checkerSize = 30;
      settings.cursorLight = theme.imageCursorLight();
      settings.cursorDark = theme.imageCursorDark();

      canvas = new ScenePanel<SceneData>(this, new ImageScene());
      canvas.setSceneData(settings.copy());

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

    public void setImage(Image image) {
      this.image = image;
      imageSize = new VecD(image.getWidth(), image.getHeight(), 0);
      updateScaleLimits();
      setScale(zoomToFit ? scaleImageToViewFit : scaleImageToView);
      refresh();
    }

    private void refresh() {
      settings.image = image;
      settings.transform = calcTransform();
      canvas.setSceneData(settings.copy());
    }

    public void setPreviewPixel(Pixel previewPixel) {
      settings.previewPixel = previewPixel;
      refresh();
    }

    protected void scrollBy(int dx, int dy) {
      imageOffset = imageOffset
          .add(-dx / scaleImageToView, -dy / scaleImageToView, 0)
          .clamp(imageOffsetMin, imageOffsetMax);
      updateScrollbars();
      refresh();
    }

    protected void setBackgroundMode(BackgroundMode backgroundMode, Color backgroundColor) {
      settings.backgroundMode = backgroundMode;
      settings.backgroundColor = backgroundColor;
      refresh();
    }

    protected void setFlipped(boolean flipped) {
      settings.flipped = flipped ^ naturallyFlipped;
      refresh();
    }

    protected boolean isChannelEnabled(int channel) {
      return settings.channels[channel];
    }

    protected void setChannelEnabled(int channel, boolean enabled) {
      settings.channels[channel] = enabled;
      refresh();
    }

    public Pixel getPixel(Point point) {
      VecD imageNormalized = calcInvTransform().multiply(pointToNDC(point));
      VecD imageTexel = imageNormalized.multiply(0.5).add(0.5, 0.5, 0.0).multiply(imageSize);
      int x = (int)imageTexel.x;
      int y = (int)imageTexel.y;
      if (x < 0 || y < 0 || x >= image.getWidth() || y >= image.getHeight()) {
        return Pixel.OUT_OF_BOUNDS;
      }
      float u = (float)imageNormalized.x * 0.5f + 0.5f;
      float v = (float)imageNormalized.y * (settings.flipped ? 0.5f : 0.5f) + 0.5f;
      int sampleY = settings.flipped ? (image.getHeight() - y - 1) : y;
      return new Pixel(x, y, u, v, image.getData().getPixel(x, sampleY));
    }

    public void setZoomToFit(boolean zoomToFit) {
      this.zoomToFit = zoomToFit;
      if (zoomToFit) {
        setScale(scaleImageToViewFit);
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
      double scale = scaleImageToView * (1 - 0.05f * amount);

      if (cursor != null) {
        VecD ndc = pointToNDC(cursor);
        VecD imageNormPreScale = calcInvTransform().multiply(ndc);
        setScale(scale);
        VecD imageNormPostScale = calcInvTransform().multiply(ndc);
        VecD imageNormDelta = imageNormPostScale.subtract(imageNormPreScale);
        imageOffset = imageOffset
            .add(imageNormDelta.multiply(imageSize).multiply(0.5))
            .clamp(imageOffsetMin, imageOffsetMax);
      } else {
        setScale(scale);
      }
      updateScrollbars();
      refresh();
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

    /**
     * @return a transform that can be used to convert the texture quad [-1, -1] to [1, 1] into
     * NDC coordinates based on the current scale and offset.
     */
    private MatD calcTransform() {
      return MatD.makeScale(new VecD(2, 2, 0).safeDivide(viewSize))
              .scale(scaleImageToView)
              .translate(imageOffset)
              .scale(imageSize.multiply(0.5));
    }

    /**
     * @return the inverse of the matrix returned from {@link #calcTransform()}.
     */
    private MatD calcInvTransform() {
      return MatD.makeScale(new VecD(2, 2, 0).safeDivide(imageSize))
          .translate(imageOffset.negate())
          .scale(1.0 / scaleImageToView)
          .scale(viewSize.multiply(0.5));
    }

    private static double clamp(double x, double min, double max) {
      return Math.max(Math.min(x, max), min);
    }

    private void setScale(double scale) {
      scaleImageToView = clamp(scale, scaleImageToViewMin, scaleImageToViewMax);

      VecD viewSizeSubBorder = viewSize.subtract(BORDER_SIZE * 2);

      imageOffsetMax = imageSize
          .subtract(viewSizeSubBorder.divide(scaleImageToView))
          .multiply(0.5)
          .max(VecD.ZERO);
      imageOffsetMin = imageOffsetMax.negate();
      imageOffset = imageOffset.clamp(imageOffsetMin, imageOffsetMax);
    }

    private void onResize() {
      Rectangle area = canvas.getClientArea();
      viewSize = new VecD(area.width, area.height, 0);
      updateScaleLimits();
      if (zoomToFit) {
        setScale(scaleImageToViewFit);
      }
      updateScrollbars();
      refresh();
    }

    private void updateScaleLimits() {
      VecD viewSpace = viewSize.subtract(settings.borderWidth).max(VecD.ZERO);
      scaleImageToViewFit = viewSpace.safeDivide(imageSize).minXY();
      scaleImageToViewMax = Math.max(MAX_ZOOM_FACTOR, scaleImageToViewFit);
      // The smallest zoom factor to see the whole image or that causes the larger dimension to be
      // no less than MIN_ZOOM_WIDTH pixels.
      scaleImageToViewMin = Math.min(MIN_ZOOM_SIZE.safeDivide(imageSize).minXY(), scaleImageToViewFit);
    }

    private void updateScrollbars() {
      for (int i = 0; i < scrollbars.length; i++) {
        ScrollBar scrollbar = scrollbars[i];
        int val = (int)(imageOffset.get(i) * scaleImageToView); // offset in view pixels
        int min = (int)(imageOffsetMin.get(i) * scaleImageToView); // min movement in view pixels
        int max = (int)(imageOffsetMax.get(i) * scaleImageToView); // max movement in view pixels
        int rng = max - min;
        if (rng == 0) {
          scrollbar.setEnabled(false);
          scrollbar.setValues(0, 0, 1, 1, 1, 1);
        } else {
          int view = (int)this.viewSize.get(i);
          scrollbar.setEnabled(true);
          scrollbar.setValues(
              max - val,        // selection
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
          int max = (int)(imageOffsetMax.get(i) * scaleImageToView); // max movement in view pixels
          int val = max - scrollbar.getSelection();
          imageOffset = imageOffset.set(i, val / scaleImageToView);
        }
      }
      refresh();
    }
  }

  private static class ImageScene implements Scene<SceneData> {
    private static final int PREVIEW_WIDTH = 19; // Should be odd, so center pixel looks nice.
    private static final int PREVIEW_HEIGHT = 11; // Should be odd, so center pixel looks nice.
    private static final int PREVIEW_SIZE = 7;

    private Shader shader;
    private Texture texture;
    private SceneData data;

    private final float[] uRange = new float[] { 0, 1 };
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
      Image oldImage = (this.data != null) ? this.data.image : null;
      if (oldImage != newData.image) {
        // Release old texture, create new.
        if (texture != null) {
          texture.delete();
        }
        texture = renderer
            .newTexture(GL11.GL_TEXTURE_2D)
            .setMinMagFilter(GL11.GL_LINEAR, GL11.GL_NEAREST)
            .setBorderColor(newData.borderColor);
        newData.image.getData().uploadToTexture(texture);
        shader.setUniform("uTexture", texture);

        // Get range limits, update uniforms.
        PixelInfo info = newData.image.getData().getInfo();
        uRange[0] = info.getMin();
        uRange[1] = info.getMax() - info.getMin();
        shader.setUniform("uRange", uRange);
      }
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
      if (texture != null) {
        drawImage(renderer);
        drawPreview(renderer);
      }
    }

    @Override
    public void resize(Renderer renderer, int width, int height) {
      // Ignore.
    }

    private void drawBackground(Renderer renderer) {
      switch (data.backgroundMode) {
        case Checkerboard:
          renderer.drawChecker(data.transform, data.checkerLight, data.checkerDark, data.checkerSize);
          break;
        case SolidColor:
          renderer.drawSolid(data.transform, data.backgroundColor);
          break;
        default:
          throw new AssertionError();
      }
      renderer.drawBorder(data.transform, data.borderColor, data.borderWidth);
    }

    private void drawImage(Renderer renderer) {
      texture.setWrapMode(GL12.GL_CLAMP_TO_EDGE, GL12.GL_CLAMP_TO_EDGE);
      shader.setUniform("uTextureSize", new float[] { 1, 1 });
      shader.setUniform("uTextureOffset", new float[] { 0, 0 });
      shader.setUniform("uChannels", uChannels);
      shader.setUniform("uFlipped", data.flipped ? 1 : 0);
      renderer.drawQuad(data.transform, shader);
    }

    private void drawPreview(Renderer renderer) {
      if (data.previewPixel == Pixel.OUT_OF_BOUNDS) {
        return;
      }

      int width = PREVIEW_WIDTH * PREVIEW_SIZE;
      int height = PREVIEW_HEIGHT * PREVIEW_SIZE;
      int x = data.borderWidth;
      int y = renderer.getViewHeight() - height - data.borderWidth;

      renderer.drawBorder(x, y, width, height, data.borderColor, data.borderWidth);

      float[] texScale = new float[] {
          (float)PREVIEW_WIDTH / data.image.getWidth(),
          (float)PREVIEW_HEIGHT / data.image.getHeight()
      };
      float[] texOffset = new float[] {
          (float)(data.previewPixel.x - PREVIEW_WIDTH / 2) / data.image.getWidth(),
          (float)(data.previewPixel.y - PREVIEW_HEIGHT / 2) / data.image.getHeight()
      };

      texture.setWrapMode(GL13.GL_CLAMP_TO_BORDER, GL13.GL_CLAMP_TO_BORDER);
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
