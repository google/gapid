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

import com.google.gapid.proto.service.Service;
import java.util.TreeMap;
import java.util.Map;
import java.util.HashMap;
import java.util.Arrays;

/**
 * Status utilities
 */
public class StatusWatcher {
  enum State {
    RUNNING,
    BLOCKED,
    ROOT
  }

  public static class Task {
    Task(Task p, long[] i, String n, boolean background) {
      indices = i;
      children = new TreeMap<Long, Task>();
      progress = 0;
      state = State.RUNNING;
      parent = p;
      name = n;
      this.background = background;

    }
    Task() { 
      indices = new long[]{0};
      children = new TreeMap<Long, Task>();
      progress = 0;
      state = State.ROOT;
      parent = null;
      name = "Root";
      this.background = false;
    }


    static interface Runner {
      public void op(Task t);
    }

    public synchronized void ForAncestorsAndSelf(Runner op) {
      if (state == State.ROOT) {
        return;
      }
      if (parent != null) {
        parent.ForAncestorsAndSelf(op);
      }
      op.op(this);
    }


    public synchronized void ForChildrenAndSelf(Runner op) {
      for(Map.Entry<Long,Task> entry : children.entrySet()) {
        entry.getValue().ForAncestorsAndSelf(op);
      }
      if (state == State.ROOT) {
        return;
      }
      op.op(this);
    }



    private final long[] indices;
    private final Task parent;
    private final String name;

    public long[] getIndices() {
      return indices;
    }

    public String getName() {
      return name;
    }

    private TreeMap<Long, Task> children;
    private int progress;
    private State state;
    private boolean background;
    
    public synchronized boolean HasInLineage(Task p) {
      
      if (p == null) { 
        return false;
      }
      if (p == this) {
        return true;
      }
      if (parent != null) {
        return parent.HasInLineage(p);
      }
      return false;
    }

    public boolean IsBackground(){ 
      return background;
    }


    public synchronized void SetBlocked(boolean newVal){ 
      state = newVal? State.BLOCKED: State.RUNNING;
    }

    public synchronized boolean IsBlocked() {
      return state == State.BLOCKED;
    }

    public synchronized void UpdateProgress(int newVal) {
      progress = newVal;
    }

    public synchronized int getProgress() {
      return progress;
    }

    public synchronized void AddChild(Task p) {
      this.children.put(p.indices[p.indices.length - 1], p);
    }

    public synchronized void RemoveFromParent() {
      if (parent != null) {
        parent.RemoveChild(this);
      }
    }

    public synchronized void RemoveChild(Task p) {
      // This is our child
      this.children.remove(p.indices[p.indices.length-1]);
    }

    public synchronized Task GetLeftmostDescendant(boolean includeBackground) {
      if (children.size() == 0) {
        return this;
      }
      
      for(Map.Entry<Long,Task> entry : children.entrySet()) {
        if (!entry.getValue().IsBlocked() && (!entry.getValue().IsBackground() || includeBackground)) {
          return entry.getValue().GetLeftmostDescendant(includeBackground);
        }
      }
      
      return this;
    }
  }

  /**
   * Listener is the interface implemented by types that want to listen to status updates
   */
  public interface Listener {
    public void onAdd(Task t);
    public void onRemove(Task t);
    public void onUpdate(Task t);
    public void onStatusChange(Task t);
    public void onMemoryUpdate(long used, long max);
  }

  private static final Listener NULL_LISTENER = new Listener()
  {
    public @Override void onAdd(Task t) { /* intentionally empty */ }
    public @Override void onRemove(Task t)  { /* intentionally empty */ }
    public @Override void onUpdate(Task t)  { /* intentionally empty */ }
    public @Override void onStatusChange(Task t)  { /* intentionally empty */ }
    public @Override void onMemoryUpdate(long used, long max)  { /* intentionally empty */ }
  };

  private static Listener listener = NULL_LISTENER;
  private static final Task rootTask = new Task();
  private static final Task currentActiveTask = rootTask;
  private static final HashMap<Long, Task> allTasks = new HashMap<Long, Task>();
  private static long usedMemory = 0;
  private static long maxMemory = 0;

  public static void init() { 
    allTasks.put((long)0, rootTask);
  }

  public static void setListener(Listener newListener) {
    listener = (newListener == null) ? NULL_LISTENER : newListener;
  }

  private static void addTask(Service.TaskUpdate task) {
    long parent = task.getParent();
    Task parentTask = allTasks.get(parent);
    if (parentTask == null) {
      parentTask = rootTask;
    }

    long[] p_indices = parentTask.getIndices();
    long[] indices = new long[p_indices.length + 1];
    for (int i = 0; i < p_indices.length; i++) {
      indices[i] = p_indices[i];
    }
    indices[indices.length - 1] = task.getId();

    Task nt = new Task(
      parentTask,
      indices,
      task.getName(),
      task.getBackground()
    );

    parentTask.AddChild(nt);
    listener.onAdd(nt);
    allTasks.put(task.getId(), nt);
  }

  private static Task getTask(Service.TaskUpdate task) {
    long id = task.getId();
    return allTasks.get(id);
  }

  private static void removeTask(Service.TaskUpdate task) {
    Task t = getTask(task);
    if (t == null) {
      return;
    }
    t.RemoveFromParent();
    t.ForChildrenAndSelf((tsk) -> listener.onRemove(tsk));
  }

  private static void updateTask(Service.TaskUpdate task) {
    Task t = getTask(task);
    if (t == null) {
      return;
    }
    t.UpdateProgress(task.getCompletePercent());
    listener.onUpdate(t);
  }

  private static void onStatusChange(Service.TaskUpdate task) {
    Task t = getTask(task);
    if (t == null) {
      return;
    }
    if (task.getStatus() == Service.TaskStatus.BLOCKED) {
      t.ForAncestorsAndSelf((tsk) -> tsk.SetBlocked(true));
    } else if (task.getStatus() == Service.TaskStatus.UNBLOCKED) {
      t.ForAncestorsAndSelf((tsk) -> tsk.SetBlocked(false));
    }
    t.ForAncestorsAndSelf((tsk) -> listener.onStatusChange(tsk));
  }

  public static Task getFirstTask(boolean includeBackground) {
    Task t = rootTask.GetLeftmostDescendant(includeBackground);
    if (t == rootTask) {
      return null;
    }
    return t;
  }

  private static void processTask(Service.TaskUpdate task) {
    switch(task.getStatus()) {
      case STARTING: {
        addTask(task);
        break;
      }
      case FINISHED: {
        removeTask(task);
      } break;
      case PROGRESS: {
        updateTask(task);
      } break;
      case BLOCKED:
      case UNBLOCKED: {
        onStatusChange(task);
      } break;
      case EVENT: {
      } break;
      default:
        break;
    }
  }

  private static void processMemory(Service.MemoryStatus mem) {
    usedMemory = mem.getTotalHeap();
    if (usedMemory > maxMemory) {
      maxMemory = usedMemory;
    }
    listener.onMemoryUpdate(usedMemory, maxMemory);
  }

  public static void notifyMessage(Service.ServerStatusResponse message) {
    if (message.hasMemory()) {
      processMemory(message.getMemory());
    } else if (message.hasTask()) {
      processTask(message.getTask());
    }
  }
}
