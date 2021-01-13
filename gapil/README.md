# Graphics API Language

The Graphics API Language is used to describe in detail the interface and required behaviour of a graphics API.
From these API files, much of GAPII, GAPIR and GAPIS is generated.

## Data types

### Builtin types

Typename | Description
-------- | -----------
`void`   | used to denote no return value, or a pointer to untyped memory (void\*)
`string` | a sequence of characters
`bool`   | architecture dependent sized boolean value
`char`   | architecture dependent sized character value
`int`    | architecture dependent sized signed integer
`uint`   | architecture dependent sized unsigned integer
`size`   | architecture dependent sized integer used for sizes
`s8`     | 8-bit signed integer
`u8`     | 8-bit unsigned integer
`s16`    | 16-bit signed integer
`u16`    | 16-bit unsigned integer
`s32`    | 32-bit signed integer
`u32`    | 32-bit unsigned integer
`s64`    | 64-bit signed integer
`u64`    | 64-bit unsigned integer
`f32`    | 32-bit floating-pointer number
`f64`    | 64-bit floating-pointer number

### Named types

A number of different types can be defined within the API file.

Each of the following types can be preceded with any number of [annotations](#annotations).

#### Class

Class types define objects that can hold fields and methods.

Class instances can be used as value types or be shared using references.

Classes are declared using the syntax:

```cpp
class name {
  field_type_1 field_name_1
  ...
  field_type_N field_name_N
}
```

#### Enum

Enum types define a collection of name-integer pairs within a named scope.

Enums may contain multiple names with the same value.

Enums are declared using the syntax:

```cpp
enum name {
  name_1 = value_1
  ...
  name_N = value_N
}
```

#### Bitfield

Bitfield types define a collection of name-integer pairs within a named scope.

Bitfields have special operators for performing bitwise tests.

Bitfields are declared using the syntax:

```cpp
bitfield name {
  name_1 = value_1
  ...
  name_N = value_N
}
```

#### Alias

Aliases create a new type derived from an existing type. Templates may chose to
emit a new type for aliases in the target code-generated language.

Aliased types may have annotations.

Aliases are declared using the syntax:

```cpp
type base_type alias_name
```

### Map
Maps are a set of key-value pairs.

```cpp
map!(u32, u32) uint_to_uint_map
```

The key must be a comparable type (or string).
Maps have the following operations

```
x := uint_to_uint_map[2]; // Access
uint_to_uint_map[2] = x; // Insert
for index, key, value in uint_to_uint_map {} // Iteration
```

The Access operation returns the value in question, or a default
value if it did not exist in the map.

The insert operations inserts the given value at the given
location in the map.

Iterating a map will iterate over the indices/keys/values in the map.

### DenseMap

DenseMaps are a specialization of Map

```cpp
dense_map!(u32, u32) uint_to_uint_map
```

They have the same semantics as Map, although their underlying
storage is different. They are optimized for a set of small sequential
keys. They take an amount of storage on the order of the largest
key inserted into the map. They must be keyed on an unsigned integer
value.

## Commands

Commands are declared with a C-style function signature, prefixed with `cmd`.

```cpp
cmd GLboolean glIsEnabled(GLenum capability) {
  // command body
}
```

Each command may be prefixed with one or more annotations.

```cpp
@frame_end
cmd EGLBoolean eglSwapBuffers(EGLDisplay display, void* surface) {
  // ...
}
```

## Global fields

At the top-level scope of an API file, fields can be declared with optional
initializers.

```cpp
f32 aGlobalF32      // Default-initialized to 0
u8  aGlobalU8  = 4  // Initialized to 4
```

These fields are initialized before any command is executed

## Statements

### Builtin statements

#### read(T[] src)

`read` is used to signal that the specified memory slice is read by the command.

For GAPII the read statement will instruct the interceptor to observe src before
invoking the real driver function.

For GAPIR the read statement will instruct the replay system to fill the
corresponding memory range with the observed memory before invoking the call.

The read statement is an implicit pre-fence statement.

#### write(T[] dst)

`write` is used to signal that the specified memory slice is written by the
command.

For GAPII the write statement will instruct the interceptor to observe dst after
invoking the real driver function.

For GAPIR the write statement will instruct the replay system to perform any
output-value remappings after invoking the call.

The write statement is an implicit post-fence statement.

#### copy(T[] dst, T[] src)

`copy` will copy min(len(src), len(dst)) elements from src to dst, and perform
the corresponding read() and write() logic on src and dst, respectively.

The copy statement is an implicit pre and post-fence statement.

#### fence

Statements need to be split into those that are executed before the call to the
function, and those that need to be executed after the call to the function.

For example consider a function that performs a read-modify-write operation on
the provided buffer:

```cpp
cmd void RMW(u8* buffer, u32 size) {
  read(buffer[0:size])
  // implicit fence
  write(buffer[0:size])
}
```

In this example, the GAPII interceptor needs to perform a read observation on
buffer before calling the driver's `RMW` function, and a write observation after
calling the driver's `RMW` function.

For commands that do not use the explicit `fence` statement, an implicit fence
will be inserted between all pre-fence statements and all post-fence statements
in the command. The logic to find this insertion point is currently pretty
simple, and may fail with a compilation error if a single insertion point cannot
be found. In these cases you should explicitly add a `fence` statement to the
command.

##### print(fmt, ...)

`print` can be used for debugging and will issue printf like logging calls to
print messages and values to the log.

## Annotations

An annotation is a custom attribute that can be placed on named types and
commands.

Annotations can take the form:

```
@name

@name(arg1, arg2, arg3)
```

Annotations are for use by the templates when using the apic [`template`]
(../cmd/apic/template.go) or [`validate`]
(../cmd/apic/validate.go) commands, but otherwise have no effect.
