// Copyright 2014 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

var groups = (function() {
'use strict';

var exports = {};


////////////////////////////////////////////////////////////////////////////////
// Utility functions.


// Root URL of this page.
var GROUPS_ROOT_URL = '/auth/groups/';


// True if group name starts with '<something>/' prefix.
function isExternalGroupName(name) {
  return name.indexOf('/') != -1;
}


// True if string looks like a glob pattern (and not as group member name).
function isGlob(item) {
  // Glob patterns contain '*' and '[]' not allowed in member names.
  return item.search(/[\*\[\]]/) != -1;
}


// Trims group description to fit single line.
function trimGroupDescription(desc) {
  var firstLine = desc.split('\n')[0];
  if (firstLine.length > 55) {
    firstLine = firstLine.slice(0, 55) + '...';
  }
  return firstLine;
}


// Given an array of strings returns a one with longest substring of 'text'.
function longestMatch(items, text) {
  if (text.length === 0 || items.length === 0) {
    return null;
  }

  // Make jshint happy by moving local function declaration outside of the loop.
  var makePrefixFilter = function(prefix) {
    return function(item) {
      return item.indexOf(prefix) != -1;
    };
  };

  // Invariant: curSet is non empty subsequence of 'items', each item in curSet
  // has 'curPrefix' as a substring.
  var curPrefix = '';
  var curSet = items;
  for (var i = 0; i < text.length; i++) {
    // Attempt to increase curPrefix.
    var newPrefix = curPrefix + text[i];
    var newSet = _.filter(curSet, makePrefixFilter(newPrefix));
    // No matches at all -> curSet contains longest matches.
    if (newSet.length === 0) {
      // Could not find the first letter -> no match at all.
      if (i === 0) {
        return null;
      }
      return curSet[0];
    }
    // Carry on.
    curPrefix = newPrefix;
    curSet = newSet;
  }
  // curSet is a subset of 'items' that have 'text' as substring, pick first.
  return curSet[0];
}


////////////////////////////////////////////////////////////////////////////////
// Group chooser UI element: list of groups + 'Create new group' button.


var GroupChooser = function($element, allowCreateGroup) {
  // Root jquery DOM element.
  this.$element = $element;
  // True to show "Create a new group" item.
  this.allowCreateGroup = allowCreateGroup;
  // Currently known list of groups as shown in UI.
  this.groupList = [];
  // Just names. To avoid building this list for use in search all the time.
  this.groupNames = [];
  // Same list, but as a dict: group name -> group object.
  this.groupMap = {};
  // Mapping group name -> jQuery element, plus null -> "new group" element.
  this.groupToItemMap = {};
  // If true, selection won't change on clicks in UI.
  this.interactionDisabled = false;

  // Make group chooser use scroll bar.
  this.$element.slimScroll({height: '657px'});
};


// Loads list of groups from a server. Updates group chooser UI.
// Returns deferred.
GroupChooser.prototype.refetchGroups = function() {
  var defer = api.groups();
  var self = this;
  defer.then(function(response) {
    self.setGroupList(response.data.groups);
  });
  return defer;
};


// Updates DOM of a group chooser, resets current selection.
GroupChooser.prototype.setGroupList = function(groups) {
  var self = this;

  self.groupList = common.sortGroupsByName(groups);
  self.groupMap = {};
  self.groupToItemMap = {};
  self.groupNames = [];
  _.each(self.groupList, function(group) {
    group.href = GROUPS_ROOT_URL + group.name;
    group.isExternal = isExternalGroupName(group.name);
    group.descriptionTrimmed = trimGroupDescription(group.description);
    self.groupMap[group.name] = group;
    self.groupNames.push(group.name);
  });

  // Helper function to add children to DOM.
  var addElement = function(markup, groupName) {
    var item = $(markup);
    item.addClass('chooser-element');
    item.data('group-name', groupName);
    item.appendTo(self.$element);
    return item;
  };

  // Rebuild DOM: list of groups + 'Create new group' button.
  self.$element.addClass('list-group');
  self.$element.empty();
  _.each(self.groupList, function(group) {
    self.groupToItemMap[group.name] = addElement(
        common.render('group-chooser-item-template', group), group.name);
  });
  if (this.allowCreateGroup) {
    var templateArgs = {'href': GROUPS_ROOT_URL + NEW_GROUP_PLACEHOLDER};
    self.groupToItemMap[null] = addElement(
        common.render('group-chooser-button-template', templateArgs), null);
  }

  // Setup click event handlers. Clicks change selection.
  $('.chooser-element', self.$element).click(function() {
    if (!self.interactionDisabled) {
      self.setSelection($(this).data('group-name'), null);
    }
    return false;
  });
};


// Returns true if given group is in the list.
GroupChooser.prototype.isKnownGroup = function(name) {
  return this.groupMap.hasOwnProperty(name);
};


// Returns name of the selected group or null if 'Create new group' is selected.
// Returns 'undefined' if nothing is selected.
GroupChooser.prototype.getSelection = function() {
  var active = $('.chooser-element.active', self.$element);
  // 'group-name' attribute of 'Create new group' button is 'null'.
  return active.length ? active.data('group-name') : undefined;
};


// Highlights a group as chosen in group list.
// If |name| is null, then highlights 'Create new group' button.
// Also triggers 'selectionChanged' event, passing |state| to the handlers.
GroupChooser.prototype.setSelection = function(name, state) {
  // Nothing to do?
  if (this.getSelection() === name) {
    return;
  }
  var selectionMade = false;
  $('.chooser-element', self.$element).each(function() {
    if ($(this).data('group-name') === name) {
      $(this).addClass('active');
      selectionMade = true;
    } else {
      $(this).removeClass('active');
    }
  });
  if (selectionMade) {
    this.ensureGroupVisible(name);
    this.$element.triggerHandler(
        'selectionChanged', {group: name, state: state});
  }
};


// Selects top element.
GroupChooser.prototype.selectDefault = function() {
  var elements = $('.chooser-element', self.$element);
  if (elements.length) {
    this.setSelection(elements.first().data('group-name'), null);
  }
};


// Registers new event listener that is called whenever selection changes.
GroupChooser.prototype.onSelectionChanged = function(listener) {
  this.$element.on('selectionChanged', function(event, selection) {
    listener(selection.group, selection.state);
  });
};


// Disables an ability to change selection.
GroupChooser.prototype.setInteractionDisabled = function(disabled) {
  this.interactionDisabled = disabled;
};


// Scrolls group list so that given group (or "new group button") is visible.
GroupChooser.prototype.ensureGroupVisible = function(name) {
  var $item = this.groupToItemMap[name];
  if (!$item) {
    return;
  }

  // |pos| is position of $item relative to scrollable div origin.
  var scrollTop = this.$element.scrollTop();
  var pos = $item.position().top + scrollTop;

  // Scroll to the item if it is completely or partially invisible.
  var itemHeight = $item.outerHeight();
  var areaHeight = this.$element.height();
  if (pos < scrollTop || pos + itemHeight > scrollTop + areaHeight) {
    this.$element.slimScroll({scrollTo: pos});
  }
};


// Called when GroupChooser becomes visible in DOM (with all parent elements).
GroupChooser.prototype.madeVisible = function() {
  // ensureGroupVisible works only when groupChooser is visible in DOM, so
  // call it once more after displaying everything.
  this.ensureGroupVisible(this.getSelection());
};


////////////////////////////////////////////////////////////////////////////////
// Text field to search for groups.


var SearchBox = function($element) {
  this.$element = $element;
};


// Registers new event listener that is called whenever text changes.
SearchBox.prototype.onTextChanged = function(listener) {
  var self = this;
  this.$element.on('input', function() {
    listener(self.$element.val());
  });
};


// Registers new event listener that is called when Enter is hit.
SearchBox.prototype.onEnterKey = function(listener) {
  var self = this;
  this.$element.on('keyup', function(e) {
    if (e.keyCode == 13) {
      listener(self.$element.val());
    }
  });
};


////////////////////////////////////////////////////////////////////////////////
// Main content frame: a parent for forms to create a group or edit an existing.


var ContentFrame = function($element) {
  this.$element = $element;
  this.content = null;
  this.loading = null;
};


// Registers new event listener that is called when content is loaded and show.
ContentFrame.prototype.onContentShown = function(listener) {
  this.$element.on('contentShown', function() {
    listener();
  });
};


// Replaces frame's content with another one.
// |content| is an instance of GroupForm class.
ContentFrame.prototype.setContent = function(content) {
  if (this.content) {
    this.content.hide();
    this.content = null;
  }
  this.$element.empty();
  this.content = content;
  this.loading = null;
  if (this.content) {
    this.content.show(this.$element);
    this.$element.triggerHandler('contentShown');
  }
};


// Loads new content asynchronously using content.load(...) call.
// |content| is an instance of GroupForm class.
ContentFrame.prototype.loadContent = function(content) {
  var self = this;
  if (self.content) {
    self.content.setInteractionDisabled(true);
  }
  self.loading = content;
  var defer = content.load().then(function() {
    // Switch content only if another 'loadContent' wasn't called before.
    if (self.loading == content) {
      self.setContent(content);
    }
  }, function(error) {
    // Still loading same content?
    if (self.loading == content) {
      self.setContent(null);
      self.$element.append($(common.render('frame-error-pane', error)));
    }
  });
  return defer;
};


////////////////////////////////////////////////////////////////////////////////
// Common code for 'New group' and 'Edit group' forms.


var GroupForm = function($element, groupName) {
  this.$element = $element;
  this.groupName = groupName;
  this.visible = false;
  this.readOnly = false;
};


// Presents this form in $parent.
GroupForm.prototype.show = function($parent) {
  this.visible = true;
  this.$element.appendTo($parent);
  if (this.groupName) {
    setCurrentGroupInURL(this.groupName);
  }
};


// Hides this form.
GroupForm.prototype.hide = function() {
  this.visible = false;
  this.$element.detach();
};


// Switches focus to members box on the form.
GroupForm.prototype.focus = function() {
  $('textarea[name=membersAndGlobs]', this.$element).focus();
};


// Load contents of this from the server.
// Returns deferred.
GroupForm.prototype.load = function() {
  // Subclasses implement this. Base class just returns resolved deferred.
  var defer = $.Deferred();
  defer.resolve();
  return defer;
};


// Disables or enables controls on the form.
GroupForm.prototype.setInteractionDisabled = function(disabled) {
  if (!this.readOnly) {
    $('button', this.$element).attr('disabled', disabled);
  }
};


// Disable modification of the form.
GroupForm.prototype.makeReadonly = function() {
  this.readOnly = true;
  $('button, input, textarea', this.$element).attr('disabled', true);
};


// Shows a message on a form. |type| can be 'success' or 'error'.
GroupForm.prototype.showMessage = function(type, title, message) {
  $('#alerts', this.$element).html(
      common.getAlertBoxHtml(type, title, message));
};


// Hides a message previously shown with 'showMessage'.
GroupForm.prototype.hideMessage = function() {
  $('#alerts', this.$element).empty();
};


// Adds validators and submit handlers to the form.
GroupForm.prototype.setupSubmitHandler = function(submitCallback) {
  $('form', this.$element).validate({
    // Submit handler is only called if form passes validation.
    submitHandler: function($form) {
      // Extract data from the form.
      var name = $('input[name=name]', $form).val();
      var description = $('textarea[name=description]', $form).val();
      var owners = $('input[name=owners]', $form).val();
      var membersAndGlobs = $('textarea[name=membersAndGlobs]', $form).val();
      var nested = $('textarea[name=nested]', $form).val();

      // Splits 'value' on lines boundaries, trims spaces and returns lines
      // as an array of strings. Helper function used below.
      var splitItemList = function(value) {
        var trimmed = _.map(value.split('\n'), function(item) {
          return item.trim();
        });
        return _.filter(trimmed, function(item) {
          return !!item;
        });
      };

      // Split joined membersAndGlobs into separately 'members' and 'globs'.
      // Globs are defined by '*' or '[]' chars not allowed in member entries.
      var members = [];
      var globs = [];
      _.each(splitItemList(membersAndGlobs), function(item) {
        if (isGlob(item)) {
          globs.push(item);
        } else {
          members.push(item);
        }
      });

      // Pass data to callback. Never allow actual POST by always returning
      // false. POST is done via asynchronous request in the submit handler.
      try {
        submitCallback({
          name: name.trim(),
          description: description.trim(),
          owners: owners,
          members: common.addPrefixToItems('user', members),
          globs: common.addPrefixToItems('user', globs),
          nested: splitItemList(nested)
        });
      } finally {
        return false;
      }
    },
    // Validation rules, uses validators defined in registerFormValidators.
    rules: {
      'name': {
        required: true,
        groupName: true
      },
      'description': {
        required: true
      },
      'owners': {
        groupNameOrEmpty: true
      },
      'membersAndGlobs': {
        membersAndGlobsList: true
      },
      'nested': {
        groupList: true
      }
    }
  });
};


////////////////////////////////////////////////////////////////////////////////
// Form to view\edit existing group.


var EditGroupForm = function(groupName) {
  // Call parent constructor.
  GroupForm.call(this, null, groupName);
  // Name of the group this form operates on.
  this.groupName = groupName;
  // Last-Modified header of content (once loaded).
  this.lastModified = null;
  // Called when 'Delete group' action is invoked.
  this.onDeleteGroup = null;
  // Called when group form is submitted.
  this.onUpdateGroup = null;
};


// Inherit from GroupForm.
EditGroupForm.prototype = Object.create(GroupForm.prototype);


// Loads contents of this from the server.
EditGroupForm.prototype.load = function() {
  var self = this;
  var defer = api.groupRead(this.groupName);
  defer.then(function(response) {
    self.buildForm(response.data.group, response.headers['Last-Modified']);
  });
  return defer;
};


// Builds DOM element with this form given group object.
EditGroupForm.prototype.buildForm = function(group, lastModified) {
  // Prepare environment for template.
  group = _.clone(group);
  group.changeLogUrl = common.getChangeLogURL('AuthGroup', group.name);
  group.lookupUrl = common.getLookupURL(group.name);
  group.fullListingUrl = common.getGroupListingURL(group.name);

  // Join members and globs list into single UI list.
  var members = common.stripPrefixFromItems('user', group.members || []);
  var globs = common.stripPrefixFromItems('user', group.globs || []);
  var membersAndGlobs = [].concat(members, globs);

  // Assert that they can be split apart later.
  if (!_.all(members, function(item) { return !isGlob(item); })) {
    console.log(members);
    throw 'Invalid members list';
  }
  if (!_.all(globs, isGlob)) {
    console.log(globs);
    throw 'Invalid glob list';
  }

  // Convert list of strings to a single text blob.
  group.membersAndGlobs = membersAndGlobs.join('\n') + '\n';
  group.nested = (group.nested || []).join('\n') + '\n';
  group.isExternal = isExternalGroupName(group.name);

  this.$element = $(common.render('edit-group-form-template', group));
  this.lastModified = lastModified;

  if (group.isExternal) {
    // Read-only UI for external groups.
    this.makeReadonly();
  } else {
    // 'Delete' button handler. Asks confirmation and calls 'onDeleteGroup'.
    var self = this;
    $('#delete-btn', this.$element).click(function() {
      common.confirm('Delete this group?').done(function() {
        self.onDeleteGroup(self.groupName, self.lastModified);
      });
    });

    // Add validation and submit handler.
    this.setupSubmitHandler(function(group) {
      self.onUpdateGroup(group, self.lastModified);
    });
  }

  // Activate tooltips on utility buttons.
  $('#full-listing-button', this.$element).tooltip();
  $('#lookup-button', this.$element).tooltip();
  $('#change-log-button', this.$element).tooltip();
};


////////////////////////////////////////////////////////////////////////////////
// 'Create new group' form.


// It must be an invalid group name (to avoid collisions).
var NEW_GROUP_PLACEHOLDER = 'new!';


var NewGroupForm = function(onSubmitGroup) {
  // Call parent constructor.
  GroupForm.call(
      this, $(common.render('new-group-form-template')), NEW_GROUP_PLACEHOLDER);

  // Add validation and submit handler.
  this.setupSubmitHandler(function(group) {
    onSubmitGroup(group);
  });
};


// Inherit from GroupForm.
NewGroupForm.prototype = Object.create(GroupForm.prototype);


////////////////////////////////////////////////////////////////////////////////
// Address bar manipulation (to put the currently selected group in URL).


var initializeGroupInURL = function() {
  // This code serves two purposes:
  //  1. It converts old-style anchor-based URLs into new ones.
  //  2. It initializes {'group': ...} entry in the history.
  var groupName = getCurrentGroupInURL() || common.getAnchor();
  if (groupName) {
    window.history.replaceState(
      {'group': groupName}, null, GROUPS_ROOT_URL + groupName);
  }
};


var getCurrentGroupInURL = function() {
  var p = window.location.pathname;
  if (p.startsWith(GROUPS_ROOT_URL)) {
    return p.slice(GROUPS_ROOT_URL.length);
  }
  return '';
};


var setCurrentGroupInURL = function(groupName) {
  if (getCurrentGroupInURL() != groupName) {
    console.log('Pushing to history', groupName);
    window.history.pushState(
      {'group': groupName}, null, GROUPS_ROOT_URL + groupName);
  }
};


var onCurrentGroupInURLChange = function(cb) {
  window.onpopstate = function(event) {
    var s = event.state;
    if (s && s.hasOwnProperty('group')) {
      console.log('Navigated back to', s.group);
      cb(s.group);
    };
  };
};


////////////////////////////////////////////////////////////////////////////////
// Main entry point, sets up all high-level UI logic.


// Wrapper around a REST API call that originated from some form.
// Locks UI while call is running, refreshes a list of groups once it completes.
var waitForResult = function(defer, groupChooser, form) {
  // Deferred triggered when update is finished (successfully or not). Return
  // values of this function.
  var done = $.Deferred();

  // Lock UI while running the request, unlock once it finishes.
  groupChooser.setInteractionDisabled(true);
  form.setInteractionDisabled(true);
  done.always(function() {
    groupChooser.setInteractionDisabled(false);
    form.setInteractionDisabled(false);
  });

  // Hide previous error message (if any).
  form.hideMessage();

  // Wait for request to finish, refetch the list of groups and trigger |done|.
  defer.then(function(response) {
    // Call succeeded: refetch the list of groups and return the result.
    groupChooser.refetchGroups().then(function() {
      done.resolve(response);
    }, function(error) {
      // Show page-wide error message, since without the list of groups the page
      // is useless.
      common.presentError(error.text);
      done.reject(error);
    });
  }, function(error) {
    // Show error message on the form, since it's local error with the request.
    form.showMessage('error', 'Oh snap!', error.text);
    done.reject(error);
  });

  return done.promise();
};


// Sets up jQuery.validate validators for group form fields.
var registerFormValidators = function() {
  // Regular expressions for form fields.
  var groupRe = /^[0-9a-zA-Z_\-\.\/@]{3,80}$/;
  var membersRe = /^((user|bot|project|service|anonymous)\:)?[\w\-\+\%\.\@\*\[\]]+$/;

  // Splits |value| on lines boundary and checks that each line matches 're'.
  // Helper function use in validators below.
  var validateItemList = function(re, value) {
    return _.reduce(value.split('\n'), function(acc, item) {
      return acc && (!item || re.test(item));
    }, true);
  };

  // ID (as used in 'rules' section of $form.validate) -> [checker, error msg].
  var validators = {
    'groupName': [
      function(value, element) { return groupRe.test(value); },
      'Invalid group name'
    ],
    'groupNameOrEmpty': [
      function(value, element) { return !value || groupRe.test(value); },
      'Invalid group name'
    ],
    'membersAndGlobsList': [
      _.partial(validateItemList, membersRe),
      'Invalid member entry'
    ],
    'groupList': [
      _.partial(validateItemList, groupRe),
      'Invalid group name'
    ]
  };

  // Actually register them all.
  _.each(validators, function(value, key) {
    $.validator.addMethod(key, value[0], value[1]);
  });
};


exports.onContentLoaded = function() {
  // Setup global UI elements.
  var groupChooser = new GroupChooser($('#group-chooser'), config.is_admin);
  var searchBox = new SearchBox($('#search-box'));
  var mainFrame = new ContentFrame($('#main-content-pane'));

  // Setup form validators used in group forms.
  registerFormValidators();

  // Called to setup 'Create new group' flow.
  var startNewGroupFlow = function() {
    var form = new NewGroupForm(function(groupObj) {
      var request = api.groupCreate(groupObj);
      waitForResult(request, groupChooser, form).done(function() {
        // Pass 'CREATE' as |state| to group chooser. It will eventually
        // be passed to 'startEditGroupFlow' after group form for new group
        // loads.
        groupChooser.setSelection(groupObj.name, 'CREATE');
      });
    });
    mainFrame.loadContent(form);
  };

  // Called to setup 'Edit the group' flow (including deletion of a group).
  // |state| is whatever was passed to 'groupChooser.setSelection' as a second
  // argument or null if selection changed due to user action.
  var startEditGroupFlow = function(groupName, state) {
    var form = new EditGroupForm(groupName);

    // Called when 'Delete' button is clicked.
    form.onDeleteGroup = function(groupName, lastModified) {
      var request = api.groupDelete(groupName, lastModified);
      waitForResult(request, groupChooser, form).done(function() {
        groupChooser.selectDefault();
      });
    };

    // Called when 'Update' button is clicked.
    form.onUpdateGroup = function(groupObj, lastModified) {
      var request = api.groupUpdate(groupObj, lastModified);
      waitForResult(request, groupChooser, form).done(function() {
        // Pass 'UPDATE' as |state| to group chooser. It will eventually
        // be passed to 'startEditGroupFlow' after group form reloads.
        groupChooser.setSelection(groupObj.name, 'UPDATE');
      });
    };

    // Once group loads, show status message based on the operation performed.
    // It is passed as |state| here.
    mainFrame.loadContent(form).done(function() {
      if (state == 'CREATE') {
        form.showMessage('success', 'Group created.', '');
      } else if (state == 'UPDATE') {
        form.showMessage('success', 'Group updated.', '');
      }
    });
  };

  // Attach event handlers.
  groupChooser.onSelectionChanged(function(selection, state) {
    if (selection === null) {
      startNewGroupFlow();
    } else {
      startEditGroupFlow(selection, state);
    }
  });


  // Allow to select groups via search box.
  searchBox.onTextChanged(function(text) {
    var found = longestMatch(groupChooser.groupNames, text);
    if (found) {
      groupChooser.setSelection(found, null);
    }
  });

  // Focus on group members box if "Enter" is hit.
  searchBox.onEnterKey(function(text) {
    if (mainFrame.content) {
      mainFrame.content.focus();
    }
  });

  // Converts ".../groups#groupName" to ".../groups/groupName", to make sure
  // old-style URLs still work.
  initializeGroupInURL();

  // Helper function that selects a group based on current URL.
  var jumpToCurrentGroup = function(selectDefault) {
    var current = getCurrentGroupInURL();
    if (current == NEW_GROUP_PLACEHOLDER) {
      groupChooser.setSelection(null, null);
    } else if (groupChooser.isKnownGroup(current)) {
      groupChooser.setSelection(current, null);
    } else if (selectDefault) {
      groupChooser.selectDefault();
    }
  };

  // Load and show data.
  groupChooser.refetchGroups().then(function() {
    // Show a group specified in the URL (or a default one).
    jumpToCurrentGroup(true);
    // Only start paying attention to URL changes when the state is loaded.
    onCurrentGroupInURLChange(function() { jumpToCurrentGroup(false); });
  }, function(error) {
    common.presentError(error.text);
  });

  // Present the page only when main content pane is loaded.
  mainFrame.onContentShown(function() {
    common.presentContent();
    groupChooser.madeVisible();
  });

  // Enable XSRF token auto-updater.
  api.setXSRFTokenAutoupdate(true);
};

return exports;

}());
