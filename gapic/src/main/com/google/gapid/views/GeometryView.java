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
import static com.google.gapid.views.ErrorDialog.showErrorDialog;
import static com.google.gapid.widgets.Widgets.createBaloonToolItem;
import static com.google.gapid.widgets.Widgets.createComposite;
import static com.google.gapid.widgets.Widgets.createLabel;
import static com.google.gapid.widgets.Widgets.createSeparator;
import static com.google.gapid.widgets.Widgets.createToggleToolItem;
import static com.google.gapid.widgets.Widgets.createToolItem;
import static com.google.gapid.widgets.Widgets.exclusiveSelection;
import static com.google.gapid.widgets.Widgets.withMargin;
import static java.util.logging.Level.WARNING;

import com.google.common.base.Supplier;
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
import com.google.gapid.models.Geometries;
import com.google.gapid.models.Geometries.VertexSemantics;
import com.google.gapid.models.Models;
import com.google.gapid.proto.service.Service.ClientAction;
import com.google.gapid.proto.service.vertex.Vertex;
import com.google.gapid.util.Loadable;
import com.google.gapid.util.Loadable.Message;
import com.google.gapid.util.Messages;
import com.google.gapid.widgets.DialogBase;
import com.google.gapid.widgets.LoadablePanel;
import com.google.gapid.widgets.ScenePanel;
import com.google.gapid.widgets.Theme;
import com.google.gapid.widgets.Widgets;

import org.eclipse.jface.dialogs.IDialogConstants;
import org.eclipse.jface.window.Window;
import org.eclipse.swt.SWT;
import org.eclipse.swt.layout.FillLayout;
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
import java.util.Arrays;
import java.util.logging.Logger;

/**
 * View that displays the 3D geometry of the last draw call within the current selection.
 */
public class GeometryView extends Composite
    implements Tab, Capture.Listener, CommandStream.Listener, Geometries.Listener {
  private static final Logger LOG = Logger.getLogger(GeometryView.class.getName());

  private final Models models;
  protected final LoadablePanel<ScenePanel<GeometryScene.Data>> loading;
  protected final ScenePanel<GeometryScene.Data> canvas;
  private final Label statusBar;
  protected GeometryScene.Data data = GeometryScene.Data.DEFAULTS;
  private final IsoSurfaceCameraModel camera =
      new IsoSurfaceCameraModel(new CylindricalCameraModel());
  private ToolItem originalModelItem, facetedModelItem;
  private Geometry.DisplayMode displayMode = Geometry.DisplayMode.TRIANGLES;
  private Geometry.DisplayMode desiredDisplayMode = Geometry.DisplayMode.TRIANGLES;
  private ToolItem renderAsTriangles, renderAsLines, renderAsPoints;
  private ToolItem configureItem, saveItem;

  public GeometryView(Composite parent, Models models, Widgets widgets) {
    super(parent, SWT.NONE);
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
    models.geos.addListener(this);
    addListener(SWT.Dispose, e -> {
      models.capture.removeListener(this);
      models.commands.removeListener(this);
      models.geos.removeListener(this);
    });

    configureItem.setEnabled(false);
    originalModelItem.setEnabled(false);
    facetedModelItem.setEnabled(false);
    saveItem.setEnabled(false);

    checkOpenGLAndShowMessage(null, null);
  }

  private ToolBar createToolbar(Theme theme) {
    ToolBar bar = new ToolBar(this, SWT.VERTICAL | SWT.FLAT);
    createBaloonToolItem(bar, theme.yUp(), shell -> {
      Composite c = createComposite(shell, new FillLayout(SWT.VERTICAL), SWT.BORDER);
      ToolBar b = new ToolBar(c, SWT.HORIZONTAL | SWT.FLAT);
      exclusiveSelection(
        createToggleToolItem(b, theme.yUp(), e -> {
          models.analytics.postInteraction(View.Geometry, ClientAction.YUp);
          setSceneData(data.withGeometry(new Geometry(data.geometry.model, false, false), displayMode));
        }, "Select Y-Up Axis"),
        createToggleToolItem(b, theme.yDown(), e -> {
          models.analytics.postInteraction(View.Geometry, ClientAction.YDown);
          setSceneData(data.withGeometry(new Geometry(data.geometry.model, false, true), displayMode));
        }, "Select Y-Down Axis"),
        createToggleToolItem(b, theme.zUp(), e -> {
          models.analytics.postInteraction(View.Geometry, ClientAction.ZUp);
          setSceneData(data.withGeometry(new Geometry(data.geometry.model, true, false), displayMode));
        }, "Select Z-Up Axis"),
        createToggleToolItem(b, theme.zDown(), e -> {
          models.analytics.postInteraction(View.Geometry, ClientAction.ZDown);
          setSceneData(data.withGeometry(new Geometry(data.geometry.model, true, true), displayMode));
        }, "Select Z-Down Axis")
      );
    }, "Choose up axis");
    createToolItem(bar, theme.windingCCW(), e -> {
      boolean cw = data.winding == GeometryScene.Winding.CCW; // cw represent the new value.
      models.analytics.postInteraction(
          View.Geometry, cw ? ClientAction.WindingCW : ClientAction.WindingCCW);
      ((ToolItem)e.widget).setImage(cw ? theme.windingCW() : theme.windingCCW());
      setSceneData(data.withToggledWinding());
    }, "Toggle triangle winding");
    configureItem = createToolItem(bar, theme.settings(), e -> {
      models.analytics.postInteraction(View.Geometry, ClientAction.VertexSemantics);
      SemanticsDialog dialog = new SemanticsDialog(getShell(), theme, models.geos.getSemantics());
      if (dialog.open() == Window.OK) {
        models.geos.updateSemantics(dialog.getSemantics());
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
          setModel(models.geos.getData().original);
        }, "Use original normals"),
        facetedModelItem = createToggleToolItem(bar, theme.faceted(), e -> {
          models.analytics.postInteraction(View.Geometry, ClientAction.Faceted);
          setModel(models.geos.getData().faceted);
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
        Geometries.Data mesh = models.geos.getData();
        try (Writer out = new FileWriter(objFile)) {
          ObjWriter.write(out, originalModelItem.getSelection() ? mesh.original : mesh.faceted);
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
    onCaptureLoadingStart(false);
    if (models.capture.isLoaded() && models.commands.isLoaded()) {
      onCommandsLoaded();
      if (models.commands.getSelectedCommands() != null) {
        onGeometryLoadingStart();
        if (models.geos.isLoaded()) {
          onGeometryLoaded(null);
        }
      }
    }
  }

  @Override
  public void onCaptureLoadingStart(boolean maintainState) {
    checkOpenGLAndShowMessage(Info, Messages.LOADING_CAPTURE);
  }

  @Override
  public void onCaptureLoaded(Loadable.Message error) {
    if (error != null) {
      checkOpenGLAndShowMessage(Error, Messages.CAPTURE_LOAD_FAILURE);
    }
  }

  @Override
  public void onCommandsLoaded() {
    if (!models.commands.isLoaded()) {
      checkOpenGLAndShowMessage(Error, Messages.CAPTURE_LOAD_FAILURE);
    } else if (models.commands.getSelectedCommands() == null) {
      checkOpenGLAndShowMessage(Info, Messages.SELECT_DRAW_CALL);
    }
  }

  @Override
  public void onCommandsSelected(CommandIndex path) {
    if (path == null) {
      checkOpenGLAndShowMessage(Info, Messages.SELECT_DRAW_CALL);
    }
  }

  @Override
  public void onGeometryLoadingStart() {
    if (canvas.isOpenGL()) {
      loading.startLoading();

      configureItem.setEnabled(false);
      originalModelItem.setEnabled(false);
      facetedModelItem.setEnabled(false);
      saveItem.setEnabled(false);
      statusBar.setText("");
    }
  }

  @Override
  public void onGeometryLoaded(Message error) {
    if (!canvas.isOpenGL()) {
      return;
    } else if (error != null) {
      loading.showMessage(error);
      return;
    }

    Geometries.Data meshes = models.geos.getData();
    if (!meshes.hasFaceted()) {
      loading.showMessage(Info, Messages.SELECT_DRAW_CALL);
      return;
    }

    loading.stopLoading();

    configureItem.setEnabled(SemanticsDialog.shouldShowUi(meshes.semantics));
    originalModelItem.setEnabled(meshes.hasOriginal());
    facetedModelItem.setEnabled(meshes.hasFaceted());
    saveItem.setEnabled(true);

    if (meshes.hasOriginal()) {
      setModel(meshes.original);
      originalModelItem.setSelection(true);
      facetedModelItem.setSelection(false);
    } else {
      setModel(meshes.faceted);
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

    setSceneData(data.withGeometry(new Geometry(model, data.geometry.zUp, data.geometry.flipUpAxis), displayMode));
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

  private static class SemanticsDialog extends DialogBase {
    private static final String[] SEMANTIC_NAMES = Arrays.stream(Vertex.Semantic.Type.values())
        .filter(t -> t != Vertex.Semantic.Type.UNRECOGNIZED)
        .map(t -> (t == Vertex.Semantic.Type.Unknown ? "Other" : t.name()))
        .toArray(String[]::new);

    private VertexSemantics semantics;
    private Supplier<VertexSemantics> onOk;

    public SemanticsDialog(Shell parent, Theme theme, VertexSemantics semantics) {
      super(parent, theme);
      this.semantics = semantics;
    }

    public static boolean shouldShowUi(VertexSemantics semantics) {
      return semantics.elements.length > 0;
    }

    public VertexSemantics getSemantics() {
      return semantics;
    }

    @Override
    public String getTitle() {
      return Messages.GEO_SEMANTICS_TITLE;
    }

    @Override
    protected Control createDialogArea(Composite parent) {
      Composite area = (Composite)super.createDialogArea(parent);
      onOk = createUi(area, semantics);
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
        semantics = onOk.get();
      }
      super.buttonPressed(buttonId);
    }

    @SuppressWarnings("ProtocolBufferOrdinal")
    private static Supplier<VertexSemantics> createUi(Composite parent, VertexSemantics semantics) {
      createLabel(parent, Messages.GEO_SEMANTICS_HINT);

      Composite panel = createComposite(parent, withMargin(new GridLayout(3, false), 10, 5));
      panel.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));

      Combo[] inputs = new Combo[semantics.elements.length];
      for (int i = 0; i < semantics.elements.length; i++) {
        int idx = i;
        createLabel(panel, semantics.elements[i].name);
        createLabel(panel, semantics.elements[i].type);
        inputs[i] = Widgets.createDropDown(panel);
        inputs[i].setItems(SEMANTIC_NAMES);
        inputs[i].select(semantics.elements[i].semantic.ordinal());
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
        VertexSemantics result = semantics.copy();
        for (int i = 0; i < inputs.length; i++) {
          int sel = inputs[i].getSelectionIndex();
          // The last enum element is "UNRECOGNIZED", which is invalid.
          if (sel < 0 || sel >= Vertex.Semantic.Type.values().length - 1) {
            result.assign(i, Vertex.Semantic.Type.Unknown);
          } else {
            result.assign(i, Vertex.Semantic.Type.values()[sel]);
          }
        }
        return result;
      };
    }
  }
}
