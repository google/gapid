import os

from . import helpers

class gapid_env(object):
    def __init__(self, gapid_install_dir):
        self.install_dir = gapid_install_dir
        self.gapir = os.path.join(self.install_dir, "gapir.exe")
        self.printer = os.path.join(self.install_dir, "printer.exe")
        self.current_trace = None
        self.ignore_debug_messages = False
        self.ignore_layer_info_messages = False

        if not os.path.isfile(self.printer):
            helpers.on_error(self, "Cannot find printer")
        if not os.path.isfile(self.gapir):
            helpers.on_error(self, "Cannot find gapir")
        
        helpers.status_message(self, "GAPID environment initialized")
        helpers.status_message(self, f"    Gapir: {self.gapir}")
        helpers.status_message(self, f"    Printer: {self.printer}")

        