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

import static com.google.gapid.util.GeoUtils.bottomLeft;
import static com.google.gapid.util.Loadable.MessageType.Info;
import static com.google.gapid.widgets.Widgets.createComposite;
import static com.google.gapid.widgets.Widgets.createMenuItem;
import static com.google.gapid.widgets.Widgets.createSeparator;
import static com.google.gapid.widgets.Widgets.createToolItem;

import com.google.gapid.image.FetchedImage;
import com.google.gapid.models.Analytics.View;
import com.google.gapid.models.Capture;
import com.google.gapid.models.CommandStream.CommandIndex;
import com.google.gapid.models.Models;
import com.google.gapid.models.Resources;
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
import com.google.gapid.util.Messages;
import com.google.gapid.util.Paths;
import com.google.gapid.widgets.ImagePanel;
import com.google.gapid.widgets.Theme;
import com.google.gapid.widgets.Widgets;

import org.eclipse.swt.SWT;
import org.eclipse.swt.layout.FillLayout;
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
import java.util.function.Consumer;
import java.util.logging.Logger;

/**
 * View that displays a selected texture resource of the current capture.
 */
public class TextureView extends Composite
    implements Tab, Capture.Listener, Resources.Listener {
  protected static final Logger LOG = Logger.getLogger(TextureView.class.getName());

  public TextureView(Composite parent, Service.Resource texture, Models models, Widgets widgets) {
    super(parent, SWT.NONE);
    setLayout(new FillLayout());

    TextureWidget view = new TextureWidget(this, true, models, widgets);
    view.loadTexture(texture);
  }

  @Override
  public Control getControl() {
    return this;
  }

  public static class TextureWidget extends Composite {
    private final boolean pinned;
    private final Models models;
    private final SingleInFlight rpcController = new SingleInFlight();
    private final ToolItem pinItem;
    private final GotoAction gotoAction;
    protected final ImagePanel imagePanel;
    private Service.Resource textureResource = null;

    public TextureWidget(Composite parent, boolean pinned, Models models, Widgets widgets) {
      super(parent, SWT.NONE);
      this.pinned = pinned;
      this.models = models;
      this.gotoAction = new GotoAction(this, models, widgets.theme,
          a -> models.commands.selectCommands(CommandIndex.forCommand(a), true));

      setLayout(new FillLayout(SWT.VERTICAL));
      Composite imageAndToolbar = createComposite(this, new GridLayout(2, false));
      ToolBar toolBar = new ToolBar(imageAndToolbar, SWT.VERTICAL | SWT.FLAT);
      imagePanel = new ImagePanel(imageAndToolbar, View.TextureView, models.analytics, widgets);

      toolBar.setLayoutData(new GridData(SWT.LEFT, SWT.FILL, false, true));
      imagePanel.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));

      if (pinned) {
        pinItem = createToolItem(
            toolBar, widgets.theme.pinned(), e -> { /* ignore */ }, "Pinned texture");
      } else {
        pinItem = createToolItem(toolBar, widgets.theme.pin(),
            e -> models.resources.pinTexture(textureResource), "Pin this texture");
      }
      createSeparator(toolBar);
      imagePanel.createToolbar(toolBar, widgets.theme);
      gotoAction.createToolItem(toolBar);

      addListener(SWT.Dispose, e -> {
        gotoAction.dispose();
      });

      clear();
    }

    public void clear() {
      textureResource = null;
      pinItem.setEnabled(false);
      gotoAction.clear();
      imagePanel.showMessage(Info, Messages.SELECT_TEXTURE);
    }

    public void loadTexture(Service.Resource texture) {
      if (texture == null) {
        clear();
        return;
      }
      textureResource = texture;

      imagePanel.startLoading();
      Path.ResourceData path = models.resources.getResourcePath(texture);
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
      gotoAction.setCommandIds(texture.getAccessesList(), path.getAfter());
    }

    protected void setImage(FetchedImage result) {
      imagePanel.setImage(result);
      pinItem.setEnabled(!pinned);
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
          models.analytics.postInteraction(View.TextureView, ClientAction.ShowReferences);
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
                models.analytics.postInteraction(View.TextureView, ClientAction.GotoReference);
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
  }
}
