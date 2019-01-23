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

/**
 * Status utilities
 */
public class StatusSummaryUpdater implements StatusWatcher.Listener {
    private final Listener listener;

    private static final String[] suffixes = {
      new String("B"),
      new String("KB"),
      new String("MB"),
      new String("GB"),
      new String("TB"),
      new String("PB"),
      new String("EB"),
    };
    public static String BytesToHuman(long bytes) {
      int i = 0;
      long remainderBytes = 0;
      while(bytes > 1024) {
        remainderBytes = bytes & 0x3FF;
        bytes >>= 10;
        i++;
      }
      if (i == 0) {
        return String.format("%d%s", bytes, suffixes[i]);
      } else {
        return String.format("%.3f%s", bytes + remainderBytes/1024.0, suffixes[i]);
      }
    }

    public class Summary {
      Summary(long u, long m, long n, String[] tsk, int prog) {
        usedMemory = u;
        maxMemory = m;
        numBlocked = n;
        longestTask = tsk;
        longestTaskProgress = prog;
      }
      public long usedMemory;
      public long maxMemory;
      public long numBlocked;

      public String[] longestTask;
      public int longestTaskProgress;
    }

    /** Callback interface */
    public interface Listener {
      /** Called whenever a new release is found. */
      void onStatusSummaryUpdated(Summary s);
    }
    
    public StatusSummaryUpdater(Listener l) {
      listener = l;
      currentTask = null;
    }
    
    public @Override void onAdd(StatusWatcher.Task t) {
      if (t.HasInLineage(currentTask) || currentTask == null) {
        currentTask = t;
        scheduleUpdate();
      }
    }

    public @Override void onRemove(StatusWatcher.Task t) {
      if (t == currentTask) {
        currentTask = StatusWatcher.getFirstTask(false);
      }
      scheduleUpdate();
    }

    public @Override void onUpdate(StatusWatcher.Task t) {
      if (t == currentTask) {
        scheduleUpdate();
      }
    }

    public @Override void onStatusChange(StatusWatcher.Task t) {
      if (t.IsBlocked()) {
        numBlocked += 1;
      } else {
        numBlocked -= 1;
      }
      if (t == currentTask) {
        scheduleUpdate();
      }
    }

    public @Override void onMemoryUpdate(long used, long max) {
      usedMemory = used;
      maxMemory = max;
      scheduleUpdate();
    }

    private void scheduleUpdate() {
      if (currentTask == null) {
        listener.onStatusSummaryUpdated(new Summary(
          usedMemory,
          maxMemory,
          numBlocked,
          new String[]{"Idle"},
          0
        ));
      } else {
        class Updater implements  StatusWatcher.Task.Runner {
          Updater(int length) {
            s = new String[length];
            i = 0;
          }
          String[] s;
          int i;
          public @Override void op(StatusWatcher.Task t) {
            s[i++] = t.getName();
          }

          public String[] getString() { return s; }
        }
        Updater u = new Updater(currentTask.getIndices().length - 1);
        currentTask.ForAncestorsAndSelf(u); 
        listener.onStatusSummaryUpdated(new Summary(
          usedMemory,
          maxMemory,
          numBlocked,
          u.getString(),
          currentTask.getProgress()
        ));
      }
    }

    private StatusWatcher.Task currentTask = null;
    private long usedMemory = 0;
    private long maxMemory = 0;
    private long numBlocked = 0;
}
