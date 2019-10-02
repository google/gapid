// Copyright 2015 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

var change_log = (function() {
'use strict';

var exports = {};


// Keys in change dict present in all kinds of changes.
var KNOWN_CHANGE_KEYS = [
  'change_type',
  'target',
  'auth_db_rev',
  'who',
  'when',
  'comment',
  'app_version'
];


// Parses change target string ('AuthGroup$name') into components, adds a human
// readable title and URL to a change log for the target.
var parseTarget = function(t) {
  var kind = t.split('$', 1)[0];
  var name = t.substring(kind.length + 1);

  // Recognize some known targets.
  var title = name;
  var targetURL = null;
  switch (kind) {
    case 'AuthGroup':
      targetURL = '/auth/groups/' + name;
      break;
    case 'AuthIPWhitelist':
      targetURL = '/auth/ip_whitelists';
      break;
    case 'AuthIPWhitelistAssignments':
      title = 'IP whitelist assignment';
      targetURL = '/auth/ip_whitelists';
      break;
    case 'AuthGlobalConfig':
      title = 'Global config';
      targetURL = '/auth/oauth_config';
      break;
  }

  return {
    kind: kind,
    name: name,
    title: title,
    changeLogURL: common.getChangeLogURL(kind, name),
    targetURL: targetURL
  };
};


// Given a change dict, returns text blob to show in "Change details" box.
var changeToTextBlob = function(c) {
  var text = '';

  // First visit known keys present in all changes. That way they are always
  // in top in the text representation (and in predefined order).
  _.each(KNOWN_CHANGE_KEYS, function(key) {
    var val = c[key];
    if (val) {
      text += key + ': ' + val + '\n';
    }
  });

  // Then visit the rest (in stable order).
  var keys = _.keys(c);
  keys.sort();
  _.each(keys, function(key) {
    if (KNOWN_CHANGE_KEYS.indexOf(key) != -1) {
      return;
    }
    var val = c[key];
    if (val instanceof Array) {
      if (val.length) {
        text += key + ':\n';
        _.each(val, function(item) { text += '  ' + item + '\n'; });
      }
    } else if (val) {
      text += key + ': ' + val + '\n';
    }
  });

  return text;
};


// Removes 'user:' prefix from strings if it is present.
var stripUserPrefix = function(x) {
  return common.stripPrefix('user', x);
};


// Converts UTC timestamps to readable strings, strips identity prefixes, etc.
var beautifyChange = function(obj) {
  var target = parseTarget(obj.target);
  return _.extend(_.clone(obj), {
    when: common.utcTimestampToString(obj.when),
    who: stripUserPrefix(obj.who),
    targetTitle: target.title,
    changeLogURL: target.changeLogURL,
    revisionURL: common.getChangeLogRevisionURL(obj.auth_db_rev)
  });
};


// Offload HTML escaping to Handlebars.
var listTemplate = Handlebars.compile(
    '{{#each items}}<li>{{this}}</li>{{/each}}');


// Returns HTML with a list of items or a single item if len(items) == 1.
var listToHTML = function(items) {
  if (items.length == 1) {
    return _.escape(items[0]);
  }
  return listTemplate({items: items});
};


// Change type => function to format tooltip body.
var TOOLTIP_FORMATTERS = {
  'GROUP_MEMBERS_ADDED': function(c) {
    return listToHTML(_.map(c.members, stripUserPrefix));
  },
  'GROUP_MEMBERS_REMOVED': function(c) {
    return listToHTML(_.map(c.members, stripUserPrefix));
  },
  'GROUP_GLOBS_ADDED': function(c) {
    return listToHTML(_.map(c.globs, stripUserPrefix));
  },
  'GROUP_GLOBS_REMOVED': function(c) {
    return listToHTML(_.map(c.globs, stripUserPrefix));
  },
  'GROUP_NESTED_ADDED': function(c) {
    return listToHTML(c.nested);
  },
  'GROUP_NESTED_REMOVED': function(c) {
    return listToHTML(c.nested);
  },
  'GROUP_DESCRIPTION_CHANGED': function(c) {
    return _.escape(c.description);
  },
  'GROUP_OWNERS_CHANGED': function(c) {
    return _.escape(c.old_owners) + ' &rarr; ' + _.escape(c.owners);
  },
  'GROUP_CREATED': function(c) {
    return _.escape(c.description);
  }
};


// Represents a table with AuthDB change log (with pagination).
var ChangeLogTable = function($element, $modal, target, revision) {
  // Root element for change log table.
  this.$element = $element;
  // Modal dialog with details of a single change log entry.
  this.$modal = $modal;
  // If set, limits change log queries to given target.
  this.target = target;
  // If set, limits change log queries to specific revision only.
  this.revision = revision;
  // Page size. Choose so that single page fits on screen (well, some screen).
  this.limit = 15;
  // Cursor for the current page.
  this.cursor = null;
  // Cursor for the next page (if any).
  this.nextCursor = null;
  // Stack of cursors for previous pages (if any).
  this.prevCursor = [];
  // True if UI is locked (when doing AJAX).
  this.locked = false;
  // Latest change log passed to 'setData'.
  this.changeLog = null;

  // Add the table and pager to DOM.
  this.$tableDiv = $('<div>');
  this.$pager = $(common.render('change-log-pager-template'));
  $element.append(this.$tableDiv);
  $element.append(this.$pager);
  this.updatePagerButtons();

  // Attach events to pager buttons.
  var that = this;
  $('#next', this.$pager).click(function() {
    that.nextClicked();
    return false;
  });
  $('#prev', this.$pager).click(function() {
    that.prevClicked();
    return false;
  });
};


// Refetches current page, returns corresponding defer.
ChangeLogTable.prototype.refresh = function() {
  var that = this;
  return this.fetchCursor(this.cursor, function(data) {
    that.nextCursor = data.cursor;
    that.setData(data.changes);
  });
};


// Lock UI, fetches given cursor, calls callback, unlock UI. Returns overall
// defer for this whole operation.
ChangeLogTable.prototype.fetchCursor = function(cursor, callback) {
  var defer = api.queryChangeLog(
      this.target, this.revision, this.limit, cursor);
  var that = this;
  this.lockUI();
  defer.then(function(response) {
    callback(response.data);
    that.unlockUI();
    that.updatePagerButtons();
  }, function(error) {
    that.displayError(error);
    that.unlockUI();
    that.updatePagerButtons();
  });
  return defer;
};


// Updates DOM based on the response from server.
ChangeLogTable.prototype.setData = function(changes) {
  var tmpl = {changes: _.map(changes, beautifyChange)};
  var table = $(common.render('change-log-table-template', tmpl));
  this.$tableDiv.empty().append(table);
  this.changeLog = changes;

  // Present full change info on click.
  var that = this;
  $('.view-change', table).click(function() {
    if (that.locked) {
      return false;
    }
    var idx = $(this).data('change-idx');
    that.presentChange(that.changeLog[idx]);
    return false;
  });

  // Put some change details in the tooltip.
  $('.view-change', table).parent().tooltip({
    placement: 'right',
    html: true,
    title: function() {
      if (that.locked) {
        return null;
      }
      var idx = $('.view-change', this).data('change-idx');
      var change = that.changeLog[idx];
      var formatter = TOOLTIP_FORMATTERS[change.change_type];
      return formatter ? formatter(change) : null;
    }
  });
};


// Called when 'Older ->' button is clicked.
ChangeLogTable.prototype.nextClicked = function() {
  if (!this.nextCursor || this.locked) {
    return;
  }
  var next = this.nextCursor;
  var that = this;
  return this.fetchCursor(next, function(data) {
    that.prevCursor.push(that.cursor);
    that.cursor = next;
    that.nextCursor = data.cursor;
    that.setData(data.changes);
  });
};


// Called when '<- Newer' button is clicked.
ChangeLogTable.prototype.prevClicked = function() {
  if (this.prevCursor.length === 0 || this.locked) {
    return;
  }
  var prev = this.prevCursor[this.prevCursor.length - 1];
  var that = this;
  return this.fetchCursor(prev, function(data) {
    that.prevCursor.pop();
    that.cursor = prev;
    that.nextCursor = data.cursor;
    that.setData(data.changes);
  });
};


// Shows a popup with details of a single change.
ChangeLogTable.prototype.presentChange = function(change) {
  if (this.locked) {
    return;
  }
  $('#details-text', this.$modal).text(changeToTextBlob(change));
  this.$modal.modal('show');
};


// Shows error message.
ChangeLogTable.prototype.displayError = function(error) {
  // TODO(vadimsh): It is a hack - modifies global page state instead of a state
  // of a change log element.
  common.presentError(error.text);
};


// Locks UI actions before AJAX.
ChangeLogTable.prototype.lockUI = function() {
  this.locked = true;
  this.$modal.modal('hide');
};


// Unlocks UI actions after AJAX.
ChangeLogTable.prototype.unlockUI = function() {
  this.locked = false;
};


// Disables or enables pager buttons based on 'prevCursor' and 'nextCursor'.
// Do not visually react to 'locked' state, since in practice it just adds
// annoying flashing when paging.
ChangeLogTable.prototype.updatePagerButtons = function() {
  var hasPrev = this.prevCursor.length > 0;
  $('#next', this.$pager).parent().toggleClass('disabled', !this.nextCursor);
  $('#prev', this.$pager).parent().toggleClass('disabled', !hasPrev);
  this.$pager.toggleClass('hide', !this.nextCursor && !hasPrev);
};


// Called when HTML body of a page is loaded.
exports.onContentLoaded = function() {
  var target = common.getQueryParameter('target');
  var authDbRev = common.getQueryParameter('auth_db_rev');
  if (authDbRev) {
    authDbRev = parseInt(authDbRev);
  }

  // Generate header.
  var parsed;
  if (target) {
    parsed = parseTarget(target);
  } else {
    parsed = {
      title: 'Global Log',
      kind: null,
      targetURL: null
    };
  }
  $('#change-log-header').html(common.render('change-log-header-template', {
    authDbRev: authDbRev,
    title: parsed.title,
    kind: parsed.kind,
    targetURL: parsed.targetURL
  }));

  // Build the table, fetch initial page.
  var tableDiv = $('#change-log-table');
  var modalDiv = $('#show-change-details');
  var changeTable = new ChangeLogTable(tableDiv, modalDiv, target, authDbRev);
  changeTable.refresh().then(function(response) {
    common.presentContent();
  }, function(error) {
    common.presentError(error.text);
  });
};


return exports;
}());
