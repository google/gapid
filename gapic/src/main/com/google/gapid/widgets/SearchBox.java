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

import com.google.gapid.util.Events;

import org.eclipse.swt.SWT;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Button;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Text;

import java.util.regex.Pattern;
import java.util.regex.PatternSyntaxException;

/**
 * A search box widget supporting regex searches.
 */
public class SearchBox extends Composite {
  public SearchBox(Composite parent) {
    super(parent, SWT.NONE);
    setLayout(new GridLayout(2, false));

    Text text = new Text(this, SWT.SINGLE | SWT.SEARCH | SWT.ICON_SEARCH | SWT.ICON_CANCEL);
    Button regex = Widgets.createCheckbox(this, "Regex", true);

    text.setLayoutData(new GridData(SWT.FILL, SWT.TOP, true, false));
    regex.setLayoutData(new GridData(SWT.RIGHT, SWT.TOP, false, false));

    text.addListener(SWT.DefaultSelection, e -> {
      notifyListeners(Events.Search,
          Events.newSearchEvent(SearchBox.this, text.getText(), regex.getSelection()));
    });
  }

  public static Pattern getPattern(String text, boolean regex) {
    Pattern result = null;
    if (regex) {
      try {
        result = Pattern.compile(text, Pattern.CASE_INSENSITIVE);
      } catch (PatternSyntaxException e) {
        // Ignore.
      }
    }
    if (result == null) {
      result = Pattern.compile(Pattern.quote(text), Pattern.CASE_INSENSITIVE);
    }
    return result;
  }
}
