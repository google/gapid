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
import static com.google.gapid.util.Paths.command;
import static com.google.gapid.util.Ranges.last;
import static com.google.gapid.widgets.Widgets.createBaloonToolItem;
import static com.google.gapid.widgets.Widgets.createComposite;
import static com.google.gapid.widgets.Widgets.createSeparator;
import static com.google.gapid.widgets.Widgets.createToggleToolItem;
import static com.google.gapid.widgets.Widgets.exclusiveSelection;

import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.Server.GapisInitException;
import com.google.gapid.image.FetchedImage;
import com.google.gapid.image.MultiLevelImage;
import com.google.gapid.models.AtomStream;
import com.google.gapid.models.Capture;
import com.google.gapid.models.Devices;
import com.google.gapid.models.Models;
import com.google.gapid.proto.service.Service.CommandRange;
import com.google.gapid.proto.service.Service.RenderSettings;
import com.google.gapid.proto.service.Service.WireframeMode;
import com.google.gapid.proto.service.gfxapi.GfxAPI.FramebufferAttachment;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.proto.service.path.Path.ImageInfo;
import com.google.gapid.rpclib.futures.FutureController;
import com.google.gapid.rpclib.futures.SingleInFlight;
import com.google.gapid.rpclib.rpccore.Rpc;
import com.google.gapid.rpclib.rpccore.RpcException;
import com.google.gapid.server.Client;
import com.google.gapid.server.Client.DataUnavailableException;
import com.google.gapid.util.Loadable;
import com.google.gapid.util.Messages;
import com.google.gapid.util.UiErrorCallback;
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
    implements Tab, Capture.Listener, Devices.Listener, AtomStream.Listener {
  private static final Logger LOG = Logger.getLogger(FramebufferView.class.getName());
  private static final int MAX_SIZE = 0xffff;
  private static final RenderSettings RENDER_SHADED = RenderSettings.newBuilder()
      .setMaxHeight(MAX_SIZE).setMaxWidth(MAX_SIZE)
      .setWireframeMode(WireframeMode.None)
      .build();
  private static final RenderSettings RENDER_OVERLAY = RenderSettings.newBuilder()
      .setMaxHeight(MAX_SIZE).setMaxWidth(MAX_SIZE)
      .setWireframeMode(WireframeMode.Overlay)
      .build();
  private static final RenderSettings RENDER_WIREFRAME = RenderSettings.newBuilder()
      .setMaxHeight(MAX_SIZE).setMaxWidth(MAX_SIZE)
      .setWireframeMode(WireframeMode.All)
      .build();

  private final Client client;
  private final Models models;
  private final FutureController rpcController = new SingleInFlight();
  protected final ImagePanel imagePanel;
  protected final Loadable loading;
  private RenderSettings renderSettings = RENDER_SHADED;
  private FramebufferAttachment target = FramebufferAttachment.Color0;
  private ToolItem targetItem;

  public FramebufferView(Composite parent, Client client, Models models, Widgets widgets) {
    super(parent, SWT.NONE);
    this.client = client;
    this.models = models;

    setLayout(new GridLayout(2, false));

    ToolBar toolBar = createToolBar(widgets.theme);
    imagePanel = new ImagePanel(this, widgets);
    loading = imagePanel.getLoading();

    toolBar.setLayoutData(new GridData(SWT.LEFT, SWT.FILL, false, true));
    imagePanel.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));

    imagePanel.createToolbar(toolBar, widgets.theme, false);

    models.capture.addListener(this);
    models.devices.addListener(this);
    models.atoms.addListener(this);
    addListener(SWT.Dispose, e -> {
      models.capture.removeListener(this);
      models.devices.removeListener(this);
      models.atoms.removeListener(this);
    });
  }

  private ToolBar createToolBar(Theme theme) {
    ToolBar bar = new ToolBar(this, SWT.VERTICAL | SWT.FLAT);
    targetItem = createBaloonToolItem(bar, theme.colorBuffer0(), shell -> {
      Composite c = createComposite(shell, new FillLayout(SWT.VERTICAL), SWT.BORDER);
      ToolBar b = new ToolBar(c, SWT.HORIZONTAL | SWT.FLAT);
      exclusiveSelection(
          createToggleToolItem(b, theme.colorBuffer0(),
              e -> updateRenderTarget(FramebufferAttachment.Color0, theme.colorBuffer0()),
              "Show 1st color buffer"),
          createToggleToolItem(b, theme.colorBuffer1(),
              e -> updateRenderTarget(FramebufferAttachment.Color1, theme.colorBuffer1()),
              "Show 2nd color buffer"),
          createToggleToolItem(b, theme.colorBuffer2(),
              e -> updateRenderTarget(FramebufferAttachment.Color2, theme.colorBuffer2()),
              "Show 3rd color buffer"),
          createToggleToolItem(b, theme.colorBuffer3(),
              e -> updateRenderTarget(FramebufferAttachment.Color3, theme.colorBuffer3()),
              "Show 4th color buffer"),
          createToggleToolItem(b, theme.depthBuffer(),
              e -> updateRenderTarget(FramebufferAttachment.Depth, theme.depthBuffer()),
              "Show depth buffer"));
    }, "Choose framebuffer attachment to display");
    createSeparator(bar);
    exclusiveSelection(
        createToggleToolItem(bar, theme.wireframeNone(), e -> {
          renderSettings = RENDER_SHADED;
          updateBuffer();
        }, "Render shaded geometry"),
        createToggleToolItem(bar, theme.wireframeOverlay(), e -> {
          renderSettings = RENDER_OVERLAY;
          updateBuffer();
        }, "Render shaded geometry and overlay wireframe of last draw call"),
        createToggleToolItem(bar, theme.wireframeAll(), e -> {
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
    loading.showMessage(Info, Messages.LOADING_CAPTURE);
  }

  @Override
  public void onCaptureLoaded(GapisInitException error) {
    if (error != null) {
      imagePanel.setImage(null);
      loading.showMessage(Error, Messages.CAPTURE_LOAD_FAILURE);
    }
  }

  @Override
  public void onAtomsLoaded() {
    updateBuffer();
  }

  @Override
  public void onAtomsSelected(CommandRange range) {
    updateBuffer();
  }

  @Override
  public void onReplayDeviceChanged() {
    updateBuffer();
  }

  private void updateRenderTarget(FramebufferAttachment attachment, Image icon) {
    target = attachment;
    targetItem.setImage(icon);
    updateBuffer();
  }

  private void updateBuffer() {
    CommandRange atomPath = models.atoms.getSelectedAtoms();
    if (atomPath == null) {
      loading.showMessage(Info, Messages.SELECT_ATOM);
    } else if (!models.devices.hasReplayDevice()) {
      loading.showMessage(Error, Messages.NO_REPLAY_DEVICE);
    } else {
      loading.startLoading();
      Rpc.listen(FetchedImage.load(client, getImageInfoPath(
          command(models.atoms.getPath(), last(atomPath)))), rpcController,
          new UiErrorCallback<FetchedImage, MultiLevelImage, String>(this, LOG) {
        @Override
        protected ResultOrError<MultiLevelImage, String> onRpcThread(
            Rpc.Result<FetchedImage> result) throws RpcException, ExecutionException {
          try {
            return success(result.get());
          } catch (DataUnavailableException e) {
            return error(e.getMessage());
          }
        }

        @Override
        protected void onUiThreadSuccess(MultiLevelImage result) {
          imagePanel.setImage(result);
        }

        @Override
        protected void onUiThreadError(String error) {
          loading.showMessage(Error, error);
        }
      });
    }
  }

  private ListenableFuture<ImageInfo> getImageInfoPath(Path.Command atomPath) {
    return client.getFramebufferAttachment(
        models.devices.getReplayDevice(), atomPath, target, renderSettings);
  }
}
