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

  public static interface Listener {
    public void onStatus(String status);
    public void onHeap(long heap);
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
}
