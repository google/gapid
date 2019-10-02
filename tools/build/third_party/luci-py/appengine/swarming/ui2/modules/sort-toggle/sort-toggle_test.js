// Copyright 2018 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

import 'modules/sort-toggle'

describe('sort-toggle', function() {
  // A reusable HTML element in which we create our element under test.
  const container = document.createElement('div');
  document.body.appendChild(container);

  afterEach(function() {
    container.innerHTML = '';
  });

//===============TESTS START====================================

  // calls the test callback with one element 'ele', a created <sort-toggle>.
  function createElement(test) {
    return window.customElements.whenDefined('sort-toggle').then(() => {
      container.innerHTML = `<sort-toggle></sort-toggle>`;
      expect(container.firstElementChild).toBeTruthy();
      test(container.firstElementChild);
    });
  }

  it('displays both arrows when currentKey != key', function(done) {
    createElement((ele) => {
      ele.currentKey = 'alpha';
      ele.key = 'beta';
      let arrow = ele.querySelector('arrow-drop-down-icon-sk');
      expect(arrow).toBeTruthy();
      expect(arrow.attributes.hidden).toBeFalsy();
      arrow = ele.querySelector('arrow-drop-up-icon-sk');
      expect(arrow).toBeTruthy();
      expect(arrow.attributes.hidden).toBeFalsy();
      done();
    });
  });

  it('displays only one arrow when the active choice', function(done) {
    createElement((ele) => {
      ele.currentKey = 'beta';
      ele.key = 'beta';
      ele.direction = 'asc';

      let arrow = ele.querySelector('arrow-drop-down-icon-sk');
      expect(arrow).toBeTruthy();
      expect(arrow.attributes.hidden).toBeTruthy('We should only display up arrow');
      arrow = ele.querySelector('arrow-drop-up-icon-sk');
      expect(arrow).toBeTruthy();
      expect(arrow.attributes.hidden).toBeFalsy();

      ele.direction = 'desc';
      arrow = ele.querySelector('arrow-drop-down-icon-sk');
      expect(arrow).toBeTruthy();
      expect(arrow.attributes.hidden).toBeFalsy();
      arrow = ele.querySelector('arrow-drop-up-icon-sk');
      expect(arrow).toBeTruthy();
      expect(arrow.attributes.hidden).toBeTruthy('We should only display down arrow');

      done();
    });
  });

  it('emits a sort-change event and toggles on click', function(done) {
    createElement((ele) => {
      ele.currentKey = 'beta';
      ele.key = 'beta';
      ele.direction = 'asc';

      ele.addEventListener('sort-change', (e) => {
        expect(e.detail.direction).toBe('desc');
        expect(e.detail.key).toBe('beta');

        expect(ele.direction).toBe('desc');
        let arrow = ele.querySelector('arrow-drop-down-icon-sk');
        expect(arrow).toBeTruthy();
        expect(arrow.attributes.hidden).toBeFalsy();
        arrow = ele.querySelector('arrow-drop-up-icon-sk');
        expect(arrow).toBeTruthy();
        expect(arrow.attributes.hidden).toBeTruthy('We should only display down arrow');

        done();
      });

      ele.click();
    });
  });

  it('defaults to ascending when changing "currentKey"', function(done) {
    createElement((ele) => {
      ele.currentKey = 'alpha';
      ele.key = 'beta';
      ele.direction = 'asc';

      ele.addEventListener('sort-change', (e) => {
        // No matter what the old direction was, the first
        // click on change should be ascending
        expect(e.detail.direction).toBe('asc');
        expect(e.detail.key).toBe('beta');

        expect(ele.direction).toBe('asc');
        done();
      });

      ele.click();
    });
  });
});