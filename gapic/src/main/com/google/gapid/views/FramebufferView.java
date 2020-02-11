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
import static com.google.gapid.widgets.Widgets.createBaloonToolItem;
import static com.google.gapid.widgets.Widgets.createComposite;
import static com.google.gapid.widgets.Widgets.createSeparator;
import static com.google.gapid.widgets.Widgets.createToggleToolItem;
import static com.google.gapid.widgets.Widgets.exclusiveSelection;

import com.google.gapid.image.FetchedImage;
import com.google.gapid.image.MultiLayerAndLevelImage;
import com.google.gapid.models.Analytics.View;
import com.google.gapid.models.Capture;
import com.google.gapid.models.CommandStream;
import com.google.gapid.models.CommandStream.CommandIndex;
import com.google.gapid.models.Devices;
import com.google.gapid.models.Models;
import com.google.gapid.proto.device.Device;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.Service.ClientAction;
import com.google.gapid.proto.service.api.API;
import com.google.gapid.rpc.Rpc;
import com.google.gapid.rpc.RpcException;
import com.google.gapid.rpc.SingleInFlight;
import com.google.gapid.rpc.UiErrorCallback;
import com.google.gapid.server.Client.DataUnavailableException;
import com.google.gapid.util.Loadable;
import com.google.gapid.util.Messages;
import com.google.gapid.widgets.ImagePanel;
import com.google.gapid.widgets.Theme;
import com.google.gapid.widgets.Widgets;

import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Image;
import org.eclipse.swt.layout.FillLayout;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.ToolBar;
import org.eclipse.swt.widgets.ToolItem;

import java.util.concurrent.ExecutionException;
import java.util.logging.Logger;

/**
 * View that displays the framebuffer at the current selection in an {@link ImagePanel}.
 */
public class FramebufferView extends Composite
    implements Tab, Capture.Listener, Devices.Listener, CommandStream.Listener {
  private static final Logger LOG = Logger.getLogger(FramebufferView.class.getName());
  private static final int MAX_SIZE = 0xffff;
  private static final Service.RenderSettings RENDER_SHADED = Service.RenderSettings.newBuilder()
      .setMaxHeight(MAX_SIZE).setMaxWidth(MAX_SIZE)
      .setDrawMode(Service.DrawMode.NORMAL)
      .build();
  private static final Service.RenderSettings RENDER_OVERLAY = Service.RenderSettings.newBuilder()
      .setMaxHeight(MAX_SIZE).setMaxWidth(MAX_SIZE)
      .setDrawMode(Service.DrawMode.WIREFRAME_OVERLAY)
      .build();
  private static final Service.RenderSettings RENDER_WIREFRAME = Service.RenderSettings.newBuilder()
      .setMaxHeight(MAX_SIZE).setMaxWidth(MAX_SIZE)
      .setDrawMode(Service.DrawMode.WIREFRAME_ALL)
      .build();

  private final Models models;
  private final SingleInFlight rpcController = new SingleInFlight();
  protected final ImagePanel imagePanel;
  private Service.RenderSettings renderSettings = RENDER_SHADED;
  private API.FramebufferAttachment target = API.FramebufferAttachment.Color0;
  private ToolItem targetItem;

  public FramebufferView(Composite parent, Models models, Widgets widgets) {
    super(parent, SWT.NONE);
    this.models = models;

    setLayout(new GridLayout(2, false));

    ToolBar toolBar = createToolBar(widgets.theme);
    imagePanel = new ImagePanel(this, View.Framebuffer, models.analytics, widgets, true);

    toolBar.setLayoutData(new GridData(SWT.LEFT, SWT.FILL, false, true));
    imagePanel.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));

    imagePanel.createToolbar(toolBar, widgets.theme);
    // Work around for https://bugs.eclipse.org/bugs/show_bug.cgi?id=517480
    Widgets.createSeparator(toolBar);

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
    targetItem = createBaloonToolItem(bar, theme.colorBuffer0(), shell -> {
      models.analytics.postInteraction(View.Framebuffer, ClientAction.ShowTargets);
      Composite c = createComposite(shell, new FillLayout(SWT.VERTICAL), SWT.BORDER);
      ToolBar b = new ToolBar(c, SWT.HORIZONTAL | SWT.FLAT);
      exclusiveSelection(
          createToggleToolItem(b, theme.colorBuffer0(), e -> {
            models.analytics.postInteraction(View.Framebuffer, ClientAction.Color0);
            updateRenderTarget(API.FramebufferAttachment.Color0, theme.colorBuffer0());
          }, "Show 1st color buffer"),
          createToggleToolItem(b, theme.colorBuffer1(), e -> {
            models.analytics.postInteraction(View.Framebuffer, ClientAction.Color1);
            updateRenderTarget(API.FramebufferAttachment.Color1, theme.colorBuffer1());
          }, "Show 2nd color buffer"),
          createToggleToolItem(b, theme.colorBuffer2(), e -> {
            models.analytics.postInteraction(View.Framebuffer, ClientAction.Color2);
            updateRenderTarget(API.FramebufferAttachment.Color2, theme.colorBuffer2());
          }, "Show 3rd color buffer"),
          createToggleToolItem(b, theme.colorBuffer3(), e -> {
            models.analytics.postInteraction(View.Framebuffer, ClientAction.Color3);
            updateRenderTarget(API.FramebufferAttachment.Color3, theme.colorBuffer3());
          }, "Show 4th color buffer"),
          createToggleToolItem(b, theme.depthBuffer(), e -> {
            models.analytics.postInteraction(View.Framebuffer, ClientAction.Depth);
            updateRenderTarget(API.FramebufferAttachment.Depth, theme.depthBuffer());
          }, "Show depth buffer"));
    }, "Choose framebuffer attachment to display");
    createSeparator(bar);
    exclusiveSelection(
        createToggleToolItem(bar, theme.wireframeNone(), e -> {
          models.analytics.postInteraction(View.Framebuffer, ClientAction.Shaded);
          renderSettings = RENDER_SHADED;
          updateBuffer();
        }, "Render shaded geometry"),
        createToggleToolItem(bar, theme.wireframeOverlay(), e -> {
          models.analytics.postInteraction(View.Framebuffer, ClientAction.OverlayWireframe);
          renderSettings = RENDER_OVERLAY;
          updateBuffer();
        }, "Render shaded geometry and overlay wireframe of last draw call"),
        createToggleToolItem(bar, theme.wireframeAll(), e -> {
          models.analytics.postInteraction(View.Framebuffer, ClientAction.Wireframe);
          renderSettings = RENDER_WIREFRAME;
          updateBuffer();
        }, "Render wireframe geometry"));
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
      updateBuffer();
    }
  }

  @Override
  public void onCaptureLoadingStart(boolean maintainState) {
    imagePanel.setImage(null);
    imagePanel.showMessage(Info, Messages.LOADING_CAPTURE);
  }

  @Override
  public void onCaptureLoaded(Loadable.Message error) {
    if (error != null) {
      imagePanel.setImage(null);
      imagePanel.showMessage(Error, Messages.CAPTURE_LOAD_FAILURE);
    }
  }

  @Override
  public void onCommandsLoaded() {
    updateBuffer();
  }

  @Override
  public void onCommandsSelected(CommandIndex range) {
    updateBuffer();
  }

  @Override
  public void onReplayDeviceChanged(Device.Instance dev) {
    updateBuffer();
  }

  private void updateRenderTarget(API.FramebufferAttachment attachment, Image icon) {
    target = attachment;
    targetItem.setImage(icon);
    updateBuffer();
  }

  private void updateBuffer() {
    CommandIndex command = models.commands.getSelectedCommands();
    if (command == null) {
      imagePanel.showMessage(Info, Messages.SELECT_COMMAND);
    } else if (!models.devices.hasReplayDevice()) {
      imagePanel.showMessage(Error, Messages.NO_REPLAY_DEVICE);
    } else {
      imagePanel.startLoading();
      rpcController.start().listen(models.images.getFramebuffer(command, target, renderSettings),
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
}
