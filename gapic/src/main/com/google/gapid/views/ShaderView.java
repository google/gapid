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

import static com.google.gapid.util.Loadable.MessageType.Info;
import static com.google.gapid.widgets.Widgets.createBoldLabel;
import static com.google.gapid.widgets.Widgets.createButton;
import static com.google.gapid.widgets.Widgets.createComposite;
import static com.google.gapid.widgets.Widgets.createGroup;
import static com.google.gapid.widgets.Widgets.createLabel;
import static com.google.gapid.widgets.Widgets.createScrolledComposite;
import static com.google.gapid.widgets.Widgets.createStandardTabFolder;
import static com.google.gapid.widgets.Widgets.createStandardTabItem;
import static com.google.gapid.widgets.Widgets.disposeAllChildren;
import static com.google.gapid.widgets.Widgets.withLayoutData;

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
import com.google.gapid.util.Experimental;
import com.google.gapid.util.Loadable;
import com.google.gapid.util.Messages;
import com.google.gapid.widgets.LoadablePanel;
import com.google.gapid.widgets.Widgets;

import org.eclipse.jface.text.ITextOperationTarget;
import org.eclipse.jface.text.source.SourceViewer;
import org.eclipse.swt.SWT;
import org.eclipse.swt.custom.ST;
import org.eclipse.swt.custom.ScrolledComposite;
import org.eclipse.swt.custom.StyledText;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.graphics.Rectangle;
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

import java.util.Optional;
import java.util.concurrent.ExecutionException;
import java.util.logging.Logger;

/**
 * View allowing the inspection and editing of a shader resource.
 */
public class ShaderView extends Composite
    implements Tab, Capture.Listener, Resources.Listener {
  protected static final Logger LOG = Logger.getLogger(ShaderView.class.getName());

  private final Models models;
  private final ShaderWidget shaderView;
  private Service.Resource shaderResource = null;

  private boolean pinned = false;

  public ShaderView(Composite parent, Models models, Widgets widgets) {
    super(parent, SWT.NONE);
    this.models = models;

    setLayout(new FillLayout());
    shaderView = new ShaderWidget(this, models, widgets);

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
      shaderView.loadShader(models.resources.getSelectedShader());
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
    shaderView.clear();
    if (!pinned) {
      shaderResource = null;
    }
  }

  @Override
  public void onCaptureLoaded(Loadable.Message error) {
    shaderView.clear();
    if (!pinned) {
      shaderResource = null;
    }

  }

  @Override
  public void onResourcesLoaded() {
    shaderView.clear();
    if (!pinned) {
      shaderResource = null;
    }
  }

  @Override
  public void onShaderSelected(Service.Resource shader) {
    if (!pinned) {
      shaderResource = shader;
      shaderView.loadShader(shader);
    }
  }

  public static class ShaderWidget extends Composite {
    private static final String EMPTY_SHADER =
        "// No source attached to this shader at this point in the trace.";

    private final SingleInFlight rpcController = new SingleInFlight();
    private final Models models;
    private final LoadablePanel<Composite> loading;
    private final TabFolder tabFolder;
    private final Group spirvGroup;
    private final Group sourceGroup;
    private final Group statGroup;
    private final Label crossCompileLabel;
    private final GridData crossCompileGridData;
    private final SourceViewer spirvViewer;
    private final SourceViewer sourceViewer;
    private final Optional<Button> pushButton;
    private TabItem sourceTab;
    private Service.Resource shaderResource = null;
    private API.Shader shaderMessage = null;


    public ShaderWidget(Composite parent, Models models, Widgets widgets) {
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
      spirvViewer.setEditable(false);
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
      sourceViewer.setEditable(false);
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

      // TODO(b/188434910): Shader editing is disabled as it doesn't work right
      // now. Enable it once the issue is fixed.
      if (!Experimental.enableUnstableFeatures(models.settings)) {
        pushButton = Optional.empty();
      } else {
        pushButton = Optional.of(createButton(loading.getContents(), "Push Changes", e -> {
          if (shaderResource != null && shaderMessage != null) {
            models.analytics.postInteraction(View.ShaderView, ClientAction.Edit);
            models.resources.updateResource(shaderResource, API.ResourceData.newBuilder()
                .setShader(shaderMessage.toBuilder().setSource(spirvViewer.getDocument().get()))
                .build());
          }
        }));
        pushButton.ifPresent(b -> {
          b.setLayoutData(new GridData(SWT.RIGHT, SWT.BOTTOM, false, false));
          b.setEnabled(false);
        });
        spirvViewer.setEditable(true);
        sourceViewer.setEditable(true);
      }

      statGroup = createGroup(loading.getContents(), "Static Analysis");
      statGroup.setFont(widgets.theme.subTitleFont());
      statGroup.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, false));
      statGroup.setLayout(new GridLayout(1, false));
    }

    public void clear() {
      shaderResource = null;
      shaderMessage = null;
      pushButton.ifPresent(b -> b.setEnabled(false));
      loading.showMessage(Info, Messages.SELECT_SHADER);
    }

    public void loadShader(Service.Resource shader) {
      if (shader == null) {
        clear();
        return;
      }

      loading.startLoading();
      rpcController.start().listen(models.resources.loadResource(shader),
          new UiCallback<API.ResourceData, API.Shader>(this, LOG) {
        @Override
        protected API.Shader onRpcThread(Rpc.Result<API.ResourceData> result)
            throws RpcException, ExecutionException {
          return result.get().getShader();
        }

        @Override
        protected void onUiThread(API.Shader result) {
          setShader(shader, result);
        }
      });
    }

    public void setShader(Service.Resource resource, API.Shader shader) {
      loading.stopLoading();

      shaderResource = resource;
      shaderMessage = shader;

      pushButton.ifPresent(b -> b.setEnabled(true));
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

      disposeAllChildren(statGroup);
      createStaticAnalysisGroup(statGroup, shaderMessage.getStaticAnalysis());
      statGroup.requestLayout();
    }

    private static void createStaticAnalysisGroup(
        Group dataGroupComposite, API.Shader.StaticAnalysis staticAnalysis) {
      ScrolledComposite scrollComposite = withLayoutData(createScrolledComposite(dataGroupComposite,
          new FillLayout(), SWT.V_SCROLL | SWT.H_SCROLL),
          new GridData(SWT.FILL, SWT.FILL, true, true));

      GridLayout gridLayout = new GridLayout(2, false);
      gridLayout.marginWidth = 5;
      gridLayout.marginHeight = 5;
      gridLayout.horizontalSpacing = -1;
      gridLayout.verticalSpacing = -1;
      Composite contentComposite = createComposite(scrollComposite, gridLayout);

      createStaticAnalysisStat(
          contentComposite, "ALU Instructions:", staticAnalysis.getAluInstructions());
      createStaticAnalysisStat(
          contentComposite, "Texture Instructions:", staticAnalysis.getTextureInstructions());
      createStaticAnalysisStat(
          contentComposite, "Branch Instructions:", staticAnalysis.getBranchInstructions());
      createStaticAnalysisStat(
          contentComposite, "Peak Temporary Register Pressure:", staticAnalysis.getTempRegisters());

      scrollComposite.setContent(contentComposite);
      scrollComposite.setExpandVertical(true);
      scrollComposite.setExpandHorizontal(true);
      scrollComposite.addListener(SWT.Resize, event -> {
        Rectangle scrollArea = scrollComposite.getClientArea();

        int currentNumColumns = gridLayout.numColumns;
        int numChildren = contentComposite.getChildren().length;
        Point tableSize = contentComposite.computeSize(SWT.DEFAULT, SWT.DEFAULT, true);

        if (tableSize.x < scrollArea.width) {
          while (tableSize.x < scrollArea.width && gridLayout.numColumns < numChildren) {
            gridLayout.numColumns += 2;
            tableSize = contentComposite.computeSize(SWT.DEFAULT, SWT.DEFAULT, true);
          }

          if (tableSize.x > scrollArea.width) {
            gridLayout.numColumns -= 2;
          }
        } else {
          while (tableSize.x > scrollArea.width && gridLayout.numColumns >= 4) {
            gridLayout.numColumns -= 2;
            tableSize = contentComposite.computeSize(SWT.DEFAULT, SWT.DEFAULT, true);
          }
        }

        if (gridLayout.numColumns != currentNumColumns) {
          scrollComposite.layout();
        }

        scrollComposite.setMinHeight(contentComposite.computeSize(scrollArea.width, SWT.DEFAULT).y);
      });
    }

    private static void createStaticAnalysisStat(Composite parent, String statText, int statData) {
      GridLayout statsLayout = new GridLayout(1, false);
      statsLayout.marginHeight = 5;
      statsLayout.marginWidth = 5;

      Composite keyComposite = withLayoutData(createComposite(parent, statsLayout, SWT.BORDER),
          new GridData(SWT.FILL, SWT.TOP, false, false));

      withLayoutData( createBoldLabel(keyComposite, statText),
          new GridData(SWT.RIGHT, SWT.CENTER, true, true));

      Composite valueComposite = withLayoutData(createComposite(parent, statsLayout, SWT.BORDER),
          new GridData(SWT.FILL, SWT.TOP, false, false));
      withLayoutData(createLabel(valueComposite, Integer.toString(statData)),
          new GridData(SWT.LEFT, SWT.CENTER, true, true));
    }

    private static boolean isKey(Event e, int stateMask, int keyCode) {
      return (e.stateMask & stateMask) == stateMask && e.keyCode == keyCode;
    }
  }
}
