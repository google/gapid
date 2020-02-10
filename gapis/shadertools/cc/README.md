# Shadertools, libmanager

/**
 * Karolina Gawryszczak
 * 18.10.2016
 **/

Library libmanager provides 2 function:

```
	-> code_with_debug_info_t* convertGlsl(const char*, size_t, options_t*);
		1. compiles GLSL source code in ES version to spirv code (using glslang tool)
		2. makes some changes in compiled spirv code (using my class SpvManager; those changes depend on provided options).
		3. decompile changed spirv code to source code in desktop version (using spirv-cross tool)
		4. again, compiles source code in desktop version (again, using glslang tool)

	-> void deleteGlslCodeWithDebug(code_with_debug_info_t*);
```

Options:

```
	typedef struct options_t {
  		bool is_fragment_shader;
  		bool is_vertex_shader;
  		bool prefix_names;
  			/* Remaps all user declaration names to avoid names collisions between different GLSL versions
  				with default prefix or 'name_prefix' if given. */
  		const char* names_prefix;   /* optional */
  		bool add_outputs_for_inputs; 
  			/* Creates outputs for inputs with default or given 'output_prefix'. */
  		const char* output_prefix;    /* optional */
  		bool make_debuggable;
  			/* Inserts variables prints. */
  		bool check_after_changes;
  			/* Check if changed source code compiles, using glslang. */
  		bool disassemble;
  			/* Return disassemble code after all changes. */
	} options_t;
```


Usage:

Simple library usage is shown in main.cpp program.

```
./main <vertex_or_fragment_shader_filename>`
```

Tests:

- script tests/run_tests.py
  - script takes one argument 'step', which is a number.
    Variable 'width' means how many files you want to test.
    Files from (step * width) to ((step + 1) * width) will be tested.
  - Creates output files in test/shaders/ directory.
    Output file name: <test_name>.out3
