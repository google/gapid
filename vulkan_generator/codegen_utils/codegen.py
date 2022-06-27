# Copyright (C) 2022 Google Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""This module contains the utility functions that make generating c++ code nicer"""

from typing import Dict
from typing import List
from typing import Optional

import textwrap

def indent_characters(depth : int = 1) -> str: # return the correct indentation for a given indent depth
    indent_symbols = "  "

    ret = ""
    for _ in range(depth):
        ret = ret +indent_symbols

    return ret

def indent_code(code: str, depth : int = 1) -> str: # return the correct indentation for a given indent depth
    if depth <= 0:
        return code

    return indent_code(textwrap.indent(code, indent_characters(1)), depth -1)

def generated_license_header() -> str:
    return textwrap.dedent("""
        // Copyright (C) 2022 Google Inc.
        //
        // This file is generated code. It was created by the AGI code generator.
        //
        // Licensed under the Apache License, Version 2.0 (the "License");
        // you may not use this file except in compliance with the License.
        // You may obtain a copy of the License at
        //
        //      http://www.apache.org/licenses/LICENSE-2.0
        //
        // Unless required by applicable law or agreed to in writing, software
        // distributed under the License is distributed on an "AS IS" BASIS,
        // WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
        // See the License for the specific language governing permissions and
        // limitations under the License.
        
        """)

def comment_code(code: str, comment: str) -> str:
    ''' Adds a comment to your code. Makes a basic effort to be pretty. '''
    # if there's no code or the code is just whitespace, we'll just return the comment
    if len(code) == 0 or code.isspace():
        return "/* " +comment + " */\n"

    # one-line code snippets get commented "in line"
    if code.count("\n") == 0 or code.count("\n") == 1 and code[-2:-1] == "\n":
        return code + " /* " +comment + " */\n"

    # multi-line code snippets get a comment above the code
    return "/* " +comment + " */\n" +code

def create_function_signature(name: str, return_type: str, arguments: Optional[Dict[str, str]]) -> str:
    ''' Create the signature for a function, ie: int fibonacci(int n) '''
    ret = return_type + " " + name + "("

    for arg_name, arg_type in arguments.items():
        ret = ret + arg_type + " " + arg_name + ", "

    # Strip the trailing comma from the last argument
    if arguments:
        ret = ret[0:-2]

    ret = ret + ")"

    return ret

def create_function_declaration(name: str,
                                return_type: str = "void",
                                arguments: Optional[Dict[str, str]] = None) -> str:
    ''' Create a declaration for a function, ie: int fibonacci(int n); '''
    return create_function_signature(name, return_type, arguments,) + ";"

def create_function_definition(name: str,
                               return_type: str = "void",
                               arguments: Optional[Dict[str, str]] = None,
                               code: str = "") -> str:
    ''' Create the signature for a function, ie: int fibonacci(int n) { /* code here */ }'''
    return create_function_signature(name, return_type, arguments) + " {\n" + indent_code(code) +"\n}"

def create_exception_declaration(name: str, base_class: str = "std::exception") -> str:
    return "class " + name + " : public " + base_class +" {};"

def create_class_definition(name: str,
                            public_inheritance: Optional[List[str]] = None,
                            protected_inheritance: Optional[List[str]] = None,
                            private_inheritance: Optional[List[str]] = None,
                            public_members: Optional[List[str]] = None,
                            protected_members: Optional[List[str]] = None,
                            private_members: Optional[List[str]] = None,
                            public_functions: Optional[List[str]] = None,
                            protected_functions: Optional[List[str]] = None,
                            private_functions: Optional[List[str]] = None
                            ) -> str:

    ''' Create the full definition for a class including inheritance, all members, etc'''

    def inheritance_helper(inheritance_type : str, inheritance : List[str]) -> str:
        if not inheritance:
            return ""
        inheritance_string = inheritance_type + " "
        for base in inheritance:
            inheritance_string = inheritance_string + base + ", "
        return inheritance_string

    def member_helper(member_type : str, members : List[str]) -> str:
        if not members:
            return ""
        members_string = indent_characters() + member_type +":\n"
        for member in members:
            members_string = members_string + indent_characters(2) + member + "\n"
        return members_string + "\n"

    ret = "class " + name

    inheritance = public_inheritance or protected_inheritance or private_inheritance
    if inheritance:
        ret = ret +" : "

    ret = ret +inheritance_helper("public", public_inheritance)
    ret = ret +inheritance_helper("protected", protected_inheritance)
    ret = ret +inheritance_helper("private", private_inheritance)

    # Strip the trailing comma from the last inherited base class
    if inheritance:
        ret = ret[0:-2]

    ret = ret + "\n{"

    functions = public_functions or protected_functions or private_functions
    members = public_members or protected_members or private_members
    contents = functions or members
    if contents:
        ret = ret + "\n"

    ret = ret +member_helper("public", public_functions)
    ret = ret +member_helper("protected", protected_functions)
    ret = ret +member_helper("private", private_functions)

    ret = ret +member_helper("public", public_members)
    ret = ret +member_helper("protected", protected_members)
    ret = ret +member_helper("private", private_members)

    ret = ret + "};\n"

    return ret



