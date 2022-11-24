from . import gapid_env
from . import helpers
import subprocess
import json
import time

MID_EXECUTION = 1 << 0

class trace(object):
    def __init__(self, file, stream):
        self.start_time = time.localtime()
        data = json.load(stream)
        self.commands = []
        self.annotations = {}
        self.observations = {}
        self.executed_commands = {}
        self.file = file

        i = 0
        for x in data:
            if x["name"] == "Annotation":
                if i in self.annotations:
                    self.annotations[i].append(x)
                else:
                    self.annotations[i] = [x]
            elif x["name"] == "MemoryObservation":
                if i in self.observations:
                    self.observations[i].append(x)
                else:
                    self.observations[i] = [x]
            else:
                self.commands.append(x)
                i = i + 1
        self.end_time = time.localtime()
        self.compute_command_buffer_contents()

    def compute_command_buffer_contents(self):
        command_buffer_contents = {}

        for i in range(len(self.commands)):
            cmd = self.commands[i]
            if cmd["name"] == "vkBeginCommandBuffer":
                command_buffer_contents[cmd["commandBuffer"]] = [i]
                continue
            if cmd["name"].startswith('vkCmd'):
                if not cmd["commandBuffer"] in command_buffer_contents:
                    command_buffer_contents[cmd["commandBuffer"]] = [i]
                command_buffer_contents[cmd["commandBuffer"]].append(i)
                continue
            if cmd["name"] == "vkEndCommandBuffer":
                if not cmd["commandBuffer"] in command_buffer_contents:
                    command_buffer_contents[cmd["commandBuffer"]] = [i]
                command_buffer_contents[cmd["commandBuffer"]].append(i)
            if cmd["name"] == "vkQueueSubmit":
                cbs = []
                for j in range(cmd["submitCount"]):
                    cbs.extend(command_buffer_contents[x] for x in cmd["pSubmits"][j]["pCommandBuffers"]
                        if x in command_buffer_contents)
                self.executed_commands[i] = cbs

    def get_stats(self):
        stats = {}
        stats["Total Commands"] = len(self.commands)
        stats["Frame Count"] = len([x for x in self.commands if x["name"] == "vkQueuePresentKHR"])
        stats["Loading time"] = f"{time.mktime(self.end_time) - time.mktime(self.start_time)}s"
        
        stats["MEC Commands"] = len([x for x in self.commands if x["tracer_flags"] & MID_EXECUTION])
        stats["Application Commands"] = len(self.commands) - stats["MEC Commands"]
        if (stats["Frame Count"] > 0):
            stats["Average Commands Per Frame"] = (len(self.commands) - stats["MEC Commands"]) / stats["Frame Count"]
        if stats["MEC Commands"] == 0:
            del stats["MEC Commands"]
        return stats

def load_trace(env, file):
    env.current_trace = file
    helpers.status_message(env, f"Loading trace {file}")
    sp = subprocess.Popen([env.printer, file], stdout=subprocess.PIPE)
    t = trace(file, sp.stdout)
    helpers.status_message(env, f"Loaded trace {file}")
    return t
