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
package com.google.gapid.widgets;

import static com.google.gapid.util.Paths.command;
import static com.google.gapid.widgets.Widgets.createCheckbox;
import static com.google.gapid.widgets.Widgets.createComposite;
import static com.google.gapid.widgets.Widgets.createLabel;
import static com.google.gapid.widgets.Widgets.createSpinner;
import static com.google.gapid.widgets.Widgets.createTextbox;

import com.google.common.collect.Iterables;
import com.google.common.collect.Lists;
import com.google.gapid.models.Models;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.rpclib.rpccore.Rpc;
import com.google.gapid.rpclib.rpccore.RpcException;
import com.google.gapid.rpclib.schema.Constant;
import com.google.gapid.rpclib.schema.ConstantSet;
import com.google.gapid.rpclib.schema.Dynamic;
import com.google.gapid.rpclib.schema.Field;
import com.google.gapid.rpclib.schema.Method;
import com.google.gapid.rpclib.schema.Primitive;
import com.google.gapid.rpclib.schema.Type;
import com.google.gapid.server.Client;
import com.google.gapid.service.atom.Atom;
import com.google.gapid.service.snippets.Labels;
import com.google.gapid.service.snippets.SnippetObject;
import com.google.gapid.util.PrefixTree;
import com.google.gapid.util.UiCallback;
import com.google.gapid.views.Formatter;

import org.eclipse.jface.dialogs.TitleAreaDialog;
import org.eclipse.jface.fieldassist.ContentProposal;
import org.eclipse.jface.fieldassist.ContentProposalAdapter;
import org.eclipse.jface.fieldassist.IContentProposal;
import org.eclipse.jface.fieldassist.IContentProposalProvider;
import org.eclipse.jface.fieldassist.TextContentAdapter;
import org.eclipse.jface.viewers.ArrayContentProvider;
import org.eclipse.jface.viewers.ComboViewer;
import org.eclipse.jface.viewers.LabelProvider;
import org.eclipse.jface.window.Window;
import org.eclipse.swt.SWT;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.layout.RowLayout;
import org.eclipse.swt.widgets.Button;
import org.eclipse.swt.widgets.Combo;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Label;
import org.eclipse.swt.widgets.Shell;
import org.eclipse.swt.widgets.Spinner;
import org.eclipse.swt.widgets.Text;

import java.io.IOException;
import java.math.BigInteger;
import java.nio.ByteBuffer;
import java.util.Arrays;
import java.util.Collection;
import java.util.Collections;
import java.util.List;
import java.util.concurrent.ExecutionException;
import java.util.logging.Level;
import java.util.logging.Logger;

public class AtomEditor {
  private static final Logger LOG = Logger.getLogger(AtomEditor.class.getName());

  private final Client client;
  protected final Models models;

  public AtomEditor(Client client, Models models) {
    this.client = client;
    this.models = models;
  }

  public void showEditPopup(Shell parent, long index) {
    EditDialog dialog = new EditDialog(parent, models.atoms.getAtom(index));
    if (dialog.open() == Window.OK) {
      Rpc.listen(client.set(getSetPath(index), getSetValue(dialog.newAtom)),
          new UiCallback<Path.Any, Path.Any>(parent, LOG) {
        @Override
        protected Path.Any onRpcThread(Rpc.Result<Path.Any> result)
            throws RpcException, ExecutionException {
          return result.get();
        }

        @Override
        protected void onUiThread(Path.Any newPath) {
          // TODO this should probably be able to handle any path.
          models.capture.updateCapture(newPath.getCommand().getCommands().getCapture(), null);
        }
      });
    }
  }

  private Path.Any getSetPath(long index) {
    return Path.Any.newBuilder()
        .setCommand(command(models.atoms.getPath(), index))
        .build();
  }

  private static Service.Value getSetValue(Dynamic atom) {
    try {
      return Service.Value.newBuilder()
          .setObject(Client.encode(atom))
          .build();
    } catch (IOException e) {
      LOG.log(Level.SEVERE, "Failed to encode atom " + atom);
      return Service.Value.getDefaultInstance();
    }
  }

  private static class EditDialog extends TitleAreaDialog {
    private final Atom atom;
    private final List<Editor<?>> editors = Lists.newArrayList();
    public Dynamic newAtom;

    public EditDialog(Shell parentShell, Atom atom) {
      super(parentShell);
      this.atom = atom;
    }

    @Override
    public void create() {
      super.create();
      setTitle("Edit " + atom.getName());
    }

    @Override
    protected void configureShell(Shell newShell) {
      super.configureShell(newShell);
      newShell.setText("Edit " + atom.getName() + "...");
    }

    @Override
    protected Control createDialogArea(Composite parent) {
      Composite area = (Composite)super.createDialogArea(parent);

      Composite container = createComposite(area, new GridLayout(2, false));
      container.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));

      int resultIndex = atom.getResultIndex();
      int extrasIndex = atom.getExtrasIndex();
      for (int i = 0; i < atom.getFieldCount(); i++) {
        if (i == resultIndex || i == extrasIndex) {
          continue;
        }

        SnippetObject snippetObject = SnippetObject.param(atom, i);
        Object value = atom.getFieldValue(i);
        Field field = atom.getFieldInfo(i);

        String typeString = field.getType() instanceof Primitive ?
            " (" + ((Primitive)field.getType()).getMethod() + ")" : "";
        createLabel(container, atom.getFieldInfo(i).getName() + typeString + ":");
        Editor<?> editor = Editor.getFor(container, field.getType(), value, snippetObject, i);
        editor.control.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, false));
        editors.add(editor);
      }

      return area;
    }

    @Override
    protected boolean isResizable() {
      return true;
    }

    @Override
    protected void okPressed() {
      newAtom = ((Dynamic)atom.unwrap()).copy();
      for (Editor<?> editor : editors) {
        editor.update(newAtom);
      }

      super.okPressed();
    }

    private abstract static class Editor<C extends Control> {
      private static final int MAX_DROP_DOWN_SIZE = 1000;

      public final int fieldIndex;
      public final C control;

      public Editor(int fieldIndex, C control) {
        this.fieldIndex = fieldIndex;
        this.control = control;
      }

      public abstract void update(Dynamic atom);

      public static Editor<?> getFor(
          Composite parent,Type type, Object value, SnippetObject snippetObject, int fieldIndex) {
        if (type instanceof Primitive) {
          final Primitive primitive = (Primitive) type;

          Collection<Constant> constant = Formatter.findConstant(snippetObject, primitive);
          if (constant.size() >= 1) {
            // handle enum
            List<Constant> constants = Arrays.asList(ConstantSet.lookup(type).getEntries());
            // if we have a set of preferred constants, use them.
            Labels labels = Labels.fromSnippets(snippetObject.getSnippets());
            if (labels != null) {
              List<Constant> preferred = labels.preferred(constants);
              if (preferred.containsAll(constant)) {
                constants = preferred;
              }
            }

            // TODO change this to actually check if this ConstantSet is flags
            if (constant.size() == 1) {
              Constant current = Iterables.get(constant, 0);
              if (constants.size() > MAX_DROP_DOWN_SIZE) {
                return new ConstantEditor(parent, constants, current, fieldIndex);
              } else {
                return new EnumEditor(parent, constants, current, fieldIndex);
              }
            } else {
              return new FlagEditor(parent, constants, constant, fieldIndex);
            }
          }

          Method method = primitive.getMethod();
          if (method == Method.Bool) {
            return new BooleanEditor(parent, ((Boolean) value).booleanValue(), fieldIndex);
          } else if (method == Method.Float32 || method == Method.Float64) {
            return new FloatEditor(parent, ((Number) value).doubleValue(), fieldIndex);
          } else if (method == Method.String) {
            return new StringEditor(parent, String.valueOf(value), fieldIndex);
          } else if (method == Method.Uint8) {
            return new IntEditor(parent, ((Number) value).intValue(), 0, 255, fieldIndex);
          } else if (method == Method.Int8) {
            return new IntEditor(parent, ((Number) value).intValue(), -128, 127, fieldIndex);
          } else if (method == Method.Uint16) {
            return new IntEditor(parent, ((Number) value).intValue(), 0, 65535, fieldIndex);
          } else if (method == Method.Int16) {
            return new IntEditor(parent, ((Number) value).intValue(), -32768, 32767, fieldIndex);
          } else if (method == Method.Int32) {
            return new IntEditor(
                parent, ((Number) value).intValue(), 0x80000000, 0x7fffffff, fieldIndex);
          } else if (method == Method.Uint32 || method == Method.Int64) {
            return new LongEditor(
                parent, BigInteger.valueOf(((Number) value).longValue()), fieldIndex);
          } else if (method == Method.Uint64) {
            ByteBuffer buffer = ByteBuffer.allocate(Long.BYTES);
            buffer.putLong(((Number) value).longValue());
            return new LongEditor(parent, new BigInteger(1, buffer.array()), fieldIndex);
          }
        }
        return new NoEditEditor(parent, type, snippetObject);
      }
    }

    private static class NoEditEditor extends Editor<Label> {
      public NoEditEditor(Composite parent, Type type, SnippetObject value) {
        super(-1, new Label(parent, SWT.NONE));
        control.setText(Formatter.toString(value, type));
      }

      @Override
      public void update(Dynamic atom) {
        // Noop.
      }
    }

    private static class EnumEditor extends Editor<Combo> {
      private final ComboViewer viewer;

      public EnumEditor(
          Composite parent, List<Constant> constants, Constant value, int fieldIndex) {
        super(fieldIndex, new Combo(parent, SWT.DROP_DOWN | SWT.READ_ONLY));
        viewer = new ComboViewer(control);
        viewer.setContentProvider(ArrayContentProvider.getInstance());
        viewer.setLabelProvider(new LabelProvider());
        viewer.setUseHashlookup(true);
        viewer.setInput(constants);
        control.select(constants.indexOf(value));
      }

      @Override
      public void update(Dynamic atom) {
        atom.setFieldValue(
            fieldIndex, ((Constant)viewer.getElementAt(control.getSelectionIndex())).getValue());
      }
    }

    private static class ConstantEditor extends Editor<Text> {
      private static final int MAX_PROPOSALS = 1000;

      protected final PrefixTree<ConstantValue> lookup;

      public ConstantEditor(
          Composite parent, List<Constant> constants, Constant value, int fieldIndex) {
        super(fieldIndex, Widgets.createTextbox(parent, value.getName()));

        constants = Lists.newArrayList(constants);
        // Reverse order. The prefix tree returns elements in LIFO order.
        Collections.sort(constants, (c1, c2) -> c2.getName().compareTo(c1.getName()));
        lookup = PrefixTree.of(constants.stream().map(ConstantValue::new).iterator());

        IContentProposalProvider cpp = new IContentProposalProvider(){
          @Override
          public IContentProposal[] getProposals(String contents, int position) {
            List<IContentProposal> result = Lists.newArrayList();
            lookup.find(contents.substring(0, position).toLowerCase(), v -> {
              result.add(v);
              return result.size() < MAX_PROPOSALS;
            });
            return result.toArray(new IContentProposal[result.size()]);
          }
        };
        ContentProposalAdapter adapter = new ContentProposalAdapter(
            control, new TextContentAdapter(), cpp, null, null);
        adapter.setProposalAcceptanceStyle(ContentProposalAdapter.PROPOSAL_REPLACE);
      }

      @Override
      public void update(Dynamic atom) {
        ConstantValue value = lookup.get(control.getText().toLowerCase());
        if (value != null) { // TODO
          atom.setFieldValue(fieldIndex, value.constant.getValue());
        }
      }

      private static class ConstantValue extends ContentProposal implements PrefixTree.Value {
        public final Constant constant;

        public ConstantValue(Constant constant) {
          super(constant.getName(), constant.toString());
          this.constant = constant;
        }

        @Override
        public String getKey() {
          return constant.getName().toLowerCase();
        }
      }
    }

    private static class FlagEditor extends Editor<Composite> {
      private final List<Constant> constants;

      public FlagEditor(
          Composite parent, List<Constant> constants, Collection<Constant> value, int fieldIndex) {
        super(fieldIndex, createComposite(parent, new RowLayout(SWT.VERTICAL)));
        this.constants = constants;

        for (Constant constant : constants) {
          createCheckbox(control, String.valueOf(constant), value.contains(constant));
        }
      }

      @Override
      public void update(Dynamic atom) {
        long value = 0;
        Control[] children = control.getChildren();
        for (int i = 0; i < children.length; i++) {
          if (((Button)children[i]).getSelection()) {
            value |= ((Number)constants.get(i).getValue()).longValue();
          }
        }
        atom.setFieldValue(fieldIndex, value);
      }
    }

    private static class BooleanEditor extends Editor<Button> {
      public BooleanEditor(Composite parent, boolean value, int fieldIndex) {
        super(fieldIndex, createCheckbox(parent, value));
      }

      @Override
      public void update(Dynamic atom) {
        atom.setFieldValue(fieldIndex, control.getSelection());
      }
    }

    private static class IntEditor extends Editor<Spinner> {
      public IntEditor(Composite parent, int value, int min, int max, int fieldIndex) {
        super(fieldIndex, createSpinner(parent, value, min, max));
      }

      @Override
      public void update(Dynamic atom) {
        atom.setFieldValue(fieldIndex, control.getSelection());
      }
    }

    private static class LongEditor extends Editor<Text> {
      public LongEditor(Composite parent, BigInteger value, int fieldIndex) {
        super(fieldIndex, createTextbox(parent, String.valueOf(value)));
      }

      @Override
      public void update(Dynamic atom) {
        try {
          atom.setFieldValue(fieldIndex, new BigInteger(control.getText()));
        } catch (NumberFormatException e) {
          // TODO.
        }
      }
    }

    private static class FloatEditor extends Editor<Text> {
      public FloatEditor(Composite parent, double value, int fieldIndex) {
        super(fieldIndex, createTextbox(parent, String.valueOf(value)));
      }

      @Override
      public void update(Dynamic atom) {
        try {
          atom.setFieldValue(fieldIndex, Double.parseDouble(control.getText()));
        } catch (NumberFormatException e) {
          // TODO.
        }
      }
    }

    private static class StringEditor extends Editor<Text> {
      public StringEditor(Composite parent, String value, int fieldIndex) {
        super(fieldIndex, createTextbox(parent, value));
      }

      @Override
      public void update(Dynamic atom) {
        atom.setFieldValue(fieldIndex, control.getText());
      }
    }
  }
}
