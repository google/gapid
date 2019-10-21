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
package com.google.gapid.util;

import com.google.common.collect.Sets;
import com.google.gapid.widgets.Widgets;

import org.eclipse.swt.SWT;
import org.eclipse.swt.widgets.Widget;

import java.util.Set;
import java.util.concurrent.atomic.AtomicInteger;
import java.util.function.Consumer;

public class Keyboard {
  private final int delay;
  private final Set<Integer> down = Sets.newHashSet();
  private int modifiers = 0;
  private final AtomicInteger scheduled = new AtomicInteger();

  public Keyboard(Widget owner, int delay, Consumer<Keyboard> onKey) {
    this.delay = delay;
    owner.addListener(SWT.KeyDown, e -> {
      if ((e.keyCode & SWT.MODIFIER_MASK) == e.keyCode) {
        modifiers = e.stateMask | e.keyCode;
      } else if (down.add(e.keyCode)) {
        modifiers = e.stateMask;
        schedule(owner, onKey);
      }
    });

    owner.getDisplay().addFilter(SWT.KeyUp, e -> {
      if ((e.keyCode & SWT.MODIFIER_MASK) == e.keyCode) {
        modifiers = e.stateMask & ~e.keyCode;
      } else {
        down.remove(e.keyCode);
        if (down.isEmpty()) {
          scheduled.incrementAndGet(); // cancel any next onKey.
        }
      }
    });
  }

  private void schedule(Widget owner, Consumer<Keyboard> onKey) {
    onKey.accept(this);
    int id = scheduled.incrementAndGet();
    Widgets.scheduleIfNotDisposed(owner, delay, () -> {
      if (scheduled.get() == id) {
        schedule(owner, onKey);
      }
    });
  }

  public boolean isKeyDown(int code) {
    return down.contains(code);
  }

  public boolean hasMod(int mask) {
    return (modifiers & mask) == mask;
  }
}
