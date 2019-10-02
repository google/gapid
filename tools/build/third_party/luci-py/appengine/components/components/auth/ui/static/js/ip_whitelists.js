// Copyright 2014 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

var ip_whitelists = (function() {
'use strict';

var exports = {};


// Multiline string with subnets -> list of subnet strings.
var splitSubnetsList = function(subnets) {
  var mapper = function(str) { return str.trim(); };
  var filter = function(str) { return str !== ''; };
  return _.filter(_.map(subnets.split('\n'), mapper), filter);
};


////////////////////////////////////////////////////////////////////////////////
// Selector is a combo box with IP whitelist names and "Create new" item.


var Selector = function($element, readonly) {
  this.$element = $element;
  this.readonly = readonly;
  this.onCreateWhitelist = null;
  this.onWhitelistSelected = null;

  var that = this;
  $element.change(function() {
    that.onSelectionChanged();
  });
};


// Rebuilds the list.
Selector.prototype.populate = function(whitelists, selection) {
  this.$element.empty();

  var that = this;
  var selected = null;
  var addToSelector = function(name, data) {
    var option = $(document.createElement('option'));
    option.text(name);
    option.data('selector-data', data);
    that.$element.append(option);
    if (selected === null || name == selection) {
      selected = option;
    }
  };

  // All whitelists.
  _.each(whitelists, function(whitelist) {
    addToSelector(whitelist.name, whitelist);
  });

  // Separator and "New list" option.
  if (!this.readonly) {
    addToSelector('----------------------------', 'SEPARATOR');
    addToSelector('Create new IP whitelist', 'CREATE');
  } else {
    // Empty list looks ugly, put something in there.
    if (selected === null) {
      addToSelector('No IP whitelists', 'SEPARATOR');
    }
  }

  // Make the selection.
  selected.attr('selected', 'selected');
  this.onSelectionChanged();
};


// Called whenever selected item in combo box changes.
Selector.prototype.onSelectionChanged = function() {
  var selectedOption = $('option:selected', this.$element);
  var selectedData = selectedOption.data('selector-data');
  if (selectedData === 'SEPARATOR') {
    if (this.onWhitelistSelected !== null) {
      this.onWhitelistSelected(null);
    }
  } else if (selectedData === 'CREATE') {
    if (this.onWhitelistSelected !== null) {
      this.onWhitelistSelected(null);
    }
    if (this.onCreateWhitelist !== null) {
      this.onCreateWhitelist();
    }
  } else {
    if (this.onWhitelistSelected !== null) {
      this.onWhitelistSelected(selectedData);
    }
  }
};


////////////////////////////////////////////////////////////////////////////////
// "Create new IP whitelist" modal dialog.


var NewWhitelistDialog = function($element) {
  this.$element = $element;
  this.$alerts = $('#alerts-box', $element);

  this.onCreateWhitelist = null;

  var that = this;
  $('#create-btn', $element).on('click', function() {
    that.onCreateClicked();
  });
};


// Cleans previous values and alerts, presents the dialog.
NewWhitelistDialog.prototype.show = function() {
  $('input[name="name"]', this.$element).val('');
  $('input[name="description"]', this.$element).val('');
  $('textarea[name="subnets"]', this.$element).val('');
  this.$alerts.empty();
  this.$element.modal('show');
};


// Cleans alerts, hides the dialog.
NewWhitelistDialog.prototype.hide = function() {
  this.$alerts.empty();
  this.$element.modal('hide');
};


// Displays error message in the dialog.
NewWhitelistDialog.prototype.showError = function(text) {
  this.$alerts.html(common.getAlertBoxHtml('error', 'Oh snap!', text));
};


// Called when "Create" button is clicked. Invokes onCreateWhitelist callback.
NewWhitelistDialog.prototype.onCreateClicked = function() {
  if (this.onCreateWhitelist === null) {
    return;
  }

  var name = $('input[name="name"]', this.$element).val();
  var desc = $('input[name="description"]', this.$element).val();
  var list = $('textarea[name="subnets"]', this.$element).val();

  this.onCreateWhitelist({
    name: name,
    description: desc,
    subnets: splitSubnetsList(list)
  });
};


////////////////////////////////////////////////////////////////////////////////
// The panel with information about some selected IP whitelist.


var WhitelistPane = function($element) {
  this.$element = $element;
  this.$alerts = $('#alerts-box', $element);

  this.ipWhitelist = null;
  this.lastModified = null;

  this.onUpdateWhitelist = null;
  this.onDeleteWhitelist = null;

  var that = this;
  $('#update-btn', $element).on('click', function() { that.onUpdateClick(); });
  $('#delete-btn', $element).on('click', function() { that.onDeleteClick(); });
};


// Shows the pane.
WhitelistPane.prototype.show = function() {
  this.$element.show();
};


// Hides the pane.
WhitelistPane.prototype.hide = function() {
  this.$element.hide();
};


// Displays error message in the pane.
WhitelistPane.prototype.showError = function(text) {
  this.$alerts.html(common.getAlertBoxHtml('error', 'Oh snap!', text));
};


// Displays success message in the pane.
WhitelistPane.prototype.showSuccess = function(text) {
  this.$alerts.html(common.getAlertBoxHtml('success', 'Done!', text));
};


// Fills in the form with details about some IP whitelist.
WhitelistPane.prototype.populate = function(ipWhitelist) {
  // TODO(vadimsh): Convert ipWhitelist.modified_ts to a value compatible with
  // 'If-Unmodified-Since' header and put it into this.lastModified.
  this.ipWhitelist = ipWhitelist;
  $('input[name="description"]', this.$element).val(ipWhitelist.description);
  $('textarea[name="subnets"]', this.$element).val(
      (ipWhitelist.subnets || []).join('\n'));
  this.$alerts.empty();
};


// Called whenever 'Update' button is clicked.
WhitelistPane.prototype.onUpdateClick = function() {
  if (!this.onUpdateWhitelist) {
    return;
  }

  var desc = $('input[name="description"]', this.$element).val();
  var list = $('textarea[name="subnets"]', this.$element).val();

  var updatedIpWhitelist = _.clone(this.ipWhitelist);
  updatedIpWhitelist.description = desc;
  updatedIpWhitelist.subnets = splitSubnetsList(list);

  this.onUpdateWhitelist(updatedIpWhitelist, this.lastModified);
};


// Called whenever 'Delete' button is clicked.
WhitelistPane.prototype.onDeleteClick = function() {
  if (this.onDeleteWhitelist) {
    this.onDeleteWhitelist(this.ipWhitelist.name, this.lastModified);
  }
};


////////////////////////////////////////////////////////////////////////////////
// Top level logic.


// Fetches all IP whitelists, adds them to the selector, selects some.
var reloadWhitelists = function(selector, selection) {
  var done = $.Deferred();
  api.ipWhitelists().then(function(response) {
    selector.populate(response.data.ip_whitelists, selection);
    common.presentContent();
    done.resolve(response);
  }, function(error) {
    common.presentError(error.text);
    done.reject(error);
  });
  return done.promise();
};


// Called when HTML body of a page is loaded.
exports.onContentLoaded = function() {
  var readonly = config.auth_service_config_locked || !config.is_admin;
  var selector = new Selector($('#ip-whitelists-selector'), readonly);
  var newListDialog = new NewWhitelistDialog($('#create-ip-whitelist'));
  var whitelistPane = new WhitelistPane($('#selected-ip-whitelist'));

  // Enable\disable UI interactions on the page.
  var setInteractionDisabled = function(disabled) {
    common.setInteractionDisabled(selector.$element, disabled);
    common.setInteractionDisabled(newListDialog.$element, disabled);
    common.setInteractionDisabled(whitelistPane.$element, disabled);
  };

  // Disable UI, wait for defer, reload whitelists, enable UI.
  var wrapDefer = function(defer, selection) {
    var done = $.Deferred();
    setInteractionDisabled(true);
    defer.then(function(response) {
      reloadWhitelists(selector, selection).done(function() {
        setInteractionDisabled(false);
        done.resolve(response);
      });
    }, function(error) {
      setInteractionDisabled(false);
      done.reject(error);
    });
    return done.promise();
  };

  // Show the dialog when 'Create IP whitelist' item is selected.
  selector.onCreateWhitelist = function() {
    newListDialog.show();
  };

  // Update whitelistPane with selected whitelist on a selection changes.
  selector.onWhitelistSelected = function(ipWhitelist) {
    if (ipWhitelist === null) {
      whitelistPane.hide();
    } else {
      whitelistPane.populate(ipWhitelist);
      whitelistPane.show();
    }
  };

  // Wire dialog's "Create" button.
  newListDialog.onCreateWhitelist = function(ipWhitelist) {
    // Some minimal client side validation, otherwise handlers nay return 404.
    if (!ipWhitelist.name.match(/^[0-9a-zA-Z_\-\+\.\ ]{2,200}$/)) {
      newListDialog.showError('Invalid IP whitelist name.');
      return;
    }
    var defer = wrapDefer(api.ipWhitelistCreate(ipWhitelist), ipWhitelist.name);
    defer.then(function(response) {
      newListDialog.hide();
      whitelistPane.showSuccess('Created.');
    }, function(error) {
      newListDialog.showError(error.text || 'Unknown error');
    });
  };

  // Wire 'Delete whitelist' button.
  whitelistPane.onDeleteWhitelist = function(name, lastModified) {
    var defer = wrapDefer(api.ipWhitelistDelete(name, lastModified), null);
    defer.fail(function(error) {
      whitelistPane.showError(error.text || 'Unknown error');
    });
  };

  // Wire 'Update whitelist' button.
  whitelistPane.onUpdateWhitelist = function(ipWhitelist, lastModified) {
    var defer = wrapDefer(
        api.ipWhitelistUpdate(ipWhitelist, lastModified), ipWhitelist.name);
    defer.then(function(response) {
      whitelistPane.showSuccess('Updated.');
    }, function(error) {
      whitelistPane.showError(error.text || 'Unknown error');
    });
  };

  // Initial data fetch.
  whitelistPane.hide();
  reloadWhitelists(selector, null);

  // Enable XSRF token auto-updater.
  api.setXSRFTokenAutoupdate(true);
};


return exports;
}());
