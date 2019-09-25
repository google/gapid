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

import static com.google.gapid.proto.service.api.API.ResourceType.PipelineResource;
import static com.google.gapid.util.Loadable.MessageType.Error;
import static com.google.gapid.util.Loadable.MessageType.Info;
import static com.google.gapid.widgets.Widgets.createComposite;
import static com.google.gapid.widgets.Widgets.createTextarea;
import static com.google.gapid.widgets.Widgets.createStandardTabFolder;
import static com.google.gapid.widgets.Widgets.createStandardTabItem;
import static com.google.gapid.widgets.Widgets.disposeAllChildren;
import static java.util.logging.Level.FINE;

import com.google.gapid.models.Models;
import com.google.gapid.models.Resources;
import com.google.gapid.models.Capture;
import com.google.gapid.models.CommandStream;
import com.google.gapid.models.CommandStream.CommandIndex;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.api.API;
import com.google.gapid.rpc.Rpc;
import com.google.gapid.rpc.RpcException;
import com.google.gapid.rpc.SingleInFlight;
import com.google.gapid.rpc.UiCallback;
import com.google.gapid.util.Loadable;
import com.google.gapid.util.Messages;
import com.google.gapid.widgets.LoadablePanel;
import com.google.gapid.widgets.Widgets;

import org.eclipse.swt.SWT;
import org.eclipse.swt.widgets.TabFolder;
import org.eclipse.swt.widgets.TabItem;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Text;
import org.eclipse.swt.layout.FillLayout;

import java.util.List;
import java.util.concurrent.ExecutionException;
import java.util.logging.Logger;

/**
 * View the displays the information for each stage of the pipeline.
 */
public class PipelineView extends Composite
    implements Tab, Capture.Listener, Resources.Listener, CommandStream.Listener {
  protected static final Logger LOG = Logger.getLogger(ShaderView.class.getName());

  private List<Data> pipelines;
  private Models models;
  private Widgets widgets;

  private final LoadablePanel<Composite> loading;
  private TabFolder folder;

  public PipelineView(Composite parent, Models models, Widgets widgets) {
    super(parent, SWT.NONE);
    this.models = models;
    this.widgets = widgets;

    setLayout(new FillLayout());

    loading = LoadablePanel.create(this, widgets,
        panel -> createComposite(panel, new FillLayout(SWT.VERTICAL)));

    models.resources.addListener(this);
    models.commands.addListener(this);
    addListener(SWT.Dispose, e -> {
      models.resources.removeListener(this);
      models.commands.removeListener(this);
    });
  }

  @Override
  public void reinitialize() {
    updatePipelines();
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
 
  }

  @Override
  public void onCommandsLoaded() {
    if (!models.commands.isLoaded()) {
      loading.showMessage(Info, Messages.CAPTURE_LOAD_FAILURE);
    } else {
      loading.stopLoading();
    }
  }

  @Override
  public void onCommandsSelected(CommandIndex path) {
    updatePipelines();
  }


  public void updatePipelines() {
    disposeAllChildren(loading.getContents());
    folder = createStandardTabFolder(loading.getContents());

    Resources.ResourceList resources = models.resources.getResources(API.ResourceType.PipelineResource);

    if (!resources.isEmpty()) {
      resources.stream()
        .map(r -> new Data(r.resource))
        .forEach(data -> {
          getPipeline(data, folder);
        });
    }
    
    loading.getContents().requestLayout();
  }

  public void getPipeline(Data data, TabFolder folder) {
    Rpc.listen(models.resources.loadResource(data.info),
        new UiCallback<API.ResourceData, API.Pipeline>(this, LOG) {
      @Override
      protected API.Pipeline onRpcThread(Rpc.Result<API.ResourceData> result)
          throws RpcException, ExecutionException {
        API.Pipeline pipeline = result.get().getPipeline();
        data.resource = pipeline;
        return pipeline;
      }

      @Override
      protected void onUiThread(API.Pipeline result) {
        if (result.getBound()) {
          List<API.Stage> stages = result.getStagesList();

          for (API.Stage stage : stages) {
            createStandardTabItem(folder, stage.getDebugName());
          }
        }
      }
    });
  }

  @Override
  public Control getControl() {
    return this;
  }

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
  }
}
