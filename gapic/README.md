## Setting up Eclipse to build the AGI UI

1. Create the `bazel-external` link in the AGI checkout directory:
   - Linux:  
     `ln -s $(bazel info output_base)/external bazel-external`
   - Windows Command Prompt:  
     ``for /f usebackq %F in (`bazel info output_base`) do mklink /J bazel-external "%F/external"``
2. Create and open a new Eclipse workspace.
3. Open the Eclipse Preferences and navigate to **General** -> **Workspace** -> **Linked Resources**.
4. Click **New...** to define a new path variable:
   1. Set the name to `GAPIC_PLATFORM_SRC`.
   2. Set the value to the your platform's folder inside the gapic/src/platform folder of the AGI checkout, e.g. `<gitroot>/gapic/src/platform/<OS>`
   3. Click **OK** to create the variable.
5. Click **Apply and Close** to dismiss the preferences dialog.
6. Run the bazel build to build all the generated code.
7. Select **File** -> **Import** and then **General** -> **Existing Projects into Workspace** and click **Next**.
   1. Enter the location of your AGI checkout into the root directory box.
   2. Click **Select All**. You should see a project named gapic.
   3. **IMPORTANT**: Uncheck **Copy projects into workspace**
   4. Click **Finish**.
8. Set up **Run Configurations**.
   1. Select **Java Application**.
   2. Press the button for a new configuration.
   3. Choose a name, e.g. "Default".
   4. In **Main** tab:
      - In **Project** click **Browse**. **gapic** should be selected. Click **Ok**.
      - In **Main class** click **Search**, select **Main - com.google.gapid** and click **Ok**.
   5. In **Arguments** tab:
      - In **Program arguments** give the `--gapid` flag. E.g.: `--gapid <path_to_agi>/bazel-bin/pkg`, replace `<path_to_agi>` with your AGI checkout path. Optionally add more arguments.
   6. In **JRE** tab:
      - If AGI should run with a different JRE than Eclipse, choose it in **Alternate JRE**. It might have to be added as **Installed JREs** first.
   7. In **Classpath** tab:
      - Select **User Entries** and click **Add JARs**. Select `/gapic/bazel-external/org_lwjgl_core/lwjgl-natives-<OS>.jar` for your OS and click **Ok**. Folder name and jar name may vary based on your operating system.
      - Repeat for `/gapic/bazel-external/org_lwjgl_opengl/lwjgl-opengl-natives-<OS>.jar`.
   8. Click **Apply** and **Run** to test the configuration.
