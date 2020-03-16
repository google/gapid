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

import static com.google.gapid.util.Arrays.last;
import static com.google.gapid.util.Colors.BLACK;
import static com.google.gapid.util.Colors.DARK_LUMINANCE_THRESHOLD;
import static com.google.gapid.util.Colors.WHITE;
import static com.google.gapid.util.Colors.getLuminance;
import static com.google.gapid.util.Colors.rgb;
import static com.google.gapid.util.Loadable.MessageType.Error;
import static com.google.gapid.util.Loadable.MessageType.Info;
import static com.google.gapid.widgets.Widgets.createComposite;
import static com.google.gapid.widgets.Widgets.createGroup;
import static com.google.gapid.widgets.Widgets.createStandardTabFolder;
import static com.google.gapid.widgets.Widgets.createStandardTabItem;
import static com.google.gapid.widgets.Widgets.createTableViewer;
import static com.google.gapid.widgets.Widgets.createTreeColumn;
import static com.google.gapid.widgets.Widgets.createTreeViewer;
import static com.google.gapid.widgets.Widgets.disposeAllChildren;
import static com.google.gapid.widgets.Widgets.packColumns;
import static com.google.gapid.widgets.Widgets.scheduleIfNotDisposed;
import static com.google.gapid.widgets.Widgets.sorting;
import static java.util.logging.Level.FINE;

import com.google.common.collect.Lists;
import com.google.common.collect.Maps;
import com.google.common.primitives.UnsignedLongs;
import com.google.gapid.lang.glsl.GlslSourceConfiguration;
import com.google.gapid.models.Analytics.View;
import com.google.gapid.models.Capture;
import com.google.gapid.models.CommandStream;
import com.google.gapid.models.CommandStream.CommandIndex;
import com.google.gapid.models.Models;
import com.google.gapid.models.Resources;
import com.google.gapid.models.Settings;
import com.google.gapid.proto.core.pod.Pod;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.Service.ClientAction;
import com.google.gapid.proto.service.api.API;
import com.google.gapid.rpc.Rpc;
import com.google.gapid.rpc.RpcException;
import com.google.gapid.rpc.SingleInFlight;
import com.google.gapid.rpc.UiCallback;
import com.google.gapid.util.Events;
import com.google.gapid.util.Loadable;
import com.google.gapid.util.Messages;
import com.google.gapid.util.ProtoDebugTextFormat;
import com.google.gapid.widgets.LoadablePanel;
import com.google.gapid.widgets.SearchBox;
import com.google.gapid.widgets.Theme;
import com.google.gapid.widgets.Widgets;

import org.eclipse.jface.text.ITextOperationTarget;
import org.eclipse.jface.text.source.SourceViewer;
import org.eclipse.jface.viewers.ArrayContentProvider;
import org.eclipse.jface.viewers.ITreeContentProvider;
import org.eclipse.jface.viewers.LabelProvider;
import org.eclipse.jface.viewers.TableViewer;
import org.eclipse.jface.viewers.TreeViewer;
import org.eclipse.jface.viewers.Viewer;
import org.eclipse.jface.viewers.ViewerFilter;
import org.eclipse.swt.SWT;
import org.eclipse.swt.custom.ST;
import org.eclipse.swt.custom.SashForm;
import org.eclipse.swt.custom.StyledText;
import org.eclipse.swt.graphics.Image;
import org.eclipse.swt.graphics.ImageData;
import org.eclipse.swt.graphics.PaletteData;
import org.eclipse.swt.layout.FillLayout;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Button;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Event;
import org.eclipse.swt.widgets.Group;
import org.eclipse.swt.widgets.TabFolder;
import org.eclipse.swt.widgets.TreeItem;

import java.util.ArrayList;
import java.util.Collections;
import java.util.Comparator;
import java.util.List;
import java.util.Map;
import java.util.concurrent.ExecutionException;
import java.util.function.Consumer;
import java.util.logging.Logger;
import java.util.regex.Pattern;

/**
 * View allowing the inspection and editing of the shader resources.
 */
public class ShaderView extends Composite
    implements Tab, Capture.Listener, CommandStream.Listener, Resources.Listener {
  protected static final Logger LOG = Logger.getLogger(ShaderView.class.getName());

  protected final Models models;
  private final Widgets widgets;
  private final SingleInFlight shaderRpcController = new SingleInFlight();
  private final SingleInFlight programRpcController = new SingleInFlight();
  private final LoadablePanel<Composite> loading;
  private boolean uiBuiltWithPrograms;

  public ShaderView(Composite parent, Models models, Widgets widgets) {
    super(parent, SWT.NONE);
    this.models = models;
    this.widgets = widgets;

    setLayout(new FillLayout());

    loading = LoadablePanel.create(this, widgets,
        panel -> createComposite(panel, new FillLayout(SWT.VERTICAL)));
    updateUi(true);

    models.capture.addListener(this);
    models.commands.addListener(this);
    models.resources.addListener(this);
    addListener(SWT.Dispose, e -> {
      models.capture.removeListener(this);
      models.commands.removeListener(this);
      models.resources.removeListener(this);
    });
  }

  private Control createShaderTab(Composite parent) {
    ShaderPanel panel = new ShaderPanel(parent, models, widgets.theme, Type.shader((data, src) -> {
      API.Shader shader = (data == null) ? null : (API.Shader)data.resource;
      if (shader != null) {
        models.analytics.postInteraction(View.Shaders, ClientAction.Edit);
        models.resources.updateResource(data.info, API.ResourceData.newBuilder()
            .setShader(shader.toBuilder()
                .setSource(src))
            .build());
      }
    }));
    panel.addListener(SWT.Selection, e -> {
      models.analytics.postInteraction(View.Shaders, ClientAction.SelectShader);
      getShaderSource((Data)e.data, panel::setSource);
    });
    return panel;
  }

  private Control createProgramTab(Composite parent) {
    SashForm splitter = new SashForm(parent, SWT.VERTICAL);

    ShaderPanel panel = new ShaderPanel(splitter, models, widgets.theme, Type.program());
    Composite uniformsGroup = createGroup(splitter, "Uniforms");
    UniformsPanel uniforms = new UniformsPanel(uniformsGroup);

    splitter.setWeights(models.settings.getSplitterWeights(Settings.SplitterWeights.Uniforms));

    panel.addListener(SWT.Selection, e -> {
      models.analytics.postInteraction(View.Shaders, ClientAction.SelectProgram);
      getProgramSource((Data)e.data, program ->
          scheduleIfNotDisposed(uniforms, () -> uniforms.setUniforms(program)), panel::setSource);
    });
    splitter.addListener(SWT.Dispose, e ->
      models.settings.setSplitterWeights(Settings.SplitterWeights.Uniforms, splitter.getWeights()));
    return splitter;
  }

  private void getShaderSource(Data data, Consumer<ShaderPanel.Source[]> callback) {
    shaderRpcController.start().listen(models.resources.loadResource(data.info),
        new UiCallback<API.ResourceData, ShaderPanel.Source>(this, LOG) {
      @Override
      protected ShaderPanel.Source onRpcThread(Rpc.Result<API.ResourceData> result)
          throws RpcException, ExecutionException {
        API.Shader shader = result.get().getShader();
        data.resource = shader;
        return ShaderPanel.Source.of(shader);
      }

      @Override
      protected void onUiThread(ShaderPanel.Source result) {
        callback.accept(new ShaderPanel.Source[] { result });
      }
    });
  }

  private void getProgramSource(
      Data data, Consumer<API.Program> onProgramLoaded, Consumer<ShaderPanel.Source[]> callback) {
    programRpcController.start().listen(models.resources.loadResource(data.info),
        new UiCallback<API.ResourceData, ShaderPanel.Source[]>(this, LOG) {
      @Override
      protected ShaderPanel.Source[] onRpcThread(Rpc.Result<API.ResourceData> result)
          throws RpcException, ExecutionException {
        API.Program program = result.get().getProgram();
        data.resource = program;
        onProgramLoaded.accept(program);
        return ShaderPanel.Source.of(program);
      }

      @Override
      protected void onUiThread(ShaderPanel.Source[] result) {
        callback.accept(result);
      }
    });
  }

  @Override
  public Control getControl() {
    return this;
  }

  @Override
  public void reinitialize() {
    onCaptureLoadingStart(false);
    updateUi(false);
    updateLoading();
  }

  @Override
  public void onCaptureLoadingStart(boolean maintainState) {
    loading.showMessage(Info, Messages.LOADING_CAPTURE);
  }

  @Override
  public void onCaptureLoaded(Loadable.Message error) {
    if (error != null) {
      loading.showMessage(Error, Messages.CAPTURE_LOAD_FAILURE);
    }
  }

  @Override
  public void onCommandsLoaded() {
    if (!models.commands.isLoaded()) {
      loading.showMessage(Info, Messages.CAPTURE_LOAD_FAILURE);
    } else {
      updateLoading();
    }
  }

  @Override
  public void onCommandsSelected(CommandIndex path) {
    updateLoading();
  }

  @Override
  public void onResourcesLoaded() {
    if (!models.resources.isLoaded()) {
      loading.showMessage(Info, Messages.CAPTURE_LOAD_FAILURE);
    } else {
      updateUi(false);
      updateLoading();
    }
  }

  private void updateLoading() {
    if (models.commands.isLoaded() && models.resources.isLoaded()) {
      if (models.commands.getSelectedCommands() == null) {
        loading.showMessage(Info, Messages.SELECT_COMMAND);
      } else {
        loading.stopLoading();
      }
    }
  }

  private void updateUi(boolean force) {
    boolean hasPrograms = true;
    if (models.resources.isLoaded()) {
      hasPrograms = models.resources.getResources().stream()
          .filter(r -> r.getType() == API.ResourceType.ProgramResource)
          .findAny()
          .isPresent();
    } else if (!force) {
      return;
    }

    if (force || hasPrograms != uiBuiltWithPrograms) {
      LOG.log(FINE, "(Re-)creating the shader UI, force: {0}, programs: {1}",
          new Object[] { force, hasPrograms });
      uiBuiltWithPrograms = hasPrograms;
      disposeAllChildren(loading.getContents());

      if (hasPrograms) {
        TabFolder folder = createStandardTabFolder(loading.getContents());
        createStandardTabItem(folder, "Shaders", createShaderTab(folder));
        createStandardTabItem(folder, "Programs", createProgramTab(folder));
      } else {
        createShaderTab(loading.getContents());
      }
      loading.getContents().requestLayout();
    }
  }

  /**
   * Panel displaying the source code of a shader or program.
   */
  private static class ShaderPanel extends Composite
      implements Resources.Listener, CommandStream.Listener {
    private final Models models;
    private final Theme theme;
    protected final Type type;
    private final TreeViewer shaderViewer;
    private final Composite sourceComposite;
    private final Button pushButton;
    private ViewerFilter keywordSearchFilter;
    private SourceViewer shaderSourceViewer;
    private boolean lastUpdateContainedAllShaders = false;
    private List<Data> shaders = Collections.emptyList();

    public ShaderPanel(Composite parent, Models models, Theme theme, Type type) {
      super(parent, SWT.NONE);
      this.models = models;
      this.theme = theme;
      this.type = type;

      setLayout(new FillLayout(SWT.HORIZONTAL));

      SashForm splitter = new SashForm(this, SWT.HORIZONTAL);
      Composite treeViewerContainer= createComposite(splitter, new GridLayout(1, false), SWT.BORDER);
      Composite sourcesContainer = createComposite(splitter, new GridLayout(1, false));

      if (type.type == API.ResourceType.ShaderResource) {
        splitter.setWeights(models.settings.getSplitterWeights(Settings.SplitterWeights.Shaders));
        splitter.addListener(SWT.Dispose, e -> models.settings.setSplitterWeights(
            Settings.SplitterWeights.Shaders, splitter.getWeights()));
      } else if (type.type == API.ResourceType.ProgramResource) {
        splitter.setWeights(models.settings.getSplitterWeights(Settings.SplitterWeights.Programs));
        splitter.addListener(SWT.Dispose, e -> models.settings.setSplitterWeights(
            Settings.SplitterWeights.Programs, splitter.getWeights()));
      }

      SearchBox searchBox = new SearchBox(treeViewerContainer, true);
      searchBox.setLayoutData(new GridData(SWT.FILL, SWT.TOP, true, false));
      searchBox.addListener(Events.Search, e -> updateSearchFilter(e.text, (e.detail & Events.REGEX) != 0));

      shaderViewer = createShaderSelector(treeViewerContainer);
      shaderViewer.getTree().setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));

      sourceComposite = createComposite(sourcesContainer, new FillLayout(SWT.VERTICAL));
      sourceComposite.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));

      if (type.isEditable()) {
        pushButton = Widgets.createButton(sourcesContainer, "Push Changes",
            e -> type.updateShader(
                (Data)getLastSelection().getData(),
                shaderSourceViewer.getDocument().get()));
        pushButton.setLayoutData(new GridData(SWT.RIGHT, SWT.BOTTOM, false, false));
        pushButton.setEnabled(false);
      } else {
        pushButton = null;
      }

      models.commands.addListener(this);
      models.resources.addListener(this);
      addListener(SWT.Dispose, e -> {
        models.commands.removeListener(this);
        models.resources.removeListener(this);
      });

      updateShaders();
    }

    private TreeViewer createShaderSelector(Composite parent) {
      TreeViewer treeViewer = createTreeViewer(parent, SWT.FILL);
      treeViewer.getTree().setHeaderVisible(true);
      treeViewer.setContentProvider(createShaderContentProvider());
      treeViewer.setLabelProvider(new LabelProvider());
      treeViewer.getControl().addListener(SWT.Selection, e -> updateSelection());

      sorting(treeViewer,
          createTreeColumn(treeViewer, "ID", Data::getId,
              (d1, d2) -> UnsignedLongs.compare(d1.getSortId(), d2.getSortId())),
          createTreeColumn(treeViewer, "Label", Data::getLabel,
              Comparator.comparing(Data::getLabel)));
      return treeViewer;
    }

    private static ITreeContentProvider createShaderContentProvider() {
      return new ITreeContentProvider() {
        @SuppressWarnings("unchecked")
        @Override
        public Object[] getElements(Object inputElement) {
          return ((List<Data>) inputElement).toArray();
        }

        @Override
        public boolean hasChildren(Object element) {
          return false;
        }

        @Override
        public Object getParent(Object element) {
          return null;
        }

        @Override
        public Object[] getChildren(Object element) {
          return null;
        }
      };
    }

    private static ViewerFilter createSearchFilter(Pattern pattern) {
      return new ViewerFilter() {
        @Override
        public boolean select(Viewer viewer, Object parentElement, Object element) {
          return !(element instanceof Data) || ((Data)element).matches(pattern);
        }
      };
    }

    private SourceViewer createSourcePanel(Composite parent, Source source) {
      Group group = createGroup(parent, source.label);
      SourceViewer viewer =
          new SourceViewer(group, null, SWT.MULTI | SWT.H_SCROLL | SWT.V_SCROLL | SWT.BORDER);
      StyledText textWidget = viewer.getTextWidget();
      viewer.setEditable(type.isEditable());
      textWidget.setFont(theme.monoSpaceFont());
      textWidget.setKeyBinding(ST.SELECT_ALL, ST.SELECT_ALL);
      viewer.configure(new GlslSourceConfiguration(theme));
      viewer.setDocument(GlslSourceConfiguration.createDocument(source.source));
      textWidget.addListener(SWT.KeyDown, e -> {
        if (isKey(e, SWT.MOD1, 'z') && !isKey(e, SWT.MOD1 | SWT.SHIFT, 'z')) {
          viewer.doOperation(ITextOperationTarget.UNDO);
        } else if (isKey(e, SWT.MOD1, 'y') || isKey(e, SWT.MOD1 | SWT.SHIFT, 'z')) {
          viewer.doOperation(ITextOperationTarget.REDO);
        }
      });
      return viewer;
    }

    private static boolean isKey(Event e, int stateMask, int keyCode) {
      return (e.stateMask & stateMask) == stateMask && e.keyCode == keyCode;
    }

    public void clearSource() {
      shaderSourceViewer = null;
      disposeAllChildren(sourceComposite);
      if (pushButton != null) {
        pushButton.setEnabled(false);
      }
    }

    public void setSource(Source[] sources) {
      clearSource();
      SashForm sourceSash = new SashForm(sourceComposite, SWT.VERTICAL);
      for (Source source : sources) {
        shaderSourceViewer = createSourcePanel(sourceSash, source);
      }
      sourceSash.requestLayout();
      if (sources.length > 0 && pushButton != null) {
        pushButton.setEnabled(true);
      }
    }

    @Override
    public void onResourcesLoaded() {
      lastUpdateContainedAllShaders = false;
      updateShaders();
    }

    @Override
    public void onCommandsSelected(CommandIndex path) {
      updateShaders();
    }

    private TreeItem getLastSelection() {
      return last(shaderViewer.getTree().getSelection());
    }

    private void updateShaders() {
      if (models.resources.isLoaded() && models.commands.getSelectedCommands() != null) {
        Resources.ResourceList resources = models.resources.getResources(type.type);

        // If we previously had created the dropdown with all the shaders and didn't skip any
        // this time, the dropdown does not need to change.
        if (!lastUpdateContainedAllShaders || !resources.complete) {
          shaders = new ArrayList<Data>();
          if (!resources.isEmpty()) {
            resources.stream()
                .map(r -> new Data(r.resource))
                .forEach(shaders::add);
          }
          lastUpdateContainedAllShaders = resources.complete;

          // Because TreeViewer will dispose old TreeItems after refresh,
          // memorize and select with index, rather than object.
          TreeItem selection = getLastSelection();
          int selectionIndex = -1;
          if (selection != null) {
            selectionIndex = shaderViewer.getTree().indexOf(selection);
          }
          shaderViewer.setInput(shaders);
          packColumns(shaderViewer.getTree());

          if (selectionIndex >= 0 && selectionIndex < shaderViewer.getTree().getItemCount()) {
            selection = shaderViewer.getTree().getItem(selectionIndex);
            shaderViewer.getTree().setSelection(selection);
          }
        } else {
          for (Data data : shaders) {
            data.resource = null;
          }
        }

        updateSelection();
      }
    }

    private void updateSelection() {
      if (shaderViewer.getTree().getSelectionCount() < 1) {
        clearSource();
      } else {
        Event event = new Event();
        event.data = getLastSelection().getData();
        notifyListeners(SWT.Selection, event);
      }
    }

    private void updateSearchFilter(String text, boolean isRegex) {
      // Keyword Filter is dynamic.
      // Remove the previous one each time to avoid filter accumulation.
      if (keywordSearchFilter != null) {
        shaderViewer.removeFilter(keywordSearchFilter);
        keywordSearchFilter = null;
      }
      if (!text.isEmpty()) {
        Pattern pattern = SearchBox.getPattern(text, isRegex);
        keywordSearchFilter = createSearchFilter(pattern);
        shaderViewer.addFilter(keywordSearchFilter);
      }
    }

    public static class Source {
      private static final Source EMPTY_PROGRAM = new Source("Program",
          "// No shaders attached to this program at this point in the trace.");
      private static final String EMPTY_SHADER =
          "// No source attached to this shader at this point in the trace.";

      public final String label;
      public final String source;

      public Source(String label, String source) {
        this.label = label;
        this.source = source;
      }

      public static Source of(API.Shader shader) {
        return new Source(shader.getType() + " Shader",
            shader.getSource().isEmpty() ? EMPTY_SHADER : shader.getSource());
      }

      public static Source[] of(API.Program program) {
        if (program.getShadersCount() == 0) {
          return new Source[] { EMPTY_PROGRAM };
        }

        Source[] source = new Source[program.getShadersCount()];
        for (int i = 0; i < source.length; i++) {
          source[i] = of(program.getShaders(i));
        }
        return source;
      }
    }

    protected static interface UpdateShader {
      public void updateShader(Data data, String newSource);
    }
  }

  /**
   * Panel displaying the uniforms of the current shader program.
   */
  private static class UniformsPanel extends Composite {
    private final TableViewer table;
    private final Map<API.Uniform, Image> images = Maps.newIdentityHashMap();

    public UniformsPanel(Composite parent) {
      super(parent, SWT.NONE);
      setLayout(new FillLayout(SWT.VERTICAL));

      table = createTableViewer(
          this, SWT.BORDER | SWT.MULTI | SWT.FULL_SELECTION | SWT.H_SCROLL | SWT.V_SCROLL);
      table.setContentProvider(new ArrayContentProvider());

      Widgets.<API.Uniform>createTableColumn(table, "Location",
          uniform -> String.valueOf(uniform.getUniformLocation()));
      Widgets.<API.Uniform>createTableColumn(table, "Name", API.Uniform::getName);
      Widgets.<API.Uniform>createTableColumn(table, "Type",
          uniform -> String.valueOf(uniform.getType()));
      Widgets.<API.Uniform>createTableColumn(table, "Format",
          uniform -> String.valueOf(uniform.getFormat()));
      Widgets.<API.Uniform>createTableColumn(table, "Value", uniform -> {
        Pod.Value value = uniform.getValue().getPod(); // TODO
        switch (uniform.getType()) {
          case Int32: return String.valueOf(value.getSint32Array().getValList());
          case Uint32: return String.valueOf(value.getUint32Array().getValList());
          case Bool: return String.valueOf(value.getBoolArray().getValList());
          case Float: return String.valueOf(value.getFloat32Array().getValList());
          case Double: return String.valueOf(value.getFloat64Array().getValList());
          default: return ProtoDebugTextFormat.shortDebugString(value);
        }
      },this::getImage);
      packColumns(table.getTable());
      addListener(SWT.Dispose, e -> clearImages());
    }

    public void setUniforms(API.Program program) {
      List<API.Uniform> uniforms = Lists.newArrayList(program.getUniformsList());
      Collections.sort(uniforms, (a, b) -> a.getUniformLocation() - b.getUniformLocation());
      clearImages();
      table.setInput(uniforms);
      table.refresh();
      packColumns(table.getTable());
      table.getTable().requestLayout();
    }

    private Image getImage(API.Uniform uniform) {
      if (!images.containsKey(uniform)) {
        Image image = null;
        Pod.Value value = uniform.getValue().getPod(); // TODO
        switch (uniform.getType()) {
          case Float: {
            List<Float> values = value.getFloat32Array().getValList();
            if ((values.size() == 3 || values.size() == 4) &&
                isColorRange(values.get(0)) && isColorRange(values.get(1)) &&
                isColorRange(values.get(2))) {
              image = createImage(values.get(0), values.get(1), values.get(2));
            }
            break;
          }
          case Double: {
            List<Double> values = value.getFloat64Array().getValList();
            if ((values.size() == 3 || values.size() == 4) &&
                isColorRange(values.get(0)) && isColorRange(values.get(1)) &&
                isColorRange(values.get(2))) {
              image = createImage(values.get(0), values.get(1), values.get(2));
            }
            break;
          }
          default:
            // Not a color.
        }
        images.put(uniform, image);
      }
      return images.get(uniform);
    }

    private static boolean isColorRange(double v) {
      return v >= 0 && v <= 1;
    }

    private Image createImage(double r, double g, double b) {
      ImageData data = new ImageData(12, 12, 1, new PaletteData(
          (getLuminance(r, g, b) < DARK_LUMINANCE_THRESHOLD) ? WHITE : BLACK, rgb(r, g, b)), 1,
          new byte[] {
              0, 0, 127, -31, 127, -31, 127, -31, 127, -31, 127, -31, 127, -31, 127, -31, 127, -31,
              127, -31, 127, -31, 0, 0
          } /* Square of 1s with a border of 0s (and padding to 2 bytes per row) */);
      return new Image(getDisplay(), data);
    }

    private void clearImages() {
      for (Image image : images.values()) {
        if (image != null) {
          image.dispose();
        }
      }
      images.clear();
    }
  }

  /**
   * Distinguishes between shaders and programs.
   */
  private static class Type implements ShaderPanel.UpdateShader {
    public final API.ResourceType type;
    public final ShaderPanel.UpdateShader onSourceEdited;

    public Type(API.ResourceType type, ShaderPanel.UpdateShader onSourceEdited) {
      this.type = type;
      this.onSourceEdited = onSourceEdited;
    }

    public static Type shader(ShaderPanel.UpdateShader onSourceEdited) {
      return new Type(API.ResourceType.ShaderResource, onSourceEdited);
    }

    public static Type program() {
      return new Type(API.ResourceType.ProgramResource, null);
    }

    @Override
    public void updateShader(Data data, String newSource) {
      onSourceEdited.updateShader(data, newSource);
    }

    public boolean isEditable() {
      return onSourceEdited != null;
    }
  }

  /**
   * Shader or program data.
   */
  private static class Data {
    public final Service.Resource info;
    public Object resource;

    public Data(Service.Resource info) {
      this.info = info;
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

    public boolean matches(Pattern pattern) {
      return pattern.matcher(getId()).find() || pattern.matcher(getLabel()).find();
    }
  }
}
