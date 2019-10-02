// Copyright 2018 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

import 'modules/swarming-index'

describe('swarming-index', function() {
  // Instead of using import, we use require. Otherwise,
  // the concatenation trick we do doesn't play well with webpack, which would
  // leak dependencies (e.g. bot-list's 'column' function to task-list) and
  // try to import things multiple times.
  const { fetchMock, MATCHED, UNMATCHED } = require('fetch-mock');
  const { expectNoUnmatchedCalls, mockAppGETs } = require('modules/test_util');

  beforeEach(function(){
    // These are the default responses to the expected API calls (aka 'matched')
    // They can be overridden for specific tests, if needed.
    mockAppGETs(fetchMock, {
      get_bootstrap_token: false
    });

    fetchMock.post('/_ah/api/swarming/v1/server/token', 403);

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

  // calls the test callback with one element 'ele', a created <swarming-index>.
  // We can't put the describes inside the whenDefined callback because
  // that doesn't work on Firefox (and possibly other places).
  function createElement(test) {
    return window.customElements.whenDefined('swarming-index').then(() => {
      container.innerHTML =
          `<swarming-index client_id=for_test testing_offline=true>
          </swarming-index>`;
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

  function becomeAdmin() {
    // overwrite the default fetchMock behaviors for this run to return
    // what an admin would see.
    fetchMock.get('/_ah/api/swarming/v1/server/permissions', {
      get_bootstrap_token: true
    }, { overwriteRoutes: true });
    fetchMock.post('/_ah/api/swarming/v1/server/token', {
      bootstrap_token: '8675309JennyDontChangeYourNumber8675309'
    }, { overwriteRoutes: true });
  }

//===============TESTS START====================================

  describe('html structure', function() {
    it('contains swarming-app as its only child', function(done) {
      createElement((ele) => {
        expect(ele.children.length).toBe(1);
        expect(ele.children[0].tagName).toBe('swarming-app'.toUpperCase());
        done();
      });
    });

    describe('when not logged in', function() {
      it('tells the user they should log in', function(done) {
        createElement((ele) => {
          const serverVersion = ele.querySelector('swarming-app>main .server_version');
          expect(serverVersion).toBeTruthy();
          expect(serverVersion.innerText).toContain('must log in');
          done();
        })
      })
      it('does not display the bootstrapping section', function(done) {
        createElement((ele) => {
          const sectionHeaders = ele.querySelectorAll('swarming-app>main h2');
          expect(sectionHeaders).toBeTruthy();
          expect(sectionHeaders.length).toBe(2);
          done();
        })
      });
    });

    describe('when logged in as unauthorized user', function() {

      function notAuthorized() {
        // overwrite the default fetchMock behaviors to have everything return 403.
        fetchMock.get('/_ah/api/swarming/v1/server/details', 403,
                      { overwriteRoutes: true });
        fetchMock.get('/_ah/api/swarming/v1/server/permissions', {},
                      { overwriteRoutes: true });
      }

      beforeEach(notAuthorized);

      it('tells the user to try a different account', function(done){
        createElement((ele) => {
          userLogsIn(ele, () => {
            const serverVersion = ele.querySelector('swarming-app>main .server_version');
            expect(serverVersion).toBeTruthy();
            expect(serverVersion.innerText).toContain('different account');
            done();
          });
        });
      });
      it('does not displays the bootstrapping section', function(done){
        createElement((ele) => {
          userLogsIn(ele, () => {
            const sectionHeaders = ele.querySelectorAll('swarming-app>main h2');
            expect(sectionHeaders).toBeTruthy();
            expect(sectionHeaders.length).toBe(2);
            done();
          });
        });
      });
      it('does not display the bootstrap token', function(done){
        createElement((ele) => {
          userLogsIn(ele, () => {
            const commandBox = ele.querySelector('swarming-app>main .command');
            expect(commandBox).toBeNull();
            done();
          });
        });
      });
    });

    describe('when logged in as user (no bootstrap_token)', function() {
      it('displays the server version', function(done) {
        createElement((ele) => {
          userLogsIn(ele, () => {
            const serverVersion = ele.querySelector('swarming-app>main .server_version');
            expect(serverVersion).toBeTruthy();
            expect(serverVersion.innerText).toContain('1234-abcdefg');
            done();
          });
        });
      });
      it('does not displays the bootstrapping section', function(done) {
        createElement((ele) => {
          userLogsIn(ele, () => {
            const sectionHeaders = ele.querySelectorAll('swarming-app>main h2');
            expect(sectionHeaders).toBeTruthy();
            expect(sectionHeaders.length).toBe(2);
            done();
          });
        });
      });
      it('does not display the bootstrap token', function(done) {
        createElement((ele) => {
          userLogsIn(ele, () => {
            const commandBox = ele.querySelector('swarming-app>main .command');
            expect(commandBox).toBeNull();
            done();
          });
        });
      });
    });

    describe('when logged in as admin (boostrap_token)', function() {
      beforeEach(becomeAdmin);

      it('displays the server version', function(done) {
        createElement((ele) => {
          userLogsIn(ele, () => {
            const serverVersion = ele.querySelector('swarming-app>main .server_version');
            expect(serverVersion).toBeTruthy();
            expect(serverVersion.innerText).toContain('1234-abcdefg');
            done();
          });
        });
      });
      it('displays the bootstrapping section', function(done) {
        createElement((ele) => {
          userLogsIn(ele, () => {
            const sectionHeaders = ele.querySelectorAll('swarming-app>main h2');
            expect(sectionHeaders).toBeTruthy();
            expect(sectionHeaders.length).toBe(3);
            done();
          });
        });
      });
      it('displays the bootstrap token', function(done) {
        createElement((ele) => {
          userLogsIn(ele, () => {
            // There are several of these, but we'll just check one of them.
            const commandBox = ele.querySelector('swarming-app>main .command');
            expect(commandBox).toBeTruthy();
            expect(commandBox.innerText).toContain('8675309');
            done();
          });
        });
      });
    });
  }); // end describe('html structure')

  describe('api calls', function() {
    it('makes no API calls when not logged in', function(done) {
      createElement((ele) => {
        fetchMock.flush(true).then(() => {
          // MATCHED calls are calls that we expect and specified in the
          // beforeEach at the top of this file.
          let calls = fetchMock.calls(MATCHED, 'GET');
          expect(calls.length).toBe(0);
          calls = fetchMock.calls(MATCHED, 'POST');
          expect(calls.length).toBe(0);

          expectNoUnmatchedCalls(fetchMock);
          done();
        });
      });
    });

    it('does not request a token when a normal user logs in', function(done) {
      createElement((ele) => {
        userLogsIn(ele, () => {
          // swarming-app makes some GETs and swarming-app_test.js tests that.
          const calls = fetchMock.calls(MATCHED, 'POST');
          expect(calls.length).toBe(0);

          expectNoUnmatchedCalls(fetchMock);
          done();
        });
      });
    });

    it('fetches a token when an admin logs in', function(done) {
      becomeAdmin();
      createElement((ele) => {
        userLogsIn(ele, () => {
           // swarming-app makes the GETs and swarming-app_test.js tests that.

          const calls = fetchMock.calls(MATCHED, 'POST');
          const posts = calls.map((c) => c[0]);
          expect(calls.length).toBe(1);
          expect(posts).toContain('/_ah/api/swarming/v1/server/token');

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
  }); // end describe('api calls')
});
