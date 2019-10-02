// Copyright 2017 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

var lookup = (function() {
'use strict';

var exports = {};


// A defer with the currently executing lookup operation.
//
// Used internally inside 'lookup' to ensure UI follows only the last initiated
// lookup operation.
var currentLookupOp = null;


// Extracts a string to lookup from the current page URL.
var principalFromURL = function() {
  return common.getQueryParameter('p');
};


// Returns root URL to this page with '?p=...' filled in.
var thisPageURL = function(principal) {
  return common.getLookupURL(principal);
};


// Sets the initial history state and hooks up to onpopstate callback.
var initializeHistory = function(cb) {
  var p = principalFromURL();
  window.history.replaceState({'principal': p}, null, thisPageURL(p));

  window.onpopstate = function(event) {
    var s = event.state;
    if (s && s.hasOwnProperty('principal')) {
      cb(s.principal);
    }
  };
};


// Updates the history state on lookup request, to put '&p=...' there.
var updateHistory = function(principal) {
  if (principal != principalFromURL()) {
    window.history.pushState(
        {'principal': principal}, null, thisPageURL(principal));
  }
};


// lookup initiates a lookup request.
//
// Fills in the lookup edit box and updates browser history to match the given
// value.
var lookup = function(principal) {
  $('#lookup-input').val(principal);
  updateHistory(principal);

  // Reset the state before starting the lookup.
  resetUIState();
  currentLookupOp = null;

  if (!principal) {
    return;
  }

  // Normalize the principal to the form the API expects. As everywhere in the
  // UI, we assume 'user:' prefix is implied in emails and globs. In addition,
  // emails all have '@' symbol and (unlike external groups such as google/a@b)
  // don't have '/', and globs all have '*' symbol. Everything else is assumed
  // to be a group.
  var isEmail = principal.indexOf('@') != -1 && principal.indexOf('/') == -1;
  var isGlob = principal.indexOf('*') != -1;
  if ((isEmail || isGlob) && principal.indexOf(':') == -1) {
    principal = 'user:' + principal;
  }

  // Show in the UI that we are searching.
  setBusyIndicator(true);

  // Ask the backend to do the lookup for us. Ignore the result if some other
  // lookup has been started while we were waiting.
  currentLookupOp = api.fetchRelevantSubgraph(principal);
  currentLookupOp.then(function(response) {
    if (this === currentLookupOp) {
      setBusyIndicator(false);
      currentLookupOp = null;
      setLookupResults(principal, response.data.subgraph);
    }
  }, function(error) {
    if (this === currentLookupOp) {
      setBusyIndicator(false);
      currentLookupOp = null;
      setErrorText(error.text);
    }
  });
};


// Resets the state of the lookup UI to the default one.
var resetUIState = function() {
  setBusyIndicator(false);
  setErrorText('');
  setLookupResults('', null);
};


// Displays or hides "we are searching now" indicator.
var setBusyIndicator = function(isBusy) {
  var $indicator = $('#lookup-busy-indicator');
  if (isBusy) {
    $indicator.show();
  } else {
    $indicator.hide();
  }
};


// Displays (if text is not empty) or hides (if it is empty) the error box.
var setErrorText = function(text) {
  var $box = $('#lookup-error-box');
  $('#lookup-error-message', $box).text(text);
  if (text) {
    $box.show();
  } else {
    $box.hide();
  }
};


// Called when the API replies with the results to render them.
var setLookupResults = function(principal, subgraph) {
  var $box = $('#lookup-results-box');
  var $content = $('#lookup-results');
  if (!subgraph) {
    $box.hide();
    $('[data-toggle="popover"]', $content).popover('destroy');
    $content.empty();
  } else {
    var vars = interpretLookupResults(subgraph);
    $content.html(common.render('lookup-results-template', vars));

    // Enable fancy HTML-formated popovers that display list of inclusion paths
    // grabbing them from vars.includers[...].includesIndirectly.
    $('[data-toggle="popover"]', $content).popover({
      container: 'body',
      html: true,
      title: 'Included via',
      content: function () {
        var includer = vars.includers[$(this).attr('data-group-name')];
        return common.render('indirect-group-popover', includer);
      }
    });

    // Intercept clicks on group links and reuse the current page instead of
    // opening a new one. Web 2.0! With hax.
    //
    // TODO(vadimsh): Get rid of flickering. We load data too fast, progress
    // indicator appears and almost immediately disappears.
    $('.group-link', $content).click(function() {
      lookup($(this).attr('data-group-name'));
      return false;
    });

    $box.show();
  }
};


// Takes subgraph returned by API and produces an environment for the template.
//
// See 'lookup-results-template' template in lookup.html to see how the vars
// are used.
var interpretLookupResults = function(subgraph) {
  // Note: the principal is always represented by nodes[0] per API guarantee.
  var nodes = subgraph.nodes;
  var principal = nodes[0];

  // Map {group name => Includer objects (see 'includer' below)}.
  var includers = {};

  // Returns the corresponding includer from 'includers', possibly creating it.
  var includer = function(group) {
    if (!includers.hasOwnProperty(group)) {
      includers[group] = {
        'name': group,
        'href': common.getLookupURL(group),
        'includesDirectly': false,
        'includesViaGlobs': [],  // list of globs (with stripped 'user:')
        'includesIndirectly': [] // list of lists of groups with inclusion paths
      };
    }
    return includers[group];
  };


  // Enumerates ALL paths from root to leafs along 'IN' edges, calling 'visitor'
  // for each path, including incomplete paths too. Paths are represented as
  // arrays of node objects (values of 'nodes' array).
  var enumeratePaths = function(current, visitor) {
    visitor(current);
    var last = current[current.length-1];
    if (last.edges && last.edges['IN']) {
      _.each(last.edges['IN'], function(idx) {
        var node = nodes[idx];
        console.assert(current.indexOf(node) == -1); // no cycles!
        current.push(node);
        enumeratePaths(current, visitor);
        current.pop();
      });
    }
  };

  // For each path from root to 'last' (that is not necessary a leaf) analyze
  // the path and update corresponding 'includer' object.
  enumeratePaths([principal], function(path) {
    console.assert(path.length > 0);
    console.assert(path[0] === principal);
    if (path.length == 1) {
      return;  // the trivial [principal] path
    }

    var last = path[path.length-1];
    if (last.kind != 'GROUP') {
      return;  // we are interested only in examining groups, skip GLOBs
    }

    var inc = includer(last.value);
    if (path.length == 2) {
      // The entire path is 'principal -> last', meaning 'last' includes the
      // principal directly.
      inc.includesDirectly = true;
    } else if (path.length == 3 && path[1].kind == 'GLOB') {
      // The entire path is 'principal -> GLOB -> last', meaning 'last' includes
      // the principal via the glob.
      inc.includesViaGlobs.push(common.stripPrefix('user', path[1].value));
    } else {
      // Some arbitrarily long indirect inclusion path. Just record all group
      // names in it (skipping globs). Skip the root principal itself (path[0])
      // and the currently analyzed node (path[-1]), not useful information, it
      // is same for all paths.
      var groupNames = [];
      for (var i = 1; i < path.length-1; i++) {
        if (path[i].kind == 'GROUP') {
          groupNames.push(path[i].value);
        }
      }
      inc.includesIndirectly.push(groupNames);
    }
  });


  // Finally massage the findings for easier display. Note that directIncluders
  // and indirectIncluders are NOT disjoint sets.
  var directIncluders = [];
  var indirectIncluders = [];
  _.each(includers, function(inc) {
    if (inc.includesDirectly || inc.includesViaGlobs.length != 0) {
      directIncluders.push(inc);
    }
    if (inc.includesIndirectly.length != 0) {
      // Long inclusion paths look like data dumps in UI and don't really fit.
      // The most interesting components are at the ends, so keep only them.
      inc.includesIndirectly = shortenInclusionPaths(inc.includesIndirectly);
      indirectIncluders.push(inc);
    }
  });
  directIncluders = common.sortGroupsByName(directIncluders);
  indirectIncluders = common.sortGroupsByName(indirectIncluders);

  // If looking up a group, show a link to the main group page.
  var groupHref = '';
  if (principal.kind == 'GROUP') {
    groupHref = common.getGroupPageURL(principal.value);
  }

  // TODO(vadimsh): Display ownership information too.
  // TODO(vadimsh): Display included groups if the principal is a group.

  return {
    'principalName': common.stripPrefix('user', principal.value),
    'principalIsGroup': principal.kind == 'GROUP',
    'groupHref': groupHref,
    'includers': includers,  // used to construct popovers with additional info
    'directIncluders': directIncluders,
    'indirectIncluders': indirectIncluders
  };
};


// For each long path in the list kicks out the middle (replacing it with '').
var shortenInclusionPaths = function(paths) {
  var out = [];
  var seen = {};  // path.join('\n') => true

  _.each(paths, function(path) {
    if (path.length <= 3) {
      out.push(path); // short enough already
      return;
    }
    var shorter = [path[0], '', path[path.length-1]];
    var key = shorter.join('\n');
    if (!seen[key]) {
      seen[key] = true;
      out.push(shorter);
    }
  });

  return out;
};


// Called when HTML body of a page is loaded.
exports.onContentLoaded = function() {
  // Setup a reaction to clicking Enter while in the edit box.
  $('#lookup-form').submit(function(event) {
    lookup($('#lookup-input').val());
    return false;
  });
  // Setup a reaction to the back button.
  initializeHistory(lookup);
  // Do the lookup of a principal provided through URL (if any).
  lookup(principalFromURL());
  // Show the UI before the lookup has finished, we have the busy indicator.
  common.presentContent();
};


return exports;
}());
