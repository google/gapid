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
package com.google.gapid.perfetto.models;

import com.google.common.base.Preconditions;
import com.google.common.collect.ImmutableList;
import com.google.common.collect.Lists;
import com.google.common.collect.Maps;
import com.google.gapid.perfetto.canvas.Panel;
import com.google.gapid.perfetto.views.CopyablePanel;
import com.google.gapid.perfetto.views.State;

import java.util.List;
import java.util.Map;

/**
 * Information about what tracks are shown in the UI.
 */
public class TrackConfig {
  public static final TrackConfig EMPTY = new TrackConfig(ImmutableList.of());

  public final ImmutableList<Element<?>> elements;

  public TrackConfig(ImmutableList<Element<?>> elements) {
    this.elements = elements;
  }

  public abstract static class Element<T> {
    public final String id;
    public final String name;
    protected final T uiFactory;

    public Element(String id, String name, T uiFactory) {
      this.id = id;
      this.name = name;
      this.uiFactory = uiFactory;
    }

    public abstract CopyablePanel<?> createUi(State.ForSystemTrace state);
  }

  public static class Track<T extends CopyablePanel<T>> extends Element<Track.UiFactory<T>> {
    public Track(String id, String name, UiFactory<T> uiFactory) {
      super(id, name, uiFactory);
    }

    @Override
    public T createUi(State.ForSystemTrace state) {
      return uiFactory.createPanel(state);
    }

    public interface UiFactory<T extends Panel> {
      public T createPanel(State.ForSystemTrace state);
    }
  }

  public static class Group extends Element<Group.UiFactory> {
    public final ImmutableList<Element<?>> tracks;

    public Group(String id, String name, ImmutableList<Element<?>> tracks, UiFactory uiFactory) {
      super(id, name, uiFactory);
      this.tracks = tracks;
    }

    @Override
    public CopyablePanel<?> createUi(State.ForSystemTrace state) {
      ImmutableList.Builder<CopyablePanel<?>> children = ImmutableList.builder();
      tracks.forEach(track -> children.add(track.createUi(state)));
      return uiFactory.createPanel(state, children.build());
    }

    public interface UiFactory {
      public CopyablePanel<?> createPanel(
          State.ForSystemTrace state, ImmutableList<CopyablePanel<?>> children);
    }
  }

  public static class LabelGroup extends Group {
    public LabelGroup(
        String id, String name, ImmutableList<Element<?>> tracks, UiFactory uiFactory) {
      super(id, name, tracks, uiFactory);
    }
  }

  public static class Builder {
    private static final ElementBuilder ROOT = new ElementBuilder("root", "", false, null);

    private final Map<String, ElementBuilder> tracks = Maps.newHashMap();
    private final Map<String, List<ElementBuilder>> groups = Maps.newHashMap();

    public Builder() {
    }

    public Builder addGroup(String parent, String id, String name, Group.UiFactory ui) {
      groups.computeIfAbsent(parent, $ -> Lists.newArrayList())
          .add(newElement(id, name, false, ui));
      return this;
    }

    public Builder addLabelGroup(String parent, String id, String name, Group.UiFactory ui) {
      groups.computeIfAbsent(parent, $ -> Lists.newArrayList())
          .add(newElement(id, name, true, ui));
      return this;
    }

    public Builder addTrack(String parent, String id, String name, Track.UiFactory<Panel> ui) {
      groups.computeIfAbsent(parent, $ -> Lists.newArrayList())
          .add(newElement(id, name, false, ui));
      return this;
    }

    private ElementBuilder newElement(String id, String name, boolean label, Object uiFactory) {
      Preconditions.checkState(!tracks.containsKey(id));
      ElementBuilder track = new ElementBuilder(id, name, label, uiFactory);
      tracks.put(id, track);
      return track;
    }

    public TrackConfig build() {
      if (tracks.isEmpty() && groups.isEmpty()) {
        return TrackConfig.EMPTY;
      }
      return new TrackConfig(buildGroup(null).tracks);
    }

    private Group buildGroup(String id) {
      Preconditions.checkState(groups.containsKey(id));
      ElementBuilder track = (id == null) ? ROOT : tracks.get(id);
      Preconditions.checkState(track != null);
      ImmutableList.Builder<Element<?>> children = ImmutableList.builder();
      for (ElementBuilder child : groups.get(id)) {
        if (groups.containsKey(child.id)) {
          children.add(buildGroup(child.id));
        } else {
          children.add(child.track());
        }
      }
      return track.group(children.build());
    }

    private static class ElementBuilder {
      public final String id;
      public final String name;
      public final boolean label;
      public final Object uiFactory;

      public ElementBuilder(String id, String name, boolean label, Object uiFactory) {
        this.id = id;
        this.name = name;
        this.label = label;
        this.uiFactory = uiFactory;
      }

      @SuppressWarnings("unchecked")
      public <T extends CopyablePanel<T>> Track<T> track() {
        return new Track<T>(id, name, (Track.UiFactory<T>)uiFactory);
      }

      public Group group(ImmutableList<Element<?>> tracks) {
        if (label) {
          return new LabelGroup(id, name, tracks, (Group.UiFactory)uiFactory);
        } else {
          return new Group(id, name, tracks, (Group.UiFactory)uiFactory);
        }
      }
    }
  }
}
