# Polymer Swarming web UI

This contains all the Polymer 1.X elements used in swarming.


## Building

To build the pages for deploying, run:

    make vulcanize

This combines all of the elements needed to display the page into several
"single-page" apps, like the bot-list.
These are checked into version control so that they may be easily redeployed w/o
having to rebuild the pages if there were no changes.

To vulcanize and run appengine locally, run:

    make local_deploy

To run appengine locally without vulcanizing (preferred debugging mode), run:

    make debug_local_deploy

To access the demo pages on localhost:9050, run:

    make run


## Prerequisites

You will need to install a recent version of node.js, npm, and bower. You can
either install it through `bash`, `apt` or manually download and install from
the web site https://nodejs.org/en/download/.


### npm via bash

To install in the local user (as `~/nodejs` in this example), use:

    echo prefix = ~/nodejs >> ~/.npmrc
    mkdir ~/nodejs
    cd ~/nodejs
    curl https://nodejs.org/dist/v6.10.3/node-v6.10.3-linux-x64.tar.xz | tar xJ --strip-components=1
    export PATH="$PATH:$HOME/nodejs/bin"
    npm install -g bower


### npm via apt

    sudo apt-get install npm nodejs-legacy
    # node and npm installed at this point are ancient, we need to update
    sudo npm install npm@latest -g
    # uninstall old npm
    sudo apt-get purge npm
    # make sure npm shows version 3.X or newer
    npm -v
    # you may need to add /usr/local/bin/npm to your superuser path
    # or just use /usr/local/bin/npm instead of npm below
    sudo npm cache clean -f
    sudo npm install -g n
    sudo n stable

    # should return 6.x or higher
    node -v

    sudo npm install -g bower
