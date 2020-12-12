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
package com.google.gapid.models;

import static com.google.gapid.util.Paths.compare;
import static com.google.gapid.util.Paths.framebufferAttachmentsAfter;
import static com.google.gapid.util.Paths.isNull;
import static com.google.gapid.util.Paths.pipelinesAfter;
import static com.google.gapid.util.Paths.resourceAfter;
import static java.util.Collections.emptyList;
import static java.util.logging.Level.WARNING;

import com.google.common.collect.Lists;
import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.models.CommandStream.CommandIndex;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.api.API;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.rpc.Rpc;
import com.google.gapid.rpc.Rpc.Result;
import com.google.gapid.rpc.RpcException;
import com.google.gapid.rpc.UiCallback;
import com.google.gapid.server.Client;
import com.google.gapid.util.Events;
import com.google.gapid.util.MoreFutures;

import org.eclipse.swt.widgets.Shell;

import java.util.List;
import java.util.Objects;
import java.util.concurrent.ExecutionException;
import java.util.logging.Logger;
import java.util.stream.Stream;

/**
 * Model containing the capture resources (textures, shaders, etc.) metadata.
 */
public class Resources extends CaptureDependentModel.ForValue<Resources.Data, Resources.Listener>
    implements Follower.Listener {
  private static final Logger LOG = Logger.getLogger(Resources.class.getName());

  protected final Capture capture;
  private final CommandStream commands;

  protected Service.Resource selectedTexture = null;

  public Resources(Shell shell, Analytics analytics, Client client, Capture capture,
      Devices devices, CommandStream commands, Follower follower) {
    super(LOG, shell, analytics, client, Listener.class, capture, devices);
    this.capture = capture;
    this.commands = commands;

    follower.addListener(this);
  }

  @Override
  protected Path.Any getSource(Capture.Data data) {
    return Path.Any.newBuilder()
        .setResources(Path.Resources.newBuilder()
            .setCapture(data.path))
        .build();
  }

  @Override
  protected boolean shouldLoad(Capture.Data data) {
    return data.isGraphics();
  }

  @Override
  protected Data unbox(Service.Value value, Path.Device device) {
    return new Data(device, value.getResources());
  }

  @Override
  protected void fireLoadStartEvent() {
    selectTexture(null);
  }

  @Override
  protected void fireLoadedEvent() {
    listeners.fire().onResourcesLoaded();
  }

  @Override
  public void onResourceFollowed(Path.ResourceData path) {
    Resources.Resource r = getResource(path);

    if (r != null) {
      switch (r.resource.getType()) {
        case TextureResource:
          selectTexture(r.resource);
          break;
        default:
          LOG.log(WARNING, "Unknown follow path result: " + path);
      }
    } else {
      LOG.log(WARNING, "Path resolved to null resource: " + path);
    }
  }

  public List<Service.ResourcesByType> getResources() {
    return getData().resources.getTypesList();
  }

  public ResourceList getResources(Path.ResourceType type) {
    if (!isLoaded() || commands.getSelectedCommands() == null) {
      return new ResourceList(type, emptyList(), false);
    }
    return getData().getResources(commands.getSelectedCommands().getCommand(), type);
  }

  public Resource getResource(Path.ResourceData path) {
    if (!isLoaded() || commands.getSelectedCommands() == null) {
      return null;
    }
    return getData().getResource(path);
  }

  public Path.ResourceData getResourcePath(Service.Resource resource) {
    CommandIndex after = commands.getSelectedCommands();
    return (after == null) ? null : Path.ResourceData.newBuilder()
        .setAfter(after.getCommand())
        .setID(resource.getID())
        .build();
  }

  public ListenableFuture<API.ResourceData> loadResource(Service.Resource resource) {
    CommandIndex after = commands.getSelectedCommands();
    if (after == null) {
      return Futures.immediateFailedFuture(new RuntimeException("No command selected"));
    }

    // TODO: don't get the device via getData
    return MoreFutures.transform(
        client.get(resourceAfter(after, resource.getID()), getData().device),
        Service.Value::getResourceData);
  }

  public ListenableFuture<API.MultiResourceData> loadBoundPipelines() {
    CommandIndex after = commands.getSelectedCommands();
    if (after == null) {
      return Futures.immediateFailedFuture(new RuntimeException("No command selected"));
    }

    // TODO: don't get the device via getData
    return MoreFutures.transform(
        client.get(pipelinesAfter(after), getData().device),
        Service.Value::getMultiResourceData);
  }


  public ListenableFuture<Service.FramebufferAttachments> loadFramebufferAttachments() {
    CommandIndex after = commands.getSelectedCommands();
    if (after == null) {
      return Futures.immediateFailedFuture(new RuntimeException("No command selected"));
    }

    // TODO: don't get the device via getData
    return MoreFutures.transform(
      client.get(framebufferAttachmentsAfter(after), getData().device),
      Service.Value::getFramebufferAttachments);
  }

  public void updateResource(Service.Resource resource, API.ResourceData data) {
    CommandIndex after = commands.getSelectedCommands();
    if (after == null) {
      return;
    }

 // TODO: don't get the device via getData
    Rpc.listen(client.set(
        resourceAfter(after, resource.getID()), getData().device, Service.Value.newBuilder()
            .setResourceData(data)
            .build()),
        new UiCallback<Path.Any, Path.Capture>(shell, LOG) {
      @Override
      protected Path.Capture onRpcThread(Result<Path.Any> result)
          throws RpcException, ExecutionException {
        // TODO this should probably be able to handle any path.
        return result.get().getResourceData().getAfter().getCapture();
      }

      @Override
      protected void onUiThread(Path.Capture result) {
        capture.updateCapture(result, null);
      }
    });
  }

  public Service.Resource getSelectedTexture() {
    return selectedTexture;
  }

  public void selectTexture(Service.Resource texture) {
    if (!Objects.equals(selectedTexture, texture)) {
      selectedTexture = texture;
      listeners.fire().onTextureSelected(selectedTexture);
    }
  }

  public static class Data extends DeviceDependentModel.Data {
    public final Service.Resources resources;

    public Data(Path.Device device, Service.Resources resources) {
      super(device);
      this.resources = resources;
    }

    public ResourceList getResources(Path.Command after, Path.ResourceType type) {
      List<Resource> list = Lists.newArrayList();
      boolean complete = true;
      for (Service.ResourcesByType rs : resources.getTypesList()) {
        if (rs.getType() != type) {
          continue;
        }

        for (Service.Resource r : rs.getResourcesList()) {
          if (compare(firstAccess(r), after) <= 0) {
            Path.Command deleted = r.getDeleted();
            list.add(new Resource(r, !isNull(deleted) && compare(deleted, after) <= 0));
          } else {
            complete = false;
          }
        }
      }
      return new ResourceList(type, list, complete);
    }

    public Resource getResource(Path.ResourceData path) {
      for (Service.ResourcesByType rs : resources.getTypesList()) {
        for (Service.Resource r : rs.getResourcesList()) {
          if (path.getID().equals(r.getID())) {
            Path.Command deleted = r.getDeleted();
            return new Resource(r, !isNull(deleted) && compare(deleted, path.getAfter()) <= 0);
          }
        }
      }

      return null;
    }

    private static Path.Command firstAccess(Service.Resource info) {
      return (info.getAccessesCount() == 0) ? null : info.getAccesses(0);
    }
  }

  public static class ResourceList {
    public final Path.ResourceType type;
    public final List<Resource> resources;
    public final boolean complete;

    public ResourceList(Path.ResourceType type, List<Resource> resources, boolean complete) {
      this.type = type;
      this.resources = resources;
      this.complete = complete;
    }

    public boolean isEmpty() {
      return resources.isEmpty();
    }

    public Stream<Resource> stream() {
      return resources.stream();
    }
  }

  public static class Resource {
    public final Service.Resource resource;
    public final boolean deleted;

    public Resource(Service.Resource resource, boolean deleted) {
      this.resource = resource;
      this.deleted = deleted;
    }
  }

  @SuppressWarnings("unused")
  public static interface Listener extends Events.Listener {
    /**
     * Event indicating that the resources metadata has been loaded.
     */
    public default void onResourcesLoaded() { /* empty */ }

    /**
     * Event indicating that a texture resource has been selected.
     */
    public default void onTextureSelected(Service.Resource texture) { /* empty */ }
  }
}
