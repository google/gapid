/*
 * Copyright (C) 2020 Google Inc.
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
import static com.google.gapid.util.Loadable.MessageType.Error;
import static com.google.gapid.util.Loadable.MessageType.Info;
import static com.google.gapid.widgets.Widgets.createComposite;
import static com.google.gapid.widgets.Widgets.createTreeColumn;
import static com.google.gapid.widgets.Widgets.createTreeViewer;
import static com.google.gapid.widgets.Widgets.packColumns;
import static com.google.gapid.widgets.Widgets.sorting;

import com.google.common.primitives.UnsignedLongs;
import com.google.gapid.models.Analytics.View;
import com.google.gapid.models.Capture;
import com.google.gapid.models.CommandStream;
import com.google.gapid.models.CommandStream.CommandIndex;
import com.google.gapid.models.Models;
import com.google.gapid.models.Resources;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.Service.ClientAction;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.util.Loadable;
import com.google.gapid.util.Messages;
import com.google.gapid.widgets.LoadablePanel;
import com.google.gapid.widgets.Widgets;

import org.eclipse.jface.viewers.ITreeContentProvider;
import org.eclipse.jface.viewers.LabelProvider;
import org.eclipse.jface.viewers.StructuredSelection;
import org.eclipse.jface.viewers.TreeViewer;
import org.eclipse.swt.SWT;
import org.eclipse.swt.layout.FillLayout;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.TreeItem;

import java.util.ArrayList;
import java.util.List;
import java.util.logging.Logger;

/**
 * Displays a list of shader resources of the current capture.
 */
public class ShaderList extends Composite
    implements Tab, Capture.Listener, Resources.Listener, CommandStream.Listener {
  protected static final Logger LOG = Logger.getLogger(ShaderList.class.getName());

  protected final Models models;
  private final LoadablePanel<Composite> loading;
  private final TreeViewer shaderViewer;
  private boolean lastUpdateContainedAllShaders = false;

  public ShaderList(Composite parent, Models models, Widgets widgets) {
    super(parent, SWT.NONE);
    this.models = models;

    setLayout(new FillLayout());

    loading = LoadablePanel.create(this, widgets,
        panel -> createComposite(panel, new GridLayout(1, false)));

    shaderViewer = createTreeViewer(loading.getContents(), SWT.FILL);
    shaderViewer.getTree().setHeaderVisible(true);
    shaderViewer.setContentProvider(new ITreeContentProvider() {
      @SuppressWarnings("unchecked")
      @Override
      public Object[] getElements(Object inputElement) {
        return ((List<Data>)inputElement).toArray();
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
    });
    shaderViewer.setLabelProvider(new LabelProvider());
    shaderViewer.getControl().addListener(SWT.Selection, e -> {
      models.analytics.postInteraction(View.Shaders, ClientAction.SelectShader);
      Service.Resource selectedShader = null;
      if (shaderViewer.getTree().getSelectionCount() >= 1) {
        selectedShader = ((Data)getSelection().getData()).info;
      }
      models.resources.selectShader(selectedShader);
    });

    sorting(shaderViewer,
        createTreeColumn(shaderViewer, "ID", Data::getHandle,
            (d1, d2) -> UnsignedLongs.compare(d1.getSortId(), d2.getSortId())));
    shaderViewer.getTree().setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));

    models.capture.addListener(this);
    models.resources.addListener(this);
    models.commands.addListener(this);
    addListener(SWT.Dispose, e -> {
      models.capture.removeListener(this);
      models.resources.removeListener(this);
      models.commands.removeListener(this);
    });
  }

  @Override
  public Control getControl() {
    return this;
  }

  @Override
  public void reinitialize() {
    updateShaders();
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
  public void onResourcesLoaded() {
    lastUpdateContainedAllShaders = false;
    if (!models.resources.isLoaded()) {
      loading.showMessage(Info, Messages.CAPTURE_LOAD_FAILURE);
    } else {
      updateShaders();
    }
  }

  @Override
  public void onShaderSelected(Service.Resource shader) {
    // Do nothing if shader is already selected.
    TreeItem selection = getSelection();
    if (shader != null && selection != null) {
      if (((Data)selection.getData()).info.getID().equals(shader.getID())) {
        return;
      }
    } else if (shader == null && selection == null) {
      return;
    }

    if (shader != null) {
      // Find shader in view and select it.
      TreeItem[] items = shaderViewer.getTree().getItems();
      for (int i = 0; i < items.length; i++) {
        Data data = (Data)(items[i].getData());
        if (data.info.getID().equals(shader.getID())) {
          shaderViewer.getTree().setSelection(items[i]);
          break;
        }
      }
    } else {
      shaderViewer.setSelection(StructuredSelection.EMPTY);
    }
  }

  @Override
  public void onCommandsLoaded() {
    if (!models.commands.isLoaded()) {
      loading.showMessage(Info, Messages.CAPTURE_LOAD_FAILURE);
    } else {
      updateShaders();
    }
  }

  @Override
  public void onCommandsSelected(CommandIndex path) {
    updateShaders();
  }

  private void updateShaders() {
    if (!models.commands.isLoaded() || !models.resources.isLoaded()) {
      loading.startLoading();
    } else if (models.commands.getSelectedCommands() == null) {
      loading.showMessage(Info, Messages.SELECT_COMMAND);
    } else {
      Resources.ResourceList resources = models.resources.getResources(Path.ResourceType.Shader);

      if (!lastUpdateContainedAllShaders || !resources.complete) {
        List<Data> shaders = new ArrayList<Data>();
        if (!resources.isEmpty()) {
          resources.stream()
              .map(r -> new Data(r.resource))
              .forEach(shaders::add);
        }
        lastUpdateContainedAllShaders = resources.complete;
        shaderViewer.setInput(shaders);
        packColumns(shaderViewer.getTree());

        if (shaders.isEmpty()) {
          loading.showMessage(Info, Messages.NO_SHADERS);
        } else {
          loading.stopLoading();
        }
      }
      onShaderSelected(models.resources.getSelectedShader());
    }
  }

  private TreeItem getSelection() {
    return last(shaderViewer.getTree().getSelection());
  }

  /**
   * Shader data.
   */
  private static class Data {
    public final Service.Resource info;

    public Data(Service.Resource info) {
      this.info = info;
    }

    public String getHandle() {
      return info.getHandle();
    }

    public long getSortId() {
      return info.getOrder();
    }
  }
}
