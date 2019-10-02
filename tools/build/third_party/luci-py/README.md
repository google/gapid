# LUCI

LUCI is an ensemble of services that work together to run large scale CI
(continuous integration) infrastructure to get results in *O(1)*. It is used by
the [Chromium project](http://www.chromium.org).

See [appengine/](appengine) for the services provided.


## Overview

A complete CI toolkit:

*   Strict separation between *task distribution* and *file distribution*.
    *   Swarming distributes tasks across an army (thousands) of Swarming bots.
    *   Isolate server distributes 10s of thousands of files _per task_.
*   Federated authentication and ACL.
*   git based configuration data propagation.
*   Automatic Google Compute Engine VM recycling.
*   Windows, OSX and Linux (Ubuntu), used with Android, iOS, ChromeOS.
*   Written in python 2.7 along [Go](https://go.chromium.org/luci) tools.
*   Used by the Chromium infrastructure and other teams.


## Installing

1.  Install [Google AppEngine
    SDK](https://cloud.google.com/appengine/downloads).
2.  git clone https://chromium.googlesource.com/infra/luci/luci-py


## Code layout

*   [/appengine/...](appengine) contains
    [AppEngine](https://cloud.google.com/appengine/docs/python/) server
    code.
*   [/client/...](client) contains all client code.
*   [/infra/config/...](infra/config) contains metadata to run the commit queue.


## Versioning

*   Branch `master` constains the latest code.
*   Branch `stable` contains the stable code.


## Contributing

*   Sign the [Google CLA](https://cla.developers.google.com/clas).
*   See the [coding style](CODING_STYLE.md).
*   Make sure your `user.email` and `user.name` are configured in `git config`.

Run the following to setup the code review tool and create your first review:

    git clone https://chromium.googlesource.com/chromium/tools/depot_tools.git $HOME/src/depot_tools
    export PATH="$PATH:$HOME/src/depot_tools"
    cd luci-py
    git checkout -b work origin/master

    # hack hack

    git commit -a -m "This is awesome\nR=joe@example.com"
    # This will ask for your Google Account credentials.
    git cl upload -s
    # Wait for LGTM over email.
    # Check the commit queue box in codereview website.
    # Wait for the change to be tested and landed automatically.

Use `git cl help` and `git cl help <cmd>` for more details.


## License

This project is licensed under Apache v2.0 license. See LICENSE for details.
