// Copyright 2014 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

var api = (function() {
'use strict';

var exports = {};

// Current known value of XSRF token.
var xsrf_token = null;
// Id of a timer that refetches the token.
var xsrf_token_timer = null;
// How often to refetch the token.
var xsrf_update_interval = 30 * 60 * 1000;


// Return a dict with subset of response headers used by API.
// Keys are in camel case.
var extractHeaders = function(jqXHR) {
  var headers = {};
  _.each(['Last-Modified'], function(key) {
    headers[key] = jqXHR.getResponseHeader(key);
  });
  return headers;
};


// Makes an asynchronous call to API endpoint. Returns a promise.
// On success result value is
//    {status: 'success', code: <good HTTP code>, data: <whatever API returned>}
// On failure result value is
//    {status: 'error', code: <bad HTTP code>, error: <whatever API returned>}
var call = function(type, url, data, headers) {
  // Append XSRF token header to the request if known.
  headers = _.clone(headers || {});
  if (xsrf_token) {
    headers['X-XSRF-Token'] = xsrf_token;
  }

  // Launch the request.
  var request = $.ajax({
    type: type,
    url: url,
    data: (data !== null) ? JSON.stringify(data) : null,
    cache: false,
    contentType: 'application/json; charset=UTF-8',
    dataType: 'json',
    headers: headers
  });

  // Future return value.
  var defer = $.Deferred();

  // Convert from jQuery odd signature to saner one.
  request.done(function(data, textStatus, jqXHR) {
    defer.resolve({
      status: textStatus,
      code: jqXHR.status,
      data: data,
      headers: extractHeaders(jqXHR)
    });
  });

  // Deal with few more oddities.
  request.fail(function(jqXHR, textStatus, errorThrown) {
    // Structured error object returned by API.
    var errorObj;
    // Error message string that will be logged.
    var errorMsg = jqXHR.responseText;

    // Oddly jQuery doesn't parse JSON body of a failed response (like 403), so
    // do it ourself using plain response text if Content-Type is JSON.
    var contentType = jqXHR.getResponseHeader('content-type') || '';
    if (contentType.indexOf('application/json') != -1) {
      try {
        errorObj = $.parseJSON(jqXHR.responseText);
        if (_.has(errorObj, 'text')) {
          errorMsg = errorObj.text;
        }
      } catch (err) {
        // Server tricked us by giving incorrect content type. Use response
        // string as error object.
        console.error(err);
        errorObj = jqXHR.responseText;
      }
    }

    // Assemble all know information into human readable multiline string.
    var verboseText = String.format(
        'Request to \'{0}\' failed ({1}) with code {2}.\n{3}',
        url, textStatus, jqXHR.status, errorMsg);
    console.error(verboseText);

    // And return that to waiting caller.
    defer.reject({
      status: textStatus,
      code: jqXHR.status,
      error: errorObj,
      headers: extractHeaders(jqXHR),
      text: errorMsg,
      verbose: verboseText
    });
  });

  return defer.promise();
};

// Also make 'call' available to clients of this module.
exports.call = call;


// Group object -> group object with all necessary fields present.
var normalizeGroupObj = function(group) {
  return {
    name: group.name,
    description: group.description,
    owners: group.owners,
    members: group.members || [],
    globs: group.globs || [],
    nested: group.nested || []
  };
};


// IP whitelist object -> IP whitelist object with all necessary fields present.
var normalizeIpWhitelistObj = function(ipWhitelist) {
  return {
    name: ipWhitelist.name,
    description: ipWhitelist.description,
    subnets: ipWhitelist.subnets || []
  };
};


// Sets XSRF token.
exports.setXSRFToken = function(token) {
  xsrf_token = token;
};


// Refetches XSRF token from server.
exports.updateXSRFToken = function() {
  var endpoint = '/auth/api/v1/accounts/self/xsrf_token';
  var headers = {'X-XSRF-Token-Request': '1'};
  var request = call('POST', endpoint, null, headers);
  request.then(function(response) {
    exports.setXSRFToken(response.data.xsrf_token);
  });
  return request;
};


// Enables XSRF token refresh timer.
exports.setXSRFTokenAutoupdate = function(autoUpdate) {
  if (!autoUpdate && xsrf_token_timer) {
    clearTimeout(xsrf_token_timer);
    xsrf_token_timer = null;
  }
  if (autoUpdate && !xsrf_token_timer) {
    xsrf_token_timer = setTimeout(function() {
      var timer_id = xsrf_token_timer;
      exports.updateXSRFToken().always(function() {
        if (timer_id == xsrf_token_timer) {
          xsrf_token_timer = null;
          exports.setXSRFTokenAutoupdate(true);
        }
      });
    }, xsrf_update_interval);
  }
};


// Fetches OAuth configuration.
exports.fetchOAuthConfig = function(allowCached) {
  var headers = allowCached ? {} : {'Cache-Control': 'no-cache'};
  return call('GET', '/auth/api/v1/server/oauth_config', null, headers);
};


// Stores OAuth configuration.
exports.updateOAuthConfig = function(client_id, client_secret,
                                     additional_ids, token_server_url) {
  return call('POST', '/auth/api/v1/server/oauth_config', {
    additional_client_ids: additional_ids,
    client_id: client_id,
    client_not_so_secret: client_secret,
    token_server_url: token_server_url
  });
};


// Lists all known user groups.
exports.groups = function() {
  return call('GET', '/auth/api/v1/groups');
};


// Fetches detailed information about a group.
exports.groupRead = function(name) {
  return call(
      'GET', '/auth/api/v1/groups/' + name,
      null, {'Cache-Control': 'no-cache'});
};


// Deletes a group.
exports.groupDelete = function(name, lastModified) {
  var headers = {};
  if (lastModified) {
    headers['If-Unmodified-Since'] = lastModified;
  }
  return call('DELETE', '/auth/api/v1/groups/' + name, null, headers);
};


// Creates a new group.
exports.groupCreate = function(group) {
  return call(
      'POST',
      '/auth/api/v1/groups/' + group.name,
      normalizeGroupObj(group));
};


// Updates an existing group.
exports.groupUpdate = function(group, lastModified) {
  var headers = {};
  if (lastModified) {
    headers['If-Unmodified-Since'] = lastModified;
  }
  return call(
      'PUT',
      '/auth/api/v1/groups/' + group.name,
      normalizeGroupObj(group),
      headers);
};


// Lists all known IP whitelists.
exports.ipWhitelists = function() {
  return call('GET', '/auth/api/v1/ip_whitelists');
};


// Fetches single IP whitelist given its name.
exports.ipWhitelistRead = function(name) {
  return call('GET', '/auth/api/v1/ip_whitelists/' + name);
};


// Deletes an IP whitelist.
exports.ipWhitelistDelete = function(name, lastModified) {
  var headers = {};
  if (lastModified) {
    headers['If-Unmodified-Since'] = lastModified;
  }
  return call('DELETE', '/auth/api/v1/ip_whitelists/' + name, null, headers);
};


// Creates a new IP whitelist.
exports.ipWhitelistCreate = function(ipWhitelist) {
  return call(
      'POST',
      '/auth/api/v1/ip_whitelists/' + ipWhitelist.name,
      normalizeIpWhitelistObj(ipWhitelist));
};


// Updates an existing IP whitelist.
exports.ipWhitelistUpdate = function(ipWhitelist, lastModified) {
  var headers = {};
  if (lastModified) {
    headers['If-Unmodified-Since'] = lastModified;
  }
  return call(
      'PUT',
      '/auth/api/v1/ip_whitelists/' + ipWhitelist.name,
      normalizeIpWhitelistObj(ipWhitelist),
      headers);
};


// Grabs changes matching a query.
exports.queryChangeLog = function(target, revision, limit, cursor) {
  var q = {};
  if (target) {
    q['target'] = target;
  }
  if (revision) {
    q['auth_db_rev'] = revision;
  }
  if (limit) {
    q['limit'] = limit;
  }
  if (cursor) {
    q['cursor'] = cursor;
  }
  return call('GET', '/auth/api/v1/change_log?' + $.param(q));
};


// Fetches recursive listing of a group.
exports.fetchGroupListing = function(group) {
  return call(
      'GET', '/auth/api/v1/listing/groups/' + group, null,
      {'Cache-Control': 'no-cache'});
};


// Fetches the subgraph with information related to the given principal.
//
// Refer to the backend source code to figure out what it is exactly.
exports.fetchRelevantSubgraph = function(principal) {
  return call(
      'GET', '/auth/api/v1/subgraph/' + principal, null,
      {'Cache-Control': 'no-cache'});
};


return exports;
}());
