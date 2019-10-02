// Copyright 2018 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

import 'modules/swarming-app'

describe('swarming-app', function() {
  // Instead of using import, we use require. Otherwise,
  // the concatenation trick we do doesn't play well with webpack, which would
  // leak dependencies (e.g. bot-list's 'column' function to task-list) and
  // try to import things multiple times.
  const { expectNoUnmatchedCalls, mockAppGETs } = require('modules/test_util');
  const { fetchMock, MATCHED, UNMATCHED } = require('fetch-mock');

  beforeEach(function(){
    // These are the default responses to the expected API calls (aka 'matched')
    // They can be overridden for specific tests, if needed.
    mockAppGETs(fetchMock, {
      can_like_dogs: true,
      can_like_cats: true
    });

    // Everything else
    fetchMock.catch(404);
  });

  afterEach(function() {
    // Completely remove the mocking which allows each test
    // to be able to mess with the mocked routes w/o impacting other tests.
    fetchMock.reset();
  });

  // A reusable HTML element in which we create our element under test.
  const container = document.createElement('div');
  document.body.appendChild(container);

  afterEach(function() {
    container.innerHTML = '';
  });

  // calls the test callback with one element 'ele', a created <swarming-app>.
  // We can't put the describes inside the whenDefined callback because
  // that doesn't work on Firefox (and possibly other places).
  function createElement(test) {
    return window.customElements.whenDefined('swarming-app').then(() => {
      container.innerHTML = `
          <swarming-app testing_offline=true>
            <header>
              <aside class=hideable>Menu option</aside>
            </header>
            <main></main>
            <footer></footer>
          </swarming-app>`;
      expect(container.firstElementChild).toBeTruthy();
      test(container.firstElementChild);
    });
  }

  function userLogsIn(ele, callback) {
    // The swarming-app emits the 'busy-end' event when all pending
    // fetches (and renders) have resolved.
    let ran = false;
    ele.addEventListener('busy-end', (e) => {
      if (!ran) {
        callback();
      }
      ran = true; // prevent multiple runs if the test makes the
                  // app go busy (e.g. if it calls fetch).
    });
    const login = ele.querySelector('oauth-login');
    login._logIn();
    fetchMock.flush();
  }

//===============TESTS START====================================

  describe('html injection to provided content', function() {

    it('injects login element and sidebar buttons', function(done) {
      createElement((ele) => {
        const button = ele.querySelector('header button');
        expect(button).toBeTruthy();
        const spacer = ele.querySelector('header .grow');
        expect(spacer).toBeTruthy();
        const login = ele.querySelector('header oauth-login');
        expect(login).toBeTruthy();
        const spinner = ele.querySelector('header spinner-sk');
        expect(spinner).toBeTruthy();
        const serverVersion = ele.querySelector('header .server-version');
        expect(serverVersion).toBeTruthy();
        done();
      });
    });

    it('does not inject content if missing .hideable', function(done) {
      window.customElements.whenDefined('swarming-app').then(() => {
        container.innerHTML = `
        <swarming-app testing_offline=true>
          <header>
            <aside>Menu option</aside>
          </header>
          <main></main>
          <footer></footer>
        </swarming-app>`;
        const ele = container.firstElementChild;
        expect(ele).toBeTruthy();
        const button = ele.querySelector('header button');
        expect(button).toBeNull();
        const spacer = ele.querySelector('header .grow');
        expect(spacer).toBeNull();
        const login = ele.querySelector('header oauth-login');
        expect(login).toBeNull();
        done();
      });
    });
  });  // end describe('html injection to provided content')

  describe('sidebar', function() {
    it('should toggle', function(done) {
      createElement((ele) => {
        const button = ele.querySelector('header button');
        expect(button).toBeTruthy();
        const sidebar = ele.querySelector('header aside');
        expect(sidebar).toBeTruthy();

        expect(sidebar.classList).not.toContain('shown');
        button.click();
        expect(sidebar.classList).toContain('shown');
        button.click();
        expect(sidebar.classList).not.toContain('shown');
        done();
      });
    });
  }); // end describe('sidebar')

  describe('footer', function() {
    it('has general purpose elements', function(done) {
      createElement((ele) => {
        const errorToast = ele.querySelector('footer error-toast-sk');
        expect(errorToast).toBeTruthy();
        const fab = ele.querySelector('footer .fab');
        expect(fab).toBeTruthy();

        // fab should be in an anchor tab
        expect(fab.parentElement.tagName).toEqual('A');
        expect(fab.parentElement.outerHTML).toContain('bugs.chromium.org/p/chromium/issues/entry');
        done();
      });
    });
  }); // end describe('footer')

  describe('spinner and busy property', function() {
    it('becomes busy while there are tasks to be done', function(done) {
      createElement((ele) => {
        expect(ele.busy).toBeFalsy();
        ele.addBusyTasks(2);
        expect(ele.busy).toBeTruthy();
        ele.finishedTask();
        expect(ele.busy).toBeTruthy();
        ele.finishedTask();
        expect(ele.busy).toBeFalsy();
        done();
      });
    });

    it('keeps spinner active while busy', function(done) {
      createElement((ele) => {
        const spinner = ele.querySelector('header spinner-sk');
        expect(spinner.active).toBeFalsy();
        ele.addBusyTasks(2);
        expect(spinner.active).toBeTruthy();
        ele.finishedTask();
        expect(spinner.active).toBeTruthy();
        ele.finishedTask();
        expect(spinner.active).toBeFalsy();
        done();
      });
    });

    it('emits a busy-end task when tasks finished', function(done) {
      createElement((ele) => {
        ele.addEventListener('busy-end', (e) => {
          e.stopPropagation();
          expect(ele.busy).toBeFalsy();
          done();
        });
        ele.addBusyTasks(1);

        setTimeout(()=>{
          ele.finishedTask();
        }, 10);
      });
    });
  }); // end describe('spinner and busy property')

  describe('behavior with logged-in user', function() {

    describe('html content', function() {
      it('adds a server version indicator to the header', function(done){
        createElement((ele) => {
          ele.addEventListener('server-details-loaded', (e) => {
            e.stopPropagation();
            const serverVersion = ele.querySelector('header .server-version');
            expect(serverVersion).toBeTruthy();
            expect(serverVersion.textContent).toContain('1234-abcdefg');
            done();
          });
          const serverVersion = ele.querySelector('header .server-version');
          expect(serverVersion).toBeTruthy();
          expect(serverVersion.textContent).toContain('must log in');
          userLogsIn(ele, () => {});
        });
      });
    });

    describe('api calls', function(){
      it('makes no API calls when not logged in', function(done) {
        createElement((ele) => {
          fetchMock.flush(true).then(() => {
            // MATCHED calls are calls that we expect and specified in the
            // beforeEach at the top of this file.
            const calls = fetchMock.calls(MATCHED, 'GET');
            expect(calls.length).toBe(0);

            expectNoUnmatchedCalls(fetchMock);
            done();
          });
        });
      });

      it('makes authenticated calls when a user logs in', function(done) {
        createElement((ele) => {
          userLogsIn(ele, () => {
            const calls = fetchMock.calls(MATCHED, 'GET');
            expect(calls.length).toBe(2);
            // calls is an array of 2-length arrays with the first element
            // being the string of the url and the second element being
            // the options that were passed in
            const gets = calls.map((c) => c[0]);
            expect(gets).toContain('/_ah/api/swarming/v1/server/details');
            expect(gets).toContain('/_ah/api/swarming/v1/server/permissions');

            // check authorization headers are set
            calls.forEach((c) => {
              expect(c[1].headers).toBeDefined();
              expect(c[1].headers.authorization).toContain('Bearer ');
            })

            expectNoUnmatchedCalls(fetchMock);
            done();
          });
        });
      });
    });

    describe('events', function(){
      it('emits a permissions-loaded event', function(done) {
        createElement((ele) => {
          ele.addEventListener('permissions-loaded', (e) => {
            e.stopPropagation();
            expect(ele.permissions).toBeTruthy();
            expect(ele.permissions.can_like_dogs).toBeTruthy();
            done();
          });
          expect(ele.permissions).toEqual({});
          userLogsIn(ele, () => {});
        });
      });

      it('emits a server-details-loaded event', function(done) {
        createElement((ele) => {
          ele.addEventListener('server-details-loaded', (e) => {
            e.stopPropagation();
            expect(ele.server_details).toBeTruthy();
            expect(ele.server_details.server_version).toBe('1234-abcdefg');
            done();
          });
          expect(ele.server_details.server_version).toContain('must log in');
          userLogsIn(ele, () => {});
        });
      });
    });

  }); // end describe('spinner and busy property')
});