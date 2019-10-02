// Copyright 2018 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

/** @module swarming-ui/modules/oauth-login
 * @description <h2><code>oauth-login</code></h2>
 * oauth-login is a small widget that handles an OAuth 2.0 flow with Google.
 *
 * <p>
 *  This widget either a sign in button or displays the info of the
 *  logged-in user and a sign out button. The widget fires an event
 *  when the user has logged in and clients should be able to make
 *  authenticated requests.
 * </p>
 *
 * <p>
 *  Clients should include the following JS script to supply the Google OAuth
 *  code:
 *    <script src="https://apis.google.com/js/api.js" async defer></script>
 * </p>
 *
 * @prop client_id - The Client ID for authenticating via OAuth.
 * @prop testing_offline - If true, the real OAuth flow won't be used.
 *    Instead, dummy data will be used. Ideal for local testing.
 *
 * @evt log-in The event that is fired when the user has logged in
 *             has a detail of the form:
 *
 * <pre>
 * {
 *   auth_header: "Bearer abc12d",
 *   profile: {
 *     email: 'foo@example.com',
 *     imageURL: 'http://example.com/img.png',
 *   }
 * }
 * </pre>
 * where auth_header is a string that should be used as the
 * "Authorization" header for authenticated requests.
 *
 */

import { html, render } from 'lit-html'
import { upgradeProperty } from 'elements-sk/upgradeProperty'
import { errorMessage } from 'elements-sk/errorMessage'

// gapiLoaded is a promise that resolves when the 'gapi' JS library is
// finished loading.
const gapiLoaded = new Promise((resolve, reject) => {
  const check = () => {
    if (window.gapi !== undefined) {
      resolve();
    } else {
      setTimeout(check, 10)
    }
  }
  setTimeout(check, 10)
});

const template = (ele) => {
  if (ele.auth_header) {
    return html`
<div>
  <img class=center id=avatar src="${ele._profile.imageURL}" width=30 height=30>
  <span class=center>${ele._profile.email}</span>
  <span class=center>|</span>
  <a class=center @click=${ele._logOut} href="#">Sign out</a>
</div>`;
  } else {
    return html`
<div>
  <a @click=${ele._logIn} href="#">Sign in</a>
</div>`;
  }
};

window.customElements.define('oauth-login', class extends HTMLElement {

  connectedCallback() {
    upgradeProperty(this, 'client_id');
    upgradeProperty(this, 'testing_offline');
    this._auth_header = '';
    if (this.testing_offline) {
      // For local testing, set a profile here. A real profile would be null
      // until the user logs in.
      this._profile = {
        email: 'missing@chromium.org',
        imageURL: 'http://storage.googleapis.com/gd-wagtail-prod-assets/original_images/logo_google_fonts_color_2x_web_64dp.png',
      };
    } else {
      this._profile = null;
      gapiLoaded.then(() => {
        gapi.load('auth2', () => {
          gapi.auth2.init({
            client_id: this.client_id,
          }).then(() => {
            this._maybeFireLoginEvent();
            this.render();
          }, (error) => {
            console.error(error);
            errorMessage(`Error initializing oauth: ${JSON.stringify(error)}`, 10000);
          });
        });
      });

    }
    this.render();
  }

  static get observedAttributes() {
    return ['client_id', 'testing_offline'];
  }

  /** @prop {string} auth_header the "Authorization" header that should be used
  *                  for authenticated requests. Read-only. */
  get auth_header() { return this._auth_header;}

  /** @prop {string} client_id To be used in the OAuth 2.0 flow. This is generally
                               supplied by the server. */
  get client_id() { return this.getAttribute('client_id');}
  set client_id(val) { return this.setAttribute('client_id', val);}

  /** @prop {Object} profile An object with keys email and imageURL of the
                             logged in user. Read Only. */
  get profile() { return this._profile; }

  /** @prop {bool} testing_offline Mirrors the attribute 'testing_offline'. */
  get testing_offline() { return this.hasAttribute('testing_offline');}
  set testing_offline(val) {
    if (val) {
      this.setAttribute('testing_offline', true);
    } else {
      this.removeAttribute('testing_offline');
    }
  }

  _maybeFireLoginEvent() {
    const user = gapi.auth2.getAuthInstance().currentUser.get();
    if (user.isSignedIn()) {
      const profile = user.getBasicProfile();
      this._profile = {
        email: profile.getEmail(),
        imageURL: profile.getImageUrl()
      };
      // Need the true here to get an access_token on the response.
      const auth = user.getAuthResponse(true);

      const header = `${auth.token_type} ${auth.access_token}`
      this.dispatchEvent(new CustomEvent('log-in', {
        detail: {
          'auth_header': header,
          'profile': this._profile,
        },
        bubbles: true,
      }));
      this._auth_header = header;
      return true;
    } else {
      this._profile = null;
      this._auth_header = '';
      return false;
    }
  }

  _logIn() {
    if (this.testing_offline) {
        this._auth_header = 'Bearer 12345678910-boomshakalaka';
        this.dispatchEvent(new CustomEvent('log-in', {
          detail: {
            'auth_header': this._auth_header,
            'profile': this._profile,
          },
          bubbles: true,
        }));
        this.render();
      } else {
        const auth = gapi.auth2.getAuthInstance();
        if (auth) {
          auth.signIn({
            scope: 'email',
            prompt: 'select_account',
          }).then(() => {
            if (!this._maybeFireLoginEvent()) {
              console.warn('login was not successful; maybe user canceled');
            }
            this.render();
          });
        }
      }
  }

  _logOut() {
    if (this.testing_offline) {
      this._auth_header = '';
      this.render();
      // reload the page to clear any sensitive data being displayed.
      window.location.reload();
    } else {
      const auth = gapi.auth2.getAuthInstance();
      if (auth) {
        auth.signOut().then(() => {
          this._auth_header = '';
          this._profile = null;
          // reload the page to clear any sensitive data being displayed.
          window.location.reload();
        });
      }
    }
  }

  render() {
    render(template(this), this, {eventContext: this});
  }

  attributeChangedCallback(attrName, oldVal, newVal) {
    this.render();
  }

});
