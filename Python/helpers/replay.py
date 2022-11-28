from . import trace_env
from . import helpers
from . import trace
from . import image_formats
import subprocess
import os
from importlib import reload
import socket
import ijson
import errno
import json
import base64
import itertools
from time import sleep

reload(helpers)

class layer(object):
    def __init__(self, file, config, data_callback, message_callback):
        self.file = file
        self.data_callback = data_callback
        self.config = config
        self.message_callback = message_callback

class replay_options(object):
    def __init__(self, env, trace):
        self.trace = trace
        self.use_cb = False
        self.layers = []

        def default_message_callback(timestamp, level, message):
            print(f'[{timestamp}] {level}: {message}')
        self.message_callback = default_message_callback;

    def use_callback_swapchain(self):
        self.use_cb = True
        return self
    
    def add_layer(self, path, config, data_callback=None, message_callback=None):
        self.layers.append(layer(path, config, data_callback, message_callback))
    
    def set_message_callback(self, callback):
        self.message_callback = callback

def replay(env, replay_options):
    
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        sock.settimeout(12) # timeout for listening
        sock.bind(('localhost', 0))

        sock.listen(1)
        port = sock.getsockname()[1]
        command = [env.gapir, "--socket", "localhost", "--port", str(port), replay_options.trace.file]
        environ = os.environ.copy()
        if replay_options.use_cb:
            old_layer_path = ""
            if "VK_LAYER_PATH" in environ:
                old_layer_path = environ["VK_LAYER_PATH"]
            environ["VK_LAYER_PATH"] = os.path.join(env.install_dir, "externals", "vk_callback_swapchain") + ";" + old_layer_path
            old_instance_layers = ""
            if "VK_INSTANCE_LAYERS" in environ:
                old_instance_layers = environ["VK_INSTANCE_LAYERS"]
            environ["VK_INSTANCE_LAYERS"] = "CallbackSwapchain" + ";" + old_instance_layers
        success = False
        try:
            #helpers.status_message(env, f"Trying to run {' '.join(command)}")
            o = subprocess.Popen(command, env=environ, stdout=subprocess.PIPE, bufsize=1, universal_newlines=True)
            conn, info = sock.accept()
            conn.setblocking(0)
            success = True

            conn.send((json.dumps([x.file for x in replay_options.layers]) + " ").encode())
            conn.send((json.dumps({x.file: x.config for x in replay_options.layers}) + " ").encode())
            
            @ijson.coroutine
            def test():
                while True:
                    it = (yield)
                    if "LayerIndex" in it and it["Message"] == "Object":
                        if it["LayerIndex"] < len(replay_options.layers) and replay_options.layers[it["LayerIndex"]].data_callback != None:
                            replay_options.layers[it["LayerIndex"]].data_callback(float(it['Time']), it["Content"])
                        continue
                    if "LayerIndex" in it:
                        if it["LayerIndex"] < len(replay_options.layers) and replay_options.layers[it["LayerIndex"]].message_callback != None:
                            replay_options.layers[it["LayerIndex"]].message_callback(float(it['Time']), it['Message'], it["Content"])
                            continue
                    replay_options.message_callback(float(it['Time']), it['Message'], it["Content"])
            coro = ijson.items_coro(test(), '', multiple_values=True)
            
            while True:
                x = None
                try:
                    x = conn.recv(1024*1024*16, 0)
                except socket.error as e:
                    err = e.args[0]
                    if err == errno.EAGAIN or err == errno.EWOULDBLOCK:
                        sleep(0.1)
                        continue
                    else:
                        # a "real" error occurred
                        helpers.on_error(env, str(e))
                if len(x) == 0:
                    break
                coro.send(x)
            coro.close()
            success = True
        except subprocess.CalledProcessError as e:
            print(e.stdout)
            print(e.stderr)
        if not success:
            helpers.on_error(env, f"Failed to run {command}")

def screenshot_helper(env, trace, draws, max_framebuffers_per_draw = 16, before = True, after = False, silent = False):
    if not before and not after:
        return []
    def on_message(timestamp, level, message):
        if silent:
            return
        if level == "Debug" and env.ignore_debug_messages:
            return
        helpers.log_message(env, f'[{timestamp}] {level} :: {message}')
    
    def on_layer_message(timestamp, level, message):
        if silent:
            return
        if level == "Debug" and env.ignore_debug_messages:
            return
        if level == "Info" and env.ignore_layer_info_messages:
            return
        helpers.log_message(env, f'screenshot.cpp:: [{timestamp}] {level} :: {message}')
    
    draws.sort(key=lambda x: (x.queue_submit, x.queue_submission_index, x.idx))
    grouped_draws = itertools.groupby(draws, lambda x: (x.queue_submit, x.queue_submission_index))

    config = {"screenshot_locations": [], "num_images_per_draw": max_framebuffers_per_draw}
    for x in grouped_draws:
        y = itertools.groupby(x[1], lambda x: x.queue_submission_index)
        cfg = {
            "submit_index": x[0][0],
        }
        cbis = []
        for z in y:
            indices = []
            zl = list(z[1])
            if before:
                indices.extend([a.idx for a in zl])
            if after:
                indices.extend([(a.idx) + 1 for a in zl])
            indices = sorted(indices)
            cbis.append({
                    "command_buffer_index": z[0],
                    "indices": indices
                })
        cfg["command_buffer_indices"] = cbis
        config["screenshot_locations"].append(cfg)
    opts = replay_options(env, trace)
    opts.use_callback_swapchain()
    opts.set_message_callback(on_message)
    imgs = []
    def on_data(timestamp, data):
        if type(data) == str:
            imgs.append(data)
            return
        dat = base64.b64decode(data["data"])
        img = image_formats.ToNumpyArray(data["format"], data["width"], data["height"], dat)
        imgs.append(img)
    

    opts.add_layer(os.path.abspath("screenshot.cpp"), config, data_callback=on_data, message_callback=on_layer_message)
    replay(env, opts)
    return imgs

def timestamp_helper(env, trace, renderpasses, silent = False, draw_calls=False):
    def on_message(timestamp, level, message):
        if silent:
            return
        if level == "Debug" and env.ignore_debug_messages:
            return
        helpers.log_message(env, f'[{timestamp}] {level} :: {message}')
    
    def on_layer_message(timestamp, level, message):
        if silent:
            return
        if level == "Debug" and env.ignore_debug_messages:
            return
        if level == "Info" and env.ignore_layer_info_messages:
            return
        helpers.log_message(env, f'screenshot.cpp:: [{timestamp}] {level} :: {message}')
    
    renderpasses.sort(key=lambda x: (x.queue_submit, x.queue_submission_index, x.renderpass_index))
    grouped_draws = itertools.groupby(renderpasses, lambda x: (x.queue_submit, x.queue_submission_index))

    config = {"timestamp_locations": [], "include_draw_calls": draw_calls}
    for x in grouped_draws:
        y = itertools.groupby(x[1], lambda x: x.queue_submission_index)
        cfg = {
            "submit_index": x[0][0],
            "command_buffer_indices": [{
                "command_buffer_index": z[0],
                "renderpasses_indices": [a.renderpass_index for a in z[1]]
            } for z in y]
        }
        config["timestamp_locations"].append(cfg)
    opts = replay_options(env, trace)
    opts.use_callback_swapchain()
    opts.set_message_callback(on_message)
    timestamps = []
    def on_data(timestamp, data):
        nonlocal timestamps
        timestamps.extend(data)    

    opts.add_layer(os.path.abspath("timestamps.cpp"), config, data_callback=on_data, message_callback=on_layer_message)
    replay(env, opts)
    return timestamps