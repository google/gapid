from . import trace_env
from . import helpers
import subprocess
import json
import time
import re
import itertools

MID_EXECUTION = 1 << 0

class submission(object):
    def __init__(self, command_buffer, commands, index):
        self.command_buffer = command_buffer
        self.commands = commands
        self.index = index

class queue_submit(object):
    def __init__(self, id, submissions):
        self.id = id
        self.submissions = submissions

class frame(object):
    def __init__(self, queue_submits):
        self.queue_submits = queue_submits
    
    def __repr__(self):
        return str(self.queue_submits)

class renderpass(object):
    def __init__(self, queue_submit, command_buffer, renderpass, renderpass_index, submission_idx, draw_calls, begin_idx, end_idx):
        self.queue_submit = queue_submit
        self.queue_submission_index = submission_idx
        self.command_buffer = command_buffer
        self.renderpass_index = renderpass_index
        self.renderpass = renderpass
        self.draw_calls = draw_calls
        self.begin_idx = begin_idx
        self.end_idx = end_idx

class  command(object):
    def __init__(self, queue_submit, command_buffer, idx, submission_idx, recording_idx):
        self.queue_submit = queue_submit
        self.queue_submission_index = submission_idx
        self.command_buffer = command_buffer
        self.idx = idx
        self.recording_idx = recording_idx

    def __repr__(self):
        return json.dumps({"queue_submit": self.queue_submit, "command_buffer": self.command_buffer, "command_idx": self.idx, "recording_command": self.recording_idx})

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
                    cbs.extend((x, command_buffer_contents[x]) for x in cmd["pSubmits"][j]["pCommandBuffers"]
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

    def find_command_indices(self, command_name, include_mec=False):
        return [i for i in range(len(self.commands)) if self.commands[i]["name"] == command_name and (include_mec or ((self.commands[i]["tracer_flags"] & MID_EXECUTION) == 0))]

    def get_submitted_commands_matching(self, queue_submit_index, filter):
        commands = []

        executed_cbs = self.executed_commands[queue_submit_index]
        for i in range(len(executed_cbs)):
            y = executed_cbs[i]
            matching = [(x, y[1][x]) for x in range(len(y[1])) if filter.match(self.commands[y[1][x]]["name"]) != None]
            if len(matching) != 0:
                commands.append((y[0], matching, i))
        return commands

    def get_trace_from_submission_pov(self):
        first_command = next(x for x in range(len(self.commands)) if (self.commands[x]["tracer_flags"] & MID_EXECUTION) == 0)
        frame_ends = [first_command];
        frame_ends.extend(self.find_command_indices("vkQueuePresentKHR"))
        
        frames = []
        queue_submits = self.find_command_indices("vkQueueSubmit")
        for i in range(len(frame_ends) -1):
            queue_submits_in_frame = [x for x in queue_submits if x >= frame_ends[i] and x <= frame_ends[i+1]]
            submits_for_frame = []
            for x in queue_submits_in_frame:
                submits = self.get_submitted_commands_matching(x, re.compile(".*"))
                submissions = [submission(x[0], x[1], x[2]) for x in submits]
                submits_for_frame.append(queue_submit(x, submissions))
            frames.append(frame(submits_for_frame))
        
        return frames

    def get_rendering_info(self):
        first_command = next(x for x in range(len(self.commands)) if (self.commands[x]["tracer_flags"] & MID_EXECUTION) == 0)
        frame_ends = [first_command]
        frame_ends.extend(self.find_command_indices("vkQueuePresentKHR"))
        
        frames = []
        queue_submits = self.find_command_indices("vkQueueSubmit")
        for i in range(len(frame_ends) -1):
            queue_submits_in_frame = [x for x in queue_submits if x >= frame_ends[i] and x <= frame_ends[i+1]]
            renderpasses_in_frame = []
            for queue_submit_idx in queue_submits_in_frame:
                draws = self.get_submitted_commands_matching(queue_submit_idx, re.compile(".*Draw.*"))
                command_buffers_with_renderpasses = self.get_submitted_commands_matching(queue_submit_idx, re.compile("vkCmdBeginRenderPass|vkCmdEndRenderPass"))
                if len(command_buffers_with_renderpasses) == 0:
                    # This submit didnt have any renderpasses in it
                    continue
                
                for x in command_buffers_with_renderpasses:
                    draws_for_command_buffer = next(y[1] for y in draws if y[2] == x[2])
                    for y in range(int(len(x[1]) / 2)): # Renderpasses come in pairs begin/end
                        draws_for_renderpass = [command(queue_submit_idx, x[0], z[0], x[2], z[1]) for z in draws_for_command_buffer if z[0] > x[1][2*y][0] and z[0] < x[1][2*y+1][0]]
                        rp = self.commands[x[1][2*y][1]]["pRenderPassBegin"]["renderPass"]
                        renderpasses_in_frame.append(
                            renderpass(
                                queue_submit_idx,
                                x[0],
                                rp,
                                y,
                                x[2],
                                draws_for_renderpass,
                                x[1][2*y][0],
                                x[1][2*y+1][0],
                            )
                        )
            frames.append(renderpasses_in_frame)
        return frames

def load_trace(env, file):
    env.current_trace = file
    helpers.status_message(env, f"Loading trace {file}")
    sp = subprocess.Popen([env.printer, file], stdout=subprocess.PIPE)
    t = trace(file, sp.stdout)
    helpers.status_message(env, f"Loaded trace {file}")
    return t
