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

import static com.google.gapid.glviewer.Geometry.isPolygon;
import static com.google.gapid.util.Loadable.MessageType.Error;
import static com.google.gapid.util.Loadable.MessageType.Info;
import static com.google.gapid.util.Paths.meshAfter;
import static com.google.gapid.views.ErrorDialog.showErrorDialog;
import static com.google.gapid.widgets.Widgets.createComposite;
import static com.google.gapid.widgets.Widgets.createLabel;
import static com.google.gapid.widgets.Widgets.createSeparator;
import static com.google.gapid.widgets.Widgets.createToggleToolItem;
import static com.google.gapid.widgets.Widgets.createToolItem;
import static com.google.gapid.widgets.Widgets.exclusiveSelection;
import static com.google.gapid.widgets.Widgets.withMargin;
import static java.util.Collections.emptyList;
import static java.util.logging.Level.WARNING;

import com.google.common.collect.Lists;
import com.google.common.collect.Maps;
import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.common.util.concurrent.Uninterruptibles;
import com.google.gapid.glviewer.Geometry;
import com.google.gapid.glviewer.GeometryScene;
import com.google.gapid.glviewer.camera.CylindricalCameraModel;
import com.google.gapid.glviewer.camera.IsoSurfaceCameraModel;
import com.google.gapid.glviewer.geo.Model;
import com.google.gapid.glviewer.geo.ObjWriter;
import com.google.gapid.models.Analytics.View;
import com.google.gapid.models.Capture;
import com.google.gapid.models.CommandStream;
import com.google.gapid.models.CommandStream.CommandIndex;
import com.google.gapid.models.Models;
import com.google.gapid.models.Strings;
import com.google.gapid.proto.service.Service.ClientAction;
import com.google.gapid.proto.service.api.API;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.proto.service.vertex.Vertex;
import com.google.gapid.proto.stringtable.Stringtable;
import com.google.gapid.rpc.Rpc;
import com.google.gapid.rpc.RpcException;
import com.google.gapid.rpc.SingleInFlight;
import com.google.gapid.rpc.UiErrorCallback;
import com.google.gapid.server.Client;
import com.google.gapid.server.Client.DataUnavailableException;
import com.google.gapid.util.Loadable;
import com.google.gapid.util.Messages;
import com.google.gapid.util.Paths;
import com.google.gapid.util.Streams;
import com.google.gapid.widgets.DialogBase;
import com.google.gapid.widgets.LoadablePanel;
import com.google.gapid.widgets.ScenePanel;
import com.google.gapid.widgets.Theme;
import com.google.gapid.widgets.Widgets;
import com.google.protobuf.ByteString;

import org.eclipse.jface.dialogs.IDialogConstants;
import org.eclipse.jface.window.Window;
import org.eclipse.swt.SWT;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Combo;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.FileDialog;
import org.eclipse.swt.widgets.Label;
import org.eclipse.swt.widgets.Shell;
import org.eclipse.swt.widgets.ToolBar;
import org.eclipse.swt.widgets.ToolItem;

import java.io.FileWriter;
import java.io.IOException;
import java.io.Writer;
import java.nio.ByteBuffer;
import java.nio.ByteOrder;
import java.nio.FloatBuffer;
import java.util.Arrays;
import java.util.List;
import java.util.Map;
import java.util.concurrent.ExecutionException;
import java.util.logging.Logger;

/**
 * View that displays the 3D geometry of the last draw call within the current selection.
 */
public class GeometryView extends Composite implements Tab, Capture.Listener, CommandStream.Listener {
  private static final Logger LOG = Logger.getLogger(GeometryView.class.getName());
  private static final Vertex.Semantic POSITION_0 = Vertex.Semantic.newBuilder()
      .setType(Vertex.Semantic.Type.Position)
      .setIndex(0)
      .build();
  private static final Vertex.Semantic NORMAL_0 = Vertex.Semantic.newBuilder()
      .setType(Vertex.Semantic.Type.Normal)
      .setIndex(0)
      .build();
  protected static final Vertex.BufferFormat POS_NORM_XYZ_F32 = Vertex.BufferFormat.newBuilder()
      .addStreams(Vertex.StreamFormat.newBuilder()
          .setSemantic(POSITION_0)
          .setFormat(Streams.FMT_XYZ_F32))
      .addStreams(Vertex.StreamFormat.newBuilder()
          .setSemantic(NORMAL_0)
          .setFormat(Streams.FMT_XYZ_F32))
      .build();
  private static final Stringtable.Msg NO_MESH_ERR = Strings.create("ERR_MESH_NOT_AVAILABLE");

  private final Client client;
  private final Models models;
  private final SingleInFlight rpcController = new SingleInFlight();
  protected final LoadablePanel<ScenePanel<GeometryScene.Data>> loading;
  protected final ScenePanel<GeometryScene.Data> canvas;
  private final Label statusBar;
  protected GeometryScene.Data data = GeometryScene.Data.DEFAULTS;
  private final IsoSurfaceCameraModel camera =
      new IsoSurfaceCameraModel(new CylindricalCameraModel());
  private ToolItem originalModelItem, facetedModelItem;
  private VertexSemantics vertexSemantics;
  private Model originalModel, facetedModel;
  private Geometry.DisplayMode displayMode = Geometry.DisplayMode.TRIANGLES;
  private Geometry.DisplayMode desiredDisplayMode = Geometry.DisplayMode.TRIANGLES;
  private ToolItem renderAsTriangles, renderAsLines, renderAsPoints;
  private ToolItem configureItem, saveItem;

  public GeometryView(Composite parent, Client client, Models models, Widgets widgets) {
    super(parent, SWT.NONE);
    this.client = client;
    this.models = models;

    setLayout(new GridLayout(2, false));

    ToolBar toolbar = createToolbar(widgets.theme);
    Composite content = createComposite(this, new GridLayout(1, false));
    GeometryScene scene = new GeometryScene(camera);
    loading = LoadablePanel.create(content, widgets,
        panel -> new ScenePanel<GeometryScene.Data>(panel, scene));
    canvas = loading.getContents();
    scene.bindCamera(canvas);
    statusBar = createLabel(content, "");

    toolbar.setLayoutData(new GridData(SWT.LEFT, SWT.FILL, false, true));
    content.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));
    loading.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));
    statusBar.setLayoutData(new GridData(SWT.FILL, SWT.BOTTOM, true, false));

    models.capture.addListener(this);
    models.commands.addListener(this);
    addListener(SWT.Dispose, e -> {
      models.capture.removeListener(this);
      models.commands.removeListener(this);
    });

    originalModelItem.setEnabled(false);
    facetedModelItem.setEnabled(false);
    configureItem.setEnabled(false);
    saveItem.setEnabled(false);

    checkOpenGLAndShowMessage(null, null);
  }

  private ToolBar createToolbar(Theme theme) {
    ToolBar bar = new ToolBar(this, SWT.VERTICAL | SWT.FLAT);
    createToolItem(bar, theme.yUp(), e -> {
      boolean zUp = !data.geometry.zUp;
      models.analytics.postInteraction(View.Geometry, zUp ? ClientAction.ZUp : ClientAction.YUp);
      ((ToolItem)e.widget).setImage(zUp ? theme.zUp() : theme.yUp());
      setSceneData(data.withGeometry(new Geometry(data.geometry.model, zUp), displayMode));
    }, "Toggle Y/Z up");
    createToolItem(bar, theme.windingCCW(), e -> {
      boolean cw = data.winding == GeometryScene.Winding.CCW; // cw represent the new value.
      models.analytics.postInteraction(
          View.Geometry, cw ? ClientAction.WindingCW : ClientAction.WindingCCW);
      ((ToolItem)e.widget).setImage(cw ? theme.windingCW() : theme.windingCCW());
      setSceneData(data.withToggledWinding());
    }, "Toggle triangle winding");
    configureItem = createToolItem(bar, theme.settings(), e -> {
      models.analytics.postInteraction(View.Geometry, ClientAction.VertexSemantics);
      if (new SemanticsDialog(getShell(), theme, vertexSemantics).open() == Window.OK) {
        updateModels(false);
      }
    }, "Configure vertex attributes");
    createSeparator(bar);
    exclusiveSelection(
        renderAsTriangles = createToggleToolItem(bar, theme.wireframeNone(), e -> {
          models.analytics.postInteraction(View.Geometry, ClientAction.Triangles);
          desiredDisplayMode = displayMode = Geometry.DisplayMode.TRIANGLES;
          updateRenderable();
        }, "Render as triangles"),
        renderAsLines = createToggleToolItem(bar, theme.wireframeAll(), e -> {
          models.analytics.postInteraction(View.Geometry, ClientAction.Wireframe);
          desiredDisplayMode = displayMode = Geometry.DisplayMode.LINES;
          updateRenderable();
        }, "Render as lines"),
        renderAsPoints = createToggleToolItem(bar, theme.pointCloud(), e -> {
          models.analytics.postInteraction(View.Geometry, ClientAction.Points);
          desiredDisplayMode = displayMode = Geometry.DisplayMode.POINTS;
          updateRenderable();
        }, "Render as points"));
    createSeparator(bar);
    exclusiveSelection(
        originalModelItem = createToggleToolItem(bar, theme.smooth(), e -> {
          models.analytics.postInteraction(View.Geometry, ClientAction.Smooth);
          setModel(originalModel);
        }, "Use original normals"),
        facetedModelItem = createToggleToolItem(bar, theme.faceted(), e -> {
          models.analytics.postInteraction(View.Geometry, ClientAction.Faceted);
          setModel(facetedModel);
        }, "Use computed per-face normals"));
    createSeparator(bar);
    createToolItem(bar, theme.cullingDisabled(), e -> {
      boolean cull = data.culling == GeometryScene.Culling.OFF; // cull represents the new value.
      models.analytics.postInteraction(
          View.Geometry, cull ? ClientAction.CullOn : ClientAction.CullOff);
      ((ToolItem)e.widget).setImage(cull ? theme.cullingEnabled() : theme.cullingDisabled());
      setSceneData(data.withToggledCulling());
    }, "Toggle backface culling");
    createSeparator(bar);
    exclusiveSelection(
        createToggleToolItem(bar, theme.lit(), e -> {
          models.analytics.postInteraction(View.Geometry, ClientAction.Shaded);
          setSceneData(data.withShading(GeometryScene.Shading.LIT));
        }, "Render with a lit shader"),
        createToggleToolItem(bar, theme.flat(), e -> {
          models.analytics.postInteraction(View.Geometry, ClientAction.Flat);
          setSceneData(data.withShading(GeometryScene.Shading.FLAT));
        }, "Render with a flat shader (silhouette)"),
        createToggleToolItem(bar, theme.normals(), e -> {
          models.analytics.postInteraction(View.Geometry, ClientAction.Normals);
          setSceneData(data.withShading(GeometryScene.Shading.NORMALS));
        }, "Render normals"));
    createSeparator(bar);
    saveItem = createToolItem(bar, theme.save(), e -> {
      models.analytics.postInteraction(View.Geometry, ClientAction.Save);
      FileDialog dialog = new FileDialog(getShell(), SWT.SAVE);
      dialog.setText("Save model to...");
      dialog.setFilterNames(new String[] { "OBJ Files (*.obj)" });
      dialog.setFilterExtensions(new String[] { "*.obj" });
      dialog.setOverwrite(true);
      String objFile = dialog.open();
      if (objFile != null) {
        try (Writer out = new FileWriter(objFile)) {
          ObjWriter.write(out, originalModelItem.getSelection() ? originalModel : facetedModel);
        } catch (IOException ex) {
          LOG.log(WARNING, "Failed to save model as OBJ", e);
          showErrorDialog(getShell(), models.analytics,
              "Failed to save model as OBJ:\n  " + ex.getMessage(), ex);
        }
      }
    }, "Save model as OBJ");
    return bar;
  }

  @Override
  public Control getControl() {
    return this;
  }

  @Override
  public void reinitialize() {
    updateModels(false);
  }

  @Override
  public void onCaptureLoadingStart(boolean maintainState) {
    updateModels(true);
  }

  @Override
  public void onCaptureLoaded(Loadable.Message error) {
    if (error != null) {
      checkOpenGLAndShowMessage(Error, Messages.CAPTURE_LOAD_FAILURE);
    }
  }

  @Override
  public void onCommandsLoaded() {
    vertexSemantics = null;
    updateModels(false);
  }

  @Override
  public void onCommandsSelected(CommandIndex range) {
    vertexSemantics = null;
    updateModels(false);
  }

  private void updateModels(boolean assumeLoading) {
    statusBar.setText("");
    if (!assumeLoading && models.commands.isLoaded()) {
      CommandIndex command = models.commands.getSelectedCommands();
      if (command == null) {
        checkOpenGLAndShowMessage(Info, Messages.SELECT_DRAW_CALL);
        configureItem.setEnabled(false);
        saveItem.setEnabled(false);
      } else {
        fetchMeshes(command);
      }
    } else {
      checkOpenGLAndShowMessage(Info, Messages.LOADING_CAPTURE);
      configureItem.setEnabled(false);
      saveItem.setEnabled(false);
    }
  }

  private void fetchMeshes(CommandIndex command) {
    if (!canvas.isOpenGL()) {
      return;
    }

    loading.startLoading();

    rpcController.start().listen(Futures.transformAsync(fetchMeshMetadata(command, vertexSemantics),
        semantics -> {
          ListenableFuture<Model> originalFuture = fetchModel(
              meshAfter(command, semantics.getOptions().build(), POS_NORM_XYZ_F32));
          ListenableFuture<Model> facetedFuture = fetchModel(meshAfter(
              command, semantics.getOptions().setFaceted(true).build(), POS_NORM_XYZ_F32));
          return Futures.transform(Futures.successfulAsList(originalFuture, facetedFuture),
              modelList -> new ModelLoadResult(semantics, modelList, originalFuture));
        }), new UiErrorCallback<ModelLoadResult, ModelLoadResult, String>(this, LOG) {
      @Override
      protected ResultOrError<ModelLoadResult, String> onRpcThread(Rpc.Result<ModelLoadResult> result)
          throws RpcException, ExecutionException {
        ModelLoadResult loadResult = result.get();
        List<Model> modelList = loadResult.models;
        if (modelList.get(0) == null && modelList.get(1) == null) {
          // Both failed, so get the error from the original model's call.
          try {
            Uninterruptibles.getUninterruptibly(loadResult.originalFuture);
          } catch (ExecutionException e) {
            if (e.getCause() instanceof DataUnavailableException) {
              // TODO: don't assume that it's because of not selecting a draw call.
              return success(loadResult.withoutModels());
            } else {
              throw e;
            }
          }
          // Should not get here, the future cannot both fail and succeed.
          throw new AssertionError("Future both failed and succeeded");
        } else if (modelList.get(0) != null && modelList.get(0).getNormals() == null) {
          // The original model has no normals, but the geometry doesn't support faceted normals.
          return success(loadResult.clearNthModel(0));
        } else if (modelList.get(0) != null && !isPolygon(modelList.get(0).getPrimitive())) {
          // TODO: if gapis returns an error for the faceted request, this is not needed.
          return success(loadResult.clearNthModel(1));
        } else {
          return success(loadResult);
        }
      }

      @Override
      protected void onUiThreadSuccess(ModelLoadResult result) {
        update(result.semantics, result.models);
      }

      @Override
      protected void onUiThreadError(String error) {
        loading.showMessage(Error, error);
      }
    });
  }

  private ListenableFuture<VertexSemantics> fetchMeshMetadata(
      CommandIndex command, VertexSemantics currentSemantics) {
    return Futures.transform(client.get(meshAfter(command, Paths.NODATA_MESH_OPTIONS)),
        value -> new VertexSemantics(value.getMesh(), currentSemantics));
  }

  private ListenableFuture<Model> fetchModel(Path.Any path) {
    return Futures.transformAsync(client.get(path), value -> fetchModel(value.getMesh()));
  }

  private static ListenableFuture<Model> fetchModel(API.Mesh mesh) {
    Vertex.Buffer vb = mesh.getVertexBuffer();
    float[] positions = null;
    float[] normals = null;

    for (Vertex.Stream stream : vb.getStreamsList()) {
      switch (stream.getSemantic().getType()) {
        case Position:
          positions = byteStringToFloatArray(stream.getData());
          break;
        case Normal:
          normals = byteStringToFloatArray(stream.getData());
          break;
        default:
          // Ignore.
      }
    }

    API.DrawPrimitive primitive = mesh.getDrawPrimitive();
    if (positions == null || (normals == null && isPolygon(primitive))) {
      return Futures.immediateFailedFuture(
          new DataUnavailableException(NO_MESH_ERR, new Client.Stack(() -> "")));
    }

    int[] indices = mesh.getIndexBuffer().getIndicesList().stream().mapToInt(x -> x).toArray();
    Model model = new Model(primitive, mesh.getStats(), positions, normals, indices);
    return Futures.immediateFuture(model);
  }

  private static float[] byteStringToFloatArray(ByteString bytes) {
    byte[] data = bytes.toByteArray();
    FloatBuffer buffer = ByteBuffer.wrap(data).order(ByteOrder.LITTLE_ENDIAN).asFloatBuffer();
    float[] out = new float[data.length / 4];
    buffer.get(out);
    return out;
  }

  protected void update(VertexSemantics semantics, List<Model> modelList) {
    this.vertexSemantics = semantics;
    this.configureItem.setEnabled(semantics.shouldShowUi());

    if (modelList.isEmpty()) {
      loading.showMessage(Info, Messages.SELECT_DRAW_CALL);
      saveItem.setEnabled(false);
      return;
    }

    originalModel = modelList.get(0);
    facetedModel = modelList.get(1);
    loading.stopLoading();
    originalModelItem.setEnabled(originalModel != null);
    facetedModelItem.setEnabled(facetedModel != null);
    saveItem.setEnabled(true);
    if (originalModel != null) {
      setModel(originalModel);
      originalModelItem.setSelection(true);
      facetedModelItem.setSelection(false);
    } else {
      setModel(facetedModel);
      originalModelItem.setSelection(false);
      facetedModelItem.setSelection(true);
    }
  }

  protected void setModel(Model model) {
    Geometry.DisplayMode newDisplayMode = desiredDisplayMode;
    switch (model.getPrimitive()) {
      case TriangleStrip:
      case Triangles:
        renderAsTriangles.setEnabled(true);
        renderAsLines.setEnabled(true);
        break;
      case LineLoop:
      case LineStrip:
      case Lines:
        renderAsTriangles.setEnabled(false);
        renderAsLines.setEnabled(true);
        if (newDisplayMode == Geometry.DisplayMode.TRIANGLES) {
          newDisplayMode = Geometry.DisplayMode.LINES;
        }
        break;
      case Points:
        renderAsTriangles.setEnabled(false);
        renderAsLines.setEnabled(false);
        newDisplayMode = Geometry.DisplayMode.POINTS;
        break;
      default:
        // Ignore.
    }

    renderAsTriangles.setSelection(newDisplayMode == Geometry.DisplayMode.TRIANGLES);
    renderAsLines.setSelection(newDisplayMode == Geometry.DisplayMode.LINES);
    renderAsPoints.setSelection(newDisplayMode == Geometry.DisplayMode.POINTS);
    displayMode = newDisplayMode;

    setSceneData(data.withGeometry(new Geometry(model, data.geometry.zUp), displayMode));
    statusBar.setText(model.getStatusMessage());
  }

  private void updateRenderable() {
    setSceneData(data.withGeometry(data.geometry, displayMode));
  }

  private void setSceneData(GeometryScene.Data data) {
    this.data = data;
    camera.setEmitter(data.geometry.getEmitter());
    canvas.setSceneData(data);
  }

  private void checkOpenGLAndShowMessage(Loadable.MessageType type, String text) {
    if (!canvas.isOpenGL()) {
      loading.showMessage(Error, Messages.NO_OPENGL);
    } else if (type != null && text != null) {
      loading.showMessage(type, text);
    }
  }

  private static class ModelLoadResult {
    public final VertexSemantics semantics;
    public final List<Model> models;
    public final ListenableFuture<Model> originalFuture;

    public ModelLoadResult(
        VertexSemantics semantics, List<Model> models, ListenableFuture<Model> originalFuture) {
      this.semantics = semantics;
      this.models = models;
      this.originalFuture = originalFuture;
    }

    public ModelLoadResult withoutModels() {
      return new ModelLoadResult(semantics, emptyList(), originalFuture);
    }

    public ModelLoadResult clearNthModel(int n) {
      List<Model> newModels = Lists.newArrayList(models);
      newModels.set(n, null);
      return new ModelLoadResult(semantics, newModels, originalFuture);
    }
  }

  private static class VertexSemantics {
    private static final String[] SEMANTIC_NAMES = Arrays.stream(Vertex.Semantic.Type.values())
        .filter(t -> t != Vertex.Semantic.Type.UNRECOGNIZED)
        .map(t -> (t == Vertex.Semantic.Type.Unknown ? "Other" : t.name()))
        .toArray(String[]::new);

    private final Element[] elements;

    public VertexSemantics(API.Mesh mesh, VertexSemantics current) {
      Map<String, Vertex.Semantic.Type> assigned = Maps.newHashMap();
      if (current != null) {
        for (Element e : current.elements) {
          assigned.put(e.name, e.assigned);
        }
      }

      this.elements = new Element[mesh.getVertexBuffer().getStreamsCount()];
      for (int i = 0; i < elements.length; i++) {
        elements[i] = Element.get(mesh.getVertexBuffer().getStreams(i), assigned);
      }
      Arrays.sort(elements, (e1, e2) -> e1.name.compareTo(e2.name));
    }

    public Path.MeshOptions.Builder getOptions() {
      Path.MeshOptions.Builder r = Path.MeshOptions.newBuilder();
      Arrays.stream(elements)
          .forEach(e -> r.addVertexSemantics(Path.MeshOptions.SemanticHint.newBuilder()
              .setName(e.name)
              .setType(e.assigned)));
      return r;
    }

    public boolean shouldShowUi() {
      return elements.length > 0;
    }

    // Returns the callback to be invoked when the OK button is clicked.
    @SuppressWarnings("ProtocolBufferOrdinal")
    public Runnable createUi(Composite parent) {
      createLabel(parent, Messages.GEO_SEMANTICS_HINT);

      Composite panel = createComposite(parent, withMargin(new GridLayout(3, false), 10, 5));
      panel.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));

      Combo[] inputs = new Combo[elements.length];
      for (int i = 0; i < elements.length; i++) {
        int idx = i;
        createLabel(panel, elements[i].name);
        createLabel(panel, elements[i].type);
        inputs[i] = Widgets.createDropDown(panel);
        inputs[i].setItems(SEMANTIC_NAMES);
        inputs[i].select(elements[i].assigned.ordinal());
        inputs[i].addListener(SWT.Selection, e -> {
          int sel = inputs[idx].getSelectionIndex();
          if (sel > 0) {
            // If anything but "other" is selected, clear it from any other attribute.
            for (int j = 0; j < inputs.length; j++) {
              if (j != idx && inputs[j].getSelectionIndex() == sel) {
                inputs[j].select(0);
              }
            }
          }
        });
      }

      return () -> {
        for (int i = 0; i < inputs.length; i++) {
          int sel = inputs[i].getSelectionIndex();
          // The last enum element is "UNRECOGNIZED", which is invalid.
          if (sel < 0 || sel >= Vertex.Semantic.Type.values().length - 1) {
            elements[i].assigned = Vertex.Semantic.Type.Unknown;
          } else {
            elements[i].assigned = Vertex.Semantic.Type.values()[sel];
          }
        }
      };
    }

    private static class Element {
      public final String name;
      public final String type;
      public Vertex.Semantic.Type assigned;

      public Element(String name, String type, Vertex.Semantic.Type guessed) {
        this.name = name;
        this.type = type;
        this.assigned = guessed;
      }

      public static Element get(Vertex.Stream s, Map<String, Vertex.Semantic.Type> assigned) {
        Vertex.Semantic.Type type = assigned.get(s.getName());
        if (type == null) {
          type = s.getSemantic().getType();
        }
        return new Element(s.getName(), Streams.toString(s.getFormat()), type);
      }
    }
  }

  private static class SemanticsDialog extends DialogBase {
    private final VertexSemantics semantics;
    private Runnable onOk;

    public SemanticsDialog(Shell parent, Theme theme, VertexSemantics semantics) {
      super(parent, theme);
      this.semantics = semantics;
    }

    @Override
    public String getTitle() {
      return Messages.GEO_SEMANTICS_TITLE;
    }

    @Override
    protected Control createDialogArea(Composite parent) {
      Composite area = (Composite)super.createDialogArea(parent);
      onOk = semantics.createUi(area);
      return area;
    }

    @Override
    protected void createButtonsForButtonBar(Composite parent) {
      createButton(parent, IDialogConstants.CANCEL_ID, IDialogConstants.CANCEL_LABEL, false);
      createButton(parent, IDialogConstants.OK_ID, IDialogConstants.OK_LABEL, true);
    }

    @Override
    protected void buttonPressed(int buttonId) {
      if (onOk != null && buttonId == IDialogConstants.OK_ID) {
        onOk.run();
      }
      super.buttonPressed(buttonId);
    }
  }
}
