## Setting up Eclipse to build the AGI UI

1. Create the `bazel-external` link:
   1. `ln -s $(bazel info output_base)/external bazel-external`
2. Create and open a new Eclipse workspace.
3. Open the Eclipse Preferences and navigate to **General** -> **Workspace** -> **Linked Resources**.
4. Click **New...** to define a new path variable:
   1. Set the name to `GAPIC_PLATFORM_SRC`.
   2. Set the value to the your platform's folder inside the gapic/src/platform folder of the AGI checkout.
     e.g. `<gitroot>/gapic/src/platform/linux`
   3. Click **OK** to create the variable.
5. Click **OK** to dismiss the preferences dialog.
6. Run the bazel build to build all the generated code.
7. Select **File** -> **Import** and then **General** -> **Existing Project into Workspace** and click **Next**.
   1. Enter the location of your AGI checkout into the root directory box.
   2. Click **Select All**. You should see a project named gapic.
   3. **IMPORTANT**: Uncheck **Copy projects into workspace**
   4. Click "Finish".
8. Set up Run Configurations.
   1. In Arguments tab: give the `--gapid` flag. E.g.: `--gapid <path_to_agi>/bazel-bin/pkg`, replace `<path_to_agi>` with your local project path.
   2. In Dependencies tab: add the lwjgl native jars. Find and add `/gapic/bazel-external/org_lwjgl_core_natives_linux/jar/lwjgl-3.2.0.jar`, `/gapic/bazel-external/org_lwjgl_opengl_natives_linux/jar/lwjgl-opengl-3.2.0.jar`. Folder name and jar name may vary based on your operating system.
