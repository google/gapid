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
import static com.google.gapid.proto.service.api.API.ResourceType.TextureResource;
import static com.google.gapid.util.GeoUtils.bottomLeft;
import static com.google.gapid.util.Loadable.MessageType.Error;
import static com.google.gapid.util.Loadable.MessageType.Info;
import static com.google.gapid.widgets.Widgets.createCheckbox;
import static com.google.gapid.widgets.Widgets.createComposite;
import static com.google.gapid.widgets.Widgets.createMenuItem;
import static com.google.gapid.widgets.Widgets.createTableColumn;
import static com.google.gapid.widgets.Widgets.createTableViewer;
import static com.google.gapid.widgets.Widgets.filling;
import static com.google.gapid.widgets.Widgets.ifNotDisposed;
import static com.google.gapid.widgets.Widgets.packColumns;
import static com.google.gapid.widgets.Widgets.sorting;
import static com.google.gapid.widgets.Widgets.withAsyncRefresh;
import static java.util.logging.Level.FINE;

import com.google.common.collect.Lists;
import com.google.common.collect.Maps;
import com.google.common.primitives.UnsignedLongs;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.image.FetchedImage;
import com.google.gapid.models.Analytics.View;
import com.google.gapid.models.Capture;
import com.google.gapid.models.CommandStream;
import com.google.gapid.models.CommandStream.CommandIndex;
import com.google.gapid.models.Follower;
import com.google.gapid.models.Models;
import com.google.gapid.models.Resources;
import com.google.gapid.models.Settings;
import com.google.gapid.proto.image.Image;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.Service.ClientAction;
import com.google.gapid.proto.service.api.API;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.rpc.Rpc;
import com.google.gapid.rpc.RpcException;
import com.google.gapid.rpc.SingleInFlight;
import com.google.gapid.rpc.UiCallback;
import com.google.gapid.rpc.UiErrorCallback;
import com.google.gapid.server.Client.DataUnavailableException;
import com.google.gapid.util.Loadable;
import com.google.gapid.util.Messages;
import com.google.gapid.util.Paths;
import com.google.gapid.widgets.ImagePanel;
import com.google.gapid.widgets.LoadableImage;
import com.google.gapid.widgets.LoadingIndicator;
import com.google.gapid.widgets.Theme;
import com.google.gapid.widgets.VisibilityTrackingTableViewer;
import com.google.gapid.widgets.Widgets;

import org.eclipse.jface.viewers.ArrayContentProvider;
import org.eclipse.jface.viewers.LabelProvider;
import org.eclipse.jface.viewers.TableViewer;
import org.eclipse.jface.viewers.Viewer;
import org.eclipse.jface.viewers.ViewerComparator;
import org.eclipse.jface.viewers.ViewerFilter;
import org.eclipse.swt.SWT;
import org.eclipse.swt.custom.SashForm;
import org.eclipse.swt.graphics.ImageData;
import org.eclipse.swt.internal.DPIUtil;
import org.eclipse.swt.layout.FillLayout;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.layout.RowLayout;
import org.eclipse.swt.widgets.Button;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Menu;
import org.eclipse.swt.widgets.MenuItem;
import org.eclipse.swt.widgets.TableItem;
import org.eclipse.swt.widgets.ToolBar;
import org.eclipse.swt.widgets.ToolItem;
import org.eclipse.swt.widgets.Widget;

import java.util.Collections;
import java.util.Comparator;
import java.util.List;
import java.util.Map;
import java.util.concurrent.ExecutionException;
import java.util.function.Consumer;
import java.util.logging.Logger;

/**
 * View that displays the texture resources of the current capture.
 */
public class TextureView extends Composite
    implements Tab, Capture.Listener, Resources.Listener, CommandStream.Listener, Follower.Listener {
  protected static final Logger LOG = Logger.getLogger(TextureView.class.getName());

  private final Models models;
  private final SingleInFlight rpcController = new SingleInFlight();
  private final GotoAction gotoAction;
  private final TableViewer textureTable;
  private final ImageProvider imageProvider;
  protected final ImagePanel imagePanel;

  public TextureView(Composite parent, Models models, Widgets widgets) {
    super(parent, SWT.NONE);
    this.models = models;
    this.gotoAction = new GotoAction(this, models, widgets.theme,
        a -> models.commands.selectCommands(CommandIndex.forCommand(a), true));

    setLayout(new FillLayout(SWT.VERTICAL));
    SashForm splitter = new SashForm(this, SWT.VERTICAL);

    Composite tableAndOption = createComposite(splitter, new GridLayout(1, false), SWT.BORDER);
    textureTable = createTableViewer(tableAndOption, SWT.BORDER | SWT.SINGLE | SWT.FULL_SELECTION);
    imageProvider = new ImageProvider(models, textureTable, widgets.loading);
    initTextureSelector(textureTable, imageProvider);
    Composite options =
        createComposite(tableAndOption, filling(new RowLayout(SWT.HORIZONTAL), true, false));
    Button showDeleted = createCheckbox(options, "Show deleted textures", true);

    Composite imageAndToolbar = createComposite(splitter, new GridLayout(2, false));
    ToolBar toolBar = new ToolBar(imageAndToolbar, SWT.VERTICAL | SWT.FLAT);
    imagePanel = new ImagePanel(imageAndToolbar, View.Textures, models.analytics, widgets, true);

    splitter.setWeights(models.settings.getSplitterWeights(Settings.SplitterWeights.Textures));
    addListener(SWT.Dispose, e ->
      models.settings.setSplitterWeights(Settings.SplitterWeights.Textures, splitter.getWeights()));

    textureTable.getTable().setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));
    toolBar.setLayoutData(new GridData(SWT.LEFT, SWT.FILL, false, true));
    imagePanel.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));

    imagePanel.createToolbar(toolBar, widgets.theme);
    gotoAction.createToolItem(toolBar);

    models.capture.addListener(this);
    models.commands.addListener(this);
    models.resources.addListener(this);
    models.follower.addListener(this);
    addListener(SWT.Dispose, e -> {
      models.capture.removeListener(this);
      models.commands.removeListener(this);
      models.resources.removeListener(this);
      models.follower.removeListener(this);
      gotoAction.dispose();
      imageProvider.reset();
    });

    ViewerFilter filterDeleted = new ViewerFilter() {
      @Override
      public boolean select(Viewer viewer, Object parentElement, Object element) {
        return !((Data)element).deleted;
      }
    };

    textureTable.getTable().addListener(SWT.Selection, e -> updateSelection());
    showDeleted.addListener(SWT.Selection, e -> {
      if (showDeleted.getSelection()) {
        textureTable.removeFilter(filterDeleted);
      } else {
        textureTable.addFilter(filterDeleted);
      }
    });
  }

  protected void setImage(FetchedImage result) {
    imagePanel.setImage(result);
  }

  private static void initTextureSelector(TableViewer viewer, ImageProvider imageProvider) {
    viewer.setContentProvider(ArrayContentProvider.getInstance());
    viewer.setLabelProvider(new ViewLabelProvider(imageProvider));

    sorting(viewer,
        createTableColumn(viewer, "Type", Data::getType,
            Comparator.comparing(Data::getType)),
        createTableColumn(viewer, "ID", Data::getId, d -> imageProvider.getImage(d),
            (d1, d2) -> UnsignedLongs.compare(d1.getSortId(), d2.getSortId())),
        createTableColumn(viewer, "Name", Data::getLabel,
            Comparator.comparing(Data::getLabel)),
        createTableColumn(viewer, "Width", Data::getWidth,
            Comparator.comparingInt(Data::getSortWidth)),
        createTableColumn(viewer, "Height", Data::getHeight,
            Comparator.comparingInt(Data::getSortHeight)),
        createTableColumn(viewer, "Depth", Data::getDepth,
            Comparator.comparingInt(Data::getSortDepth)),
        createTableColumn(viewer, "Layers", Data::getLayers,
            Comparator.comparingInt(Data::getSortLayers)),
        createTableColumn(viewer, "Levels", Data::getLevels,
            Comparator.comparingInt(Data::getSortLevels)),
        createTableColumn(viewer, "Format", Data::getFormat,
            Comparator.comparing(Data::getFormat)));
  }

  @Override
  public Control getControl() {
    return this;
  }

  @Override
  public void reinitialize() {
    onCaptureLoadingStart(false);
    updateTextures(true);
  }

  @Override
  public void onCaptureLoadingStart(boolean maintainState) {
    imagePanel.showMessage(Info, Messages.LOADING_CAPTURE);
    clear();
  }

  @Override
  public void onCaptureLoaded(Loadable.Message error) {
    if (error != null) {
      imagePanel.showMessage(Error, Messages.CAPTURE_LOAD_FAILURE);
      clear();
    }
  }

  @Override
  public void onResourcesLoaded() {
    updateTextures(true);
  }

  @Override
  public void onCommandsSelected(CommandIndex path) {
    updateTextures(false);
  }

  @Override
  public void onTextureFollowed(Service.Resource resource) {
    TableItem[] items = textureTable.getTable().getItems();
    for (int i = 0; i < items.length; i++) {
      Data d = (Data)(items[i].getData());

      if (d.info.getID().equals(resource.getID())) {
        textureTable.getTable().setSelection(items[i]);
        updateSelection();
        break;
      }
    }
  }

  private void updateTextures(boolean resourcesChanged) {
    if (models.resources.isLoaded() && models.commands.getSelectedCommands() != null) {
      // Memorize selection index before disposing image resource.
      // When comparator is reset, the table is refreshed, and early image disposal will introduce null error.
      ViewerComparator comparator = textureTable.getComparator();
      textureTable.setComparator(null);
      int selection = textureTable.getTable().getSelectionIndex();

      imageProvider.reset();

      Widgets.Refresher refresher = withAsyncRefresh(textureTable);
      List<Data> textures = Lists.newArrayList();
      models.resources.getResources(TextureResource).stream()
          .map(r -> new Data(r.resource, r.deleted))
          .forEach(data -> {
            textures.add(data);
            data.load(models.resources, textureTable.getTable(), refresher);
          });

      textureTable.setInput(textures);
      packColumns(textureTable.getTable());

      if (!resourcesChanged && selection >= 0 && selection < textures.size()) {
        textureTable.getTable().select(selection);
      }
      textureTable.setComparator(comparator);

      if (textures.isEmpty()) {
        imagePanel.showMessage(Info, Messages.NO_TEXTURES);
      } else if (textureTable.getTable().getSelectionIndex() < 0) {
        imagePanel.showMessage(Info, Messages.SELECT_TEXTURE);
      }
      updateSelection();
    } else {
      imagePanel.showMessage(Info, Messages.SELECT_COMMAND);
      clear();
    }
  }

  private void clear() {
    gotoAction.clear();
    textureTable.setInput(Collections.emptyList());
    textureTable.getTable().requestLayout();
    imageProvider.reset();
  }

  private void updateSelection() {
    int selection = textureTable.getTable().getSelectionIndex();
    if (selection < 0) {
      setImage(null);
      gotoAction.clear();
    } else {
      imagePanel.startLoading();
      Data data = (Data)textureTable.getElementAt(selection);
      Path.ResourceData path = models.resources.getResourcePath(data.info);
      rpcController.start().listen(models.images.getResource(path),
          new UiErrorCallback<FetchedImage, FetchedImage, String>(this, LOG) {
        @Override
        protected ResultOrError<FetchedImage, String> onRpcThread(Rpc.Result<FetchedImage> result)
            throws RpcException, ExecutionException {
          try {
            return success(result.get());
          } catch (DataUnavailableException e) {
            return error(e.getMessage());
          }
        }

        @Override
        protected void onUiThreadSuccess(FetchedImage result) {
          setImage(result);
        }

        @Override
        protected void onUiThreadError(String error) {
          setImage(null);
          imagePanel.showMessage(Info, error);
        }
      });
      gotoAction.setCommandIds(data.info.getAccessesList(), path.getAfter());
    }
  }


  /**
   * Texture metadata.
   */
  private static class Data {
    public final Service.Resource info;
    public final boolean deleted;
    protected AdditionalInfo imageInfo;

    public Data(Service.Resource info, boolean deleted) {
      this.info = info;
      this.deleted = deleted;
      this.imageInfo = AdditionalInfo.NULL;
    }

    public String getType() {
      return imageInfo.getType();
    }

    public String getId() {
      return info.getHandle();
    }

    public long getSortId() {
      return info.getOrder();
    }

    public String getLabel() {
      return info.getLabel();
    }

    public String getWidth() {
      return imageInfo.getWidth();
    }

    public int getSortWidth() {
      return imageInfo.level0.getWidth();
    }

    public String getHeight() {
      return imageInfo.getHeight();
    }

    public int getSortHeight() {
      return imageInfo.level0.getHeight();
    }

    public String getDepth() {
      return imageInfo.getDepth();
    }

    public int getSortDepth() {
      return imageInfo.level0.getDepth();
    }

    public String getLayers() {
      return imageInfo.getLayers();
    }

    public int getSortLayers() {
      return imageInfo.layerCount;
    }

    public String getLevels() {
      return imageInfo.getLevels();
    }

    public int getSortLevels() {
      return imageInfo.levelCount;
    }

    public String getFormat() {
      return imageInfo.getFormat();
    }

    public void load(Resources resources, Widget widget, Widgets.Refresher refresher) {
      Rpc.listen(resources.loadResource(info),
          new UiCallback<API.ResourceData, AdditionalInfo>(widget, LOG) {
        @Override
        protected AdditionalInfo onRpcThread(Rpc.Result<API.ResourceData> result)
            throws RpcException, ExecutionException {
          try {
            return AdditionalInfo.from(result.get());
          } catch (DataUnavailableException e) {
            LOG.log(FINE, "Texture info unavailable: {0}", e.getMessage());
            return AdditionalInfo.NULL;
          }
        }

        @Override
        protected void onUiThread(AdditionalInfo result) {
          imageInfo = result;
          refresher.refresh();
        }
      });
    }

    private static class AdditionalInfo {
      public static final AdditionalInfo NULL =
          new AdditionalInfo("<unknown>", Image.Info.getDefaultInstance(), 0, 0);
      public static final AdditionalInfo NULL_1D =
          new AdditionalInfo("1D", Image.Info.getDefaultInstance(), 0, 0);
      public static final AdditionalInfo NULL_1D_ARRAY =
          new AdditionalInfo("1D Array", Image.Info.getDefaultInstance(), 0, 0);
      public static final AdditionalInfo NULL_2D =
          new AdditionalInfo("2D", Image.Info.getDefaultInstance(), 0, 0);
      public static final AdditionalInfo NULL_2D_MS =
          new AdditionalInfo("2D Multisampled", Image.Info.getDefaultInstance(), 0, 0);
      public static final AdditionalInfo NULL_2D_ARRAY =
          new AdditionalInfo("2D Array", Image.Info.getDefaultInstance(), 0, 0);
      public static final AdditionalInfo NULL_2D_MS_ARRAY =
          new AdditionalInfo("2D Multisampled Array", Image.Info.getDefaultInstance(), 0, 0);
      public static final AdditionalInfo NULL_3D =
          new AdditionalInfo("3D", Image.Info.getDefaultInstance(), 0, 0);
      public static final AdditionalInfo NULL_CUBEMAP =
          new AdditionalInfo("Cubemap", Image.Info.getDefaultInstance(), 0, 0);
      public static final AdditionalInfo NULL_CUBEMAP_ARRAY =
          new AdditionalInfo("Cubemap Array", Image.Info.getDefaultInstance(), 0, 0);

      public final Image.Info level0;
      public final int layerCount;
      public final int levelCount;
      public final String typeLabel;

      public AdditionalInfo(String typeLabel, Image.Info level0, int layerCount, int levelCount) {
        this.level0 = level0;
        this.layerCount = layerCount;
        this.levelCount = levelCount;
        this.typeLabel = typeLabel;
      }

      public static AdditionalInfo from(API.ResourceData data) {
        API.Texture texture = data.getTexture();
        switch (texture.getTypeCase()) {
          case TEXTURE_1D: {
            API.Texture1D t = texture.getTexture1D();
            return (t.getLevelsCount() == 0) ? NULL_1D :
                new AdditionalInfo("1D", t.getLevels(0), 1, t.getLevelsCount());
          }
          case TEXTURE_1D_ARRAY: {
            API.Texture1DArray t = texture.getTexture1DArray();
            return (t.getLayersCount() == 0 || t.getLayers(0).getLevelsCount() == 0) ? NULL_1D_ARRAY :
                new AdditionalInfo("1D Array", t.getLayers(0).getLevels(0), t.getLayersCount(),
                    t.getLayers(0).getLevelsCount());
          }
          case TEXTURE_2D: {
            API.Texture2D t = texture.getTexture2D();
            AdditionalInfo nullInfo = t.getMultisampled() ? NULL_2D_MS : NULL_2D;
            return (t.getLevelsCount() == 0) ? nullInfo :
                new AdditionalInfo(nullInfo.typeLabel, t.getLevels(0), 1, t.getLevelsCount());
          }
          case TEXTURE_2D_ARRAY: {
            API.Texture2DArray t = texture.getTexture2DArray();
            AdditionalInfo nullInfo = t.getMultisampled() ? NULL_2D_MS_ARRAY : NULL_2D_ARRAY;
            return (t.getLayersCount() == 0 || t.getLayers(0).getLevelsCount() == 0) ? nullInfo :
                new AdditionalInfo(nullInfo.typeLabel, t.getLayers(0).getLevels(0), t.getLayersCount(),
                    t.getLayers(0).getLevelsCount());
          }
          case TEXTURE_3D: {
            API.Texture3D t = texture.getTexture3D();
            return (t.getLevelsCount() == 0) ? NULL_3D :
                new AdditionalInfo("3D", t.getLevels(0), 1, t.getLevelsCount());
          }
          case CUBEMAP: {
            API.Cubemap c = texture.getCubemap();
            return (c.getLevelsCount() == 0) ? NULL_CUBEMAP :
                new AdditionalInfo("Cubemap", c.getLevels(0).getNegativeX(), 1, c.getLevelsCount());
          }
          case CUBEMAP_ARRAY: {
            return NULL_CUBEMAP_ARRAY; // TODO
          }
          default:
            return NULL;
        }
      }

      public String getType() {
        return typeLabel;
      }

      public String getWidth() {
        return (layerCount == 0) ? "" : String.valueOf(level0.getWidth());
      }

      public String getHeight() {
        return (layerCount == 0) ? "" : String.valueOf(level0.getHeight());
      }

      public String getDepth() {
        return (layerCount == 0) ? "" : String.valueOf(level0.getDepth());
      }

      public String getFormat() {
        return (layerCount == 0) ? "" : level0.getFormat().getName();
      }

      public String getLayers() {
        return (layerCount == 0) ? "" : String.valueOf(layerCount);
      }

      public String getLevels() {
        return (layerCount == 0) ? "" : String.valueOf(levelCount);
      }
    }
  }

  /**
   * Action for the {@link ToolItem} that allows the user to jump to references of the currently
   * displayed texture.
   */
  private static class GotoAction {
    private static final int MAX_ITEMS = 100;

    protected final Models models;
    private final Theme theme;
    private final Consumer<Path.Command> listener;
    private final Menu popupMenu;
    private ToolItem item;
    private List<Path.Command> commandIds = Collections.emptyList();

    public GotoAction(
        Composite parent, Models models, Theme theme, Consumer<Path.Command> listener) {
      this.models = models;
      this.theme = theme;
      this.listener = listener;
      this.popupMenu = new Menu(parent);
    }

    public ToolItem createToolItem(ToolBar bar) {
      item = Widgets.createToolItem(bar, theme.jump(), e -> {
        models.analytics.postInteraction(View.Textures, ClientAction.ShowReferences);
        popupMenu.setLocation(bar.toDisplay(bottomLeft(((ToolItem)e.widget).getBounds())));
        popupMenu.setVisible(true);
        loadAllCommands(models.devices.getReplayDevicePath());
      }, "Jump to texture reference");
      item.setEnabled(!commandIds.isEmpty());
      return item;
    }

    public void dispose() {
      popupMenu.dispose();
    }

    public void clear() {
      commandIds = Collections.emptyList();
      update(null);
    }

    public void setCommandIds(List<Path.Command> ids, Path.Command selection) {
      commandIds = ids;
      update(selection);
    }

    private void update(Path.Command selection) {
      for (MenuItem child : popupMenu.getItems()) {
        child.dispose();
      }

      // If we just have one additional item, simply go above the max, rather than adding the
      // "one more item not shown" message.
      int count = (commandIds.size() <= MAX_ITEMS + 1) ? commandIds.size() : MAX_ITEMS;
      for (int i = 0; i < count; i++) {
        Path.Command id = commandIds.get(i);
        MenuItem child = createMenuItem(
            popupMenu, Formatter.commandIndex(id) + ": Loading...", 0, e -> {
              models.analytics.postInteraction(View.Textures, ClientAction.GotoReference);
              listener.accept(id);
            });
        child.setData(id);
        if ((Paths.compare(id, selection) <= 0) &&
            (i == commandIds.size() - 1 || (Paths.compare(commandIds.get(i + 1), selection) > 0))) {
          child.setImage(theme.arrow());
        }
      }

      if (count != commandIds.size()) {
        // TODO: Instead of using a popup menu, create a custom widget that can handle showing
        // all the references.
        MenuItem child = createMenuItem(
            popupMenu, (commandIds.size() - count) + " more references", 0, e -> { /* do nothing */});
        child.setEnabled(false);
     }

      item.setEnabled(!commandIds.isEmpty());
    }

    private void loadAllCommands(Path.Device device) {
      for (MenuItem child : popupMenu.getItems()) {
        if (child.getData() instanceof Path.Command) {
          Path.Command path = (Path.Command)child.getData();
          Rpc.listen(models.commands.loadCommand(path, device),
              new UiCallback<API.Command, String>(child, LOG) {
            @Override
            protected String onRpcThread(Rpc.Result<API.Command> result)
                throws RpcException, ExecutionException {
              return Formatter.commandIndex(path) + ": " +
                Formatter.toString(result.get(), models.constants::getConstants);
            }

            @Override
            protected void onUiThread(String result) {
              child.setText(result);
            }
          });
          child.setData(null);
        }
      }
    }
  }

  /**
   * Image provider for the texture selector.
   */
  private static class ImageProvider implements LoadingIndicator.Repaintable {
    private static final int SIZE = DPIUtil.autoScaleUp(18);

    private final Models models;
    private final TableViewer viewer;
    private final LoadingIndicator loading;
    private final Map<Data, LoadableImage> images = Maps.newIdentityHashMap();

    public ImageProvider(Models models, TableViewer viewer, LoadingIndicator loading) {
      this.models = models;
      this.viewer = viewer;
      this.loading = loading;
    }

    public void load(Data data) {
      LoadableImage image = getLoadableImage(data);
      if (image != null) {
        image.load();
      }
    }

    public void unload(Data data) {
      LoadableImage image = images.get(data);
      if (image != null) {
        image.unload();
      }
    }

    public org.eclipse.swt.graphics.Image getImage(Data data) {
      return getLoadableImage(data).getImage();
    }

    @Override
    public void repaint() {
      ifNotDisposed(viewer.getControl(), () -> viewer.refresh());
    }

    private LoadableImage getLoadableImage(Data data) {
      LoadableImage image = images.get(data);
      if (image == null) {
        image = LoadableImage.newBuilder(loading)
            .small()
            .forImageData(() -> loadImage(data))
            .onErrorReturnNull()
            .build(viewer.getTable(), this);
        images.put(data, image);
      }
      return image;
    }

    private ListenableFuture<ImageData> loadImage(Data data) {
      return noAlpha(models.images.getThumbnail(
          models.resources.getResourcePath(data.info), SIZE, i -> { /* noop */ }));
    }

    public void reset() {
      for (LoadableImage image : images.values()) {
        image.dispose();
      }
      images.clear();
    }
  }

  private static class ViewLabelProvider extends LabelProvider
      implements VisibilityTrackingTableViewer.Listener {
    private final ImageProvider imageProvider;

    public ViewLabelProvider(ImageProvider imageProvider) {
      this.imageProvider = imageProvider;
    }

    @Override
    public void onShow(TableItem item) {
      Object element = item.getData();
      if (element instanceof Data) {
        imageProvider.load((Data)element);
      }
    }

    @Override
    public void onHide(TableItem item) {
      Object element = item.getData();
      if (element instanceof Data) {
        imageProvider.unload((Data)element);
      }
    }
  }
}
