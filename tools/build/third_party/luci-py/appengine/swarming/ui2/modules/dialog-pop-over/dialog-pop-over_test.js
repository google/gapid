// Copyright 2019 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

import 'modules/dialog-pop-over'

describe('dialog-pop-over', function() {
  // Instead of using import, we use require. Otherwise,
  // the concatenation trick we do doesn't play well with webpack, which would
  // leak dependencies (e.g. bot-list's 'column' function to task-list) and
  // try to import things multiple times.
  const { $$ } = require('common-sk/modules/dom');

  // A reusable HTML element in which we create our element under test.
  const container = document.createElement('div');
  document.body.appendChild(container);

  afterEach(function() {
    container.innerHTML = '';
  });

//===============TESTS START====================================

  // calls the test callback with one element 'ele', a created <dialog-pop-over>.
  function createElement(test) {
    return window.customElements.whenDefined('dialog-pop-over').then(() => {
      container.innerHTML = `<dialog-pop-over>
        <div class=content></div>
        <button class=negative>Cancel</button>
        <button class=positive>OK</button>
      </dialog-pop-over>`;
      expect(container.firstElementChild).toBeTruthy();
      test(container.firstElementChild);
    });
  }

  it('appends a backdrop element', function(done) {
    createElement((ele) => {
      const backdrop = $$('.backdrop', ele);
      expect(backdrop).toBeTruthy();
      done();
    });
  });

  it('toggles .opened on .content and .backdrop on show/hide', function(done) {
    createElement((ele) => {
      const backdrop = $$('.backdrop', ele);
      const content = $$('.content', ele);
      expect(backdrop).not.toHaveClass('opened');
      expect(content).not.toHaveClass('opened');

      ele.show();
      expect(backdrop).toHaveClass('opened');
      expect(content).toHaveClass('opened');

      ele.hide();
      expect(backdrop).not.toHaveClass('opened');
      expect(content).not.toHaveClass('opened');
      done();
    });
  });
});