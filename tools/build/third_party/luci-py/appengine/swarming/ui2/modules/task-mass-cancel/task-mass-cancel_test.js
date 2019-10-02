// Copyright 2019 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

import 'modules/task-mass-cancel'

describe('task-mass-cancel', function() {
  // Instead of using import, we use require. Otherwise,
  // the concatenation trick we do doesn't play well with webpack, which would
  // leak dependencies (e.g. bot-list's 'column' function to task-list) and
  // try to import things multiple times.
  const { $, $$ } = require('common-sk/modules/dom');
  const { customMatchers, expectNoUnmatchedCalls, mockAppGETs } = require('modules/test_util');
  const { fetchMock, MATCHED, UNMATCHED } = require('fetch-mock');
  // A reusable HTML element in which we create our element under test.
  const container = document.createElement('div');
  document.body.appendChild(container);

  beforeEach(function() {
    jasmine.addMatchers(customMatchers);

    mockAppGETs(fetchMock, {delete_bot: true});

    fetchMock.get('glob:/_ah/api/swarming/v1/tasks/count?*', {'count': 17});

    // Everything else
    fetchMock.catch(404);
  });

  afterEach(function() {
    container.innerHTML = '';
    // Completely remove the mocking which allows each test
    // to be able to mess with the mocked routes w/o impacting other tests.
    fetchMock.reset();
  });

  // calls the test callback with one element 'ele', a created <task-mass-cancel>.
  function createElement(test) {
    return window.customElements.whenDefined('task-mass-cancel').then(() => {
      container.innerHTML = `<task-mass-cancel tags="pool:Chrome,os:Android" auth_header="fake"></task-mass-cancel>`;
      expect(container.firstElementChild).toBeTruthy();
      test(container.firstElementChild);
    });
  }

//===============TESTS START====================================

  it('can read in attributes', function(done) {
    createElement((ele) => {
      expect(ele.tags.length).toBe(2);
      expect(ele.tags).toContain('pool:Chrome');
      expect(ele.tags).toContain('os:Android');
      expect(ele.auth_header).toBe('fake');
      done();
    });
  });

  it('has a list of the passed in dimensions', function(done) {
    createElement((ele) => {
      const tags = $('ul li', ele);
      expect(tags.length).toBe(2);
      expect(tags[0]).toMatchTextContent('os:Android');
      done();
    });
  });

  it('makes 2 API calls to count when loading', function(done) {
    createElement((ele) => {
      ele.show();
      // The true on flush waits for res.json() to resolve too, which
      // is when we know the element has updated the _tasks.
      fetchMock.flush(true).then(() => {
        expectNoUnmatchedCalls(fetchMock);
        const calls = fetchMock.calls(MATCHED, 'GET');
        expect(calls.length).toBe(2);
        done();
      });
    });
  });

  it('makes an API call to delete after clicking', function(done) {
    createElement((ele) => {

      fetchMock.post('/_ah/api/swarming/v1/tasks/cancel', {matched: 22});

      let sawStartEvent = false;
      ele.addEventListener('tasks-canceling-started', () => {
        sawStartEvent = true;
      });

      ele.addEventListener('tasks-canceling-finished', () => {
        expect(sawStartEvent).toBeTruthy();
        expectNoUnmatchedCalls(fetchMock);

        const calls = fetchMock.calls(MATCHED, 'POST');
        expect(calls.length).toBe(1, '1 to delete');
        done();
      });

      ele._readyToCancel = true;
      ele.render();
      const button = $$('button.cancel', ele);
      button.click();
    });
  });

});