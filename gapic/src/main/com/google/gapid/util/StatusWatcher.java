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
package com.google.gapid.util;

import com.google.common.collect.Maps;
import com.google.gapid.proto.service.Service;
import com.google.gapid.server.Client;

import java.util.HashMap;
import java.util.LinkedHashMap;

/**
 * Utility class for monitoring the server status.
 */
public class StatusWatcher {
  private static final float STATUS_UPDATE_INTERVAL_S = 0.5f;
  private static final float MEMORY_UPDATE_INTERVAL_S = 1.0f;

  private final Listener listener;
  private final Task root = Task.newRoot();
  private final HashMap<Long, Task> tasks = Maps.newHashMap();
  private final ReplaySummary replays = new ReplaySummary();
  private String shownSummary = "";

  public StatusWatcher(Client client, Listener listener) {
    this.listener = listener;

    client.streamStatus(MEMORY_UPDATE_INTERVAL_S, STATUS_UPDATE_INTERVAL_S, update -> {
      switch (update.getResCase()) {
        case TASK:
          onTaskUpdate(update.getTask());
          break;
        case MEMORY:
          onMemoryUpdate(update.getMemory());
          break;
        case REPLAY:
          onReplayUpdate(update.getReplay());
          break;
        default:
          // Ignore.
      }
    });
  }

  private void onTaskUpdate(Service.TaskUpdate update) {
    String summary;
    synchronized (this) {
      switch(update.getStatus()) {
        case STARTING:
          Task parent = tasks.getOrDefault(update.getParent(), root);
          Task child = new Task(update, parent);
          tasks.put(update.getId(), child);
          parent.addChild(child);
          break;
        case FINISHED:
          tasks.getOrDefault(update.getId(), root).remove();
          break;
        case PROGRESS:
          tasks.getOrDefault(update.getId(), root).setProgress(update.getCompletePercent());
          break;
        case BLOCKED:
          tasks.getOrDefault(update.getId(), root).setBlocked(true);
          break;
        case UNBLOCKED:
          tasks.getOrDefault(update.getId(), root).setBlocked(false);
          break;
        default:
          // Ignore.
          return;
      }

      summary = root.getFirstChild(Task.State.RUNNING).getStatusLabel();
      if (shownSummary.equals(summary)) {
        return;
      }
      shownSummary = summary;
    }

    listener.onStatus(summary);
  }

  private void onMemoryUpdate(Service.MemoryStatus update) {
    listener.onHeap(update.getTotalHeap());
  }

  private void onReplayUpdate(Service.ReplayUpdate update) {
    if (replays.update(update)) {
      listener.onReplayProgress(replays.getSummary());
    }
  }

  public static interface Listener {
    public void onStatus(String status);
    public void onHeap(long heap);
    public void onReplayProgress(String status);
  }

  private static class Task {
    private final long id;
    private final Task parent;
    private final String name;
    private final LinkedHashMap<Long, Task> children = Maps.newLinkedHashMap();
    private State state;
    private int progress;

    public Task(long id, Task parent, String name, State state, int progress) {
      this.id = id;
      this.parent = parent;
      this.name = name;
      this.state = state;
      this.progress = progress;
    }

    public Task(Service.TaskUpdate update, Task parent) {
      this(update.getId(), parent, update.getName(), getState(update), update.getCompletePercent());
    }

    private static State getState(Service.TaskUpdate update) {
      return (update.getBackground()) ? State.BACKGROUND : State.RUNNING;
    }

    public static Task newRoot() {
      return new Task(-1, null, null, null, 0) {
        @Override
        public void setBlocked(boolean newVal) {
          // Don't do anything.
        }

        @Override
        public void remove() {
          // Don't do anything.
        }

        @Override
        public String getLabel() {
          return "";
        }
      };
    }

    public void setBlocked(boolean blocked){
      if (state != State.BACKGROUND) {
        state = blocked ? State.BLOCKED: State.RUNNING;
        parent.setBlocked(blocked);
      }
    }

    public void setProgress(int progress) {
      this.progress = progress;
    }

    public void addChild(Task child) {
      children.put(child.id, child);
    }

    public void remove() {
      parent.children.remove(id);
    }

    public Task getFirstChild(State inState) {
      for (Task child : children.values()) {
        if (child.state == inState) {
          return child;
        }
      }
      return this;
    }

    public Task getLeftMostDecendant(State inState) {
      for (Task child : children.values()) {
        if (child.state == inState) {
          return child.getLeftMostDecendant(inState);
        }
      }
      return this;
    }

    public String getLabel() {
      return (progress == 0) ? name : name + "<" + progress + "%>";
    }

    public String getStatusLabel() {
      Task leaf = getLeftMostDecendant(State.RUNNING);
      if (leaf == this) {
        return getLabel();
      } else {
        return getLabel() + " ... " + leaf.getLabel();
      }
    }

    private static enum State {
      BACKGROUND, RUNNING, BLOCKED,
    }
  }

  private static class ReplaySummary {
    private final HashMap<Integer, Replay> replays = Maps.newHashMap();

    private int queued = 0;
    private int started = 0;
    private int executing = 0;
    private long doneInstr = 0;
    private long totalInstr = 0;

    public ReplaySummary() {
    }

    public synchronized String getSummary() {
      StringBuilder sb = new StringBuilder();
      String sep = "";
      if (queued > 0) {
        sb.append(queued).append(" Queued");
        sep = ", ";
      }
      if (started > 0) {
        sb.append(sep).append(started).append(" Building");
        sep = ", ";
      }
      if (executing > 0) {
        sb.append(sep).append(executing).append(" Running");
        if (totalInstr > 0 && doneInstr > 0) {
          sb.append(" ").append((int)(((double)doneInstr / totalInstr) * 100)).append("%");
        } else {
          // TODO(pmuetschard): This assumes that 0 done means state reconstruction. See server side
          // for more details.
          sb.append(" - Initializing");
        }
      }
      return (sb.length() == 0) ? "Idle" : sb.toString();
    }

    public synchronized boolean update(Service.ReplayUpdate update) {
      Replay replay = replays.get(update.getReplayId());
      if (replay == null) {
        if (update.getStatus() == Service.ReplayStatus.REPLAY_FINISHED) {
          // Got a replay finished message for a replay we've not seen before. Just ignore it.
          return false;
        }
        replay = new Replay(update);
        replays.put(update.getReplayId(), replay);

        switch (replay.status) {
          case REPLAY_QUEUED:
            queued++;
            return true;
          case REPLAY_STARTED:
            started++;
            return true;
          case REPLAY_EXECUTING:
            executing++;
            doneInstr += replay.doneInstr;
            totalInstr += replay.totalInstr;
            return true;
          default:
            return false;
        }
      }

      switch (replay.status) {
        case REPLAY_QUEUED:
          switch (update.getStatus()) {
            case REPLAY_STARTED:
              queued--;
              return started(replay);
            case REPLAY_EXECUTING:
              queued--;
              return executing(replay, update);
            case REPLAY_FINISHED:
              queued--;
              return finished(replay);
            default:
              return false;
          }
        case REPLAY_STARTED:
          switch (update.getStatus()) {
            case REPLAY_EXECUTING:
              started--;
              return executing(replay, update);
            case REPLAY_FINISHED:
              started--;
              return finished(replay);
            default:
              return false;
          }
        case REPLAY_EXECUTING:
          switch (update.getStatus()) {
            case REPLAY_EXECUTING:
              return executing(replay, update);
            case REPLAY_FINISHED:
              executing--;
              return finished(replay);
            default:
              return false;
          }
        default:
          return false;
      }
    }

    private boolean started(Replay replay) {
      started++;
      replay.status = Service.ReplayStatus.REPLAY_STARTED;
      return true;
    }

    private boolean executing(Replay replay, Service.ReplayUpdate update) {
      int done = update.getFinishedInstrs();
      int total = update.getTotalInstrs();
      if (replay.status != Service.ReplayStatus.REPLAY_EXECUTING) {
        executing++;
        doneInstr += done;
        totalInstr += total;
      } else {
        doneInstr += done - replay.doneInstr;
        totalInstr += total - replay.totalInstr;
      }
      replay.status = Service.ReplayStatus.REPLAY_EXECUTING;
      replay.doneInstr = done;
      replay.totalInstr = total;
      return true;
    }

    private boolean finished(Replay replay) {
      if (replay.status == Service.ReplayStatus.REPLAY_EXECUTING) {
        doneInstr -= replay.doneInstr;
        totalInstr -= replay.totalInstr;
      }
      // TODO: we should clean these out after some time.
      replay.status = Service.ReplayStatus.REPLAY_FINISHED;
      return true;
    }

    private static class Replay {
      public Service.ReplayStatus status;
      public int doneInstr;
      public int totalInstr;

      public Replay(Service.ReplayUpdate update) {
        this.status = update.getStatus();
        this.doneInstr = update.getFinishedInstrs();
        this.totalInstr = update.getTotalInstrs();
      }
    }
  }
}
