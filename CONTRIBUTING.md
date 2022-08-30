# How to contribute

We'd love to accept your patches and contributions to this project. There are
just a few small guidelines you need to follow.

## Check out the source

Do the following to check out the Android GPU Inspector source code on GitHub:

1.  Sign up for GitHub at https://github.com/ if you don’t already have an account.
1.  (Optional) Set up an SSH key to [connect to your account using SSH].
1.  (Optional) [Add a GPG signing key to your account].
1.  Go to the project landing page at https://github.com/google/agi.
1.  [Fork the repository]. This creates a copy of the repository in your account.
1.  Create a work folder on your workstation. The rest of this document assumes `~/work`, adjust as needed.
1.  On _your_ AGI project page, [clone your copy of the repository] and add it to your local work folder:
    ```
    cd ~/work
    git clone <clone-url>
    cd agi
    ```
1.  Add the Google remote repository to your local repository:
    ```
    git remote add goog git@github.com:google/agi.git
    git fetch goog
    ```

## (Optional) Configure git

Use the following commands to configure git for AGI development:
```
# Assume the remote branch has the same name as your local branch to make pushing changes easier
git config push.default current
# Default to pushing to your fork (assuming the above directions)
git config remote.pushDefault origin
# Make git clean up all the remote tags it creates when you delete remote branches
git config fetch.prune true
git config user.name <your-name> # Add --global to make this a global setting
git config user.email <you@your-email.com> # Can also be a global setting
# If you added a GPG signing key, run the following commands:
git config user.signingkey <keyid>
git config commit.gpgsign true
```

## Build Android GPU Inspector for the first time

Follow the [build instructions] in the AGI repository.

## Sign the Contributor License Agreement

Contributions to this project must be accompanied by a Contributor License
Agreement. You (or your employer) retain the copyright to your contribution,
this simply gives us permission to use and redistribute your contributions as
part of the project. Head over to <https://cla.developers.google.com/> to see
your current agreements on file or to sign a new one.

You generally only need to submit a CLA once, so if you've already submitted one (even if it was for a different project), you probably don't need to do it again.

## Open a pull request (PR)

Do the following to contribute to the AGI project:

1.  Prepare your changes on a dedicated branch in your local repository:
    ```
    git checkout -b <my-branch>
    ```
1.  Make changes, commit the changes, and squash them into a single commit.
1.  Use the presubmit script to check code formatting and other things:
    ```
    # Install clang-format 6.0
    sudo apt-get install -y clang-format-6.0
    export CLANG_FORMAT=clang-format-6.0
    # Run the script
    ./kokoro/presubmit/presubmit.sh
    ```
1.  Fix potential issues, commit the fix, squash into a single commit again.
1.  Re-run presubmit script until it passes without warnings.
1.  Check that the tests pass:
    ```
    bazel test tests
    ```
    To skip test that requires a GPU, run:
    
    ```
    bazel test tests --test_tag_filters=needs_gpu
    ```
    
1.  Push to your GitHub repo:
    ```
    git push
    ```
1.  Visit https://github.com/google/agi to see a pop-up dialog inviting you to open a PR; click on the dialog to create a PR. See [Creating a pull request from a fork] for more information.
1.  All submissions, including submissions by project members, require review. You can request specific reviewers for your PR or leave the reviewers section blank. An AGI team member will review the request.

Consult [GitHub Help] for more information on using pull requests.

[connect to your account using SSH]: https://help.github.com/en/articles/connecting-to-github-with-ssh
[Add a GPG signing key to your account]: https://help.github.com/en/articles/adding-a-new-gpg-key-to-your-github-account
[Fork the repository]: https://help.github.com/en/articles/fork-a-repo
[clone your copy of the repository]: https://help.github.com/en/articles/cloning-a-repository
[build instructions]: https://github.com/google/agi/blob/master/BUILDING.md
[Creating a pull request from a fork]: https://help.github.com/en/articles/creating-a-pull-request-from-a-fork
[GitHub Help]: https://help.github.com/articles/about-pull-requests/
