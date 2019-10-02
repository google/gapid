// Copyright 2016 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

this.swarming = this.swarming || function() {

  var swarming = {};

  // return the longest string in an array
  swarming.longest = function(arr) {
      var most = "";
      for(var i = 0; i < arr.length; i++) {
        if (arr[i] && arr[i].length > most.length) {
          most = arr[i];
        }
      }
      return most;
    };

  swarming.stableSort = function(arr, comp) {
    if (!arr || !comp) {
      console.log("missing arguments to stableSort", arr, comp);
      return;
    }
    // We can guarantee a potential non-stable sort (like V8's
    // Array.prototype.sort()) to be stable by first storing the index in the
    // original sorting and using that if the original compare was 0.
    arr.forEach(function(e, i){
      if (e !== undefined && e !== null) {
        e.__sortIdx = i;
      }
    });

    arr.sort(function(a, b){
      // undefined and null elements always go last.
      if (a === undefined || a === null) {
        if (b === undefined || b === null) {
          return 0;
        }
        return 1;
      }
      if (b === undefined || b === null) {
        return -1;
      }
      var c = comp(a, b);
      if (c === 0) {
        return a.__sortIdx - b.__sortIdx;
      }
      return c;
    });
  }

  // postWithToast makes a post request and updates the error-toast
  // element with the response, regardless of failure.  See error-toast.html
  // for more information. The body param should be an object or undefined.
  swarming.postWithToast = function(url, msg, auth_headers, body) {
    // Keep toast displayed until we hear back from the request.
    sk.errorMessage(msg, 0);

    auth_headers["content-type"] = "application/json; charset=UTF-8";
    if (body) {
      body = JSON.stringify(body);
    }

    return sk.request("POST", url, body, auth_headers).then(function(response) {
      // Assumes response is a stringified json object
      sk.errorMessage("Request sent.  Response: "+response, 3000);
      return response;
    }).catch(function(r) {
      // Assumes r is something like
      // {response: "{\"error\":{\"message\":\"User ... \"}}", status: 403}
      var err = JSON.parse(r.response);
      console.log("Request failed", err);
      var humanReadable = (err.error && err.error.message) || JSON.stringify(err);
      sk.errorMessage("Request failed.  Reason: "+ humanReadable, 5000);
      return Promise.reject(err);
    });
  }

  // sanitizeAndHumanizeTime parses a date string or ms_since_epoch into a JS
  // Date object, assuming UTC time. It also creates a human readable form in
  // the obj under a key with a human_ prefix.  E.g.
  // swarming.sanitizeAndHumanizeTime(foo, "some_ts")
  // parses the string/int at foo["some_ts"] such that foo["some_ts"] is now a
  // Date object and foo["human_some_ts"] is the human formated version from
  // sk.human.localeTime.
  swarming.sanitizeAndHumanizeTime = function(obj, key) {
    obj["human_"+key] = "--";
    if (obj[key]) {
      if (obj[key].endsWith && !obj[key].endsWith('Z')) {
        // Timestamps from the server are missing the 'Z' that specifies Zulu
        // (UTC) time. If that's not the case, add the Z. Otherwise, some
        // browsers interpret this as local time, which throws off everything.
        // TODO(kjlubick): Should the server output milliseconds since the
        // epoch?  That would be more consistent.
        // See http://crbug.com/714599
        obj[key] += 'Z';
      }
      obj[key] = new Date(obj[key]);

      // Extract the timezone.
      var str = obj[key].toString();
      var timezone = str.substring(str.indexOf("("));

      // If timestamp is today, skip the date.
      var now = new Date();
      if (obj[key].getDate() == now.getDate() &&
          obj[key].getMonth() == now.getMonth() &&
          obj[key].getYear() == now.getYear()) {
        obj["human_"+key] = obj[key].toLocaleTimeString() + " " + timezone;
      } else {
        obj["human_"+key] = obj[key].toLocaleString() + " " + timezone;
      }
    }
  }

  // parseDuration parses a duration string into an integer number of seconds.
  // e.g:
  // swarming.parseDuration("40s") == 40
  // swarming.parseDuration("2m") == 120
  // swarming.parseDuration("1h") == 3600
  // swarming.parseDuration("foo") == null
  swarming.parseDuration = function(duration) {
    var number = duration.slice(0, -1);
    if (!/[1-9][0-9]*/.test(number)) {
      return null;
    }
    number = Number(number);

    var unit = duration.slice(-1);
    switch (unit) {
      // the fallthroughs here are intentional
      case 'h':
        number *= 60;
      case 'm':
        number *= 60;
      case 's':
        break;
      default:
        return null;
    }
    return number;
  }

  return swarming;
}();
