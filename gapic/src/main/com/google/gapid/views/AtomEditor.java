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
package com.google.gapid.views;

import static com.google.gapid.widgets.Widgets.createCheckbox;
import static com.google.gapid.widgets.Widgets.createComposite;
import static com.google.gapid.widgets.Widgets.createDropDownViewer;
import static com.google.gapid.widgets.Widgets.createEditDropDown;
import static com.google.gapid.widgets.Widgets.createLabel;
import static com.google.gapid.widgets.Widgets.createSpinner;
import static com.google.gapid.widgets.Widgets.createTextbox;

import com.google.common.collect.Lists;
import com.google.gapid.models.ConstantSets;
import com.google.gapid.models.Models;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.api.API;
import com.google.gapid.proto.service.box.Box;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.rpc.Rpc;
import com.google.gapid.rpc.RpcException;
import com.google.gapid.rpc.UiCallback;
import com.google.gapid.server.Client;
import com.google.gapid.util.Paths;
import com.google.gapid.util.Pods;
import com.google.gapid.util.PrefixTree;
import com.google.gapid.util.Values;
import com.google.gapid.widgets.DialogBase;
import com.google.gapid.widgets.Theme;
import com.google.gapid.widgets.Widgets;

import org.eclipse.jface.fieldassist.ComboContentAdapter;
import org.eclipse.jface.fieldassist.ContentProposal;
import org.eclipse.jface.fieldassist.ContentProposalAdapter;
import org.eclipse.jface.fieldassist.IContentProposal;
import org.eclipse.jface.fieldassist.IContentProposalProvider;
import org.eclipse.jface.fieldassist.IControlContentAdapter;
import org.eclipse.jface.fieldassist.TextContentAdapter;
import org.eclipse.jface.viewers.ArrayContentProvider;
import org.eclipse.jface.viewers.ComboViewer;
import org.eclipse.jface.viewers.LabelProvider;
import org.eclipse.jface.viewers.StructuredSelection;
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

import java.util.List;
import java.util.concurrent.ExecutionException;
import java.util.logging.Logger;

/**
 * Command (atom) editing dialog. Allows the user to change the parameters of a command in the
 * capture.
 */
public class AtomEditor {
  private static final Logger LOG = Logger.getLogger(AtomEditor.class.getName());

  private final Client client;
  protected final Models models;
  private final Theme theme;

  public AtomEditor(Client client, Models models, Theme theme) {
    this.client = client;
    this.models = models;
    this.theme = theme;
  }

  public static boolean shouldShowEditPopup(API.Command command) {
    return command.getParametersCount() > 0;
  }

  public void showEditPopup(Shell parent, Path.Command path, API.Command command) {
    EditDialog dialog = new EditDialog(parent, models, theme, command);
    if (dialog.open() == Window.OK) {
      Rpc.listen(client.set(Paths.toAny(path), Values.value(dialog.newAtom)),
          new UiCallback<Path.Any, Path.Any>(parent, LOG) {
        @Override
        protected Path.Any onRpcThread(Rpc.Result<Path.Any> result)
            throws RpcException, ExecutionException {
          return result.get();
        }

        @Override
        protected void onUiThread(Path.Any newPath) {
          models.capture.updateCapture(newPath.getCommand().getCapture(), null);
        }
      });
    }
  }

  /**
   * The dialog containing the editors for a given command.
   */
  private static class EditDialog extends DialogBase {
    private final Models models;
    private final API.Command atom;
    private final List<Editor<?>> editors = Lists.newArrayList();
    public API.Command newAtom;

    public EditDialog(Shell parentShell, Models models, Theme theme, API.Command atom) {
      super(parentShell, theme);
      this.models = models;
      this.atom = atom;
    }

    @Override
    public String getTitle() {
      return "Edit " + atom.getName() + "...";
    }

    @Override
    protected Control createDialogArea(Composite parent) {
      Composite area = (Composite)super.createDialogArea(parent);

      Composite container = Widgets.createComposite(area, new GridLayout(2, false));
      container.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));

      for (API.Parameter param : atom.getParametersList()) {
        Service.ConstantSet constants = models.constants.getConstants(param.getConstants());
        String typeString = Editor.getTypeString(param);
        typeString = typeString.isEmpty() ? "" : " (" + typeString + ")";
        createLabel(container, param.getName() + typeString + ":");
        Editor<?> editor = Editor.getFor(container, param, constants);
        editor.control.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, false));
        editors.add(editor);
      }

      return area;
    }

    @Override
    protected void okPressed() {
      API.Command.Builder builder = atom.toBuilder();
      for (int i = atom.getParametersCount() - 1; i >= 0; i--) {
        editors.get(i).update(builder.getParametersBuilder(i).getValueBuilder());
      }
      newAtom = builder.build();

      super.okPressed();
    }

    /**
     * Base class for the different types of editors.
     */
    private abstract static class Editor<C extends Control> {
      private static final int MAX_DROP_DOWN_SIZE = 1000;

      public final C control;

      public Editor(C control) {
        this.control = control;
      }

      public abstract void update(Box.Value.Builder param);

      public static Editor<?> getFor(
          Composite parent, API.Parameter param, Service.ConstantSet constants) {
        Box.Value value = param.getValue();
        switch (value.getValCase()) {
          case POD:
            if (constants != null && Pods.mayBeConstant(value.getPod())) {
              if (constants.getIsBitfield()) {
                return new FlagEditor(parent, constants, value);
              } else if (constants.getConstantsCount() > MAX_DROP_DOWN_SIZE) {
                return new ConstantEditor(parent, constants, value);
              } else {
                return new EnumEditor(parent, constants, value);
              }
            }

            switch (value.getPod().getValCase()) {
              case BOOL: return new BooleanEditor(parent, value);
              case UINT8: return new IntEditor(parent, value, 0, 255);
              case SINT8: return new IntEditor(parent, value, -128, 127);
              case UINT16: return new IntEditor(parent, value, 0, 65535);
              case SINT16: return new IntEditor(parent, value, -32768, 32767);
              case SINT:
              case SINT32: return new IntEditor(parent, value, 0x80000000, 0x7fffffff);
              case STRING: return new StringEditor(parent, value);
              default: // Fall through.
            }

            if (Pods.isLong(value.getPod())) {
              return new LongEditor(parent, value);
            } else if (Pods.isFloat(value.getPod())) {
              return new FloatEditor(parent, value);
            }
            break;

          default: // Fall through.
        }
        return new NoEditEditor(parent, value, constants);
      }

      public static String getTypeString(API.Parameter param) {
        Box.Value value = param.getValue();
        switch (value.getValCase()) {
          case POD: return value.getPod().getValCase().name().toLowerCase();
          default: return "";
        }
      }
    }

    /**
     * {@link Editor} for read-only values.
     */
    private static class NoEditEditor extends Editor<Label> {
      public NoEditEditor(Composite parent, Box.Value value, Service.ConstantSet constants) {
        super(new Label(parent, SWT.NONE));
        control.setText(Formatter.toString(value, constants, true));
      }

      @Override
      public void update(Box.Value.Builder param) {
        // Do nothing.
      }
    }

    /**
     * Base {@link Editor} class for enums.
     */
    private abstract static class BaseEnumEditor<C extends Control> extends Editor<C> {
      private static final int MAX_PROPOSALS = 1000;

      protected final PrefixTree<ConstantValue> lookup;

      public BaseEnumEditor(
          C control, Service.ConstantSet constants, IControlContentAdapter contentAdapter) {
        super(control);

        lookup = PrefixTree.of(constants.getConstantsList().stream()
            // Reverse order. The prefix tree returns elements in LIFO order.
            .sorted((c1, c2) -> c1.getName().compareTo(c1.getName()))
            .map(ConstantValue::new)
            .iterator());

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
        ContentProposalAdapter adapter =
            new ContentProposalAdapter(control, contentAdapter, cpp, null, null);
        adapter.setProposalAcceptanceStyle(ContentProposalAdapter.PROPOSAL_REPLACE);
      }

      protected static class ConstantValue extends ContentProposal implements PrefixTree.Value {
        public final Service.Constant constant;

        public ConstantValue(Service.Constant constant) {
          super(constant.getName(), Formatter.toString(constant));
          this.constant = constant;
        }

        @Override
        public String getKey() {
          return constant.getName().toLowerCase();
        }
      }
    }

    /**
     * {@link Editor} for enums using a drop down.
     */
    private static class EnumEditor extends BaseEnumEditor<Combo> {
      private final ComboViewer viewer;

      public EnumEditor(Composite parent, Service.ConstantSet constants, Box.Value value) {
        super(createEditDropDown(parent), constants, new ComboContentAdapter());
        viewer = createDropDownViewer(control);
        viewer.setContentProvider(ArrayContentProvider.getInstance());
        viewer.setLabelProvider(new LabelProvider() {
          @Override
          public String getText(Object element) {
            return Formatter.toString((Service.Constant)element);
          }
        });
        viewer.setInput(constants.getConstantsList());

        Service.Constant constantValue = ConstantSets.find(constants, value);
        if (!constantValue.getName().isEmpty()) {
          viewer.setSelection(new StructuredSelection(constantValue));
        } else {
          control.setText(Long.toString(Pods.getConstant(value.getPod())));
        }
      }

      @Override
      public void update(Box.Value.Builder param) {
        int selection = control.getSelectionIndex();
        if (selection < 0) {
          // The user either modified the text box or manually typed an entry.
          String text = control.getText().toLowerCase();
          int p = text.indexOf(' ');
          if (p > 0) {
            // If the user picked a value from the drop down, the constant number is part of the
            // text. Cut off everything after the first space.
            text = text.substring(0, p);
          }
          ConstantValue value = lookup.get(text);
          if (value != null) {
            Pods.setConstant(param.getPodBuilder(), value.constant.getValue());
          } else {
            // The user might have just typed in the constant value.
            try {
              Pods.setConstant(param.getPodBuilder(), Long.parseLong(text));
            } catch (NumberFormatException e) {
              // TODO.
            }
          }
        } else {
          Pods.setConstant(
              param.getPodBuilder(), ((Service.Constant)viewer.getElementAt(selection)).getValue());
        }
      }
    }

    /**
     * {@link Editor} for enums using a free from text box with auto completion suggestions.
     */
    private static class ConstantEditor extends BaseEnumEditor<Text> {
      public ConstantEditor(Composite parent, Service.ConstantSet constants, Box.Value val) {
        super(createTextbox(parent, getName(constants, val)), constants, new TextContentAdapter());
      }

      @Override
      public void update(Box.Value.Builder param) {
        ConstantValue value = lookup.get(control.getText().toLowerCase());
        if (value != null) {
          Pods.setConstant(param.getPodBuilder(), value.constant.getValue());
        } else {
          // The user might have just typed in the constant value.
          try {
            Pods.setConstant(param.getPodBuilder(), Long.parseLong(control.getText()));
          } catch (NumberFormatException e) {
            // TODO.
          }
        }
      }

      private static String getName(Service.ConstantSet constants, Box.Value value) {
        Service.Constant constantValue = ConstantSets.find(constants, value);
        if (!constantValue.getName().isEmpty()) {
          return constantValue.getName();
        } else {
          return Long.toString(Pods.getConstant(value.getPod()));
        }
      }
    }

    /**
     * {@link Editor} for flag/bitmask values.
     */
    private static class FlagEditor extends Editor<Composite> {
      private final Service.ConstantSet constants;

      public FlagEditor(Composite parent, Service.ConstantSet constants, Box.Value value) {
        super(createComposite(parent, new RowLayout(SWT.VERTICAL)));
        this.constants = constants;

        long bits = Pods.getConstant(value.getPod());
        for (Service.Constant constant : constants.getConstantsList()) {
          createCheckbox(control, Formatter.toString(constant),
              (bits & constant.getValue()) == constant.getValue());
        }
      }

      @Override
      public void update(Box.Value.Builder param) {
        long value = 0;
        Control[] children = control.getChildren();
        for (int i = 0; i < children.length; i++) {
          if (((Button)children[i]).getSelection()) {
            value |= constants.getConstants(i).getValue();
          }
        }
        Pods.setConstant(param.getPodBuilder(), value);
      }
    }

    /**
     * {@link Editor} for boolean values.
     */
    private static class BooleanEditor extends Editor<Button> {
      public BooleanEditor(Composite parent, Box.Value value) {
        super(createCheckbox(parent, value.getPod().getBool()));
      }

      @Override
      public void update(Box.Value.Builder param) {
        param.getPodBuilder().setBool(control.getSelection());
      }
    }

    /**
     * {@link Editor} for integer values.
     */
    private static class IntEditor extends Editor<Spinner> {
      public IntEditor(Composite parent, Box.Value value, int min, int max) {
        super(createSpinner(parent, Pods.getInt(value.getPod()), min, max));
      }

      @Override
      public void update(Box.Value.Builder param) {
        Pods.setInt(param.getPodBuilder(), control.getSelection());
      }
    }

    /**
     * {@link Editor} for long values.
     */
    private static class LongEditor extends Editor<Text> {
      public LongEditor(Composite parent, Box.Value value) {
        super(createTextbox(parent, String.valueOf(Pods.getLong(value.getPod()))));
      }

      @Override
      public void update(Box.Value.Builder param) {
        try {
          Pods.setLong(param.getPodBuilder(), Long.parseLong(control.getText()));
        } catch (NumberFormatException e) {
          // TODO.
        }
      }
    }

    /**
     * {@link Editor} for floating point values.
     */
    private static class FloatEditor extends Editor<Text> {
      public FloatEditor(Composite parent, Box.Value value) {
        super(createTextbox(parent, String.valueOf(Pods.getFloat(value.getPod()))));
      }

      @Override
      public void update(Box.Value.Builder param) {
        try {
          Pods.setFloat(param.getPodBuilder(), Double.parseDouble(control.getText()));
        } catch (NumberFormatException e) {
          // TODO.
        }
      }
    }

    /**
     * {@link Editor} for string values.
     */
    private static class StringEditor extends Editor<Text> {
      public StringEditor(Composite parent, Box.Value value) {
        super(createTextbox(parent, value.getPod().getString()));
      }

      @Override
      public void update(Box.Value.Builder param) {
        param.getPodBuilder().setString(control.getText());
      }
    }
  }
}
