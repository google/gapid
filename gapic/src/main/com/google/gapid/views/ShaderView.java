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
import static com.google.gapid.widgets.Widgets.createBoldLabel;
import static com.google.gapid.widgets.Widgets.createComposite;
import static com.google.gapid.widgets.Widgets.createGroup;
import static com.google.gapid.widgets.Widgets.createStandardTabFolder;
import static com.google.gapid.widgets.Widgets.createStandardTabItem;

import com.google.gapid.lang.glsl.GlslSourceConfiguration;
import com.google.gapid.models.Analytics.View;
import com.google.gapid.models.Capture;
import com.google.gapid.models.Models;
import com.google.gapid.models.Resources;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.Service.ClientAction;
import com.google.gapid.proto.service.api.API;
import com.google.gapid.rpc.Rpc;
import com.google.gapid.rpc.RpcException;
import com.google.gapid.rpc.SingleInFlight;
import com.google.gapid.rpc.UiCallback;
import com.google.gapid.util.Loadable;
import com.google.gapid.util.Messages;
import com.google.gapid.widgets.LoadablePanel;
import com.google.gapid.widgets.Widgets;

import org.eclipse.jface.text.ITextOperationTarget;
import org.eclipse.jface.text.source.SourceViewer;
import org.eclipse.swt.SWT;
import org.eclipse.swt.custom.ST;
import org.eclipse.swt.custom.StyledText;
import org.eclipse.swt.layout.FillLayout;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Button;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Event;
import org.eclipse.swt.widgets.Group;
import org.eclipse.swt.widgets.Label;
import org.eclipse.swt.widgets.TabFolder;
import org.eclipse.swt.widgets.TabItem;

import java.util.concurrent.ExecutionException;
import java.util.logging.Logger;

/**
 * View allowing the inspection and editing of a shader resource.
 */
public class ShaderView extends Composite
    implements Tab, Capture.Listener, Resources.Listener {
  protected static final Logger LOG = Logger.getLogger(ShaderView.class.getName());
  private static final String EMPTY_SHADER =
      "// No source attached to this shader at this point in the trace.";

  protected final Models models;
  private final SingleInFlight rpcController = new SingleInFlight();
  private final LoadablePanel<Composite> loading;
  private final TabFolder tabFolder;
  private final Group spirvGroup;
  private final Group sourceGroup;
  private final Label crossCompileLabel;
  private final GridData crossCompileGridData;
  private final SourceViewer spirvViewer;
  private final SourceViewer sourceViewer;
  private final Button pushButton;
  private TabItem sourceTab;

  protected Service.Resource shaderResource = null;
  protected API.Shader shaderMessage = null;
  protected boolean pinned = false;

  public ShaderView(Composite parent, Models models, Widgets widgets) {
    super(parent, SWT.NONE);
    this.models = models;

    setLayout(new FillLayout());

    loading = LoadablePanel.create(this, widgets,
        panel -> createComposite(panel, new GridLayout(1, false)));

    tabFolder = createStandardTabFolder(loading.getContents());
    tabFolder.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));

    spirvGroup = createGroup(tabFolder, "");
    spirvViewer =
        new SourceViewer(spirvGroup, null, SWT.MULTI | SWT.H_SCROLL | SWT.V_SCROLL | SWT.BORDER);
    spirvViewer.setEditable(true);
    spirvViewer.configure(new GlslSourceConfiguration(widgets.theme));
    StyledText spirvTextWidget = spirvViewer.getTextWidget();
    spirvTextWidget.setFont(widgets.theme.monoSpaceFont());
    spirvTextWidget.setKeyBinding(ST.SELECT_ALL, ST.SELECT_ALL);
    spirvTextWidget.addListener(SWT.KeyDown, e -> {
      if (isKey(e, SWT.MOD1, 'z') && !isKey(e, SWT.MOD1 | SWT.SHIFT, 'z')) {
        spirvViewer.doOperation(ITextOperationTarget.UNDO);
      } else if (isKey(e, SWT.MOD1, 'y') || isKey(e, SWT.MOD1 | SWT.SHIFT, 'z')) {
        spirvViewer.doOperation(ITextOperationTarget.REDO);
      }
    });
    TabItem spirvTab = createStandardTabItem(tabFolder, "SPIR-V");
    spirvTab.setControl(spirvGroup);

    sourceGroup = createGroup(tabFolder, "", new GridLayout(1, false));
    crossCompileGridData = new GridData(SWT.LEFT, SWT.TOP, false, false);
    crossCompileLabel =
        createBoldLabel(sourceGroup, "Source code was decompiled using SPIRV-Cross");
    crossCompileLabel.setLayoutData(crossCompileGridData);
    sourceViewer =
        new SourceViewer(sourceGroup, null, SWT.MULTI | SWT.H_SCROLL | SWT.V_SCROLL | SWT.BORDER);
    sourceViewer.setEditable(true);
    sourceViewer.configure(new GlslSourceConfiguration(widgets.theme));
    sourceViewer.getControl().setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));
    StyledText sourceTextWidget = sourceViewer.getTextWidget();
    sourceTextWidget.setFont(widgets.theme.monoSpaceFont());
    sourceTextWidget.setKeyBinding(ST.SELECT_ALL, ST.SELECT_ALL);
    sourceTextWidget.addListener(SWT.KeyDown, e -> {
      if (isKey(e, SWT.MOD1, 'z') && !isKey(e, SWT.MOD1 | SWT.SHIFT, 'z')) {
        sourceViewer.doOperation(ITextOperationTarget.UNDO);
      } else if (isKey(e, SWT.MOD1, 'y') || isKey(e, SWT.MOD1 | SWT.SHIFT, 'z')) {
        sourceViewer.doOperation(ITextOperationTarget.REDO);
      }
    });

    pushButton = Widgets.createButton(loading.getContents(), "Push Changes", e -> {
      if (shaderResource != null && shaderMessage != null) {
        models.analytics.postInteraction(View.ShaderView, ClientAction.Edit);
        models.resources.updateResource(shaderResource, API.ResourceData.newBuilder()
            .setShader(shaderMessage.toBuilder().setSource(spirvViewer.getDocument().get()))
            .build());
      }
    });
    pushButton.setLayoutData(new GridData(SWT.RIGHT, SWT.BOTTOM, false, false));
    pushButton.setEnabled(false);

    models.capture.addListener(this);
    models.resources.addListener(this);
    addListener(SWT.Dispose, e -> {
      models.capture.removeListener(this);
      models.resources.removeListener(this);
    });
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
      loadShader(models.resources.getSelectedShader());
    }
  }

  @Override
  public boolean supportsPinning() {
    return true;
  }

  @Override
  public boolean isPinnable() {
    return !pinned && shaderResource != null;
  }

  @Override
  public boolean isPinned() {
    return pinned;
  }

  @Override
  public void pin() {
    pinned = true;
  }

  @Override
  public void onCaptureLoadingStart(boolean maintainState) {
    if (!pinned) {
      loading.showMessage(Info, Messages.LOADING_CAPTURE);
    }
    clear();
  }

  @Override
  public void onCaptureLoaded(Loadable.Message error) {
    if (!pinned && error != null) {
      loading.showMessage(Error, Messages.CAPTURE_LOAD_FAILURE);
    }
    clear();
  }

  @Override
  public void onResourcesLoaded() {
    if (!pinned) {
      loading.showMessage(Info, Messages.SELECT_SHADER);
      clear();
    }
  }

  @Override
  public void onShaderSelected(Service.Resource shader) {
    if (!pinned) {
      loadShader(shader);
    }
  }

  public void setShader(API.Shader shader) {
    loading.stopLoading();
    shaderMessage = shader;
    pushButton.setEnabled(true);
    String label = shaderResource != null ? shaderResource.getHandle() : "Shader";
    spirvGroup.setText(label);
    String spirvSource = shaderMessage.getSpirvSource();
    if (spirvSource.isEmpty()) {
      spirvSource = EMPTY_SHADER;
    }
    spirvViewer.setDocument(GlslSourceConfiguration.createDocument(spirvSource));
    String source = shaderMessage.getSource();
    if (!source.isEmpty()) {
      sourceViewer.setDocument(GlslSourceConfiguration.createDocument(source));
      crossCompileLabel.setVisible(shaderMessage.getCrossCompiled());
      crossCompileGridData.exclude = !shaderMessage.getCrossCompiled();
      if (sourceTab != null) {
        sourceTab.setText(shaderMessage.getSourceLanguage());
      } else {
        sourceTab = createStandardTabItem(tabFolder, shaderMessage.getSourceLanguage());
        sourceTab.setControl(sourceGroup);
      }
      sourceGroup.layout();
    } else {
      if (sourceTab != null) {
        sourceTab.dispose();
        sourceTab = null;
      }
    }
  }

  private void clear() {
    shaderResource = null;
    if (!pinned) {
      shaderMessage = null;
    }
    pushButton.setEnabled(false);
  }

  private void loadShader(Service.Resource shader) {
    clear();
    if (shader == null) {
      loading.showMessage(Info, Messages.SELECT_SHADER);
      return;
    }

    loading.startLoading();
    shaderResource = shader;
    rpcController.start().listen(models.resources.loadResource(shaderResource),
        new UiCallback<API.ResourceData, API.Shader>(this, LOG) {
      @Override
      protected API.Shader onRpcThread(Rpc.Result<API.ResourceData> result)
          throws RpcException, ExecutionException {
        return result.get().getShader();
      }

      @Override
      protected void onUiThread(API.Shader result) {
        setShader(result);
      }
    });
  }

  private static boolean isKey(Event e, int stateMask, int keyCode) {
    return (e.stateMask & stateMask) == stateMask && e.keyCode == keyCode;
  }
}
