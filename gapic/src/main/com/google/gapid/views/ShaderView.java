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
import static com.google.gapid.widgets.Widgets.createSeparator;
import static com.google.gapid.widgets.Widgets.createStandardTabFolder;
import static com.google.gapid.widgets.Widgets.createStandardTabItem;
import static com.google.gapid.widgets.Widgets.createTableViewer;
import static com.google.gapid.widgets.Widgets.createToolItem;
import static com.google.gapid.widgets.Widgets.packColumns;
import static com.google.gapid.widgets.Widgets.withIndents;
import static com.google.gapid.widgets.Widgets.withLayoutData;
import static com.google.gapid.widgets.Widgets.withMargin;

import com.google.gapid.lang.glsl.GlslSourceConfiguration;
import com.google.gapid.models.Analytics;
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
import com.google.gapid.util.Messages;
import com.google.gapid.widgets.LoadablePanel;
import com.google.gapid.widgets.Widgets;

import org.eclipse.jface.text.ITextOperationTarget;
import org.eclipse.jface.text.source.SourceViewer;
import org.eclipse.jface.viewers.IStructuredContentProvider;
import org.eclipse.jface.viewers.TableViewer;
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
import org.eclipse.swt.widgets.FileDialog;
import org.eclipse.swt.widgets.Group;
import org.eclipse.swt.widgets.TabFolder;
import org.eclipse.swt.widgets.TabItem;
import org.eclipse.swt.widgets.ToolBar;
import org.eclipse.swt.widgets.ToolItem;

import java.io.BufferedWriter;
import java.io.FileWriter;
import java.io.IOException;
import java.io.Writer;
import java.util.Optional;
import java.util.concurrent.ExecutionException;
import java.util.logging.Logger;

/**
 * View allowing the inspection and editing of a shader resource.
 */
public class ShaderView extends Composite
    implements Tab, Capture.Listener, Resources.Listener {
  protected static final Logger LOG = Logger.getLogger(ShaderView.class.getName());

  public ShaderView(Composite parent, Service.Resource shader, Models models, Widgets widgets) {
    super(parent, SWT.NONE);
    setLayout(new FillLayout());

    ShaderWidget view = createPinned(this, models, widgets);
    view.loadShader(shader);
  }

  @Override
  public Control getControl() {
    return this;
  }

  public static ShaderWidget create(Composite parent, Models models, Widgets widgets) {
    return new ShaderWidget(parent, false, true, models, widgets);
  }

  public static ShaderWidget createPinned(Composite parent, Models models, Widgets widgets) {
    return new ShaderWidget(parent, true, true, models, widgets);
  }

  public static ShaderWidget createReadOnly(Composite parent, Models models, Widgets widgets) {
    return new ShaderWidget(parent, false, false, models, widgets);
  }

  public static class ShaderWidget extends Composite {
    private static final String EMPTY_SHADER =
        "// No source attached to this shader at this point in the trace.";

    private final SingleInFlight rpcController = new SingleInFlight();
    private final boolean pinned;
    private final Models models;
    private final LoadablePanel<Composite> loading;
    private final Group shaderGroup;
    private final ToolItem pinItem;
    private final ToolItem saveItem;
    private final TabFolder tabFolder;
    private final Composite sourceContainer;
    private final Group statGroup;
    private final TableViewer statTable;
    private final GridData crossCompileGridData;
    private final SourceViewer spirvViewer;
    private final SourceViewer sourceViewer;
    private final Optional<Button> pushButton;
    private TabItem spirvTab;
    private TabItem sourceTab;
    private Service.Resource shaderResource = null;
    private API.Shader shaderMessage = null;

    protected ShaderWidget(
        Composite parent, boolean pinned, boolean allowEditing, Models models, Widgets widgets) {
      super(parent, SWT.NONE);
      this.pinned = pinned;
      this.models = models;

      setLayout(new FillLayout());

      loading = LoadablePanel.create(this, widgets,
          panel -> createComposite(panel, new GridLayout(1, false)));

      shaderGroup = withLayoutData(
          createGroup(loading.getContents(), "Shader", new GridLayout(1, false)),
          new GridData(SWT.FILL, SWT.FILL, true, true));

      ToolBar toolBar = withLayoutData(new ToolBar(shaderGroup, SWT.HORIZONTAL | SWT.FLAT),
          new GridData(SWT.FILL, SWT.TOP, true, false));
      if (pinned) {
        pinItem = createToolItem(
            toolBar, widgets.theme.pinned(), e -> { /* ignore */ }, "Pinned shader");
      } else {
        pinItem = createToolItem(toolBar, widgets.theme.pin(),
            e -> models.resources.pinShader(shaderResource), "Pin this shader");
      }
      createSeparator(toolBar);
      saveItem = createToolItem(toolBar, widgets.theme.save(), e -> save(), "Save shader to file");

      tabFolder = withLayoutData(createStandardTabFolder(shaderGroup),
          new GridData(SWT.FILL, SWT.FILL, true, true));

      spirvViewer =
          new SourceViewer(tabFolder, null, SWT.MULTI | SWT.H_SCROLL | SWT.V_SCROLL | SWT.BORDER);
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
      spirvTab = createStandardTabItem(tabFolder, "SPIR-V", spirvViewer.getControl());

      sourceContainer = createComposite(
          tabFolder, withMargin(new GridLayout(1, false), 0, 0), SWT.NONE);
      crossCompileGridData = withIndents(new GridData(SWT.LEFT, SWT.TOP, false, false), 0, 5);
      withLayoutData(
          createBoldLabel(sourceContainer, "Source code was decompiled using SPIRV-Cross"),
          crossCompileGridData);
      sourceViewer =
          new SourceViewer(sourceContainer, null, SWT.MULTI | SWT.H_SCROLL | SWT.V_SCROLL | SWT.BORDER);
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
      if (!allowEditing || !Experimental.enableUnstableFeatures(models.settings)) {
        pushButton = Optional.empty();
      } else {
        pushButton = Optional.of(createButton(shaderGroup, "Push Changes", e -> {
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

      statGroup = withLayoutData(createGroup(loading.getContents(), "Static Analysis"),
          new GridData(SWT.FILL, SWT.BOTTOM, true, false));
      statTable = createTableViewer(statGroup, SWT.BORDER | SWT.H_SCROLL | SWT.V_SCROLL);
      Widgets.<Object[]>createTableColumn(statTable, "Statistic", (val) -> String.valueOf(val[0]));
      Widgets.<Object[]>createTableColumn(statTable, "Value", (val) -> String.valueOf(val[1]));
      statTable.setContentProvider(new IStructuredContentProvider() {
        @Override
        public Object[] getElements(Object inputElement) {
          if (!(inputElement instanceof API.ShaderExtras.StaticAnalysis)) {
            return null;
          }

          API.ShaderExtras.StaticAnalysis analysis = (API.ShaderExtras.StaticAnalysis)inputElement;
          return new Object[] {
              new Object[] { "ALU Instructions", analysis.getAluInstructions() },
              new Object[] { "Texture Instructions", analysis.getTextureInstructions() },
              new Object[] { "Branch Instructions", analysis.getBranchInstructions() },
              new Object[] { "Peak Temporary Register Pressure", analysis.getTempRegisters() },
          };
        }
      });
      statTable.setInput(API.ShaderExtras.StaticAnalysis.getDefaultInstance());
      packColumns(statTable.getTable());

      clear();
    }

    public void clear() {
      shaderResource = null;
      shaderMessage = null;
      pushButton.ifPresent(b -> b.setEnabled(false));
      pinItem.setEnabled(false);
      saveItem.setEnabled(false);
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
      rpcController.start().listen(models.resources.loadResourceExtras(resource),
          new UiCallback<API.ResourceExtras, API.ShaderExtras>(this, LOG) {
        @Override
        protected API.ShaderExtras onRpcThread(Rpc.Result<API.ResourceExtras> result)
            throws RpcException, ExecutionException {
          return result.get().getShaderExtras();
        }

        @Override
        protected void onUiThread(API.ShaderExtras result) {
          statTable.setInput(result.getStaticAnalysis());
        }
      });
      loading.stopLoading();

      shaderResource = resource;
      shaderMessage = shader;

      pushButton.ifPresent(b -> b.setEnabled(shaderResource != null));
      pinItem.setEnabled(!pinned && shaderResource != null);
      saveItem.setEnabled(shaderMessage != null);
      shaderGroup.setText(getLabel(resource));
      String spirvSource = shaderMessage.getSpirvSource();
      if (spirvSource.isEmpty()) {
        spirvSource = EMPTY_SHADER;
      }
      spirvViewer.setDocument(GlslSourceConfiguration.createDocument(spirvSource));
      String source = shaderMessage.getSource();
      if (!source.isEmpty()) {
        sourceViewer.setDocument(GlslSourceConfiguration.createDocument(source));
        crossCompileGridData.exclude = !shaderMessage.getCrossCompiled();
        if (sourceTab == null) {
          sourceTab =
              createStandardTabItem(tabFolder, shaderMessage.getSourceLanguage(), sourceContainer);
        } else {
          sourceTab.setText(shaderMessage.getSourceLanguage());
        }
      } else if (sourceTab != null) {
        tabFolder.setSelection(spirvTab);
        sourceTab.dispose();
        sourceTab = null;
      }
    }

    private void save() {
      int selection = tabFolder.getSelectionIndex();

      models.analytics.postInteraction(Analytics.View.ShaderView, ClientAction.Save);
      FileDialog dialog = new FileDialog(getShell(), SWT.SAVE);
      dialog.setText("Save shader to...");

      dialog.setFilterNames(new String[] { "Shaders" });
      dialog.setFilterExtensions(new String[] {
          (selection <= 0) ? "*.spirv" : "*." + shaderMessage.getSourceLanguage().toLowerCase(),
      });
      dialog.setOverwrite(true);
      String path = dialog.open();
      if (path != null) {
        try (Writer out = new BufferedWriter(new FileWriter(path))) {
          out.write(selection <= 0 ? shaderMessage.getSpirvSource() : shaderMessage.getSource());
        } catch (IOException e) {
          SWT.error(SWT.ERROR_IO, e);
        }
      }
    }

    private static String getLabel(Service.Resource resource) {
      if (resource == null) {
        return "Shader";
      }
      String label = resource.getHandle();
      if (!resource.getLabel().isEmpty()) {
        label += " " + resource.getLabel();
      }
      return label;
    }

    private static boolean isKey(Event e, int stateMask, int keyCode) {
      return (e.stateMask & stateMask) == stateMask && e.keyCode == keyCode;
    }
  }
}
