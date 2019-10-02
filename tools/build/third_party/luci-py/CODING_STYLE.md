# Coding style

For Markdown, use the [Google
style](https://github.com/google/styleguide/blob/gh-pages/docguide/style.md).


## General style

*   ALL_CAPS_GLOBALS at top
*   function_name
*   ClassName
*   variable_name
*   +2 space indent, a legacy from Google
*   2 empty lines between file level symbols
*   1 empty line between class members


## Line breaks

Keep line width under 80 chars. Break a line on a first syntactic structure,
prefer hanging indent of 4 spaces over vertical alignment. Prefer parentheses
for line joining over `\`. For JSON-like nested dicts and lists it's acceptable
to use 2 space indent.

Yes:

    str = (
        'Lorem ipsum dolor sit amet, consectetur adipiscing elit. Nullam diam met, '
        'tristique ac placerat sit amet: %s, %s' % (a, b))

    func_call(
        param1=...,
        param2=...,
        param3=...)

    like_json = {
      'key': {
        'a': {
          'b': 123,
          'c': 123,
          'd': 123,
        },
      },
    }

No:

    str = \
        'Lorem ipsum dolor sit amet, consectetur adipiscing elit. Nullam diam met, '

    str = ('Lorem ipsum dolor sit amet, consectetur adipiscing elit. Nullam diam '
           'met, ....')

    func_call(param1=...,
              param2=...,
              param3=...)

    fun_call(param1=..., param2=...,
        param3=...)


## Import statements

  * import modules, not symbols.
  * `from X import Y` statements, then `import X` statements within a single block


Separations:

    <import block of stdlib>
    
    <import block of import sys.path hack, e.g. test_env. Needed for now>
    
    <import block of appengine lib>
    
    <import block of appengine third_party lib, e.g. webapp2, or in third_party/>
    
    <import block of components and other application external libs>
    
    <import block of application libs>
