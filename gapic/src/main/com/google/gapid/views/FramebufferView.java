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
import static com.google.gapid.util.GeoUtils.right;
import static com.google.gapid.util.GeoUtils.top;
import static com.google.gapid.util.Loadable.MessageType.Error;
import static com.google.gapid.util.Loadable.MessageType.Info;
import static com.google.gapid.util.Logging.throttleLogRpcError;
import static com.google.gapid.widgets.Widgets.createBaloonToolItem;
import static com.google.gapid.widgets.Widgets.createComposite;
import static com.google.gapid.widgets.Widgets.createLabel;
import static com.google.gapid.widgets.Widgets.createSeparator;
import static com.google.gapid.widgets.Widgets.createToggleToolItem;
import static com.google.gapid.widgets.Widgets.createToolItem;
import static com.google.gapid.widgets.Widgets.exclusiveSelection;
import static com.google.gapid.widgets.Widgets.disposeAllChildren;
import static com.google.gapid.widgets.Widgets.scheduleIfNotDisposed;
import static com.google.gapid.widgets.Widgets.withLayoutData;

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
import com.google.gapid.widgets.HorizontalList;
import com.google.gapid.widgets.ImagePanel;
import com.google.gapid.widgets.LoadableImage;
import com.google.gapid.widgets.LoadingIndicator;
import com.google.gapid.widgets.Theme;
import com.google.gapid.widgets.Widgets;

import org.eclipse.swt.SWT;
import org.eclipse.swt.custom.SashForm;
import org.eclipse.swt.graphics.GC;
import org.eclipse.swt.graphics.Image;
import org.eclipse.swt.graphics.Rectangle;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.internal.DPIUtil;
import org.eclipse.swt.layout.FillLayout;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Button;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Event;
import org.eclipse.swt.widgets.Label;
import org.eclipse.swt.widgets.Listener;
import org.eclipse.swt.widgets.ToolBar;
import org.eclipse.swt.widgets.ToolItem;

import java.util.ArrayList;
import java.util.Collections;
import java.util.List;
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
    RENDER_OVERDRAW(MAX_SIZE, MAX_SIZE, Path.DrawMode.OVERDRAW),
    RENDER_THUMB(THUMB_SIZE, THUMB_SIZE, Path.DrawMode.NORMAL);

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
  private final AttachmentPicker picker;
  private final Label pickerLabel;

  private final GridData pickerGridData;
  private final Button pickerToggle;
  private boolean showAttachments;

  public FramebufferView(Composite parent, Models models, Widgets widgets) {
    super(parent, SWT.NONE);
    this.models = models;
    this.widgets = widgets;

    setLayout(new GridLayout(2, false));

    ToolBar toolBar = createToolBar(widgets.theme);

    Composite header = createComposite(this, new GridLayout(2, false));

    SashForm splitter = new SashForm(this, SWT.VERTICAL);
    splitter.setLayout(new GridLayout(1, false));

    picker = new AttachmentPicker(splitter);
    picker.addContentListener(SWT.MouseDown,
      e -> picker.selectAttachment(picker.getItemAt(e.x)));

    imagePanel = new ImagePanel(splitter, View.Framebuffer, models.analytics, widgets, true);

    pickerGridData = new GridData(SWT.FILL, SWT.FILL, true, false);
    showAttachments = models.settings.ui().getFramebufferPicker().getEnabled();
    pickerGridData.exclude = true;
    picker.setVisible(false);

    toolBar.setLayoutData(new GridData(SWT.FILL, SWT.FILL, false, true, 1, 2));
    header.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, false));
    splitter.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));
    picker.setLayoutData(pickerGridData);
    imagePanel.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));

    pickerLabel = withLayoutData(createLabel(header, "Attachment:"),
      new GridData(SWT.FILL, SWT.CENTER, true, true));
    pickerToggle = new Button(header, SWT.PUSH);
    pickerToggle.setLayoutData(new GridData(SWT.RIGHT, SWT.CENTER, true, false));
    pickerToggle.setText(showAttachments ? "Hide Attachments" : "Show Attachments");
    pickerToggle.addListener(SWT.Selection, e -> {
      showAttachments = !showAttachments;
      pickerGridData.exclude = !showAttachments;
      picker.setVisible(showAttachments);
      models.settings.writeUi().getFramebufferPickerBuilder().setEnabled(showAttachments);
      pickerToggle.setText(showAttachments ? "Hide Attachments" : "Show Attachments");
      splitter.layout();
    });
    pickerToggle.setEnabled(false);

    splitter.setWeights(models.settings.getSplitterWeights(Settings.SplitterWeights.Framebuffers));
    splitter.addListener(SWT.Dispose, e ->
        models.settings.setSplitterWeights(Settings.SplitterWeights.Framebuffers, splitter.getWeights()));
    splitter.setSashWidth(5);

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
    picker.setVisible(false);
    pickerGridData.exclude = true;
    pickerToggle.setEnabled(false);
    if (!models.capture.isLoaded()) {
      onCaptureLoadingStart(false);
    } else {
      loadBuffer();
    }
  }

  @Override
  public void onCaptureLoadingStart(boolean maintainState) {
    picker.setVisible(false);
    pickerGridData.exclude = true;
    pickerToggle.setEnabled(false);
    imagePanel.setImage(null);
    imagePanel.showMessage(Info, Messages.LOADING_CAPTURE);
    target = 0;
    picker.reset();
  }

  @Override
  public void onCaptureLoaded(Loadable.Message error) {
    if (error != null) {
      imagePanel.setImage(null);
      imagePanel.showMessage(Error, Messages.CAPTURE_LOAD_FAILURE);
    }
    target = 0;
    picker.reset();
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

  private void updateRenderTarget(int attachment) {
    target = attachment;
    updateBuffer();
  }

  private void loadBuffer() {
    if (models.commands.getSelectedCommands() == null) {
      imagePanel.showMessage(Info, Messages.SELECT_COMMAND);
    } else if (!models.devices.hasReplayDevice()) {
      imagePanel.showMessage(Error, Messages.NO_REPLAY_DEVICE);
    } else {
      imagePanel.startLoading();
      Rpc.listen(models.resources.loadFramebufferAttachments(),
          new UiErrorCallback<Service.FramebufferAttachments, Service.FramebufferAttachments, Loadable.Message>(this, LOG) {
        @Override
        protected ResultOrError<Service.FramebufferAttachments, Loadable.Message> onRpcThread(
            Rpc.Result<Service.FramebufferAttachments> result) {
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
        protected void onUiThreadSuccess(Service.FramebufferAttachments fbaList) {
          pickerGridData.exclude = !showAttachments;
          picker.setVisible(showAttachments);
          pickerToggle.setEnabled(true);

          if (fbaList.getAttachmentsList().size() <= target) {
            target = 0;
          }

          List<Attachment> newAttachments = new ArrayList();
          for (Service.FramebufferAttachment fba : fbaList.getAttachmentsList()) {
            newAttachments.add(new Attachment(fba.getIndex(), fba.getType(), fba.getLabel()));
          }

          picker.setData(newAttachments);
          picker.selectAttachment(target);
        }

        @Override
        protected void onUiThreadError(Loadable.Message message) {
          pickerGridData.exclude = true;
          picker.setVisible(false);
          pickerToggle.setEnabled(false);
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

  private class Attachment {
    public final int index;
    public final API.FramebufferAttachmentType type;
    public final String label;

    public LoadableImage image;

    public Attachment(int index, API.FramebufferAttachmentType type, String label) {
      this.index = index;
      this.type = type;
      this.label = label;
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

  private class AttachmentPicker extends HorizontalList implements LoadingIndicator.Repaintable {
    private static final int MIN_SIZE = 80;

    private List<Attachment> attachments = emptyList();
    private int selectedIndex = -1;

    public AttachmentPicker(Composite parent) {
      super(parent, SWT.BORDER);
    }

    @Override
    protected void paint(GC gc, int index, int x, int y, int w, int h) {
      Attachment attachment = attachments.get(index);
      if (attachment.image == null) {
        load(attachment, index);
      }

      Image toDraw;
      if (attachment.image != null) {
        toDraw = attachment.image.getImage();
      } else {
        toDraw = widgets.loading.getCurrentFrame();
        widgets.loading.scheduleForRedraw(this);
      }
      attachment.paint(gc, toDraw, x, y, w, h, selectedIndex == index);
    }

    public void load(Attachment attachment, int index) {
      CommandIndex command = models.commands.getSelectedCommands();
      if (command == null) {
        return;
      }

      attachment.image = LoadableImage.newBuilder(widgets.loading)
          .forImageData(noAlpha(models.images.getThumbnail(command, attachment.index, THUMB_SIZE,
              info -> scheduleIfNotDisposed(this, () -> setItemSize(index,
                      Math.max(MIN_SIZE, DPIUtil.autoScaleDown(info.getWidth())),
                      Math.max(MIN_SIZE, DPIUtil.autoScaleDown(info.getHeight())))))))
          .onErrorShowErrorIcon(widgets.theme)
          .build(this, this);

      attachment.image.addListener(new LoadableImage.Listener() {
        @Override
        public void onLoaded(boolean success) {
          Rectangle bounds = attachment.image.getImage().getBounds();
          setItemSize(index, Math.max(MIN_SIZE, bounds.width), Math.max(MIN_SIZE, bounds.height));
        }
      });
    }

    public void setData(List<Attachment> newAttachments) {
      reset();
      attachments = new ArrayList(newAttachments);
      setItemCount(attachments.size(), THUMB_SIZE, THUMB_SIZE);
    }

    public void reset() {
      for (Attachment attachment : attachments) {
        attachment.dispose();
      }
      attachments = Collections.emptyList();
      selectedIndex = -1;
      setItemCount(0, THUMB_SIZE, THUMB_SIZE);
      pickerLabel.setText("Attachment:");
    }

    public void selectAttachment(int index) {
      if (index >= 0 && index < attachments.size()) {
        selectedIndex = index;
        Attachment a = attachments.get(index);
        switch(a.type) {
          case OutputColor:
            pickerLabel.setText("Attachment: Color (" + a.index + ")");
            break;

          case OutputDepth:
            pickerLabel.setText("Attachment: Depth (" + a.index + ")");
            break;

          case InputColor:
            pickerLabel.setText("Attachment: Input Color (" + a.index + ")");
            break;

          case InputDepth:
            pickerLabel.setText("Attachment: InputDepth (" + a.index + ")");
            break;
        }
        updateRenderTarget(a.index);
        repaint();
      }
    }
  }  
}
