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

var newRobotModel = async function () {

  var queryArray = async function (path) {
    return new Promise(function (resolve, reject) {
      var request = new XMLHttpRequest();
      request.open("GET", path, true)
      request.setRequestHeader("Content-type", "application/json")
      var ready = false
      request.onload = function () {
        if (request.status >= 200 && request.status <= 300) {
          resolve(JSON.parse(request.responseText));
        } else {
          reject({ status: request.status, statusText: request.statusText });
        }
      };
      request.onerror = () => reject({ status: request.status, statusText: request.statusText });
      request.send()
    });
  }

  var model = {
    dimensions: [
      {
        name: "kind",
        keyOf: function (task) {
          return task.kind;
        },
        sort: function (keyA, keyB) {
          if (keyA === keyB) {
            return 0;
          } else if (keyA === "trace") {
            return -1;
          } else if (keyB === "trace") {
            return 1;
          } else if (keyA === "report") {
            return -1;
          } else if (keyB === "report") {
            return 1;
          }
        },
        items: [{ key: "trace" }, { key: "report" }, { key: "replay" }],
        displayName: function (key) {
          return key;
        },
        keyExists: function (key) {
          return key == "trace" || key == "report" || key == "replay"
        }
      },
      {
        name: "subject",
        keyOf: function (task) {
          return task.trace.subject;
        },
        items: [],
        keysToItems: {},
        source: async function (queryString) {
          var subjects = await queryArray("subjects/" + queryString);
          for (var i = 0; i < subjects.length; ++i) {
            var subject = subjects[i];
            var item = {
              key: subject.id,
              display: subject.Information.APK.package,
              underlying: subject
            }
            this.items.push(item);
            this.keysToItems[item.key] = item;
          }
        },
        displayName: function (key) {
          return this.keysToItems[key].display;
        },
        keyExists: function (key) {
          return this.keysToItems[key] != null;
        }
      },
      {
        name: "traceTarget",
        keyOf: function (task) {
          return task.trace.target;
        },
        items: [],
        keysToItems: {},
        source: async function (queryString) {
          var devices = await queryArray("devices/");
          var workers = await queryArray("workers/");
          for (var i = 0; i < devices.length; ++i) {
            var device = devices[i];
            for (var j = 0; j < workers.length; ++j) {
              var worker = workers[j];
              if (device.id == worker.target && worker.Operation.includes(2) && this.keysToItems[device.id] == null) {
                var item = {
                  key: device.id,
                  display: device.information.Configuration.Hardware.Name,
                  underlying: device
                };
                this.items.push(item)
                this.keysToItems[item.key] = item;
              }
            }
          }
        },
        displayName: function (key) {
          return this.keysToItems[key].display;
        },
        keyExists: function (key) {
          return this.keysToItems[key] != null;
        }
      },
      {
        name: "host",
        keyOf: function (task) {
          return task.host;
        },
        items: [],
        keysToItems: {},
        source: async function (queryString) {
          var devices = await queryArray("devices/");
          var workers = await queryArray("workers/");
          for (var i = 0; i < devices.length; ++i) {
            var device = devices[i];
            for (var j = 0; j < workers.length; ++j) {
              var worker = workers[j];
              if (device.id == worker.host && this.keysToItems[device.id] == null) {
                var item = {
                  key: device.id,
                  display: device.information.Configuration.Hardware.Name,
                  underlying: device
                };
                this.items.push(item)
                this.keysToItems[item.key] = item;
              }
            }
          }
        },
        displayName: function (key) {
          return this.keysToItems[key].display;
        },
        keyExists: function (key) {
          return this.keysToItems[key] != null;
        }
      },
      {
        name: "package",
        keyOf: function (task) {
          return task.package
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
          var aO = this.packageDisplayToOrder[keyA];
          var bO = this.packageDisplayToOrder[keyB];
          if (aO != null && bO != null) {
            return aO - bO
          }
          return keyA < keyB ? -1 : keyB < keyA ? 1 : 0;
        },
        packageToTrack: {},
        source: async function (queryString) {
          var packages = await queryArray("packages/");
          var tracks = await queryArray("tracks/");
          var childMap = {};
          var rootPkgs = [];
          var result = [];
          for (var i = 0; i < packages.length; ++i) {
            var package = packages[i]
            var packageItem = {
              underlying: package,
              display: "unknown - " + package.id,
              key: package.id,
            }
            this.items.push(packageItem);
            this.keysToItems[packageItem.key] = packageItem;
            if (package.information.tag != null) {
              packageItem.display = package.information.tag;
            } else if (package.information.type == 2 && package.information.cl != null) {
              packageItem.display = package.information.cl;
            } else if (package.information.uploader) {
              packageItem.display = package.information.uploader + " - " + package.id;
            }
            var id = package.id;
            if (package["parent"] != null) {
              parentId = package["parent"];
              childMap[parentId] = id;
            } else {
              rootPkgs.push(id)
            }
          }
          for (var i = 0; i < rootPkgs.length; ++i) {
            var root = rootPkgs[i];
            var packageList = [];
            this.packageDisplayToOrder[this.keysToItems[root].display] = Object.keys(this.packageDisplayToOrder).length;
            packageList.push(root);
            var childId;
            while ((childId = childMap[root]) != null) {
              this.packageDisplayToOrder[this.keysToItems[childId].display] = Object.keys(this.packageDisplayToOrder).length;
              // want tracks stored from Root -> Head
              packageList.push(childId);
              root = childId;
            }

            var head = root;
            var foundTrack = false;
            for (var j = 0; j < tracks.length; ++j) {
              var track = tracks[j];
              if (track["head"] == head) {
                trackInfo = {
                  key: track.id,
                  display: track.name,
                  underlying: track,
                  packageList: packageList,
                  headPackage: head
                };
                this.tracks[track.name] = trackInfo;
                for (var k = 0; k < packageList.length; ++k) {
                  this.packageToTrack[packageList[k]] = trackInfo
                }
                foundTrack = true;
                break
              }
            }
            if (!foundTrack) {
              this.tracks["auto"].packageList.push(...packageList);
              this.tracks["auto"].headPackage = packageList[packageList.length - 1];
            }
          }
        },
        displayName: function (key) {
          return this.keysToItems[key].display;
        },
        keyExists: function (key) {
          return this.keysToItems[key] != null;
        }
      },
      {
        name: "track",
        packageToTrack: {},
        keyOf: function (task) {
          if (this.packageToTrack[task.package] == null) {
            return "";
          }
          return this.packageToTrack[task.package].key;
        },
        items: [],
        source: function (queryString) {
          for (var i = 0; i < model.dimensions.length; ++i) {
            if (model.dimensions[i].name == "package") {
              var packageDim = model.dimensions[i];
              for (var track in packageDim.tracks) {
                if (packageDim.tracks.hasOwnProperty(track) && packageDim.tracks[track].packageList.length > 0) {
                  this.items.push(packageDim.tracks[track]);
                }
                this.packageToTrack = model.dimensions[i].packageToTrack
              }
            }
          }
        },
        displayName: function (key) {
          for (var i = 0; i < this.items.length; ++i) {
            if (this.items[i].key == key) {
              return this.items[i].display;
            }
          }
          return "";
        },
        keyExists: function (key) {
          return this.displayName(key) != "";
        }
      }
    ],
    dimensionsByName: {},

    tasks: [],
    connectTaskParentChild: function (childListMap, parentListMap, packageDim, task) {
      var findParentPackageInList = function (idList, childId) {
        // if is the index of the id's parent
        for (var i = 1; i < idList.length; ++i) {
          var id = idList[i];
          if (childId == id) {
            return id;
          }
        }
      };
      var compareTasksSimilar = function (t1, t2) {
        return t1.trace.target == t2.trace.target && t1.trace.subject == t2.trace.subject && t1.host == t2.host;
      }

      package = task.package;
      if (parentListMap[package] == null) {
        parentListMap[package] = [task];
      } else {
        parentListMap[package].push(task);
      }

      var parentPackage = findParentPackageInList(packageDim.packageToTrack[package].packageList, package);
      if (parentPackage != null) {
        if (childList[parentPackage] == null) {
          childListMap[parentPackage] = [task];
        } else {
          childListMap[parentPackage].push(task);
        }

        var parentList = parentListMap[package]
        if (parentList != null) {
          for (var i = 0; i < parentList.length; ++i) {
            if (compareTasksSimilar(task, parentList[i])) {
              task.parent = parentList[i]
            }
          }
        }
      }

      var childList = childListMap[package]
      if (childList != null) {
        for (var i = 0; i < childList.length; ++i) {
          var child = childList[i];
          if (compareTasksSimilar(task, childList[i])) {
            if (child.parent == null) {
              child.parent = task;
            } else {
              throw "A task's parent was found twice! parent: " + package + "; child: " + child + ";"
            }
          }
        }
      }
    },
    robotTasksPerKind: async function (kind, path, proc) {
      var statusMap = {
        1: {
          status: "InProgress",
          result: "Unknown"
        },
        2: {
          status: "Current",
          result: "Succeeded"
        },
        3: {
          status: "Current",
          result: "Failed"
        },
      }
      var tasks = [];
      var notCurrentTasks = [];
      var currentTasks = [];
      var childMap = {};
      var parentMap = {};
      var packageDim;
      for (var i = 0; i < this.dimensions.length; ++i) {
        var dimension = this.dimensions[i];
        if (dimension.name == "package") {
          packageDim = dimension;
          break;
        }
      }

      var entities = await queryArray(path);
      for (var i = 0; i < entities.length; ++i) {
        var entity = entities[i];
        task = {
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
        this.connectTaskParentChild(childMap, parentMap, packageDim, task);
        // TODO: connect task parent and child...
        tasks.push(task);
        if (task.status != "Current") {
          notCurrentTasks.push(task);
        } else {
          currentTasks.push(task);
        }
      }
      for (var i = 0; i < notCurrentTasks.length; ++i) {
        var task = notCurrentTasks[i];
        if (task.parent != null) {
          task.result = task.parent.result;
        }
      }
      for (var i = 0; i < currentTasks.length; ++i) {
        var task = currentTasks[i];
        if (task.parent != null && task.parent.result != task.result) {
          task.status = "Changed";
        }
      }
      return tasks;
    },
    fillRobotTasks: async function () {
      var traceMap = {};
      var tasks = [];

      var kindDim = this.dimensionsByName["kind"]
      tasks.push(...await this.robotTasksPerKind("trace", "traces/", function (entity, task) {
        task.trace = {
          target: entity.target,
          subject: entity.input.subject,
        };
        task.host = entity.host;
        task.package = entity.input.package;
        if (entity.output != null && entity.output.trace != null) {
          traceMap[entity.output.trace] = task;
        }
      }));

      var repTaskProc = function (entity, task) {
        if (traceMap[entity.input.trace] != null) {
          task.trace = traceMap[entity.input.trace].trace;
        }
        task.host = entity.host;
        task.package = entity.input.package;
      }
      tasks.push(...await this.robotTasksPerKind("report", "reports/", repTaskProc));
      tasks.push(...await this.robotTasksPerKind("replay", "replays/", repTaskProc));

      this.tasks = tasks;
    }
  }

  for (var i = 0; i < model.dimensions.length; ++i) {
    var dimension = model.dimensions[i];
    model.dimensionsByName[dimension.name] = dimension
    if (dimension.source != null) {
      await dimension.source("");
    }
  }
  return model
}
