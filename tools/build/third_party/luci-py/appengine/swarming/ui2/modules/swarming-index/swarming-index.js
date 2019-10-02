// Copyright 2018 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

/** @module swarming-ui/modules/swarming-index
 * @description <h2><code>swarming-index</code></h2>
 *
 * <p>
 *  Swarming Index is the landing page for the Swarming UI.
 *  It will have links to all other pages and a high-level overview of the fleet.
 * </p>
 *
 * <p>This is a top-level element.</p>
 *
 * @extends module:swarming-ui/SwarmingAppBoilerplate
 */

import { html, render } from 'lit-html'
import { upgradeProperty } from 'elements-sk/upgradeProperty'
import { jsonOrThrow } from 'common-sk/modules/jsonOrThrow'
import { errorMessage } from 'elements-sk/errorMessage'

import SwarmingAppBoilerplate from '../SwarmingAppBoilerplate'

import '../swarming-app'

// Don't use html for a straight string template, otherwise, it shows up
// as [object Object] when used as the href attribute.
const instancesURL = (ele) => `https://console.cloud.google.com/appengine/instances`+
    `project=${ele._project_id}&versionId=${ele.server_details.server_version}`;

const errorsURL = (project_id) =>
    `https://console.cloud.google.com/errors?project=${project_id}`;

const logsURL = (project_id) =>
    `https://console.cloud.google.com/logs/viewer?filters=status:500..599&project=${project_id}`;

const bootstrapTemplate = (ele) => html`
<div>
  <h2>Bootstrapping a bot</h2>
  To bootstrap a bot, run one of these (all links are valid for 1 hour):
  <ol>
    <li>
      <strong> TL;DR; </strong>
<pre class=command>python -c "import urllib; exec urllib.urlopen('${ele._host_url}/bootstrap?tok=${ele._bootstrap_token}').read()"</pre>
    </li>
    <li>
      Escaped version to pass as a ssh argument:
<pre class=command>'python -c "import urllib; exec urllib.urlopen('"'${ele._host_url}/bootstrap?tok=${ele._bootstrap_token}'"').read()"'</pre>
    </li>
    <li>
      Manually:
<pre class=command>mkdir bot; cd bot
rm -f swarming_bot.zip; curl -sSLOJ ${ele._host_url}/bot_code?tok=${ele._bootstrap_token}
python swarming_bot.zip</pre>
    </li>
  </ol>
</div>
`;

const template = (ele) => html`
<swarming-app id=swapp
              client_id="${ele.client_id}"
              ?testing_offline="${ele.testing_offline}">
  <header>
    <div class=title>Swarming</div>
      <aside class=hideable>
        <a href=/>Home</a>
        <a href=/botlist>Bot List</a>
        <a href=/tasklist>Task List</a>
        <a href=/bot>Bot Page</a>
        <a href=/task>Task Page</a>
      </aside>
  </header>
  <main>

    <h2>Service Status</h2>
    <div>Server Version:
      <span class=server_version> ${ele.server_details.server_version}</span>
    </div>
    <div>Bot Version: ${ele.server_details.bot_version} </div>
    <ul>
      <li>
        <!-- TODO(kjlubick) convert these linked pages to new UI-->
        <a href=/stats>Usage statistics</a>
      </li>
      <li>
        <a href=/restricted/mapreduce/status>Map Reduce Jobs</a>
      </li>
      <li>
        <a href=${instancesURL(ele)}>View version's instances on Cloud Console</a>
      </li>
      <li>
        <a href=${errorsURL(ele._project_id)}>View server errors on Cloud Console</a>
      </li>
      <li>
        <a href=${logsURL(ele._project_id)}>View logs for HTTP 5xx on Cloud Console</a>
      </li>
      <li>
        <a href="/restricted/ereporter2/report">View ereporter2 report</a>
      </li>
    </ul>

    <h2>Configuration</h2>
    <ul>
      <!-- TODO(kjlubick) convert these linked pages to new UI-->
      <li>
        <a href="/restricted/config">View server config</a>
      </li>
      <li>
        <a href="/restricted/upload/bootstrap">View/upload bootstrap.py</a>
      </li>
      <li>
        <a href="/restricted/upload/bot_config">View/upload bot_config.py</a>
      </li>
      <li>
        <a href="/auth/groups">View/edit user groups</a>
      </li>
    </ul>
    ${ele.permissions.get_bootstrap_token ? bootstrapTemplate(ele): ''}
  </main>
  <footer></footer>
</swarming-app>`;

window.customElements.define('swarming-index', class extends SwarmingAppBoilerplate {

  constructor() {
    super(template);
    this._bootstrap_token = '...';
    const idx = location.hostname.indexOf('.appspot.com');
    this._project_id = location.hostname.substring(0, idx) || 'not_found';
    this._host_url = location.origin;
  }

  connectedCallback() {
    super.connectedCallback();

    this.addEventListener('permissions-loaded', (e) => {
      if (this.permissions.get_bootstrap_token) {
        this._fetchToken();
      }
      this.render();
    });

    this.addEventListener('server-details-loaded', (e) => {
      this.render();
    });

    this.render();
  }

  _fetchToken() {
    const post_extra = {
      headers: {'authorization': this.auth_header},
      method: 'POST',
    };
    this.app.addBusyTasks(1);
    fetch('/_ah/api/swarming/v1/server/token', post_extra)
      .then(jsonOrThrow)
      .then((json) => {
        this._bootstrap_token = json.bootstrap_token;
        this.render();
        this.app.finishedTask();
      })
      .catch((e) => this.fetchError(e, 'token'));
  }

});
