// Copyright 2014 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

var services = (function() {
'use strict';

var exports = {};


// Should correspond to auth.api._process_cache_expiration_sec.
var CACHE_LAG_SEC = 30;


var listServices = function() {
  return api.call('GET', '/auth_service/api/v1/services');
};


var generateLink = function(serviсe_app_id) {
  return api.call(
      'POST',
      '/auth_service/api/v1/services/' + serviсe_app_id + '/linking_url');
};


var updateServiceListing = function() {
  var defer = listServices();

  // Show the page when first fetch completes, refetch after X sec.
  defer.always(function() {
    common.presentContent();
    setTimeout(updateServiceListing, 10000);
  });

  defer.then(function(result) {
    var services = [];
    var now = result.data.now;
    var auth_db_rev = result.data.auth_db_rev;
    var auth_code_version = result.data.auth_code_version;

    var getUpToDateStatus = function(updated_ts) {
      var dt = Math.round((now - updated_ts) / 1000000);
      if (dt < CACHE_LAG_SEC) {
        return {
          text: 'clearing cache',
          tooltip: String.format(
              'Cache clears in {0} sec.', CACHE_LAG_SEC - dt),
          label: 'warning'
        };
      }
      return {
        text: 'ok',
        tooltip: 'Has latest ACLs',
        label: 'success'
      };
    };

    var getReplicaStatus = function(service) {
      if (service.auth_db_rev == auth_db_rev.rev) {
        return getUpToDateStatus(service.push_finished_ts);
      }
      if (service.push_status === 0) {
        return {
          text: 'syncing',
          tooltip: String.format(
              '{0} rev behind', auth_db_rev.rev - service.auth_db_rev),
          label: 'warning'
        };
      }
      return {
        text: 'syncing',
        tooltip: service.push_error,
        label: 'danger'
      };
    };

    var getReplicationLag = function(service) {
      if (!service.push_started_ts || !service.push_finished_ts)
        return 'N/A';
      var lag_microsec = service.push_finished_ts - service.push_started_ts;
      return Math.round(lag_microsec / 1000.0) + ' ms';
    };

    // Add auth service itself to the list.
    services.push({
      app_id: auth_db_rev.primary_id,
      auth_code_version: auth_code_version,
      lag_ms: '0 ms',
      service_url: '/',
      status: getUpToDateStatus(auth_db_rev.ts)
    });

    // Add all replicas.
    _.each(result.data.services, function(service) {
      services.push({
        app_id: service.app_id,
        auth_code_version: service.auth_code_version || '--',
        lag_ms: getReplicationLag(service),
        service_url: service.replica_url + '/',
        status: getReplicaStatus(service)
      });
    });
    $('#services-list').html(
        common.render('services-list-template', {services: services}));
    $('#services-list .status-label').tooltip({});
    $('#services-list-alerts').empty();
  }, function(error) {
    $('#services-list-alerts').html(
        common.getAlertBoxHtml('error', 'Oh shap!', error.text));
  });
};


var initAddServiceForm = function() {
  $('#add-service-form').validate({
    submitHandler: function($form) {
      // Disable form UI while the request is in flight.
      common.setInteractionDisabled($form, true);
      // Launch the request.
      var app_id = $('input[name=serviсe_app_id]', $form).val();
      var defer = generateLink(app_id);
      // Enable UI back when it finishes.
      defer.always(function() {
        common.setInteractionDisabled($form, false);
      });
      defer.then(function(result) {
        // On success, show the URL.
        var url = result.data.url;
        $('#add-service-form-alerts').html(
            common.render('present-link-template', {url: url, app_id: app_id}));
      }, function(error) {
        // On a error, show the error message.
        $('#add-service-form-alerts').html(
            common.getAlertBoxHtml('error', 'Oh shap!', error.text));
      });
      return false;
    },
    rules: {
      'serviсe_app_id': {
        required: true
      }
    },
  });
};


// Called when HTML body of a page is loaded.
exports.onContentLoaded = function() {
  api.setXSRFTokenAutoupdate(true);
  if (config.is_admin) {
    initAddServiceForm();
  } else {
    $('#add-service-row').hide();
  }
  updateServiceListing();
};


return exports;
}());
