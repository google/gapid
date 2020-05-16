/*
 * Copyright (C) 2019 Google Inc.
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
package com.google.gapid.perfetto.views;

import static com.google.gapid.perfetto.TimeSpan.timeToString;
import static com.google.gapid.widgets.Widgets.createBoldLabel;
import static com.google.gapid.widgets.Widgets.createLabel;
import static com.google.gapid.widgets.Widgets.createTreeColumn;
import static com.google.gapid.widgets.Widgets.createTreeViewer;
import static com.google.gapid.widgets.Widgets.packColumns;
import static com.google.gapid.widgets.Widgets.withLayoutData;
import static com.google.gapid.widgets.Widgets.withSpans;

import com.google.gapid.perfetto.ThreadState;
import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.perfetto.models.CpuTrack;
import com.google.gapid.perfetto.models.CpuTrack.ByThread;
import com.google.gapid.perfetto.models.ProcessInfo;
import com.google.gapid.perfetto.models.ThreadInfo;

import org.eclipse.jface.viewers.ITreeContentProvider;
import org.eclipse.jface.viewers.LabelProvider;
import org.eclipse.jface.viewers.TreeViewer;
import org.eclipse.swt.SWT;
import org.eclipse.swt.layout.FillLayout;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Composite;

/**
 * Displays information about a list of selected CPU slices.
 */
public class CpuSlicesSelectionView extends Composite {
  public CpuSlicesSelectionView(Composite parent, State state, CpuTrack.Slices slices) {
    super(parent, SWT.NONE);
    if (slices.getCount() == 1) {
      setSingleSliceView(state, slices);
    } else if (slices.getCount() > 1) {
      setMultiSlicesView(state, slices);
    }
  }

  private void setSingleSliceView(State state, CpuTrack.Slices slice) {
    setLayout(new GridLayout(2, false));

    withLayoutData(createBoldLabel(this, "Slice:"), withSpans(new GridData(), 2, 1));

    createLabel(this, "Time:");
    createLabel(this, timeToString(slice.times.get(0) - state.getTraceTime().start));

    createLabel(this, "Duration:");
    createLabel(this, timeToString(slice.durs.get(0)));

    ThreadInfo thread = state.getThreadInfo(slice.utids.get(0));
    if (thread != null) {
      ProcessInfo process = state.getProcessInfo(thread.upid);
      if (process != null) {
        createLabel(this, "Process:");
        createLabel(this, process.getDisplay());
      }

      createLabel(this, "Thread:");
      createLabel(this, thread.getDisplay());
    }

    createLabel(this, "State:");
    createLabel(this, ThreadState.RUNNING.label);

    createLabel(this, "CPU:");
    createLabel(this, Integer.toString(slice.cpus.get(0) + 1));

    createLabel(this, "End State:");
    createLabel(this, slice.endStates.get(0).label);

    createLabel(this, "Priority:");
    createLabel(this, Integer.toString(slice.priorities.get(0)));
  }

  private void setMultiSlicesView(State state, CpuTrack.Slices slices) {
    setLayout(new FillLayout());

    CpuTrack.ByProcess[] byProcesses = CpuTrack.organizeSlicesByProcess(slices);

    TreeViewer viewer = createTreeViewer(this, SWT.NONE);
    viewer.getTree().setHeaderVisible(true);
    viewer.setContentProvider(new ITreeContentProvider() {
      @Override
      public Object[] getElements(Object inputElement) {
        return byProcesses;
      }

      @Override
      public boolean hasChildren(Object element) {
        return (element instanceof CpuTrack.ByProcess) ||
            (element instanceof CpuTrack.ByThread);
      }

      @Override
      public Object getParent(Object element) {
        return null;
      }

      @Override
      public Object[] getChildren(Object element) {
        if (element instanceof CpuTrack.ByProcess) {
          return ((CpuTrack.ByProcess)element).threads.toArray();
        } else if (element instanceof CpuTrack.ByThread) {
          return ((ByThread)element).sliceIndexes.toArray();
        }
        return null;
      }
    });
    viewer.setLabelProvider(new LabelProvider());

    createTreeColumn(viewer, "Name", el -> {
      if (el instanceof CpuTrack.ByProcess) {
        long pid = ((CpuTrack.ByProcess)el).pid;
        ProcessInfo pi = state.getProcessInfo(pid);
        return (pi == null) ? "<unknown process> [" + pid + "]" : pi.getDisplay();
      } else if (el instanceof CpuTrack.ByThread) {
        long tid = ((CpuTrack.ByThread)el).tid;
        ThreadInfo ti = state.getThreadInfo(tid);
        return (ti == null) ? "<unknown thread> [" + tid + "]" : ti.getDisplay();
      } else {
        return "Slice " + slices.ids.get((int)el);
      }
    });
    createTreeColumn(viewer, "Slice Duration", el -> {
      if (el instanceof CpuTrack.ByProcess) {
        return TimeSpan.timeToString(((CpuTrack.ByProcess)el).dur);
      } else if (el instanceof CpuTrack.ByThread) {
        return TimeSpan.timeToString(((CpuTrack.ByThread)el).dur);
      } else {
        return TimeSpan.timeToString(slices.durs.get((int)el));
      }
    });
    createTreeColumn(viewer, "Slice Start Time", el -> {
      if (el instanceof Integer) {
        return TimeSpan.timeToString(slices.times.get((int)el) - state.getTraceTime().start);
      } else {
        return "";
      }
    });
    createTreeColumn(viewer, "End State", el -> {
      if (el instanceof Integer) {
        return slices.endStates.get((int)el).label;
      } else {
        return "";
      }
    });
    createTreeColumn(viewer, "Priority", el -> {
      if (el instanceof Integer) {
        return String.valueOf(slices.priorities.get((int)el));
      } else {
        return "";
      }
    });
    viewer.setInput(slices);
    packColumns(viewer.getTree());
  }
}
