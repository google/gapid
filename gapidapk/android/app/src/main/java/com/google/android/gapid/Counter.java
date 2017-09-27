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

package com.google.android.gapid;

import java.util.ArrayList;
import java.util.Collections;
import java.util.Comparator;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

/**
 * Counter is a basic multi-threaded hierarchical scope based profiler.
 */
public class Counter {
    private static final Counter ROOT = new Counter("<root>", null);
    private static final ThreadLocal<Counter> CURRENT = new ThreadLocal<>();

    private final Map<String, Counter> children = new HashMap<>();
    private final String name;
    private final Counter parent;
    private int count = 0;
    private float time = 0;

    /**
     * time is used to profile the amount of time spent in the given scope, identified by the given
     * name.
     *
     * It is to be used with a try-with-resources block:
     * <code>
     *   try (Counter.Scope t = Counter.time("doSomething")) {
     *     doSomething();
     *   }
     * </code>
     *
     * @param name the scope name.
     * @return an {@link AutoCloseable} scope.
     */
    public static Scope time(String name) {
        Counter counter = CURRENT.get();
        if (counter == null) {
            counter = ROOT.child("Thread " + Thread.currentThread().getName());
            CURRENT.set(counter);
        }
        return new Scope(counter.child(name));
    }

    /**
     * Collect returns the timing information about all counters, across all threads.
     * If reset is true, then the counters will be all cleared back to 0.
     */
    public static String collect(boolean reset) {
        StringBuilder sb = new StringBuilder();
        synchronized (ROOT) {
            for (Counter thread : ROOT.children.values()) {
                thread.write(sb, 0);
            }
        }
        if (reset) {
            ROOT.reset();
        }
        return sb.toString();
    }

    private Counter(String name, Counter parent) {
        this.name = name;
        this.parent = parent;
    }

    private synchronized Counter child(String name) {
        Counter counter = children.get(name);
        if (counter == null) {
            counter = new Counter(name, this);
            children.put(name, counter);
        }
        return counter;
    }

    private void enter() {
        CURRENT.set(this);
    }

    private void exit(float duration) {
        synchronized (this) {
            time += duration;
            count++;
        }
        CURRENT.set(parent);
    }

    private static void indent(StringBuilder sb, int depth) {
        for (int i = 0; i < depth; i++) {
            sb.append("    ");
        }
    }

    private synchronized void reset() {
        children.clear();
        count = 0;
        time = 0;
    }

    private synchronized void write(StringBuilder sb, int depth) {
        if (count > 0) {
            indent(sb, depth);
            if (parent.time != 0) {
                sb.append((int)(100 * time / parent.time)).append("% ");
            }

            sb.append(name).append(" ")
                    .append("[total: ").append(time)
                    .append(" count: ").append(count)
                    .append(" average: ").append(time / count)
                    .append("]\n");
        } else {
            sb.append(name).append("\n");
        }

        if (children.size() > 0) {
            List<Counter> sorted = new ArrayList<>();
            sorted.addAll(children.values());
            Collections.sort(sorted, new Comparator<Counter>() {
                @Override
                public int compare(Counter a, Counter b) {
                    return Float.compare(a.time, b.time);
                }
            });
            for (Counter child : sorted) {
                child.write(sb, depth+1);
            }

            float other = time;
            for (Counter child : children.values()) {
                other -= child.time;
            }
            if (other > 0) {
                indent(sb, depth+1);
                sb.append((int)(100 * other / time)).append("% <other>\n");
            }
        }
    }

    public static class Scope implements AutoCloseable {
        private final long start;
        private final Counter counter;

        private Scope(Counter counter) {
            this.start = System.nanoTime();
            this.counter = counter;
            counter.enter();
        }

        @Override
        public void close() {
            long end = System.nanoTime();
            counter.exit((end - start) / 1000000000.0f);
        }
    }
}