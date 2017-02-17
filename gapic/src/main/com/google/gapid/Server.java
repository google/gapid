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
package com.google.gapid;

import static com.google.gapid.server.Version.GAPIC_VERSION;
import static java.util.concurrent.TimeUnit.MILLISECONDS;
import static java.util.logging.Level.INFO;
import static java.util.logging.Level.WARNING;

import com.google.gapid.models.Info;
import com.google.gapid.models.Strings;
import com.google.gapid.proto.service.GapidGrpc;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.stringtable.Stringtable;
import com.google.gapid.rpclib.rpccore.Rpc;
import com.google.gapid.rpclib.rpccore.RpcException;
import com.google.gapid.rpclib.schema.ConstantSet;
import com.google.gapid.rpclib.schema.Dynamic;
import com.google.gapid.rpclib.schema.Entity;
import com.google.gapid.rpclib.schema.Message;
import com.google.gapid.server.Client;
import com.google.gapid.server.GapidClientCache;
import com.google.gapid.server.GapidClientGrpc;
import com.google.gapid.server.GapisConnection;
import com.google.gapid.server.GapisProcess;
import com.google.gapid.server.Version;
import com.google.gapid.service.atom.AtomMetadata;
import com.google.gapid.util.Flags;
import com.google.gapid.util.Flags.Flag;

import java.io.IOException;
import java.util.List;
import java.util.concurrent.ExecutionException;
import java.util.concurrent.TimeoutException;
import java.util.logging.Logger;

public class Server {
  private static final Logger LOG = Logger.getLogger(Server.class.getName());

  private static final int FETCH_INFO_TIMEOUT_MS = 3000;
  private static final int FETCH_SCHEMA_TIMEOUT_MS = 3000;
  private static final int FETCH_STRING_TABLE_TIMEOUT_MS = 3000;

  public static final Flag<String> gapis = Flags.value(
      "gapis", "", "<host:port> of the gapis server to connect to.");

  public static final Flag<String> gapisAuthToken = Flags.value(
      "gapis-auth", "", "The auth token to use when connecting to an exisiting server.");

  public static final Flag<Boolean> useCache = Flags.value(
      "cache", true, "Whether to use a cache between the UI and the gapis server.");

  private GapisConnection gapisConnection;
  private Client client;

  public void connect(Listener listener) throws GapisInitException {
    connectToServer(listener);
    String status = "";
    try {
      status = "fetch server info";
      fetchServerInfo();
      status = "fetch schema";
      fetchSchema();
      status = "fetch string table";
      fetchStringTable();
    } catch (ExecutionException | RpcException | TimeoutException e) {
      throw new GapisInitException(GapisInitException.MESSAGE_FAILED_INIT, "Failed to " + status, e);
    }
  }

  public Client getClient() {
    return client;
  }

  public void disconnect() {
    if (gapisConnection != null) {
      gapisConnection.close();
      gapisConnection = null;
    }
  }

  private void connectToServer(Listener listener) throws GapisInitException {
    GapisConnection connection = createConnection(listener);
    if (!connection.isConnected()) {
      throw new GapisInitException(GapisInitException.MESSAGE_FAILED_CONNECT, "not connected");
    }
    gapisConnection = connection;
    try {
      GapidGrpc.GapidFutureStub service = connection.createGapidClient();
      if (useCache.get()) {
        client = new Client(new GapidClientCache(service));
      } else {
        LOG.log(WARNING, "** Not using caching in the UI, this is only meant for testing. **");
        client = new Client(new GapidClientGrpc(service));
      }
    } catch (IOException e) {
      throw new GapisInitException(
          GapisInitException.MESSAGE_FAILED_CONNECT, "unable to create client", e);
    }
  }

  private static GapisConnection createConnection(Listener listener) {
    if (gapis.get().isEmpty()) {
      return new GapisProcess() {
        @Override
        protected void onExit(int code) {
          super.onExit(code);
          listener.onServerExit(code);
        }
      }.connect();
    } else {
      return GapisConnection.create(
          gapis.get(), gapisAuthToken.get(), con -> listener.onServerExit(-1));
    }
  }

  /**
   * Requests, blocks, and then checks the server info.
   */
  private void fetchServerInfo()
      throws RpcException, TimeoutException, ExecutionException, GapisInitException {
    Service.ServerInfo info = Rpc.get(client.getSeverInfo(), FETCH_INFO_TIMEOUT_MS, MILLISECONDS);
    LOG.log(INFO, "Server info: {0}", info);
    Version gapisVersion = Version.fromProto(info);
    if (!GAPIC_VERSION.isCompatible(gapisVersion)) {
      throw new GapisInitException("Incompatible gapis version. Found: " + gapisVersion +
          ", wanted: " + GAPIC_VERSION.toPatternString(), "");
    }
    Info.setServerInfo(info);
  }

  /**
   * Requests, blocks, and then makes current the string table from the server.
   */
  private void fetchStringTable() throws ExecutionException, RpcException, TimeoutException {
    List<Stringtable.Info> infos =
        Rpc.get(client.getAvailableStringTables(), FETCH_STRING_TABLE_TIMEOUT_MS, MILLISECONDS);
    if (infos.size() == 0) {
      LOG.log(WARNING, "No string tables available");
      return;
    }
    Stringtable.Info info = infos.get(0);
    Stringtable.StringTable table =
        Rpc.get(client.getStringTable(info), FETCH_STRING_TABLE_TIMEOUT_MS, MILLISECONDS);
    Strings.setCurrent(table);
  }

  /**
   * Requests and blocks for the schema from the server.
   */
  private void fetchSchema() throws ExecutionException, RpcException, TimeoutException {
    Message schema = Rpc.get(client.getSchema(), FETCH_SCHEMA_TIMEOUT_MS, MILLISECONDS);
    LOG.log(INFO, "Schema with " + schema.entities.length + " classes, " +
        schema.constants.length + " constant sets");
    int atoms = 0;
    for (Entity type : schema.entities) {
      // Find the atom metadata, if present
      if (AtomMetadata.find(type) != null) {
        atoms++;
      }
      Dynamic.register(type);
    }
    LOG.log(INFO, "Schema with " + atoms + " atoms");
    for (ConstantSet set : schema.constants) {
      ConstantSet.register(set);
    }
  }

  public static interface Listener {
    public void onServerExit(int code);
  }

  public static class GapisInitException extends Exception {
    public static final String MESSAGE_FAILED_CONNECT = "Failed to connect to the graphics debugger";
    public static final String MESSAGE_FAILED_INIT = "Failed to initialize the graphics debugger";
    public static final String MESSAGE_TRACE_FILE_EMPTY = "Empty trace file ";
    public static final String MESSAGE_TRACE_FILE_BROKEN = "Invalid/Corrupted trace file ";
    public static final String MESSAGE_TRACE_FILE_LOAD_FAILED = "Failed to load trace file ";
    private final String userMessage;

    public GapisInitException(String userMessage, String debugMessage) {
      super(debugMessage);
      this.userMessage = userMessage;
    }

    public GapisInitException(String userMessage, String debugMessage, Throwable cause) {
      super(debugMessage, cause);
      this.userMessage = userMessage;
    }

    /**
     * @return The message to display to the user
     */
    @Override
    public String getLocalizedMessage() {
      return userMessage;
    }
  }
}
