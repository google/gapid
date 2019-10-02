// Copyright 2014 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

var common = (function() {
'use strict';

var exports = {};
var templatesCache = {};


// Simple string formatter: String.format('{0} {1}', 'a', 'b') -> 'a b'.
// Taken from http://stackoverflow.com/a/4673436
String.format = function(format) {
  var args = Array.prototype.slice.call(arguments, 1);
  var sprintfRegex = /\{(\d+)\}/g;
  var sprintf = function(match, number) {
    return number in args ? args[number] : match;
  };
  return format.replace(sprintfRegex, sprintf);
};


// Converts UTC timestamp (in microseconds) to a readable string in local TZ.
exports.utcTimestampToString = function(utc) {
  return (new Date(Number(utc / 1000.0))).toLocaleString();
};


// Returns URL to a main group page (where the group can be edited).
exports.getGroupPageURL = function(group) {
  return '/auth/groups/' + group;
};


// Returns URL to a change log page for a given target.
exports.getChangeLogURL = function(kind, name) {
  return '/auth/change_log?target=' + encodeURIComponent(kind + '$' + name);
};


// Returns URL to a change log page for a given revision.
exports.getChangeLogRevisionURL = function(rev) {
  return '/auth/change_log?auth_db_rev=' + encodeURIComponent('' + rev);
};


// Returns URL to a page with full listing of the group.
exports.getGroupListingURL = function(group) {
  return '/auth/listing?group=' + encodeURIComponent(group);
};


// Returns URL to a page with a principal lookup results.
exports.getLookupURL = function(principal) {
  if (principal) {
    return '/auth/lookup?p=' + encodeURIComponent(principal);
  }
  return '/auth/lookup';
};


// Appends '<prefix>:' to a string if it doesn't have a prefix.
exports.addPrefix = function(prefix, str) {
  if (str.indexOf(':') == -1) {
    return prefix + ':' + str;
  } else {
    return str;
  }
};


// Applies 'addPrefix' to each item of a list.
exports.addPrefixToItems = function(prefix, items) {
  return _.map(items, _.partial(exports.addPrefix, prefix));
};


// Strips '<prefix>:' from a string if it starts with it.
exports.stripPrefix = function(prefix, str) {
  if (!str) {
    return '';
  }
  if (str.slice(0, prefix.length + 1) == prefix + ':') {
    return str.slice(prefix.length + 1, str.length);
  } else {
    return str;
  }
};


// Applies 'stripPrefix' to each item of a list.
exports.stripPrefixFromItems = function(prefix, items) {
  return _.map(items, _.partial(exports.stripPrefix, prefix));
};


// Returns sorted list of groups.
//
// Groups without '-' or '/' come first, then groups with '-'. Groups that can
// be modified by a caller (based on 'caller_can_modify' field if available)
// always come before read-only groups.
exports.sortGroupsByName = function(groups) {
  return _.sortBy(groups, function(group) {
    // Note: caller_can_modify is optional, it is fine if its 'undefined'.
    var prefix = group.caller_can_modify ? 'A' : 'B';
    var name = group.name;
    if (name.indexOf('/') != -1) {
      return prefix + 'C' + name;
    }
    if (name.indexOf('-') != -1) {
      return prefix + 'B' + name;
    }
    return prefix + 'A' + name;
  });
};


// Fetches handlebars template code from #<templateId> element and renders it.
// Returns rendered string.
exports.render = function(templateId, context) {
  var template = templatesCache[templateId];
  if (!template) {
    var source = $('#' + templateId).html();
    template = Handlebars.compile(source);
    templatesCache[templateId] = template;
  }
  return template(context);
};


// Returns chunk of HTML code with form alert mark up.
// Args:
//   type: 'success' or 'error'.
//   title: title of the message, will be in bold.
//   message: body of the message.
exports.getAlertBoxHtml = function(type, title, message) {
  var cls = (type == 'success') ? 'alert-success' : 'alert-danger';
  return exports.render(
      'alert-box-template', {cls: cls, title: title, message: message});
};


// Disables or enabled input controls in an element.
exports.setInteractionDisabled = function($element, disabled) {
  $('button, input, textarea', $element).attr('disabled', disabled);
};


// Called during initial page load to show contents of a page once
// Javascript part complete building it.
exports.presentContent = function() {
  $('#content-box').show();
  $('#error-box').hide();
};


// Fatal error happened during loading. Show it instead of a content.
exports.presentError = function(errorText) {
  $('#error-box #error-message').text(errorText);
  $('#error-box').show();
  $('#content-box').hide();
};


// Double checks with user and redirects to logout url.
exports.logout = function() {
  if (!config.using_gae_auth ||
      confirm('You\'ll be signed out from ALL your google accounts.')) {
    window.location = config.logout_url;
  }
};


// Shows confirmation modal dialog. Returns deferred.
exports.confirm = function(message) {
  var defer = $.Deferred();
  if (window.confirm(message))
    defer.resolve();
  else
    defer.reject();
  return defer.promise();
};


// Used in setAnchor and onAnchorChange to filter out unneeded event.
var knownAnchor;


// Returns #anchor part of the current location.
exports.getAnchor = function() {
  return window.location.hash.substring(1);
};


// Changes #anchor part of the current location. Calling this method does not
// trigger 'onAnchorChange' event.
exports.setAnchor = function(a) {
  knownAnchor = a;
  window.location.hash = '#' + a;
};


// Sets a callback to watch for changes to #anchor part of the location.
exports.onAnchorChange = function(cb) {
  window.onhashchange = function() {
    var a = exports.getAnchor();
    if (a != knownAnchor) {
      knownAnchor = a;
      cb();
    }
  };
};


// Returns value of URL query parameter given its name.
exports.getQueryParameter = function(name) {
  // See http://stackoverflow.com/a/5158301.
  var match = new RegExp(
      '[?&]' + name + '=([^&]*)').exec(window.location.search);
  return match && decodeURIComponent(match[1].replace(/\+/g, ' '));
};


// Wrapper around 'Im busy' UI indicator.
var ProgressSpinner = function() {
  this.$element = $('#progress-spinner');
  this.counter = 0;
};


// Shows progress indicator.
ProgressSpinner.prototype.show = function() {
  this.counter += 1;
  if (this.counter == 1) {
    this.$element.removeClass('not-spinning').addClass('spinning');
  }
};


// Hides progress indicator.
ProgressSpinner.prototype.hide = function() {
  if (this.counter !== 0) {
    this.counter -= 1;
    if (this.counter === 0) {
      this.$element.removeClass('spinning').addClass('not-spinning');
    }
  }
};


// Configure state common for all pages.
exports.onContentLoaded = function() {
  // Configure form validation plugin to work with bootstrap styles.
  // See http://stackoverflow.com/a/18754780
  $.validator.setDefaults({
    highlight: function(element) {
      $(element).closest('.form-group').addClass('has-error');
    },
    unhighlight: function(element) {
      $(element).closest('.form-group').removeClass('has-error');
    },
    errorElement: 'span',
    errorClass: 'help-block',
    errorPlacement: function(error, element) {
      if(element.parent('.input-group').length) {
        error.insertAfter(element.parent());
      } else {
        error.insertAfter(element);
      }
    }
  });

  // Install 'Ajax is in progress' indicator.
  var progressSpinner = new ProgressSpinner();
  $(document).ajaxStart(function() {
    progressSpinner.show();
  });
  $(document).ajaxStop(function() {
    progressSpinner.hide();
  });
};


return exports;
}());
