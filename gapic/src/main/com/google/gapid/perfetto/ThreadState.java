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
package com.google.gapid.perfetto;

import static com.google.common.base.Preconditions.checkNotNull;

import com.google.gapid.perfetto.views.StyleConstants.Palette.BaseColor;

import org.eclipse.swt.graphics.RGBA;

import java.util.function.Supplier;

/**
 * Execution state a thread can be in.
 */
public class ThreadState {
  public static final ThreadState DEBUG = new ThreadState(
      "Debug", () -> BaseColor.MAGENTA.rgb, 9);
  public static final ThreadState EXIT_DEAD = new ThreadState(
      "Exit Dead", () -> BaseColor.MAGENTA.rgb, 8);
  public static final ThreadState RUNNABLE = new ThreadState(
      "Runnable", () -> BaseColor.GREEN.rgb, 1);
  public static final ThreadState RUNNING = new ThreadState(
      "Running", () -> BaseColor.PACIFIC_BLUE.rgb, 0);
  public static final ThreadState SLEEPING = new ThreadState(
      "Sleeping", () -> BaseColor.GREY.rgb, 5);
  public static final ThreadState STOPPED = new ThreadState(
      "Stopped", () -> BaseColor.MAGENTA.rgb, 8);
  public static final ThreadState TASK_DEAD = new ThreadState(
      "Task Dead", () -> BaseColor.MAGENTA.rgb, 8);
  public static final ThreadState UNINTR_SLEEP = new ThreadState(
      "Uninterruptible Sleep", () -> BaseColor.MAGENTA.rgb, 6);
  public static final ThreadState UNINTR_SLEEP_WAKE_KILL = new ThreadState(
      "Uninterruptible Sleep | WakeKill", () -> BaseColor.MAGENTA.rgb, 7);
  public static final ThreadState UNINTR_SLEEP_WAKING = new ThreadState(
      "Uninterruptible Sleep | Waking", () -> BaseColor.MAGENTA.rgb, 7);
  public static final ThreadState UNINTR_SLEEP_IO = new ThreadState(
      "Uninterruptible Sleep - Block I/O", () -> BaseColor.ORANGE.rgb, 3);
  public static final ThreadState UNINTR_SLEEP_WAKE_KILL_IO = new ThreadState(
      "Uninterruptible Sleep | WakeKill - Block I/O", () -> BaseColor.ORANGE.rgb, 4);
  public static final ThreadState UNINTR_SLEEP_WAKING_IO = new ThreadState(
      "Uninterruptible Sleep | Waking - Block I/O", () -> BaseColor.ORANGE.rgb, 4);
  public static final ThreadState WAKE_KILL = new ThreadState(
      "Wakekill", () -> BaseColor.MAGENTA.rgb, 8);
  public static final ThreadState WAKING = new ThreadState(
      "Waking", () -> BaseColor.MAGENTA.rgb, 8);
  public static final ThreadState ZOMBIE = new ThreadState(
      "Zombie", () -> BaseColor.MAGENTA.rgb, 8);
  public static final ThreadState NONE = new ThreadState(
      "", () -> BaseColor.DARK_ORANGE.rgb, 10);

  public final String label;
  public final Supplier<RGBA> color;
  public final int mergePriority;

  private ThreadState(String label, Supplier<RGBA> color, int mergePriority) {
    this.label = checkNotNull(label);
    this.color = color;
    this.mergePriority = mergePriority;
  }

  public ThreadState merge(ThreadState other) {
    return (mergePriority <= other.mergePriority) ? this : other;
  }

  public static ThreadState of(String state) {
    switch (state) {
      case "D":
        return UNINTR_SLEEP;
      case "DK":
        return UNINTR_SLEEP_WAKE_KILL;
      case "DW":
        return UNINTR_SLEEP_WAKING;
      case "K":
        return WAKE_KILL;
      case "r":
        return RUNNING;
      case "R":
      case "R+":
        return RUNNABLE;
      case "S":
        return SLEEPING;
      case "t":
        return DEBUG;
      case "W":
        return WAKING;
      case "X":
        return EXIT_DEAD;
      case "x":
        return TASK_DEAD;
      case "Z":
        return ZOMBIE;
      default:
        return new ThreadState("Unknown (" + state + ")", () -> BaseColor.DARK_ORANGE.rgb, 10);
    }
  }

  @Override
  public boolean equals(Object obj) {
    if (obj == this) {
      return true;
    } else if (!(obj instanceof ThreadState)) {
      return false;
    }
    return label.equals(((ThreadState)obj).label);
  }

  @Override
  public int hashCode() {
    return label.hashCode();
  }
}
