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
import static com.google.gapid.widgets.Widgets.createSeparator;
import static com.google.gapid.widgets.Widgets.createToggleToolItem;
import static com.google.gapid.widgets.Widgets.createToolItem;
import static com.google.gapid.widgets.Widgets.exclusiveSelection;

import com.google.common.collect.Lists;
import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.common.util.concurrent.Uninterruptibles;
import com.google.gapid.Server.GapisInitException;
import com.google.gapid.glviewer.Geometry;
import com.google.gapid.glviewer.Geometry.DisplayMode;
import com.google.gapid.glviewer.Viewer;
import com.google.gapid.glviewer.camera.CylindricalCameraModel;
import com.google.gapid.glviewer.camera.IsoSurfaceCameraModel;
import com.google.gapid.glviewer.geo.Model;
import com.google.gapid.models.AtomStream;
import com.google.gapid.models.AtomStream.AtomIndex;
import com.google.gapid.models.Capture;
import com.google.gapid.models.Models;
import com.google.gapid.models.Strings;
import com.google.gapid.proto.service.gfxapi.GfxAPI;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.proto.service.vertex.Vertex;
import com.google.gapid.proto.stringtable.Stringtable;
import com.google.gapid.rpclib.futures.FutureController;
import com.google.gapid.rpclib.futures.SingleInFlight;
import com.google.gapid.rpclib.rpccore.Rpc;
import com.google.gapid.rpclib.rpccore.RpcException;
import com.google.gapid.server.Client;
import com.google.gapid.server.Client.DataUnavailableException;
import com.google.gapid.util.Messages;
import com.google.gapid.util.Streams;
import com.google.gapid.util.UiErrorCallback;
import com.google.gapid.widgets.GlComposite;
import com.google.gapid.widgets.LoadablePanel;
import com.google.gapid.widgets.Theme;
import com.google.gapid.widgets.Widgets;
import com.google.protobuf.ByteString;

import org.eclipse.swt.SWT;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.ToolBar;
import org.eclipse.swt.widgets.ToolItem;

import java.nio.ByteBuffer;
import java.nio.ByteOrder;
import java.nio.FloatBuffer;
import java.util.List;
import java.util.concurrent.ExecutionException;
import java.util.logging.Logger;

/**
 * View that displays the 3D geometry of the last draw call within the current selection.
 */
public class GeometryView extends Composite implements Tab, Capture.Listener, AtomStream.Listener {
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
  private final FutureController rpcController = new SingleInFlight();
  protected final LoadablePanel<GlComposite> loading;
  private final Geometry geometry = new Geometry();
  private final IsoSurfaceCameraModel camera =
      new IsoSurfaceCameraModel(new CylindricalCameraModel());
  private final Viewer viewer = new Viewer(camera);
  private final GlComposite canvas;
  private ToolItem originalModelItem, facetedModelItem;
  private Model originalModel, facetedModel;
  private Geometry.DisplayMode displayMode = Geometry.DisplayMode.TRIANGLES;
  private Geometry.DisplayMode desiredDisplayMode = Geometry.DisplayMode.TRIANGLES;
  private ToolItem renderAsTriangles, renderAsLines, renderAsPoints;

  public GeometryView(Composite parent, Client client, Models models, Widgets widgets) {
    super(parent, SWT.NONE);
    this.client = client;
    this.models = models;

    setLayout(new GridLayout(2, false));

    ToolBar toolbar = createToolbar(widgets.theme);
    loading = LoadablePanel.create(this, widgets, panel -> createCanvas(panel));
    canvas = loading.getContents();

    toolbar.setLayoutData(new GridData(SWT.LEFT, SWT.FILL, false, true));
    loading.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));

    models.capture.addListener(this);
    models.atoms.addListener(this);
    addListener(SWT.Dispose, e -> {
      models.capture.removeListener(this);
      models.atoms.removeListener(this);
    });

    viewer.addMouseListeners(canvas);

    originalModelItem.setEnabled(false);
    facetedModelItem.setEnabled(false);
  }

  private GlComposite createCanvas(Composite parent) {
    GlComposite result = new GlComposite(parent);
    result.addListener(viewer);
    return result;
  }

  private ToolBar createToolbar(Theme theme) {
    ToolBar bar = new ToolBar(this, SWT.VERTICAL | SWT.FLAT);
    createToolItem(bar, theme.yUp(), e -> {
      boolean zUp = geometry.toggleZUp();
      ((ToolItem)e.widget).setImage(zUp ? theme.zUp() : theme.yUp());
      updateViewer();
    }, "Toggle Y/Z up");
    createToolItem(bar, theme.windingCCW(), e -> {
      Viewer.Winding winding = viewer.toggleWinding();
      ((ToolItem)e.widget).setImage((winding == Viewer.Winding.CCW) ?
          theme.windingCCW() : theme.windingCW());
      updateViewer();
    }, "Toggle triangle winding");
    createSeparator(bar);
    exclusiveSelection(
        renderAsTriangles = createToggleToolItem(bar, theme.wireframeNone(), e -> {
          desiredDisplayMode = displayMode = Geometry.DisplayMode.TRIANGLES;
          updateRenderable();
        }, "Render as triangles"),
        renderAsLines = createToggleToolItem(bar, theme.wireframeAll(), e -> {
          desiredDisplayMode = displayMode = Geometry.DisplayMode.LINES;
          updateRenderable();
        }, "Render as lines"),
        renderAsPoints = createToggleToolItem(bar, theme.pointCloud(), e -> {
          desiredDisplayMode = displayMode = Geometry.DisplayMode.POINTS;
          updateRenderable();
        }, "Render as points"));
    createSeparator(bar);
    exclusiveSelection(
        originalModelItem = createToggleToolItem(bar, theme.smooth(), e -> {
          setModel(originalModel);
        }, "Use original normals"),
        facetedModelItem = createToggleToolItem(bar, theme.faceted(), e -> {
          setModel(facetedModel);
        }, "Use computed per-face normals"));
    createSeparator(bar);
    createToolItem(bar, theme.cullingDisabled(), e -> {
      Viewer.Culling culling = viewer.toggleCulling();
      ((ToolItem)e.widget).setImage((culling == Viewer.Culling.ON) ?
          theme.cullingEnabled() : theme.cullingDisabled());
      updateViewer();
    }, "Toggle backface culling");
    createSeparator(bar);
    exclusiveSelection(
        createToggleToolItem(bar, theme.lit(), e -> {
          viewer.setShading(Viewer.Shading.LIT);
          updateViewer();
        }, "Render with a lit shader"),
        createToggleToolItem(bar, theme.flat(), e -> {
          viewer.setShading(Viewer.Shading.FLAT);
          updateViewer();
        }, "Render with a flat shader (silhouette)"),
        createToggleToolItem(bar, theme.normals(), e -> {
          viewer.setShading(Viewer.Shading.NORMALS);
          updateViewer();
        }, "Render normals"));
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
  public void onCaptureLoaded(GapisInitException error) {
    if (error != null) {
      loading.showMessage(Error, Messages.CAPTURE_LOAD_FAILURE);
    }
  }

  @Override
  public void onAtomsLoaded() {
    updateModels(false);
  }

  @Override
  public void onAtomsSelected(AtomIndex range) {
    updateModels(false);
  }

  private void updateModels(boolean assumeLoading) {
    if (!assumeLoading && models.atoms.isLoaded()) {
      AtomIndex atom = models.atoms.getSelectedAtoms();
      if (atom == null) {
        loading.showMessage(Info, Messages.SELECT_DRAW_CALL);
      } else {
        fetchMeshes(atom);
      }
    } else {
      loading.showMessage(Info, Messages.LOADING_CAPTURE);
    }
  }

  private void fetchMeshes(AtomIndex atom) {
    loading.startLoading();
    ListenableFuture<Model> originalFuture = fetchModel(
        meshAfter(atom, Path.MeshOptions.getDefaultInstance(), POS_NORM_XYZ_F32));
    ListenableFuture<Model> facetedFuture = fetchModel(meshAfter(
        atom, Path.MeshOptions.newBuilder().setFaceted(true).build(), POS_NORM_XYZ_F32));
    Rpc.listen(Futures.successfulAsList(originalFuture, facetedFuture), rpcController,
        new UiErrorCallback<List<Model>, List<Model>, String>(this, LOG) {
      @Override
      protected ResultOrError<List<Model>, String> onRpcThread(Rpc.Result<List<Model>> result)
          throws RpcException, ExecutionException {
        List<Model> modelList = result.get();
        if (modelList.get(0) == null && modelList.get(1) == null) {
          // Both failed, so get the error from the original model's call.
          try {
            Uninterruptibles.getUninterruptibly(originalFuture);
          } catch (ExecutionException e) {
            if (e.getCause() instanceof DataUnavailableException) {
              return error(e.getCause().getMessage());
            } else {
              throw e;
            }
          }
          // Should not get here, the future cannot both fail and succeed.
          throw new AssertionError("Future both failed and succeeded");
        } else if (modelList.get(0) != null && modelList.get(0).getNormals() == null) {
          // The original model has no normals, but the geometry doesn't support faceted normals.
          return success(Lists.newArrayList(null, modelList.get(0)));
        } else if (modelList.get(0) != null && !isPolygon(modelList.get(0).getPrimitive())) {
          // TODO: if gapis returns an error for the faceted request, this is not needed.
          return success(Lists.newArrayList(modelList.get(0), null));
        } else {
          return success(modelList);
        }
      }

      @Override
      protected void onUiThreadSuccess(List<Model> modelList) {
        update(modelList);
      }

      @Override
      protected void onUiThreadError(String error) {
        loading.showMessage(Error, error);
      }
    });
  }

  private ListenableFuture<Model> fetchModel(Path.Any path) {
    return Futures.transformAsync(client.get(path), value -> fetchModel(value.getMesh()));
  }

  private static ListenableFuture<Model> fetchModel(GfxAPI.Mesh mesh) {
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

    GfxAPI.DrawPrimitive primitive = mesh.getDrawPrimitive();
    if (positions == null || (normals == null && isPolygon(primitive))) {
      return Futures.immediateFailedFuture(new DataUnavailableException(NO_MESH_ERR));
    }

    int[] indices = mesh.getIndexBuffer().getIndicesList().stream().mapToInt(x -> x).toArray();
    Model model = new Model(primitive, positions, normals, indices);
    return Futures.immediateFuture(model);
  }

  private static float[] byteStringToFloatArray(ByteString bytes) {
    byte[] data = bytes.toByteArray();
    FloatBuffer buffer = ByteBuffer.wrap(data).order(ByteOrder.LITTLE_ENDIAN).asFloatBuffer();
    float[] out = new float[data.length / 4];
    buffer.get(out);
    return out;
  }

  protected void update(List<Model> modelList) {
    originalModel = modelList.get(0);
    facetedModel = modelList.get(1);
    loading.stopLoading();
    originalModelItem.setEnabled(originalModel != null);
    facetedModelItem.setEnabled(facetedModel != null);
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
    DisplayMode newDisplayMode = desiredDisplayMode;
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
        if (newDisplayMode == DisplayMode.TRIANGLES) {
          newDisplayMode = DisplayMode.LINES;
        }
        break;
      case Points:
        renderAsTriangles.setEnabled(false);
        renderAsLines.setEnabled(false);
        newDisplayMode = DisplayMode.POINTS;
        break;
      default:
        // Ignore.
    }

    setDisplayMode(newDisplayMode);
    geometry.setModel(model);
    updateRenderable();
  }

  private void setDisplayMode(DisplayMode newMode) {
    renderAsTriangles.setSelection(newMode == DisplayMode.TRIANGLES);
    renderAsLines.setSelection(newMode == DisplayMode.LINES);
    renderAsPoints.setSelection(newMode == DisplayMode.POINTS);
    displayMode = newMode;
  }

  private void updateRenderable() {
    // Repaint will happen below.
    viewer.setRenderable(geometry.asRenderable(displayMode));
    updateViewer();
  }

  private void updateViewer() {
    camera.setEmitter(geometry.getEmitter());
    canvas.paint();
  }
}
