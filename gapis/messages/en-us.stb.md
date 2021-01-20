
# ERR_UNKNOWN_DEVICE

Unknown device.

# ERR_FRAMEBUFFER_UNAVAILABLE

The framebuffer is not available at this point in the trace.

# ERR_DEPTH_BUFFER_NOT_SUPPORTED

Reading the depth buffer is not supported for GLES 2.0. Use desktop replay instead.

# ERR_NO_TEXTURE_DATA

No texture data has been associated with texture {{texture_name}} at this point in the trace.

# ERR_STATE_UNAVAILABLE

The state is not available at this point in the trace.

# ERR_VALUE_OUT_OF_BOUNDS

The value {{value}} for {{variable}} is out of bounds. Acceptable range: \[{{min}}-{{max}}\].

# ERR_SLICE_OUT_OF_BOUNDS

The slice {{from_value}}:{{to_value}} for {{from_variable}}:{{to_variable}} is out of bounds. Acceptable range: \[{{min}}-{{max}}\].

# ERR_INVALID_VALUE

Invalid value {{value:s64}}.

# ERR_INVALID_OBJECT_NAME

Invalid value {{value:s64}}. Object with this name does not exist.

# ERR_INVALID_VALUE_CHECK_EQ

Invalid value {{value:s64}}. It must be {{constraint:s64}}.

# ERR_INVALID_VALUE_CHECK_NE

Invalid value {{value:s64}}. It must not be {{constraint:s64}}.

# ERR_INVALID_VALUE_CHECK_GE

Invalid value {{value:s64}}. It must be greater than or equal to {{constraint:s64}}.

# ERR_INVALID_VALUE_CHECK_GT

Invalid value {{value:s64}}. It must be greater than {{constraint:s64}}.

# ERR_INVALID_VALUE_CHECK_LE

Invalid value {{value:s64}}. It must be less than or equal to {{constraint:s64}}.

# ERR_INVALID_VALUE_CHECK_LT

Invalid value {{value:s64}}. It must be less than {{constraint:s64}}.

# ERR_INVALID_VALUE_CHECK_LOCATION_LT

Invalid value {{value:s64}}. Location must be less than {{constraint:s64}}.

# ERR_INVALID_VALUE_CHECK_COUNT_GE

Invalid value {{value:s64}}. Count must be greater than or equal to {{constraint:s64}}.

# ERR_INVALID_VALUE_CHECK_SIZE_GE

Invalid value {{value:s64}}. Size must be greater than or equal to {{constraint:s64}}.

# ERR_INVALID_ENUM

Invalid enum {{value:u32}}.

# ERR_INVALID_OPERATION

Invalid operation.

# ERR_INVALID_OPERATION_DEFAULT_FRAMEBUFFER_BOUND

Invalid operation. Default framebuffer object is bound.

# ERR_INVALID_OPERATION_DEFAULT_VERTEX_ARRAY_BOUND

Invalid operation. Default vertex array object is bound.

# ERR_INVALID_OPERATION_OBJECT_DOES_NOT_EXIST

Invalid operation. Object {{id:u64}} does not exist.

# ERR_CONTEXT_DOES_NOT_EXIST

No context with id {{id:u64}} exists.

# ERR_NO_CONTEXT_BOUND

No context bound in thread: {{thread:u64}}

# ERR_CONTEXT_BOUND

Can not bind context with id {{id:u64}} since it is already bound on different thread.

# ERR_FIELD_DOES_NOT_EXIST

Value of type {{ty}} does not have field {{field}}.

# ERR_PARAMETER_DOES_NOT_EXIST

Command of type {{ty}} does not have parameter {{field}}.

# ERR_RESULT_DOES_NOT_EXIST

Command of type {{ty}} does not have a result value.

# ERR_MAP_KEY_DOES_NOT_EXIST

Map does not contain entry with key {{key}}.

# ERR_MESH_NOT_AVAILABLE

Mesh not available.

# ERR_MESH_HAS_NO_VERTICES

Mesh has no vertices.

# ERR_NO_PROGRAM_BOUND

No program bound.

# ERR_PROGRAM_NOT_LINKED

The program was not linked.

# ERR_INCORRECT_MAP_KEY_TYPE

Incorrect map key type. Got type {{got}}, expected type {{expected}}.

# ERR_TYPE_NOT_ARRAY_INDEXABLE

Value of type {{ty}} is not array-indexable.

# ERR_TYPE_NOT_MAP_INDEXABLE

Value of type {{ty}} is not a map-indexable.

# ERR_TYPE_NOT_SLICEABLE

Value of type {{ty}} is not sliceable.

# ERR_NIL_POINTER_DEREFERENCE

The object was nil.

# ERR_UNSUPPORTED_CONVERSION

The object cannot be cast to the requested type.

# ERR_CRITICAL

Internal error: {{err}}

# ERR_TRACE_ASSERT

Internal error in trace assert: {{reason}}

# ERR_MESSAGE

{{error}}

# ERR_INTERNAL_ERROR

Internal error: {{error}}

# ERR_REPLAY_DRIVER

Error during replay: {{replayError}}

# ERR_WRONG_CONTEXT_VERSION

Required context of at least {{reqmajor:u32}}.{{reqminor:u32}}, got {{major:u32}}.{{minor:u32}}.

# WARN_UNKNOWN_CONTEXT

The context {{id:u64}} was created before tracing begun. Context state is not known.

# ERR_VALUE_NEG

{{valname}} was negative ({{value:s64}}).

# ERR_VALUE_GE_LIMIT

{{valname}} was greater than or equal to {{limitname}}. {{valname}}: {{val:s64}}, {{limitname}}: {{limit:s64}}

# ERR_NOT_A_DRAW_CALL

The requested command range does not contain any draw calls.

# TAG_COMMAND_NAME

{{command}}

# ERR_PATH_WITHOUT_CAPTURE

The request path does not contain the required capture identifier.

# NO_NEW_BUILDS_AVAILABLE

There are no new builds available.

# ERR_INVALID_MEMORY_POOL

Pool {{pool}} not found.

# ERR_FILE_CANNOT_BE_READ

The file cannot be read.

# ERR_FILE_TOO_NEW

The file was created by a more recent version of AGI and cannot be read.

# ERR_FILE_TOO_OLD

The file was created by an old version of AGI and cannot be read.

# ERR_FILE_OPEN_GL

GLES traces are not supported in AGI. Please use GAPID instead.

# REPLAY_COMPATIBILITY_COMPATIBLE

Device can replay the capture.

# REPLAY_COMPATIBILITY_INCOMPATIBLE_OS

Device OS ({{device_os}}) is different from the one of the capture device ({{capture_os}}).

# REPLAY_COMPATIBILITY_INCOMPATIBLE_ARCHITECTURE

Device does not support {{trace_arch}} ABI architecture.

# REPLAY_COMPATIBILITY_INCOMPATIBLE_GPU

Device GPU ({{device_gpu}}) is different from the one of the capture device ({{capture_gpu}}).

# REPLAY_COMPATIBILITY_INCOMPATIBLE_DRIVER_VERSION

Device GPU driver version ({{device_driver_version}}) is different from the one of the capture device ({{capture_driver_version}}).

# REPLAY_COMPATIBILITY_INCOMPATIBLE_API_VERSION

Device graphics API version ({{device_api_version}}) is different from the one of the capture device ({{capture_api_version}}).

# REPLAY_COMPATIBILITY_INCOMPATIBLE_API

Device does not support the graphics API of the trace.

# REPLAY_COMPATIBILITY_MISSING_API

Trace file is incomplete/truncated: it does not contain any traced commands.
