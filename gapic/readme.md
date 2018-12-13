## Setting up Eclipse to build GAPIC

1. Create the `bazel-external` link:
   1. `ln -s $(bazel info output_base)/external bazel-external`
2. Create and open a new Eclipse workspace.
3. Open the Eclipse Preferences and navigate to **General** -> **Workspace** -> **Linked Resources**.
4. Click **New...** to define a new path variable:
   1. Set the name to `GAPIC_PLATFORM_SRC`.
   2. Set the value to the your platform's folder inside the gapic/src/platform folder of the gapid checkout.
     e.g. `<gitroot>/gapic/src/platform/linux`
   3. Click **OK** to create the variable.
5. Click **OK** to dismiss the preferences dialog.
6. Run the bazel build to build all the generated code.
7. Select **File** -> **Import** and then **General** -> **Existing Project into Workspace** and click **Next**.
   1. Enter the location of your gapid checkout into the root directory box.
   2. Click **Select All**. You should see a project named gapic.
   3. **IMPORTANT**: Uncheck **Copy projects into workspace**
   4. Click "Finish".
