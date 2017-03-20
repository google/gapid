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

import static com.google.gapid.util.Ranges.commands;
import static com.google.gapid.util.Ranges.first;
import static com.google.gapid.util.Ranges.last;

import com.google.common.base.Objects;
import com.google.gapid.models.ApiContext.FilteringContext;
import com.google.gapid.proto.service.Service.CommandRange;
import com.google.gapid.proto.service.Service.Value;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.server.Client;
import com.google.gapid.service.atom.Atom;
import com.google.gapid.service.atom.AtomList;
import com.google.gapid.service.atom.Observation;
import com.google.gapid.service.atom.Observations;
import com.google.gapid.service.memory.MemoryRange;
import com.google.gapid.util.Events;
import com.google.gapid.util.Events.ListenerCollection;
import com.google.gapid.util.Messages;
import com.google.gapid.util.Paths;

import org.eclipse.swt.widgets.Shell;

import java.io.IOException;
import java.util.logging.Logger;

/**
 * Model containing the API commands (atoms) of the capture.
 */
public class AtomStream extends CaptureDependentModel<AtomList> implements ApiContext.Listener {
  private static final Logger LOG = Logger.getLogger(AtomStream.class.getName());

  private final ApiContext context;
  private final ListenerCollection<Listener> listeners = Events.listeners(Listener.class);
  private CommandRange selection;

  public AtomStream(Shell shell, Client client, Capture capture, ApiContext context) {
    super(LOG, shell, client, capture);
    this.context = context;

    context.addListener(this);
  }

  @Override
  protected void reset(boolean maintainState) {
    super.reset(maintainState);
    if (!maintainState) {
      selection = null;
    }
  }

  @Override
  protected Path.Any getPath(Path.Capture capturePath) {
    return Path.Any.newBuilder()
        .setCommands(Path.Commands.newBuilder()
            .setCapture(capturePath))
        .build();
  }

  @Override
  protected AtomList unbox(Value value) throws IOException {
    return Client.decode(value.getObject());
  }

  @Override
  protected void fireLoadEvent() {
    listeners.fire().onAtomsLoaded();
    if (selection != null) {
      listeners.fire().onAtomsSelected(selection);
    }
  }

  @Override
  public void onContextSelected(FilteringContext ctx) {
    if (selection != null && !ctx.contains(selection)) {
      if (ctx.contains(last(selection))) {
        selectAtoms(last(selection), 1, false);
      } else {
        selectAtoms(ctx.findClosest(selection), false);
      }
    }
  }

  public int getAtomCount() {
    return getData().getAtoms().length;
  }

  public Atom getAtom(long index) {
    return getData().get(index);
  }

  /**
   * @return the index of the first command of the frame that contains the given command.
   */
  public int getStartOfFrame(long index) {
    Atom[] atoms = getData().getAtoms();
    for (int i = (int)index; i > 0; i--) {
      if (atoms[i - 1].isEndOfFrame()) {
        return i;
      }
    }
    return 0;
  }

  /**
   * @retrn the index of the last command of the frame that contains the given command.
   */
  public int getEndOfFrame(long index) {
    Atom[] atoms = getData().getAtoms();
    for (int i = (int)index; i < atoms.length; i++) {
      if (atoms[i].isEndOfFrame()) {
        return i;
      }
    }
    return atoms.length - 1;
  }

  public CommandRange getSelectedAtoms() {
    return selection;
  }

  public void selectAtoms(long from, long count, boolean force) {
    selectAtoms(commands(from, count), force);
  }

  public void selectAtoms(CommandRange range, boolean force) {
    if (force || !Objects.equal(selection, range)) {
      selection = range;
      context.selectContextContaining(range);
      listeners.fire().onAtomsSelected(selection);
    }
  }

  public Atom getFirstSelectedAtom() {
    return (selection == null || getData() == null) ? null : getData().get(first(selection));
  }

  public Atom getLastSelectedAtom() {
    return (selection == null || getData() == null) ? null : getData().get(last(selection));
  }

  /**
   * @return the path to the last draw command within the current selection or {@code null}.
   */
  public Path.Command getLastSelectedDrawCall() {
    if (selection == null || getData() == null) {
      return null;
    }

    FilteringContext selectedContext = context.getSelectedContext();
    for (long index = last(selection); index >= first(selection); index--) {
      if (selectedContext.contains(index) && getData().get(index).isDrawCall()) {
        return Path.Command.newBuilder()
            .setCommands(getPath().getCommands())
            .setIndex(index)
            .build();
      }
    }
    return null;
  }

  public TypedObservation[] getObservations(long index) {
    Atom atom = getAtom(index);
    if (atom.getObservationCount() == 0) {
      return NO_OBSERVATIONS;
    }

    Observations obs = atom.getObservations();
    TypedObservation[] result = new TypedObservation[obs.getReads().length + obs.getWrites().length];
    for (int i = 0; i < obs.getReads().length; i++) {
      result[i] = new TypedObservation(index, true, obs.getReads()[i]);
    }
    for (int i = 0, j = obs.getReads().length; i < obs.getWrites().length; i++, j++) {
      result[j] = new TypedObservation(index, false, obs.getWrites()[i]);
    }
    return result;
  }

  private static final TypedObservation[] NO_OBSERVATIONS = new TypedObservation[0];

  /**
   * Read or write memory observation at a specific command.
   */
  public static class TypedObservation {
    public static final TypedObservation NULL_OBSERVATION = new TypedObservation(
        0, false, new Observation().setRange(new MemoryRange().setBase(0).setSize(0))) {
      @Override
      public String toString() {
        return Messages.SELECT_OBSERVATION;
      }
    };

    private final long index;
    private final boolean read;
    private final Observation observation;

    public TypedObservation(long index, boolean read, Observation observation) {
      this.index = index;
      this.read = read;
      this.observation = observation;
    }

    public Path.Memory getPath(AtomStream atoms) {
      return Paths.memoryAfter(atoms.getPath(), index, 0, observation.getRange()).getMemory();
    }

    public boolean contains(long address) {
      return observation.getRange().contains(address);
    }

    @Override
    public String toString() {
      long base = observation.getRange().getBase(), count = observation.getRange().getSize();
      return (read ? "Read " : "Write ") + count + " byte" + (count == 1 ? "" : "s") +
          String.format(" at 0x%016x", base);
    }
  }

  public void addListener(Listener listener) {
    listeners.addListener(listener);
  }

  public void removeListener(Listener listener) {
    listeners.removeListener(listener);
  }

  public interface Listener extends Events.Listener {
    /**
     * Event indicating that the commands have finished loading.
     */
    public default void onAtomsLoaded() { /* empty */ }

    /**
     * Event indicating that the currently selected command range has changed.
     */
    @SuppressWarnings("unused")
    public default void onAtomsSelected(CommandRange range) { /* empty */ }
  }
}
