from . import gapid_env
from . import helpers
from . import trace
import subprocess
import os
from importlib import reload
import socket
import ijson
import errno
import json
from time import sleep

reload(helpers)

class layer(object):
    def __init__(self, file, config, callback):
        self.file = file
        self.callback = callback
        self.config = config


class replay_options(object):
    def __init__(self, trace):
        self.trace = trace
        self.use_cb = False
        self.layers = []

    def use_callback_swapchain(self):
        self.use_cb = True
        return self
    
    def add_layer(self, path, config, callback):
        self.layers.append(layer(path, config, callback))


def replay(env, replay_options):
    port = 55554
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.bind(('localhost', port))
        sock.listen(1)
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
            helpers.status_message(env, f"Trying to run {' '.join(command)}")
            o = subprocess.Popen(command, env=environ, stdout=subprocess.PIPE, bufsize=1, universal_newlines=True)
            conn, info = sock.accept()
            conn.setblocking(0)
            success = True

            conn.send(json.dumps([x.file for x in replay_options.layers]).encode())
            conn.send(json.dumps({x.file: x.config for x in replay_options.layers}).encode())
            
            @ijson.coroutine
            def test():
                while True:
                    it = (yield)
                    print(f"{it}")
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