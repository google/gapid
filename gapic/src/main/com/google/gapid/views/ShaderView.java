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

import static com.google.gapid.util.Colors.BLACK;
import static com.google.gapid.util.Colors.DARK_LUMINANCE_THRESHOLD;
import static com.google.gapid.util.Colors.WHITE;
import static com.google.gapid.util.Colors.getLuminance;
import static com.google.gapid.util.Colors.rgb;
import static com.google.gapid.util.Loadable.MessageType.Error;
import static com.google.gapid.util.Loadable.MessageType.Info;
import static com.google.gapid.util.Paths.resourceAfter;
import static com.google.gapid.widgets.Widgets.createComposite;
import static com.google.gapid.widgets.Widgets.createDropDownViewer;
import static com.google.gapid.widgets.Widgets.createGroup;
import static com.google.gapid.widgets.Widgets.createStandardTabFolder;
import static com.google.gapid.widgets.Widgets.createStandardTabItem;
import static com.google.gapid.widgets.Widgets.createTableViewer;
import static com.google.gapid.widgets.Widgets.disposeAllChildren;
import static com.google.gapid.widgets.Widgets.packColumns;
import static com.google.gapid.widgets.Widgets.scheduleIfNotDisposed;
import static java.util.logging.Level.FINE;

import com.google.common.collect.Lists;
import com.google.common.collect.Maps;
import com.google.gapid.lang.glsl.GlslSourceConfiguration;
import com.google.gapid.models.AtomStream;
import com.google.gapid.models.AtomStream.AtomIndex;
import com.google.gapid.models.Capture;
import com.google.gapid.models.Models;
import com.google.gapid.models.Resources;
import com.google.gapid.proto.core.pod.Pod;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.api.API;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.rpc.Rpc;
import com.google.gapid.rpc.RpcException;
import com.google.gapid.rpc.SingleInFlight;
import com.google.gapid.rpc.UiCallback;
import com.google.gapid.server.Client;
import com.google.gapid.util.Loadable;
import com.google.gapid.util.Messages;
import com.google.gapid.util.Paths;
import com.google.gapid.util.ProtoDebugTextFormat;
import com.google.gapid.widgets.LoadablePanel;
import com.google.gapid.widgets.Theme;
import com.google.gapid.widgets.Widgets;

import org.eclipse.jface.text.ITextOperationTarget;
import org.eclipse.jface.text.source.SourceViewer;
import org.eclipse.jface.viewers.ArrayContentProvider;
import org.eclipse.jface.viewers.ComboViewer;
import org.eclipse.jface.viewers.LabelProvider;
import org.eclipse.jface.viewers.TableViewer;
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

import java.util.Collections;
import java.util.List;
import java.util.Map;
import java.util.concurrent.ExecutionException;
import java.util.function.Consumer;
import java.util.logging.Logger;

/**
 * View allowing the inspection and editing of the shader resources.
 */
public class ShaderView extends Composite
    implements Tab, Capture.Listener, AtomStream.Listener, Resources.Listener {
  protected static final Logger LOG = Logger.getLogger(ShaderView.class.getName());

  private final Client client;
  protected final Models models;
  private final Widgets widgets;
  private final SingleInFlight shaderRpcController = new SingleInFlight();
  private final SingleInFlight programRpcController = new SingleInFlight();
  private final LoadablePanel<Composite> loading;
  private boolean uiBuiltWithPrograms;

  public ShaderView(Composite parent, Client client, Models models, Widgets widgets) {
    super(parent, SWT.NONE);
    this.client = client;
    this.models = models;
    this.widgets = widgets;

    setLayout(new FillLayout());

    loading = LoadablePanel.create(this, widgets,
        panel -> createComposite(panel, new FillLayout(SWT.VERTICAL)));
    updateUi(true);

    models.capture.addListener(this);
    models.atoms.addListener(this);
    models.resources.addListener(this);
    addListener(SWT.Dispose, e -> {
      models.capture.removeListener(this);
      models.atoms.removeListener(this);
      models.resources.removeListener(this);
    });
  }

  private Control createShaderTab(Composite parent) {
    ShaderPanel panel = new ShaderPanel(parent, models, widgets.theme, Type.shader((data, src) -> {
      API.Shader shader = (data == null) ? null : (API.Shader)data.resource;
      if (shader != null) {
        Service.Value value = Service.Value.newBuilder()
            .setResourceData(API.ResourceData.newBuilder()
                .setShader(shader.toBuilder()
                    .setSource(src)))
            .build();
        Rpc.listen(client.set(data.getPath(models.atoms), value),
            new UiCallback<Path.Any, Path.Capture>(this, LOG) {
          @Override
          protected Path.Capture onRpcThread(Rpc.Result<Path.Any> result)
              throws RpcException, ExecutionException {
            // TODO this should probably be able to handle any path.
            return result.get().getResourceData().getAfter().getCapture();
          }

          @Override
          protected void onUiThread(Path.Capture result) {
            models.capture.updateCapture(result, null);
          }
        });
      }
    }));
    panel.addListener(SWT.Selection, e -> getShaderSource((Data)e.data, panel::setSource));
    return panel;
  }

  private Control createProgramTab(Composite parent) {
    SashForm splitter = new SashForm(parent, SWT.VERTICAL);

    ShaderPanel panel = new ShaderPanel(splitter, models, widgets.theme, Type.program());
    Composite uniformsGroup = createGroup(splitter, "Uniforms");
    UniformsPanel uniforms = new UniformsPanel(uniformsGroup);

    splitter.setWeights(models.settings.shaderSplitterWeights);

    panel.addListener(SWT.Selection, e -> getProgramSource((Data)e.data,
        program -> scheduleIfNotDisposed(uniforms, () -> uniforms.setUniforms(program)),
        panel::setSource));
    splitter.addListener(
        SWT.Dispose, e -> models.settings.shaderSplitterWeights = splitter.getWeights());
    return splitter;
  }

  private void getShaderSource(Data data, Consumer<ShaderPanel.Source[]> callback) {
    shaderRpcController.start().listen(client.get(data.getPath(models.atoms)),
        new UiCallback<Service.Value, ShaderPanel.Source>(this, LOG) {
      @Override
      protected ShaderPanel.Source onRpcThread(Rpc.Result<Service.Value> result)
          throws RpcException, ExecutionException {
        API.Shader shader = result.get().getResourceData().getShader();
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
    programRpcController.start().listen(client.get(data.getPath(models.atoms)),
        new UiCallback<Service.Value, ShaderPanel.Source[]>(this, LOG) {
      @Override
      protected ShaderPanel.Source[] onRpcThread(Rpc.Result<Service.Value> result)
          throws RpcException, ExecutionException {
        API.Program program = result.get().getResourceData().getProgram();
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
  public void onAtomsLoaded() {
    if (!models.atoms.isLoaded()) {
      loading.showMessage(Info, Messages.CAPTURE_LOAD_FAILURE);
    } else {
      updateLoading();
    }
  }

  @Override
  public void onAtomsSelected(AtomIndex path) {
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
    if (models.atoms.isLoaded() && models.resources.isLoaded()) {
      if (models.atoms.getSelectedAtoms() == null) {
        loading.showMessage(Info, Messages.SELECT_ATOM);
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
      implements Resources.Listener, AtomStream.Listener {
    private final Models models;
    private final Theme theme;
    protected final Type type;
    private final ComboViewer shaderCombo;
    private final Composite sourceComposite;
    private final Button pushButton;
    private SourceViewer shaderSourceViewer;
    private boolean lastUpdateContainedAllShaders = false;
    private List<Data> shaders = Collections.emptyList();

    public ShaderPanel(Composite parent, Models models, Theme theme, Type type) {
      super(parent, SWT.NONE);
      this.models = models;
      this.theme = theme;
      this.type = type;

      setLayout(new GridLayout(1, false));
      shaderCombo = createShaderSelector();
      sourceComposite = createComposite(this, new FillLayout(SWT.VERTICAL));

      shaderCombo.getCombo().setLayoutData(new GridData(SWT.FILL, SWT.TOP, true, false));
      sourceComposite.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));

      if (type.isEditable()) {
        pushButton = Widgets.createButton(this, "Push Changes",
            e -> type.updateShader(
                (Data)shaderCombo.getElementAt(shaderCombo.getCombo().getSelectionIndex()),
                shaderSourceViewer.getDocument().get()));
        pushButton.setLayoutData(new GridData(SWT.RIGHT, SWT.BOTTOM, false, false));
        pushButton.setEnabled(false);
      } else {
        pushButton = null;
      }

      models.atoms.addListener(this);
      models.resources.addListener(this);
      addListener(SWT.Dispose, e -> {
        models.atoms.removeListener(this);
        models.resources.removeListener(this);
      });

      shaderCombo.getCombo().addListener(SWT.Selection, e -> updateSelection());
      updateShaders();
    }

    private ComboViewer createShaderSelector() {
      ComboViewer combo = createDropDownViewer(this);
      combo.setContentProvider(ArrayContentProvider.getInstance());
      combo.setLabelProvider(new LabelProvider());
      return combo;
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
        if (isKey(e, SWT.MOD1, 'z')) {
          viewer.doOperation(ITextOperationTarget.UNDO);
        } else if (isKey(e, SWT.MOD1, 'y') || isKey(e, SWT.MOD1 | SWT.SHIFT, 'z')) {
          viewer.doOperation(ITextOperationTarget.REDO);
        }
      });
      return viewer;
    }

    private static boolean isKey(Event e, int stateMask, int keyCode) {
      return (e.stateMask & stateMask) == e.stateMask && e.keyCode == keyCode;
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
      updateShaders();
    }

    @Override
    public void onAtomsSelected(AtomIndex path) {
      updateShaders();
    }

    private void updateShaders() {
      if (models.resources.isLoaded() && models.atoms.getSelectedAtoms() != null) {
        List<Data> newShaders = Lists.newArrayList();
        AtomIndex range = models.atoms.getSelectedAtoms();
        boolean skippedAnyShaders = false;
        for (Service.ResourcesByType bundle : models.resources.getResources()) {
          if (bundle.getType() == type.type) {
            for (Service.Resource info : bundle.getResourcesList()) {
              if (Paths.compare(firstAccess(info), range.getCommand()) <= 0) {
                if (newShaders.isEmpty()) {
                  newShaders.add(new Data(null) {
                    @Override
                    public String toString() {
                      return type.selectMessage;
                    }
                  });
                }
                newShaders.add(new Data(info));
              } else {
                skippedAnyShaders = true;
              }
            }
          }
        }

        if (!lastUpdateContainedAllShaders || skippedAnyShaders) {
          shaders = newShaders;
          lastUpdateContainedAllShaders = !skippedAnyShaders;

          int selection = shaderCombo.getCombo().getSelectionIndex();
          shaderCombo.setInput(shaders);
          shaderCombo.refresh();

          if (selection >= 0 && selection < shaders.size()) {
            shaderCombo.getCombo().select(selection);
          } else if (!shaders.isEmpty()) {
            shaderCombo.getCombo().select(0);
          }
        } else {
          for (Data data : shaders) {
            data.resource = null;
          }
        }

        updateSelection();
      }
    }

    private static Path.Command firstAccess(Service.Resource info) {
      return (info.getAccessesCount() == 0) ? null : info.getAccesses(0);
    }

    private void updateSelection() {
      int index = shaderCombo.getCombo().getSelectionIndex();
      if (index <= 0) {
        // Ignore the null item selection.
        clearSource();
      } else {
        Event event = new Event();
        event.data = shaderCombo.getElementAt(index);
        notifyListeners(SWT.Selection, event);
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
    public final String selectMessage;
    public final ShaderPanel.UpdateShader onSourceEdited;

    public Type(
        API.ResourceType type, String selectMessage, ShaderPanel.UpdateShader onSourceEdited) {
      this.type = type;
      this.selectMessage = selectMessage;
      this.onSourceEdited = onSourceEdited;
    }

    public static Type shader(ShaderPanel.UpdateShader onSourceEdited) {
      return new Type(API.ResourceType.ShaderResource, Messages.SELECT_SHADER, onSourceEdited);
    }

    public static Type program() {
      return new Type(API.ResourceType.ProgramResource, Messages.SELECT_PROGRAM, null);
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

    public Path.Any getPath(AtomStream atoms) {
      return resourceAfter(atoms.getSelectedAtoms(), info.getId());
    }

    @Override
    public String toString() {
      String handle = info.getHandle();
      String label = info.getLabel();
      return (label.isEmpty()) ? handle : handle + " [" + label + "]";
    }
  }
}
