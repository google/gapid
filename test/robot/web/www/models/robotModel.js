/*
 * Copyright (C) 2018 Google Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
/*jslint white: true*/
'use strict';

// The robot model handles retrieving and sorting all of the data from the
// instance of robot running on the server.
var newRobotModel = async function () {
  async function queryArray(path) {
    return new Promise(function (resolve, reject) {
      var request = new XMLHttpRequest();
      request.open("GET", path, true);
      request.setRequestHeader("Content-type", "application/json");
      request.onload = function () {
        if (request.status >= 200 && request.status <= 300) {
          resolve(JSON.parse(request.responseText));
        } else {
          reject({ status: request.status, statusText: request.statusText });
        }
      };
      request.onerror = () => reject({ status: request.status, statusText: request.statusText });
      request.send();
    });
  }

  var model;
  var kindDimension, subjectDimension, traceTargetDimension, hostDimension, packageDimension, trackDimension;

  kindDimension = {
    name: "kind",
    keyOf: function (task) {
      return task.kind;
    },
    sort: function (keyA, keyB) {
      if (keyA === keyB) {
        return 0;
      }
      if (keyA === "trace") {
        return -1;
      }
      if (keyB === "trace") {
        return 1;
      }
      if (keyA === "report") {
        return -1;
      }
      if (keyB === "report") {
        return 1;
      }
    },
    items: [{ key: "trace" }, { key: "report" }, { key: "replay" }],
    displayName: function (key) {
      return key;
    },
    keyExists: function (key) {
      return key === "trace" || key === "report" || key === "replay";
    }
  };
  subjectDimension = {
    name: "subject",
    keyOf: function (task) {
      return task.trace.subject;
    },
    items: [],
    keysToItems: {},
    source: async function () {
      var subjects = await queryArray("subjects/");
      subjects.forEach(function (subject) {
        var item = {
          key: subject.id,
          display: subject.Information.APK.package,
          underlying: subject
        };
        subjectDimension.items.push(item);
        subjectDimension.keysToItems[item.key] = item;
      });
    },
    displayName: function (key) {
      return subjectDimension.keysToItems[key].display;
    },
    keyExists: function (key) {
      return subjectDimension.keysToItems[key] != null;
    }
  };
  traceTargetDimension = {
    name: "traceTarget",
    keyOf: function (task) {
      return task.trace.target;
    },
    items: [],
    keysToItems: {},
    source: async function () {
      var devices = await queryArray("devices/");
      var workers = await queryArray("workers/");

      devices.forEach(function (device) {
        workers.forEach(function (worker) {
          // Create an item for each device that is also a target worker and can perform trace operations ('2')
          const traceOp = 2;
          if (device.id === worker.target && worker.Operation.includes(traceOp) && traceTargetDimension.keysToItems[device.id] == null) {
            var item = {
              key: device.id,
              display: device.information.Configuration.Hardware.Name,
              underlying: device
            };
            traceTargetDimension.items.push(item);
            traceTargetDimension.keysToItems[item.key] = item;
          }
        });
      });
    },
    displayName: function (key) {
      return traceTargetDimension.keysToItems[key].display;
    },
    keyExists: function (key) {
      return traceTargetDimension.keysToItems[key] != null;
    }
  };

  hostDimension = {
    name: "host",
    keyOf: function (task) {
      return task.host;
    },
    items: [],
    keysToItems: {},
    source: async function () {
      var devices = await queryArray("devices/");
      var workers = await queryArray("workers/");

      devices.forEach(function (device) {
        workers.forEach(function (worker) {
          // any device that is also a host worker
          if (device.id === worker.host && hostDimension.keysToItems[device.id] == null) {
            var item = {
              key: device.id,
              display: device.information.Configuration.Hardware.Name,
              underlying: device
            };
            hostDimension.items.push(item);
            hostDimension.keysToItems[item.key] = item;
          }
        });
      });
    },
    displayName: function (key) {
      return hostDimension.keysToItems[key].display;
    },
    keyExists: function (key) {
      return hostDimension.keysToItems[key] != null;
    }
  };

  packageDimension = {
    name: "package",
    keyOf: function (task) {
      return task.package;
    },
    items: [],
    keysToItems: {},
    packageDisplayToOrder: {},
    tracks: {
      "auto": {
        key: "",
        display: "auto",
        underlying: { "id": "", "name": "auto", "head": "" },
        packageList: [],
        headPackage: ""
      }
    },
    sort: function (keyA, keyB) {
      var aO = packageDimension.packageDisplayToOrder[keyA];
      var bO = packageDimension.packageDisplayToOrder[keyB];
      if (aO != null && bO != null) {
        return aO - bO;
      }
      return keyA < keyB ? -1 : keyB < keyA ? 1 : 0;
    },
    packageToTrack: {},
    done: false,
    source: async function () {
      packageDimension.packageToTrack = {};
      packageDimension.items = [];
      packageDimension.keysToItems = {};
      packageDimension.ackageDisplayToOrder = {};
      packageDimension.tracks = {
        "auto": {
          key: "",
          display: "auto",
          underlying: { "id": "", "name": "auto", "head": "" },
          packageList: [],
          headPackage: ""
        }
      };
      var packages = await queryArray("packages/");
      var tracks = await queryArray("tracks/");
      var childMap = {};
      var rootPkgs = [];

      packages.forEach(function (pkg) {
        var packageItem = {
          underlying: pkg,
          display: "unknown - " + pkg.id,
          key: pkg.id
        };
        packageDimension.items.push(packageItem);
        packageDimension.keysToItems[packageItem.key] = packageItem;
        // Figure out the proper display name for the package.
        if (pkg.information.tag != null) {
          packageItem.display = pkg.information.tag;
        } else if (pkg.information.type === 2 && pkg.information.cl != null) {
          packageItem.display = pkg.information.cl;
        } else if (pkg.information.uploader) {
          packageItem.display = pkg.information.uploader + " - " + pkg.id;
        }
        // No parent means this package is the root of a track.
        if (pkg.parent != null) {
          childMap[pkg.parent] = pkg.id;
        } else {
          rootPkgs.push(pkg.id);
        }
      });

      rootPkgs.forEach(function (root) {
        var packageList = [];
        var childId, head, foundTrack;
        // making sure packages have a clear order from root -> head.
        packageDimension.packageDisplayToOrder[packageDimension.keysToItems[root].display] = Object.keys(packageDimension.packageDisplayToOrder).length;
        packageList.push(root);
        childId = childMap[root];
        while (childId != null) {
          packageDimension.packageDisplayToOrder[packageDimension.keysToItems[childId].display] = Object.keys(packageDimension.packageDisplayToOrder).length;
          // We want tracks stored from Root -> Head.
          packageList.push(childId);
          root = childId;
        }

        head = root;
        if (tracks.every(function (track) {
          var trackInfo;
          if (track.head === head) {
            trackInfo = {
              key: track.id,
              display: track.name,
              underlying: track,
              packageList: packageList,
              headPackage: head
            };
            packageDimension.tracks[track.name] = trackInfo;
            packageList.forEach((pkg) => packageDimension.packageToTrack[pkg] = trackInfo);
            return false;
          }
          return true;
        })) {
          // If not every track failed to match the head we can store them all under the auto track.
          packageDimension.tracks.auto.packageList.push(...packageList);
          packageDimension.tracks.auto.headPackage = packageList[packageList.length - 1];
        }
      });
    },
    displayName: function (key) {
      return packageDimension.keysToItems[key].display;
    },
    keyExists: function (key) {
      return packageDimension.keysToItems[key] != null;
    }
  };

  trackDimension = {
    name: "track",
    packageToTrack: {},
    keyOf: function (task) {
      if (trackDimension.packageToTrack[task.package] == null) {
        return "";
      }
      return trackDimension.packageToTrack[task.package].key;
    },
    items: [],
    source: async function () {
      // TODO: find another way to wait, since we are already waiting for source in another
      // promise upstream, this is really wasteful.
      await packageDimension.source();

      Object.keys(packageDimension.tracks).forEach(function (track) {
        if (packageDimension.tracks[track].packageList.length > 0) {
          trackDimension.items.push(packageDimension.tracks[track]);
        }
      });
      trackDimension.packageToTrack = packageDimension.packageToTrack;
    },
    displayName: function (key) {
      var result;
      if (trackDimension.items.some(function (item) { result = item.key; return item.key === key; })) {
        return result;
      }
      return "";
    },
    keyExists: function (key) {
      return trackDimension.displayName(key) !== "";
    }
  };
  model = {
    dimensions: [
      kindDimension,
      subjectDimension,
      packageDimension,
      trackDimension,
      traceTargetDimension,
      hostDimension
    ],
    dimensionsByName: {},

    tasks: [],
    connectTaskParentChild: function (childListMap, parentListMap, packageDim, task) {
      function findParentPackageInList(idList, childId) {
        // result is the index of the id's parent
        var result;
        if (idList.slice(1).some(function (id) { result = id; return childId === id; })) {
          return result;
        }
        return null;
      }
      function compareTasksSimilar(t1, t2) {
        return t1.trace.target === t2.trace.target && t1.trace.subject === t2.trace.subject && t1.host === t2.host;
      }
      var i;
      var parentPackage, parentList, childList, pkg;

      pkg = task.package;
      if (parentListMap[pkg] == null) {
        parentListMap[pkg] = [task];
      } else {
        parentListMap[pkg].push(task);
      }

      parentPackage = findParentPackageInList(packageDimension.packageToTrack[pkg].packageList, pkg);
      if (parentPackage != null) {
        if (childListMap[parentPackage] == null) {
          childListMap[parentPackage] = [task];
        } else {
          childListMap[parentPackage].push(task);
        }

        parentList = parentListMap[pkg];
        if (parentList != null) {
          parentList.forEach(function (parent) {
            if (compareTasksSimilar(task, parent)) {
              task.parent = parent;
            }
          });
        }
      }

      childList = childListMap[pkg];
      if (childList != null) {
        childList.forEach(function (child) {
          if (compareTasksSimilar(task, child)) {
            if (child.parent == null) {
              child.parent = task;
            } else {
              throw "A task's parent was found twice! parent: " + pkg + "; child: " + child + ";";
            }
          }
        });
      }
    },
    robotTasksPerKind: async function (kind, path, proc) {
      var statusMap = {
        "1": {
          status: "InProgress",
          result: "Unknown"
        },
        "2": {
          status: "Current",
          result: "Succeeded"
        },
        "3": {
          status: "Current",
          result: "Failed"
        }
      };
      var tasks = [];
      var notCurrentTasks = [];
      var currentTasks = [];
      var childMap = {};
      var parentMap = {};
      var entities;

      entities = await queryArray(path);
      entities.forEach(function (entity) {
        var task = {
          underlying: entity,
          kind: kind,
          parent: null
        };
        if (statusMap[entity.status] != null) {
          task.status = statusMap[entity.status].status;
          task.result = statusMap[entity.status].result;
        } else {
          task.status = "Stale";
          task.result = "Unknown";
        }
        proc(entity, task);
        model.connectTaskParentChild(childMap, parentMap, packageDimension, task);
        tasks.push(task);
        if (task.status !== "Current") {
          notCurrentTasks.push(task);
        } else {
          currentTasks.push(task);
        }
      });
      // Make sure we resolve the parented task's result/status.
      notCurrentTasks.forEach(function (task) {
        if (task.parent != null) {
          task.result = task.parent.result;
        }
      });
      currentTasks.forEach(function (task) {
        if (task.parent != null && task.parent.result !== task.result) {
          task.status = "Changed";
        }
      });
      return tasks;
    },
    fillRobotTasks: async function () {
      var traceMap = {};
      var tasks = [], traceTasks, reportTasks, replayTasks;
      function repTaskProc(entity, task) {
        if (traceMap[entity.input.trace] != null) {
          task.trace = traceMap[entity.input.trace].trace;
        }
        task.host = entity.host;
        task.package = entity.input.package;
      }

      traceTasks = await model.robotTasksPerKind("trace", "traces/", function (entity, task) {
        task.trace = {
          target: entity.target,
          subject: entity.input.subject
        };
        task.host = entity.host;
        task.package = entity.input.package;
        if (entity.output != null && entity.output.trace != null) {
          traceMap[entity.output.trace] = task;
        }
      });
      tasks.push(...traceTasks);

      reportTasks = await model.robotTasksPerKind("report", "reports/", repTaskProc);
      tasks.push(...reportTasks);
      replayTasks = await model.robotTasksPerKind("replay", "replays/", repTaskProc);
      tasks.push(...replayTasks);

      model.tasks = tasks;
    }
  };

  async function sourceDimensions() {
    var sourcePromises = [];
    model.dimensions.forEach(function (dimension) {
      model.dimensionsByName[dimension.name] = dimension;
      if (dimension.source != null) {
        sourcePromises.push(dimension.source());
      }
    });
    return Promise.all(sourcePromises);
  }
  await sourceDimensions();
  return model;
};
