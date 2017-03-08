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

import static com.google.gapid.util.Ranges.command;
import static com.google.gapid.util.Ranges.commands;
import static com.google.gapid.util.Ranges.contains;
import static com.google.gapid.util.Ranges.count;
import static com.google.gapid.util.Ranges.end;
import static com.google.gapid.util.Ranges.first;
import static com.google.gapid.util.Ranges.intersection;
import static com.google.gapid.util.Ranges.last;

import com.google.common.collect.Lists;
import com.google.common.collect.Maps;
import com.google.gapid.models.ApiContext.FilteringContext;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.Service.CommandGroup;
import com.google.gapid.proto.service.Service.CommandRange;
import com.google.gapid.proto.service.Service.Hierarchy;
import com.google.gapid.proto.service.Service.Value;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.server.Client;
import com.google.gapid.service.atom.Atom;
import com.google.gapid.service.atom.AtomList;
import com.google.gapid.service.atom.DynamicAtom;
import com.google.gapid.util.Events;
import com.google.gapid.util.Events.ListenerCollection;
import com.google.gapid.util.Ranges;
import com.google.gapid.views.Formatter;

import org.eclipse.jface.viewers.TreePath;
import org.eclipse.swt.widgets.Shell;

import java.lang.ref.Reference;
import java.lang.ref.SoftReference;
import java.util.AbstractList;
import java.util.Collections;
import java.util.List;
import java.util.Map;
import java.util.Objects;
import java.util.RandomAccess;
import java.util.logging.Logger;
import java.util.regex.Pattern;

public class AtomHierarchies extends CaptureDependentModel<Service.Hierarchy[]> {
  private static final Logger LOG = Logger.getLogger(AtomHierarchies.class.getName());
  private static final int SUBGROUP_SIZE = 1000;
  private static final int SPLIT_THRESHOLD = 2500; // If we split, at least split into 3 subgroups.

  private final ListenerCollection<Listener> listeners = Events.listeners(Listener.class);

  public AtomHierarchies(Shell shell, Client client, Capture capture) {
    super(LOG, shell, client, capture);
  }

  @Override
  protected Path.Any getPath(Path.Capture capturePath) {
    return Path.Any.newBuilder()
        .setHierarchies(Path.Hierarchies.newBuilder()
            .setCapture(capturePath))
        .build();
  }

  @Override
  protected Hierarchy[] unbox(Value value) {
    Service.Hierarchies hs = value.getHierarchies();
    return hs.getListList().toArray(new Hierarchy[hs.getListCount()]);
  }

  @Override
  protected void fireLoadEvent() {
    listeners.fire().onHierarchiesLoaded();
  }

  public FilteredGroup getHierarchy(AtomList atoms, FilteringContext context) {
    return new FilteredGroup(
        null, atoms, context, firstWithContext(context.getId()).getRoot());
  }

  private Service.Hierarchy firstWithContext(Path.ID contextId) {
    if (getData() != null) {
      for (Hierarchy hierarchy : getData()) {
        if (Objects.equals(hierarchy.getContext(), contextId)) {
          return hierarchy;
        }
      }
    }
    return null;
  }

  public void addListener(Listener listener) {
    listeners.addListener(listener);
  }

  public void removeListener(Listener listener) {
    listeners.removeListener(listener);
  }

  public static interface Listener extends Events.Listener {
    public default void onHierarchiesLoaded() { /* empty */ }
  }

  public static class FilteredGroup {
    public final FilteredGroup parent;
    private final AtomList atoms;
    private final FilteringContext context;
    public /*final*/ Service.CommandGroup group; // TODO

    private Reference<?>[] children;
    private Lookup groupLookup;
    private Lookup atomLookup;

    public FilteredGroup(
        FilteredGroup parent, AtomList atoms, FilteringContext context, Service.CommandGroup group) {
      this.parent = parent;
      this.atoms = atoms;
      this.context = context;
      this.group = group;
    }

    private FilteredGroup(FilteredGroup parent, AtomList atoms, FilteringContext context,
        Service.CommandGroup group, Lookup groupLookup, Lookup atomLookup, int childCount) {
      this(parent, atoms, context, group);
      this.children = new Reference<?>[childCount];
      this.groupLookup = groupLookup;
      this.atomLookup = atomLookup;
    }

    public int getChildCount() {
      setupChildLookups();
      return children.length;
    }

    /** @return either a {@link FilteredGroup} or an {@link AtomNode}. */
    public Object getChild(int idx) {
      setupChildLookups();
      Reference<?> ref = children[idx];
      if (ref != null) {
        Object child = ref.get();
        if (child != null) {
          return child;
        }
      }

      int groupIndex = groupLookup.lookup(idx);
      Object child;
      if (groupIndex >= 0) {
        child = new FilteredGroup(this, atoms, context, group.getSubgroups(groupIndex));
      } else {
        long atomIndex = atomLookup.lookup(idx);
        assert atomIndex >= 0;
        child = new AtomNode(this, atomIndex, atoms.get(atomIndex));
      }
      children[idx] = new SoftReference<Object>(child);
      return child;
    }

    public long getIndexOfLastLeaf() {
      return last(group.getRange());
    }

    public long getIndexOfLastDrawCall() {
      setupChildLookups();
      for (int i = getChildCount() - 1; i >= 0; i--) {
        Object child = getChild(i);
        if (child instanceof AtomNode) {
          AtomNode atom = (AtomNode)child;
          if (atom.atom.isDrawCall()) {
            return atom.index;
          }
        } else {
          long result = ((FilteredGroup)child).getIndexOfLastDrawCall();
          if (result >= 0) {
            return result;
          }
        }
      }
      return -1;
    }

    public Atom getLastLeaf() {
      return atoms.get(last(group.getRange()));
    }

    public Path.Command getPathToLastLeaf(Path.Commands commandsPath) {
      return Path.Command.newBuilder()
          .setCommands(commandsPath)
          .setIndex(last(group.getRange()))
          .build();
    }

    // Start is either null, a FilteredGroup or AtomNode that is one of this groups chilren.
    public CommandRange search(
        Pattern pattern, Object start, boolean next, boolean searchSiblings) {
      setupChildLookups();

      int index = 0;
      if (start != null) {
        for (; index < children.length; index++) {
          if (getChild(index) == start) {
            if (next) {
              index++;
            }
            break;
          }
        }
      }

      for (boolean first = true; index < children.length; index++, first = false) {
        Object child = getChild(index);
        CommandRange found = null;
        if (child instanceof FilteredGroup) {
          if (next || !first) {
            found = ((FilteredGroup)child).matches(pattern);
          }
          if (found == null) {
            found = ((FilteredGroup)child).search(pattern, null, false, false);
          }
        } else if (child instanceof AtomNode) {
          found = ((AtomNode)child).matches(pattern);
        }
        if (found != null) {
          return found;
        }
      }

      // Search my parent, since nothing was found.
      if (searchSiblings && parent != null) {
        return parent.search(pattern, this, true, true);
      }
      return null;
    }

    private CommandRange matches(Pattern pattern) {
      return pattern.matcher(group.getName()).find() ? group.getRange() : null;
    }

    public TreePath getTreePathTo(CommandRange range) {
      List<Object> segments = Lists.newArrayList();
      if (getTreePathTo(range, segments)) {
        return new TreePath(segments.toArray());
      }
      return null;
    }

    private boolean getTreePathTo(CommandRange range, List<Object> parents) {
      int found = Collections.binarySearch(new ChildList(this), null, (child, ignored) -> {
        if (child instanceof FilteredGroup) {
          CommandRange childRange = ((FilteredGroup)child).group.getRange();
          if (contains(childRange, last(range))) {
            return 0;
          } else if (last(range) > last(childRange)) {
            return -1;
          } else {
            return 1;
          }
        } else if (child instanceof AtomNode) {
          return Long.compare(((AtomNode)child).index, last(range));
        } else {
          throw new AssertionError();
        }
      });

      if (found >= 0) {
        Object child = getChild(found);
        parents.add(child);
        if (child instanceof FilteredGroup) {
          CommandRange groupRange = ((FilteredGroup)child).group.getRange();
          if (!groupRange.equals(range)) {
            return ((FilteredGroup)child).getTreePathTo(range, parents);
          }
        }
        return true;
      }
      return false;
    }

    private void setupChildLookups() {
      if (children == null) {
        Map<CommandRange, Integer> atomMap = Maps.newHashMap();
        Map<CommandRange, Integer> groupMap = Maps.newHashMap();
        List<CommandRange> atomRanges = Lists.newArrayList();
        List<CommandRange> groupRanges = Lists.newArrayList();
        int count = 0;
        long next = first(group.getRange());
        for (int groupIndex = 0; groupIndex < group.getSubgroupsCount(); groupIndex++) {
          Service.CommandGroup subGroup = group.getSubgroups(groupIndex);
          CommandRange.Builder range = CommandRange.newBuilder()
              .setFirst(next)
              .setCount(first(subGroup.getRange()) - next);
          if (range.getCount() > 0) {
            List<CommandRange> intersection = intersection(context.getRanges(atoms), range);
            for (CommandRange r : intersection) {
              CommandRange key = commands(count, count(r));
              atomMap.put(key, (int)first(r));
              atomRanges.add(key);
              count += count(r);
            }
          }

          if (context.contains(subGroup.getRange())) {
            CommandRange lastRange = groupRanges.isEmpty() ?
                null : groupRanges.get(groupRanges.size() - 1);
            if (lastRange != null && end(lastRange) == count &&
                (groupMap.get(lastRange) + lastRange.getCount()) == groupIndex) {
              long orgGroupIndex = groupMap.remove(lastRange);
              groupRanges.remove(groupRanges.size() - 1);
              lastRange = lastRange.toBuilder().setCount(count(lastRange) + 1).build(); // TODO
              groupMap.put(lastRange, (int)orgGroupIndex);
              groupRanges.add(lastRange);
            } else {
              CommandRange newRange = command(count);
              groupMap.put(newRange, groupIndex);
              groupRanges.add(newRange);
            }
            count++;
          }
          next = end(subGroup.getRange());
        }

        CommandRange.Builder range = CommandRange.newBuilder()
            .setFirst(next)
            .setCount(end(group.getRange()) - next);
        if (range.getCount() > 0) {
          List<CommandRange> intersection = intersection(context.getRanges(atoms), range);
          for (CommandRange r : intersection) {
            CommandRange key = commands(count, count(r));
            atomMap.put(key, (int)first(r));
            atomRanges.add(key);
            count += count(r);
          }
        }

        groupLookup = Lookup.create(groupRanges, groupMap);
        atomLookup = Lookup.create(atomRanges, atomMap);
        if (count >= SPLIT_THRESHOLD) {
          split(count);
        } else {
          children = new Reference<?>[count];
        }
      }
    }

    private void split(int count) {
      List<Service.CommandGroup> newGroups = Lists.newArrayList();
      List<FilteredGroup> newFilteredGroups = Lists.newArrayList();
      long next = first(group.getRange());
      for (int idx = 0; idx < count; idx += SUBGROUP_SIZE) {
        Lookup subGroupLookup = groupLookup.subrange(idx, idx + SUBGROUP_SIZE - 1, true);
        Lookup subAtomLookup = atomLookup.subrange(idx, idx + SUBGROUP_SIZE - 1, false);
        List<Service.CommandGroup> subGroups = subGroupLookup.map(group.getSubgroupsList());

        int childCount = Math.min(count - idx, SUBGROUP_SIZE);
        int subGroupIdx = subGroupLookup.lookup(childCount - 1);
        long newNext = (subGroupIdx < 0) ? subAtomLookup.lookup(childCount - 1) + 1 :
            end(subGroups.get(subGroupIdx).getRange());

        CommandGroup subGroup = CommandGroup.newBuilder()
            .setName("Commands [" + next + " - " + (newNext - 1) + "]")
            .setRange(CommandRange.newBuilder()
                .setFirst(next)
                .setCount(newNext - next))
            .addAllSubgroups(subGroups)
            .build();
        newGroups.add(subGroup);
        newFilteredGroups.add(new FilteredGroup(
            this, atoms, context, subGroup, subGroupLookup, subAtomLookup, childCount));
        next = newNext;
      }

      group = group.toBuilder().clearSubgroups().addAllSubgroups(newGroups).build(); // TODO
      children = newFilteredGroups.stream()
          .map(g -> new SoftReference<>(g)).toArray(l -> new Reference<?>[l]);
      groupLookup = new OneToOne(commands(0, children.length));
      atomLookup = Lookup.EMPTY;
    }

    private static interface Lookup {
      public static final Lookup EMPTY = new Lookup() {
        @Override
        public int lookup(int idx) {
          return -1;
        }

        @Override
        public Lookup subrange(int from, int to, boolean computeOffset) {
          return EMPTY;
        }

        @Override
        public <T> List<T> map(List<T> source) {
          return Collections.emptyList();
        }
      };

      public int lookup(int idx);

      @SuppressWarnings("unused")
      public default Lookup subrange(int from, int to, boolean computeOffset) {
        throw new UnsupportedOperationException();
      }

      @SuppressWarnings("unused")
      public default <T> List<T> map(List<T> source) {
        throw new UnsupportedOperationException();
      }

      public static Lookup create(List<CommandRange> ranges, Map<CommandRange, Integer> map) {
        return ranges.isEmpty() ? EMPTY :
            new RangedLookup(ranges.toArray(new CommandRange[ranges.size()]), map, 0);
      }
    }

    private static class RangedLookup implements Lookup {
      private final CommandRange[] ranges;
      private final Map<CommandRange, Integer> map;
      private final int offset;

      public RangedLookup(CommandRange[] ranges, Map<CommandRange, Integer> map, int offset) {
        assert ranges.length == map.size();
        this.ranges = ranges;
        this.map = map;
        this.offset = offset;
      }

      @Override
      public int lookup(int idx) {
        int result = contains(ranges, idx);
        if (result >= 0) {
          CommandRange groupIndexRange = ranges[result];
          return offset + map.get(groupIndexRange) + (idx - (int)first(groupIndexRange));
        }
        return -1;
      }

      @Override
      public Lookup subrange(int from, int to, boolean computeOffset) {
        int start = contains(ranges, from), end = contains(ranges, to);
        if (start < 0) {
          start = (-start - 1);
        }
        if (end < 0) {
          end = (-end - 1) - 1;
        }

        if (end < start) {
          return EMPTY;
        }
        CommandRange[] newRanges = new CommandRange[(end - start) + 1];
        Map<CommandRange, Integer> newMap = Maps.newHashMap();
        for (int i = 0; i < newRanges.length; i++) {
          CommandRange oldRange = ranges[start + i];
          long startOffset = first(oldRange) - from;
          long newStart = Math.max(0, startOffset);
          long newEnd = Math.min(to - from + 1, end(oldRange) - from);
          CommandRange newRange = commands(newStart, newEnd - newStart);
          newRanges[i] = newRange;
          newMap.put(newRange, map.get(oldRange) - (int)Math.min(0, startOffset));
        }

        return new RangedLookup(
            newRanges, newMap, computeOffset ? -newMap.get(newRanges[0]) : offset);
      }

      @Override
      public <T> List<T> map(List<T> source) {
        CommandRange last = ranges[ranges.length - 1];
        int end = map.get(last) + (int)last.getCount();
        int start = map.get(ranges[0]);
        return source.subList(start, end);
      }
    }

    private static class OneToOne implements Lookup {
      private final CommandRange range;

      public OneToOne(CommandRange range) {
        this.range = range;
      }

      @Override
      public int lookup(int idx) {
        return contains(range, idx) ? idx : -1;
      }
    }

    private static class ChildList extends AbstractList<Object> implements RandomAccess {
      private final FilteredGroup group;

      public ChildList(FilteredGroup group) {
        this.group = group;
      }

      @Override
      public Object get(int index) {
        return group.getChild(index);
      }

      @Override
      public int size() {
        return group.getChildCount();
      }
    }
  }

  public static class AtomNode {
    public final FilteredGroup parent;
    public final long index;
    public final Atom atom;

    public AtomNode(FilteredGroup parent, long index, Atom atom) {
      this.parent = parent;
      this.index = index;
      this.atom = atom;
    }

    public CommandRange matches(Pattern pattern) {
      return (pattern.matcher(Formatter.toString((DynamicAtom)atom)).find()) ?
          Ranges.command(index) : null;
    }
  }
}
