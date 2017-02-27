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

import static com.google.gapid.util.Paths.stateAfter;
import static java.util.logging.Level.SEVERE;

import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.Service.CommandRange;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.proto.stringtable.Stringtable.Msg;
import com.google.gapid.rpclib.futures.FutureController;
import com.google.gapid.rpclib.futures.SingleInFlight;
import com.google.gapid.rpclib.rpccore.Rpc;
import com.google.gapid.rpclib.rpccore.RpcException;
import com.google.gapid.rpclib.schema.Dynamic;
import com.google.gapid.server.Client;
import com.google.gapid.server.Client.DataUnavailableException;
import com.google.gapid.util.Events;
import com.google.gapid.util.Events.ListenerCollection;
import com.google.gapid.util.PathStore;
import com.google.gapid.util.UiErrorCallback;

import org.eclipse.swt.widgets.Shell;

import java.io.IOException;
import java.util.concurrent.ExecutionException;
import java.util.logging.Logger;

/**
 * Model managing the API state object of the currently selected command.
 */
public class ApiState {
  protected static final Logger LOG = Logger.getLogger(ApiState.class.getName());

  private final Shell shell;
  private final Client client;
  private final ListenerCollection<Listener> listeners = Events.listeners(Listener.class);
  private final FutureController rpcController = new SingleInFlight();
  private final PathStore statePath = new PathStore();
  private final PathStore selection = new PathStore();
  private Dynamic state;

  public ApiState(Shell shell, Client client, Follower follower, AtomStream atoms) {
    this.shell = shell;
    this.client = client;

    atoms.addListener(new AtomStream.Listener() {
      @Override
      public void onAtomsSelected(CommandRange path) {
        loadState(atoms.getPath(), path);
      }
    });
    follower.addListener(new Follower.Listener() {
      @Override
      public void onStateFollowed(Path.Any path) {
        selectPath(path, true);
      }
    });
  }

  protected void loadState(Path.Any atomsPath, CommandRange range) {
    if (statePath.updateIfNotNull(stateAfter(atomsPath, range))) {
      // we are making a request for a new state, this means our current state is old and irrelevant
      state = null;
      listeners.fire().onStateLoadingStart();
      Rpc.listen(client.get(statePath.getPath()), rpcController,
          new UiErrorCallback<Service.Value, Dynamic, DataUnavailableException>(shell, LOG) {
        @Override
        protected ResultOrError<Dynamic, DataUnavailableException> onRpcThread(
            Rpc.Result<Service.Value> result) throws RpcException, ExecutionException {
          try {
            return success(Client.decode(result.get().getObject()));
          } catch (DataUnavailableException e) {
            return error(e);
          } catch (IOException e) {
            LOG.log(SEVERE, "Error decoding state", e);
            return error(new DataUnavailableException(Msg.getDefaultInstance()));
          }
        }

        @Override
        protected void onUiThreadSuccess(Dynamic result) {
          update(result);
        }

        @Override
        protected void onUiThreadError(DataUnavailableException error) {
          update(error);
        }
      });
    }
  }

  protected void update(Dynamic newState) {
    state = newState;
    listeners.fire().onStateLoaded(null);
  }

  protected void update(DataUnavailableException error) {
    listeners.fire().onStateLoaded(error);
  }

  public Path.Any getPath() {
    return statePath.getPath();
  }

  public Dynamic getState() {
    return state;
  }

  public Path.Any getSelectedPath() {
    return selection.getPath();
  }

  public void selectPath(Path.Any path, boolean force) {
    if (selection.update(path) || force) {
      listeners.fire().onStateSelected(path);
    }
  }

  public void addListener(Listener listener) {
    listeners.addListener(listener);
  }

  public void removeListener(Listener listener) {
    listeners.removeListener(listener);
  }

  @SuppressWarnings("unused")
  public static interface Listener extends Events.Listener {
    /**
     * Event indicating that the state is currently being loaded.
     */
    public default void onStateLoadingStart()  { /* empty */ }

    /**
     * Event indicating that the state has finished loading.
     *
     * @param error the loading error or {@code null} if loading was successful.
     */
    public default void onStateLoaded(DataUnavailableException error) { /* empty */ }

    /**
     * Event indicating that the portion of the state that is selected has changed.
     */
    public default void onStateSelected(Path.Any path) { /* empty */ }
  }
}
