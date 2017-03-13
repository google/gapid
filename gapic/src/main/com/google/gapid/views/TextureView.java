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

import static com.google.gapid.util.Loadable.MessageType.Error;
import static com.google.gapid.util.Loadable.MessageType.Info;
import static com.google.gapid.util.Paths.resourceAfter;
import static com.google.gapid.util.Ranges.last;
import static com.google.gapid.widgets.Widgets.withSpans;

import com.google.common.collect.Lists;
import com.google.gapid.Server.GapisInitException;
import com.google.gapid.image.FetchedImage;
import com.google.gapid.models.AtomStream;
import com.google.gapid.models.Capture;
import com.google.gapid.models.Models;
import com.google.gapid.models.Resources;
import com.google.gapid.proto.image.Image;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.Service.CommandRange;
import com.google.gapid.proto.service.gfxapi.GfxAPI;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.rpclib.futures.FutureController;
import com.google.gapid.rpclib.futures.SingleInFlight;
import com.google.gapid.rpclib.rpccore.Rpc;
import com.google.gapid.rpclib.rpccore.Rpc.Result;
import com.google.gapid.rpclib.rpccore.RpcException;
import com.google.gapid.server.Client;
import com.google.gapid.server.Client.DataUnavailableException;
import com.google.gapid.service.atom.AtomList;
import com.google.gapid.util.Loadable;
import com.google.gapid.util.Messages;
import com.google.gapid.util.UiErrorCallback;
import com.google.gapid.widgets.ImagePanel;
import com.google.gapid.widgets.Theme;
import com.google.gapid.widgets.Widgets;

import org.eclipse.jface.viewers.ArrayContentProvider;
import org.eclipse.jface.viewers.ComboViewer;
import org.eclipse.jface.viewers.LabelProvider;
import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.graphics.Rectangle;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Menu;
import org.eclipse.swt.widgets.MenuItem;
import org.eclipse.swt.widgets.ToolBar;
import org.eclipse.swt.widgets.ToolItem;

import java.util.Collections;
import java.util.List;
import java.util.concurrent.ExecutionException;
import java.util.function.LongConsumer;
import java.util.logging.Logger;

/**
 * View that displays the texture resources of the current capture.
 */
public class TextureView extends Composite
    implements Tab, Capture.Listener, Resources.Listener, AtomStream.Listener {
  private static final Logger LOG = Logger.getLogger(TextureView.class.getName());

  private final Client client;
  private final Models models;
  private final FutureController rpcController = new SingleInFlight();
  private final GotoAction gotoAction;
  private final ComboViewer textureCombo;
  protected final Loadable loading;
  private final ImagePanel imagePanel;

  public TextureView(Composite parent, Client client, Models models, Widgets widgets) {
    super(parent, SWT.NONE);
    this.client = client;
    this.models = models;
    this.gotoAction =
        new GotoAction(this, widgets.theme, a -> models.atoms.selectAtoms(a, 1, true));

    setLayout(new GridLayout(2, false));
    ToolBar toolBar = new ToolBar(this, SWT.VERTICAL | SWT.FLAT);
    textureCombo = createTextureSelector();
    imagePanel = createImagePanel(widgets);
    loading = imagePanel.getLoading();

    toolBar.setLayoutData(withSpans(new GridData(SWT.LEFT, SWT.FILL, false, true), 1, 2));
    textureCombo.getCombo().setLayoutData(new GridData(SWT.FILL, SWT.TOP, true, false));
    imagePanel.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));

    imagePanel.createToolbar(toolBar, widgets.theme, true);
    gotoAction.createToolItem(toolBar);

    models.capture.addListener(this);
    models.atoms.addListener(this);
    models.resources.addListener(this);
    addListener(SWT.Dispose, e -> {
      models.capture.removeListener(this);
      models.atoms.removeListener(this);
      models.resources.removeListener(this);
      gotoAction.dispose();
    });

    textureCombo.getCombo().addListener(SWT.Selection, e -> updateSelection());
  }

  protected void setImage(FetchedImage result) {
    imagePanel.setImage(result);
  }

  private ComboViewer createTextureSelector() {
    ComboViewer combo = new ComboViewer(this, SWT.READ_ONLY);
    combo.setContentProvider(ArrayContentProvider.getInstance());
    combo.setLabelProvider(new LabelProvider());
    combo.setUseHashlookup(true);
    combo.getCombo().setVisibleItemCount(10);
    return combo;
  }

  private ImagePanel createImagePanel(Widgets widgets) {
    ImagePanel panel = new ImagePanel(this, widgets);
    return panel;
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
    loading.showMessage(Info, Messages.LOADING_CAPTURE);
    clear();
  }

  @Override
  public void onCaptureLoaded(GapisInitException error) {
    if (error != null) {
      loading.showMessage(Error, Messages.CAPTURE_LOAD_FAILURE);
      clear();
    }
  }

  @Override
  public void onResourcesLoaded() {
    updateTextures(true);
  }

  @Override
  public void onAtomsSelected(CommandRange path) {
    updateTextures(false);
  }

  private void updateTextures(boolean resourcesChanged) {
    if (models.resources.isLoaded() && models.atoms.getSelectedAtoms() != null) {
      List<Data> textures = Lists.newArrayList();
      for (Service.ResourcesByType resources : models.resources.getResources()) {
        addTextures(textures, resources);
      }

      int selection = textureCombo.getCombo().getSelectionIndex();
      textureCombo.setInput(textures);
      textureCombo.refresh();

      if (!resourcesChanged && selection >= 0 && selection < textures.size()) {
        textureCombo.getCombo().select(selection);
      }

      if (textures.isEmpty()) {
        loading.showMessage(Info, Messages.NO_TEXTURES);
      } else if (textureCombo.getCombo().getSelectionIndex() <= 0) {
        textureCombo.getCombo().select(0);
        loading.showMessage(Info, Messages.SELECT_TEXTURE);
      }
      updateSelection();
    } else {
      loading.showMessage(Info, Messages.SELECT_ATOM);
      clear();
    }
  }

  private void clear() {
    gotoAction.clear();
    textureCombo.setInput(Collections.emptyList());
    textureCombo.refresh();
  }

  private void updateSelection() {
    int selection = textureCombo.getCombo().getSelectionIndex();
    if (selection < 0) {
      setImage(null);
      gotoAction.clear();
    } else if (selection == 0) {
      // Ignore the null item selection;
    } else {
      loading.startLoading();
      Data data = (Data)textureCombo.getElementAt(selection);
      Rpc.listen(FetchedImage.load(client, data.path.getResourceData()), rpcController,
          new UiErrorCallback<FetchedImage, FetchedImage, String>(this, LOG) {
        @Override
        protected ResultOrError<FetchedImage, String> onRpcThread(Result<FetchedImage> result)
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
          loading.showMessage(Error, error);
        }
      });
      gotoAction.setAtomIds(models.atoms.getData(), data.info.getAccessesList(),
          data.path.getResourceData().getAfter().getIndex());
    }
  }

  private void addTextures(List<Data> textures, Service.ResourcesByType resources) {
    if (resources == null || resources.getResourcesList().size() == 0) {
      return;
    }

    String typeLabel = getTypeLabel(resources.getType());
    if (typeLabel == null) {
      // Ignore non-texture resources (and unknown texture types).
      return;
    }

    CommandRange range = models.atoms.getSelectedAtoms();
    for (Service.Resource info : resources.getResourcesList()) {
      if (firstAccess(info) <= last(range)) {
        if (textures.isEmpty()) {
          textures.add(Data.NULL_DATA);
        }
        textures.add(
            new Data(resourceAfter(models.atoms.getPath(), range, info.getId()), info, typeLabel));
      }
    }
  }

  private static long firstAccess(Service.Resource info) {
    return (info.getAccessesCount() == 0) ? 0 : info.getAccesses(0);
  }

  private static String getTypeLabel(GfxAPI.ResourceType type) {
    if (type == null) {
      return null;
    }

    switch (type) {
      case Texture1DResource: return "1D";
      case Texture2DResource: return "2D";
      case Texture3DResource: return "3D";
      case CubemapResource: return "Cubemap";
      default: return null;
    }
  }

  /**
   * Texture metadata.
   */
  private static class Data {
    public static final Data NULL_DATA = new Data(null, null, null) {
      @Override
      public String toString() {
        return Messages.SELECT_TEXTURE;
      }
    };

    public final Path.Any path;
    public final Service.Resource info;
    public final String typeLabel;

    public Image.Info2D imageInfo;
    public String extraLabel;

    public Data(Path.Any path, Service.Resource info, String typeLabel) {
      this.path = path;
      this.info = info;
      this.typeLabel = typeLabel;
    }

    @Override
    public String toString() {
      return typeLabel + " " + info.getName() +
             (imageInfo == null ? "" : " - " + getInfoString()) +
             (extraLabel == null ? "" : " " + extraLabel);
    }

    private String getInfoString() {
      return imageInfo.getFormat() + " - " + imageInfo.getWidth() + "x" + imageInfo.getHeight();
    }
  }

  /**
   * Action for the {@link ToolItem} that allows the user to jump to references of the currently
   * displayed texture.
   */
  private static class GotoAction {
    private final Theme theme;
    private final LongConsumer listener;
    private final Menu popupMenu;
    private ToolItem item;
    private List<Long> atomIds = Collections.emptyList();

    public GotoAction(Composite parent, Theme theme, LongConsumer listener) {
      this.theme = theme;
      this.listener = listener;
      this.popupMenu = new Menu(parent);
    }

    public ToolItem createToolItem(ToolBar bar) {
      item = Widgets.createToolItem(bar, theme.jump(), e -> {
        Rectangle rect = ((ToolItem)e.widget).getBounds();
        popupMenu.setLocation(bar.toDisplay(new Point(rect.x, rect.y + rect.height)));
        popupMenu.setVisible(true);
      }, "Jump to texture reference");
      item.setEnabled(!atomIds.isEmpty());
      return item;
    }

    public void dispose() {
      popupMenu.dispose();
    }

    public void clear() {
      atomIds = Collections.emptyList();
      update(null, -1);
    }

    public void setAtomIds(AtomList atoms, List<Long> ids, long selection) {
      atomIds = ids;
      update(atoms, selection);
    }

    private void update(AtomList atoms, long selection) {
      for (MenuItem child : popupMenu.getItems()) {
        child.dispose();
      }

      for (int i = 0; i < atomIds.size(); i++) {
        long id = atomIds.get(i);
        MenuItem child = Widgets.createMenuItem(popupMenu, id + ": " + atoms.get(id).getName(), 0,
            e -> listener.accept(id));
        if (id <= selection && (i == atomIds.size() - 1 || atomIds.get(i + 1) > selection)) {
          child.setImage(theme.arrow());
        }
      }
      item.setEnabled(!atomIds.isEmpty());
    }
  }
}
