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

import static com.google.gapid.util.GeoUtils.right;
import static com.google.gapid.util.GeoUtils.top;
import static com.google.gapid.util.Loadable.MessageType.Error;
import static com.google.gapid.util.Loadable.MessageType.Info;
import static com.google.gapid.util.Logging.throttleLogRpcError;
import static com.google.gapid.widgets.Widgets.createBaloonToolItem;
import static com.google.gapid.widgets.Widgets.createComposite;
import static com.google.gapid.widgets.Widgets.createSeparator;
import static com.google.gapid.widgets.Widgets.createToggleToolItem;
import static com.google.gapid.widgets.Widgets.createToolItem;
import static com.google.gapid.widgets.Widgets.exclusiveSelection;
import static com.google.gapid.widgets.Widgets.disposeAllChildren;

import static java.util.Collections.emptyList;

import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.image.FetchedImage;
import com.google.gapid.image.MultiLayerAndLevelImage;
import com.google.gapid.models.Analytics.View;
import com.google.gapid.models.Capture;
import com.google.gapid.models.CommandStream;
import com.google.gapid.models.CommandStream.CommandIndex;
import com.google.gapid.models.Devices;
import com.google.gapid.models.Models;
import com.google.gapid.models.Settings;
import com.google.gapid.proto.device.Device;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.Service.ClientAction;
import com.google.gapid.proto.service.api.API;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.rpc.Rpc;
import com.google.gapid.rpc.RpcException;
import com.google.gapid.rpc.SingleInFlight;
import com.google.gapid.rpc.UiErrorCallback;
import com.google.gapid.server.Client.DataUnavailableException;
import com.google.gapid.util.Loadable;
import com.google.gapid.util.Messages;
import com.google.gapid.util.MoreFutures;
import com.google.gapid.util.Paths;
import com.google.gapid.widgets.Balloon;
import com.google.gapid.widgets.ImagePanel;
import com.google.gapid.widgets.Theme;
import com.google.gapid.widgets.Widgets;

import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Image;
import org.eclipse.swt.graphics.Rectangle;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.layout.FillLayout;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Event;
import org.eclipse.swt.widgets.Listener;
import org.eclipse.swt.widgets.ToolBar;
import org.eclipse.swt.widgets.ToolItem;

import java.util.List;
import java.util.ArrayList;
import java.util.concurrent.ExecutionException;
import java.util.logging.Logger;

/**
 * View that displays the framebuffer at the current selection in an {@link ImagePanel}.
 */
public class FramebufferView extends Composite
    implements Tab, Capture.Listener, Devices.Listener, CommandStream.Listener {
  private static final Logger LOG = Logger.getLogger(FramebufferView.class.getName());
  private static final int MAX_SIZE = 0xffff;

  private enum RenderSetting {
    RENDER_SHADED(MAX_SIZE, MAX_SIZE, Path.DrawMode.NORMAL),
    RENDER_OVERLAY(MAX_SIZE, MAX_SIZE, Path.DrawMode.WIREFRAME_OVERLAY),
    RENDER_WIREFRAME(MAX_SIZE, MAX_SIZE, Path.DrawMode.WIREFRAME_ALL),
    RENDER_OVERDRAW(MAX_SIZE, MAX_SIZE, Path.DrawMode.OVERDRAW);

    public final int maxWidth;
    public final int maxHeight;
    public final Path.DrawMode drawMode;

    private RenderSetting(int maxWidth, int maxHeight, Path.DrawMode drawMode) {
      this.maxWidth = maxWidth;
      this.maxHeight = maxHeight;
      this.drawMode = drawMode;
    }
  
    public Path.RenderSettings getRenderSettings(Settings settings) {
      return Paths.renderSettings(maxWidth, maxHeight, drawMode, 
        settings.preferences().getDisableReplayOptimization());
    }
  }

  private final Models models;
  private final Widgets widgets;
  private final SingleInFlight rpcController = new SingleInFlight();
  protected final ImagePanel imagePanel;
  private RenderSetting renderSettings;
  private int target = 0;
  private API.FramebufferAttachmentType targetType = API.FramebufferAttachmentType.OutputColor;
  private ToolItem targetItem;
  private AttachmentListener attachmentListener;
  private ToolBar toolBar;

  public FramebufferView(Composite parent, Models models, Widgets widgets) {
    super(parent, SWT.NONE);
    this.models = models;
    this.widgets = widgets;

    setLayout(new GridLayout(2, false));

    attachmentListener = new AttachmentListener(widgets.theme);

    toolBar = createToolBar(widgets.theme);
    imagePanel = new ImagePanel(this, View.Framebuffer, models.analytics, widgets, true);

    toolBar.setLayoutData(new GridData(SWT.LEFT, SWT.FILL, false, true));
    imagePanel.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));

    imagePanel.createToolbar(toolBar, widgets.theme);
    // Work around for https://bugs.eclipse.org/bugs/show_bug.cgi?id=517480
    Widgets.createSeparator(toolBar);

    renderSettings = RenderSetting.RENDER_SHADED;

    models.capture.addListener(this);
    models.devices.addListener(this);
    models.commands.addListener(this);
    addListener(SWT.Dispose, e -> {
      models.capture.removeListener(this);
      models.devices.removeListener(this);
      models.commands.removeListener(this);
    });
  }

  private ToolBar createToolBar(Theme theme) {
    ToolBar bar = new ToolBar(this, SWT.VERTICAL | SWT.FLAT);
    targetItem = createToolItem(bar, theme.lit(), attachmentListener, "Choose framebuffer attachment to display");
    createSeparator(bar);
    exclusiveSelection(
        createToggleToolItem(bar, theme.wireframeNone(), e -> {
          models.analytics.postInteraction(View.Framebuffer, ClientAction.Shaded);
          renderSettings = RenderSetting.RENDER_SHADED;
          updateBuffer();
        }, "Render shaded geometry"),
        createToggleToolItem(bar, theme.wireframeOverlay(), e -> {
          models.analytics.postInteraction(View.Framebuffer, ClientAction.OverlayWireframe);
          renderSettings = RenderSetting.RENDER_OVERLAY;
          updateBuffer();
        }, "Render shaded geometry and overlay wireframe of last draw call"),
        createToggleToolItem(bar, theme.wireframeAll(), e -> {
          models.analytics.postInteraction(View.Framebuffer, ClientAction.Wireframe);
          renderSettings = RenderSetting.RENDER_WIREFRAME;
          updateBuffer();
        }, "Render wireframe geometry"),
        createToggleToolItem(bar, theme.overdraw(), e -> {
          models.analytics.postInteraction(View.Framebuffer, ClientAction.Overdraw);
          renderSettings = RenderSetting.RENDER_OVERDRAW;
          updateBuffer();
        }, "Render overdraw"));
    createSeparator(bar);
    return bar;
  }

  @Override
  public Control getControl() {
    return this;
  }

  @Override
  public void reinitialize() {
    if (!models.capture.isLoaded()) {
      onCaptureLoadingStart(false);
    } else {
      loadBuffer();
    }
  }

  @Override
  public void onCaptureLoadingStart(boolean maintainState) {
    imagePanel.setImage(null);
    imagePanel.showMessage(Info, Messages.LOADING_CAPTURE);
    target = 0;
    attachmentListener.reset(emptyList());
  }

  @Override
  public void onCaptureLoaded(Loadable.Message error) {
    if (error != null) {
      imagePanel.setImage(null);
      imagePanel.showMessage(Error, Messages.CAPTURE_LOAD_FAILURE);
    }
    target = 0;
    attachmentListener.reset(emptyList());
  }

  @Override
  public void onCommandsLoaded() {
    loadBuffer();
  }

  @Override
  public void onCommandsSelected(CommandIndex range) {
    loadBuffer();
  }

  @Override
  public void onReplayDeviceChanged(Device.Instance dev) {
    loadBuffer();
  }

  private void updateRenderTarget(int attachment, Image icon) {
    target = attachment;
    targetItem.setImage(icon);
    updateBuffer();
  }

  private ListenableFuture<FetchedImage> loadAndUpdate() {
    CommandIndex command = models.commands.getSelectedCommands();
    if (command == null) {
      return Futures.immediateFailedFuture(new RuntimeException("No command selected"));
    }

    return MoreFutures.transformAsync(models.resources.loadFramebufferAttachments(), fbaList -> {
      attachmentListener.reset(fbaList.getAttachmentsList());
      if (fbaList.getAttachmentsList().size() <= target) {
        target = 0;
      }
      targetType = fbaList.getAttachmentsList().get(target).getType();
      return models.images.getFramebuffer(command, target, renderSettings.getRenderSettings(models.settings));
    });
  }

  private void loadBuffer() {
    if (!models.devices.hasReplayDevice()) {
      imagePanel.showMessage(Error, Messages.NO_REPLAY_DEVICE);
    } else {
      imagePanel.startLoading();
      Rpc.listen(loadAndUpdate(),
          new UiErrorCallback<FetchedImage, MultiLayerAndLevelImage, Loadable.Message>(this, LOG) {
        @Override
        protected ResultOrError<MultiLayerAndLevelImage, Loadable.Message> onRpcThread(
            Rpc.Result<FetchedImage> result) {
          try {
            return success(result.get());
          }  catch (DataUnavailableException e) {
            return error(Loadable.Message.error(e));
          } catch (RpcException e) {
            models.analytics.reportException(e);
            return error(Loadable.Message.error(e));
          } catch (ExecutionException e) {
            models.analytics.reportException(e);
            throttleLogRpcError(LOG, "Failed to load framebuffer attachments", e);
            return error(Loadable.Message.error(e.getCause().getMessage()));
          }
        }

        @Override
        protected void onUiThreadSuccess(MultiLayerAndLevelImage result) {
          imagePanel.setImage(result);

          switch (targetType) {
            case OutputColor:
              targetItem.setImage(widgets.theme.lit());
              break;

            case OutputDepth:
              targetItem.setImage(widgets.theme.depthBuffer());
              break;
          }
        }

        @Override
        protected void onUiThreadError(Loadable.Message message) {
          imagePanel.showMessage(message);
        }
      });
    }
  }

  private void updateBuffer() {
    CommandIndex command = models.commands.getSelectedCommands();
    if (command == null) {
      imagePanel.showMessage(Info, Messages.SELECT_COMMAND);
    } else if (!models.devices.hasReplayDevice()) {
      imagePanel.showMessage(Error, Messages.NO_REPLAY_DEVICE);
    } else {
      imagePanel.startLoading();
      rpcController.start().listen(models.images.getFramebuffer(command, target, renderSettings.getRenderSettings(models.settings)),
          new UiErrorCallback<FetchedImage, MultiLayerAndLevelImage, Loadable.Message>(this, LOG) {
        @Override
        protected ResultOrError<MultiLayerAndLevelImage, Loadable.Message> onRpcThread(
            Rpc.Result<FetchedImage> result) throws RpcException, ExecutionException {
          try {
            return success(result.get());
          } catch (DataUnavailableException e) {
            return error(Loadable.Message.info(e));
          } catch (RpcException e) {
            return error(Loadable.Message.error(e));
          }
        }

        @Override
        protected void onUiThreadSuccess(MultiLayerAndLevelImage result) {
          imagePanel.setImage(result);
        }

        @Override
        protected void onUiThreadError(Loadable.Message message) {
          imagePanel.showMessage(message);
        }
      });
    }
  }

  private class AttachmentListener implements Listener {
    private List<Service.FramebufferAttachment> fbaList;
    private final Theme theme;

    public AttachmentListener(Theme theme) {
      this.theme = theme;
      fbaList = emptyList();
    }

    public void reset(List<Service.FramebufferAttachment> fbaList) {
      this.fbaList = fbaList;
    }

    @Override
    public void handleEvent(Event e) {
      Rectangle b = ((ToolItem)e.widget).getBounds();
      Balloon.createAndShow(toolBar, shell -> {
        models.analytics.postInteraction(View.Framebuffer, ClientAction.ShowTargets);
        Composite c = createComposite(shell, new FillLayout(SWT.VERTICAL), SWT.BORDER);
        ToolBar tb = new ToolBar(c, SWT.HORIZONTAL | SWT.FLAT);
        if (!fbaList.isEmpty()) {
          List<ToolItem> fbaItems = new ArrayList<ToolItem>();
          for (Service.FramebufferAttachment fba : fbaList) {
            switch(fba.getType()) {
              case OutputColor:
                fbaItems.add(createToggleToolItem(tb, theme.lit(),
                  x -> updateRenderTarget(fba.getIndex(), theme.lit()),
                  "Show " + fba.getLabel()));
                break;
  
              case OutputDepth:
                fbaItems.add(createToggleToolItem(tb, theme.depthBuffer(),
                  x -> updateRenderTarget(fba.getIndex(), theme.depthBuffer()),
                  "Show " + fba.getLabel()));
                break;
            }
          }
          exclusiveSelection(fbaItems);
        }
      }, new Point(right(b) + 2, top(b)));
    }
  }
}
