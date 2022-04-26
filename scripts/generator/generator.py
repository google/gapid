class generator():
    def __init__(self, file):
        self.depth = ""
        self.file = file

    def print(self, str):
        print(f"{self.depth}{str}", file=self.file)

    def print_scoping(self, str):
        print(f"{self.depth[:-1]}{str}", file=self.file)

    def line(self):
        print(f"", file=self.file)

    def enter_scope(self, str):
        print(f"{self.depth}{str}", file=self.file)
        self.depth += "  "

    def leave_scope(self, str):
        self.depth = self.depth[:-2]
        print(f"{self.depth}{str}", file=self.file)

    def post_leave_scope(self, str):
        print(f"{self.depth}{str}", file=self.file)
        self.depth = self.depth[:-2]

    def leave_enter_scope(self, str):
        print(f"{self.depth[:-2]}{str}", file=self.file)

    def get_depth(self):
        return len(self.depth)
