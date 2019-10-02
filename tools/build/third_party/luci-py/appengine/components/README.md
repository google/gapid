components/
-----------

Modules and tools in this directory are shared between all AppEngine servers in
this repository.

Contents of this directory:

  - [components/](components) contains reusable code to be used from AppEngine
    application.  `components` directory must be symlinked to an app directory.
    Must contain only code that needs to be deployed to GAE The only exception
    is unit tests with regexp '.+_test\.py' or 'test_.+', which are acceptable.
  - [test_support/](test_support) reusable code that can only be used from
    tests, not from AppEngine. This code must not be deployed to AppEngine.
  - [tool_support/](tool_support) reusable code that can only be used from tests
    *and* tools, not from AppEngine. This code must not be deployed to
    AppEngine.
  - [tests/](tests) contains *smoke tests* that depend on
    `//appengine/components/components/` thus can't be located there.
  - [tools/](tools) utilities to manage applications on AppEngine. This code
    must not be deployed to AppEngine.

