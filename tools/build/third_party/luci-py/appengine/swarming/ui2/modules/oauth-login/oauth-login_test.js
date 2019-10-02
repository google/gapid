// Copyright 2018 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

import 'modules/oauth-login'

describe('oauth-login', function() {
  // A reusable HTML element in which we create our element under test.
  const container = document.createElement('div');
  document.body.appendChild(container);

  afterEach(function() {
    container.innerHTML = '';
  });

//===============TESTS START====================================

  describe('testing-offline true', function() {

    // calls the test callback with one element 'ele', a created <oauth-login>.
    function createElement(test) {
      return window.customElements.whenDefined('oauth-login').then(() => {
        container.innerHTML = `<oauth-login client_id=fake testing_offline=true></oauth-login>`;
        expect(container.firstElementChild).toBeTruthy();
        expect(container.firstElementChild.testing_offline).toBeTruthy();
        test(container.firstElementChild);
      });
    }

    it('starts off logged out', function(done) {
      createElement((ele) => {
        expect(ele.auth_header).toBe('');
        expect(ele.client_id).toBe('fake');
        done();
      });
    });

    it('triggers a log-in custom event on login', function(done) {
      createElement((ele) => {
        ele.addEventListener('log-in', (e) => {
          e.stopPropagation();
          expect(e.detail).toBeDefined();
          expect(e.detail.auth_header).toContain('Bearer ');
          done();
        });
        ele._logIn();
      });
    });

    it('has auth_header set after log-in', function(done) {
      createElement((ele) => {
        ele._logIn();
        expect(ele.auth_header).toContain('Bearer ');
        done();
      });
    });

  }); // end describe('testing-offline true')

  describe('testing-offline false', function() {

    // calls the test callback with one element 'ele', a created <oauth-login>.
    function createElement(test) {
      return window.customElements.whenDefined('oauth-login').then(() => {
        container.innerHTML = `<oauth-login client_id=fake></oauth-login>`;
        expect(container.firstElementChild).toBeTruthy();
        expect(container.firstElementChild.testing_offline).toBeFalsy();
        test(container.firstElementChild);
      });
    }

    beforeEach(function(){
      // Stub this out to return a blank promise, which allows any inits
      // called to gracefully do nothing.
      window.gapi = {
        auth2: {
          init: jasmine.createSpy('init').and.returnValue(new Promise(()=>{})),
        },
        load: jasmine.createSpy('load').and.returnValue(new Promise(()=>{})),
      };
    });

    it('starts off logged out', function(done) {
      createElement((ele) => {
        expect(ele.auth_header).toBe('');
        expect(ele.client_id).toBe('fake');
        done();
      });
    });

    it('calls gapi.signIn on call to _logIn', function(done) {
      createElement((ele) => {
        // createSpyObj is a little different than createSpy in that it can make
        // multiple spies at once (the array) and packages them in an object
        // such that you can do something like mockAuthInstance.signIn()
        const mockAuthInstance = jasmine.createSpyObj('authInstance', ['signIn']);

        mockAuthInstance.signIn.and.callFake((obj) => {
          expect(obj.scope).toBe('email');
          expect(obj.prompt).toBe('select_account');
          done();
          return new Promise(()=>{});
        });
        window.gapi = {
          auth2: {
            getAuthInstance: jasmine.createSpy('getAuthInstance').and.returnValue(mockAuthInstance),
          },
          load: jasmine.createSpy('load').and.returnValue(new Promise(()=>{})),
        };

        ele._logIn();
      });
    });

    it('calls gapi.signOut on call to _logOut', function(done) {
      createElement((ele) => {
        const mockAuthInstance = jasmine.createSpyObj('authInstance', ['signOut']);

        mockAuthInstance.signOut.and.callFake(() => {
          done();
          return new Promise(()=>{});
        });
        window.gapi = {
          auth2: {
            getAuthInstance: jasmine.createSpy('getAuthInstance').and.returnValue(mockAuthInstance),
          },
          load: jasmine.createSpy('load').and.returnValue(new Promise(()=>{})),
        };

        ele._logOut();
      });
    });
  }); // end describe('testing-offline false')

});
