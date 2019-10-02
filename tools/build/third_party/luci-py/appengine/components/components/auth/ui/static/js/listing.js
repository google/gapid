// Copyright 2017 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

var listing = (function() {
'use strict';

var exports = {};


// Sorts list of objects (in place) by 'principal' field.
var sortByPrincipal = function(list) {
  list.sort(function(a, b) {
    if (a.principal < b.principal) return -1;
    if (a.principal > b.principal) return 1;
    return 0;
  });
};


// Removes 'user:' prefix from 'principal' field, adds lookupUrl field.
var preparePrincipalList = function(list) {
  _.each(list, function(obj) {
    obj.principal = common.stripPrefix('user', obj.principal);
    obj.lookupUrl = common.getLookupURL(obj.principal);
  });
};


// Returns "<num> <noun>(s)", adding 's' only if necessary.
var pluralized = function(num, noun) {
  if (num == 1) {
    return "1 " + noun;
  }
  return String(num) + " " + noun + "s";
};


// Renders listing UI in the given $element.
var renderListing = function(group, listing, $element) {
  // API returns listings unordered.
  sortByPrincipal(listing.members);
  sortByPrincipal(listing.globs);
  sortByPrincipal(listing.nested);

  // Beautify strings by stripping default 'user:' prefix, calculate URLs to
  // Lookup pages.
  preparePrincipalList(listing.members);
  preparePrincipalList(listing.globs);

  // Compute nested groups href's.
  _.each(listing.nested, function(obj) {
    obj.listingURL = common.getGroupListingURL(obj.principal);
  });

  // Render.
  $element.html(common.render('listing-template', {
    'group': group,
    'members': listing.members,
    'globs': listing.globs,
    'nested': listing.nested,
    'emptyListing': listing.members.length + listing.globs.length == 0,
    'emptyNested': listing.nested.length == 0,
    'membersCountStr': pluralized(listing.members.length, 'member'),
    'globsCountStr': pluralized(listing.globs.length, 'glob'),
    'nestedCountStr': pluralized(listing.nested.length, 'nested group')
  }));
};


// Called when HTML body of a page is loaded.
exports.onContentLoaded = function() {
  var group = common.getQueryParameter('group');
  if (!group) {
    common.presentError('Missing "group" query parameter');
    return;
  }
  api.fetchGroupListing(group).then(function(response) {
    renderListing(group, response.data['listing'], $('#listing-container'));
    common.presentContent();
  }, function(error) {
    common.presentError(error.text);
  });
};


return exports;
}());
