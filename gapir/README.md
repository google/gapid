# Graphics API Replay (GAPIR)

GAPIR is a stack-based virtual machine that can execute programs formed from a
very small instruction set.

## Evaluation of existing VMs

Before embarking on building a new virtual machine from scratch, we evaluated
our needs, and compared it to a number of existing, lightweight, open-source VMs
([Lua], [Parrot], [Neko], etc).

We opted for building a custom VM because:

* Our [required instruction set](#opcodes) was significantly
  smaller than those provided by other VMs. We have no need for functions or
  any type of control flow, and by reducing the instruction set to only what
  we absolutely require, we’ve avoided unnecessary complexity in testing, and
  generation of the command stream.
* We have no need for standard libraries (math functions, io functions, etc),
  which for some VMs come bundled with, and can be tricky to separate.
* We desired a [very custom memory system](#memory-pools) that would have been
  difficult to fit into other VMs.
* Some of the VMs of interest had licences that were incompatible with our
  needs.
* Our speed requirements are very high, we do profiling based on the VM
  playback, we need as little overhead as possible per draw call.


## Memory pools

GAPIR has 3 distinct types of memory pools.

### Volatile memory

Volatile memory is pre-allocated memory that is free to be modified by any
opcode during execution. It can be used for temporary or semi-persistent
storage.

### Constant memory

Along with a sequence of opcodes, a replay request contains a block of constant
data. This may be read from at any point in the execution of the replay, but is
immutable for the entire replay.

### Absolute pointers

Memory that’s not allocated by the replay system may still need to be read or
written to in order to perform a replay. Pointers returned by [`glGetString`]
[glGetString] or [`glMapBufferRange`][glMapBufferRange] are examples
of memory that’s not allocated by the replay system, but may need to be
accessed.


## Data types

The AGI virtual-machine supports the following primitive data types:

Type            | Description
--------------- | ----------------------------------------------------
Bool            | true / false value
Int8            | 8-bit signed integer
Int16           | 16-bit signed integer
Int32           | 32-bit signed integer
Int64           | 64-bit signed integer
Uint8           | 8-bit unsigned integer
Uint16          | 16-bit unsigned integer
Uint32          | 32-bit unsigned integer
Uint64          | 64-bit unsigned integer
Float           | 32-bit floating point number
Double          | 64-bit floating point number
AbsolutePointer | Pointer to an [absolute address](#absolute-pointers)
ConstantPointer | Pointer within the [constant pool](#constant-memory)
VolatilePointer | Pointer within the [volatile pool](#volatile-memory)

### Stack

The VM uses a standard LIFO stack where each element is a type-value pair.
The size of the stored elements are unified to the size of the largest storable
type and all of the elements are aligned.

Each operation, except for `CLONE`, consumes the operands from the current stack
and pushes the result back to the stack.

## Opcodes

Each opcode is 32 bits long where the first 6 bits are the instruction code and
the rest of the bits contain the instruction data. This leaves room for
additional instructions to be added in the future.

Notation: `<field_name:field_size_in_bits>`

### `CALL(push-return, api, function)` [-{arg-count} (any type) / +{push-return} (any type)]
`<code:6> <padding:1> <push-return:1> <padding:4> <api:4> <function id:16>`

Calls the specified function in the given API and if push-return is 1 then saves the
return value to the stack; otherwise the return value is discarded.

The arguments are popped from the stack and they are type-checked with the arguments
of the called function.

The arguments have to be pushed onto the stack in order (the last argument is on the
top of the stack).

Function IDs in range 0xff00-0xffff are reserved.

### `PUSH_I(type, data)` [+1 (type)]
`<code:6> <type:6> <data:20>`

Pushes `data` to the top of the stack.

If the data type is an integer or a pointer type, then the data is copied into the
least-significant-bits of the target word, sign-extending if the type is signed.

If the data type is a float or double, then the value is written to the sign and
exponent bits of the floating point number, and the fractional bits are set to 0.

### `LOAD_C(type, address)` [+1 (type)]
`<code:6> <type:6> <constant-address:20>`

Pushes data loaded from `constant-address` to the top of the stack.

### `LOAD_V(type, address)` [+1 (type)]
`<code:6> <type:6> <volatile-address:20>`

Pushes data loaded from `volatile-address` to the top of the stack.

### `LOAD(type)` [-1 (pointer) / +1 (type)]
`<code:6> <type:6> <padding:20>`

Pops a memory address from the top of the stack and pushes the data at that
address to the top of the stack

### `POP(count)` [-{count} (any type)]
`<code:6> <count:26>`

Pops and discards `count` values from the top of the stack.

### `STORE_V(volatile-address)` [-1 (any type)]
`<code:6> <volatile-address:26>`

Pops the top value from the the stack and saves it to `volatile-address`.
All pointer values, regardless of the pointer type on the stack, will be stored as an
absolute pointer address.

### `STORE()` [-2 (pointer, any type)]
`<code:6> <padding:26>`

Pops the target address and then the value from the top of the stack, and then stores
the value to the target address.
All pointer values, regardless of the pointer type on the stack, will be stored as an
absolute pointer address.

### `RESOURCE(resource-id)` [-1 (pointer)]
`<code:6> <resource-id:26>`

Pops the address from the top of the stack and then loads the resource `resource-id`
to that address.

### `POST()` [-2 (uint32_t, pointer)]
`<code:6> <padding:26>`

Pops size and then a pointer from the top of the stack and posts size bytes of
data from the address to the server.

### `COPY(count)` [-2 (pointer, pointer)]
`<code:6> <count:26>`

Pops the target address then the source address from the top of the stack, and
then copies `count` bytes from source to target.

### `CLONE(n)` [+1 (any type)]
`<code:6> <n:26>`

Copies the n-th element from the top of the stack to the new top of the stack.

### `STRCPY()` [-2 (pointer, pointer)]
`<code:6> <max-count:26>`

Pops the target address then the source address from the top of the stack, and
then copies at most `max-count` minus one bytes from source to target. If the
`max-count` is greater than the source string length, then the target will be
padded with 0s. The destination buffer will always be 0-terminated.

### `EXTEND(value)` [no change]
`<code:6> <value:26>`

Extends the value at the top of the stack with the given data, in-place.

If the data type of the top of the stack is an integer or a pointer type, then the
value on the stack is left-shifted by 26 bits and is bitwise-OR’ed with the
specified value.

If the data type is a float or double, then the fractional part of the floating
point value on the stack is left-shifted by 26 bits and is bitwise-OR’ed with the
specified value. Bits shifted beyond the fractional part of the floating point
number are discarded.

### `ADD(value)` [no change]
`<code:6> <count:26>`

Pops and sums `count` values from the top of the stack, and then pushes the
result to the top of the stack.

All summed value types must be equal.

### `LABEL(value)` [no change]
`<code:6> <value:26>`

Set the current debug label to `value`.
The label value is displayed in debug messages or in the case of a crash.

### `JUMPLABEL(value)` [no change]
`<code:6> <value:26>`

Add a jump label to store the current execute instruction index so that later
a jump instruction can jump to this instruction and start execution from there.

### `JUMPNZ(value)` [no change]
`<code:6> <value:26>`

Jump to the instruction specified by the jump label and start execution from there
if the value on the top of the stack is not zero. Otherwise it is a Nop.

### `JUMPZ(value)` [no change]
`<code:6> <value:26>`

Jump to the instruction specified by the jump label and start execution from there
if the value on the top of the stack is zero. Otherwise it is a Nop.

### `NOTIFICATION()` [-2 (uint32_t, pointer)]
`<code:6> <padding:26>`

Pops size and then a pointer from the top of the stack and streams back `size` bytes of
data from the address to the server via the notification message.

### `WAIT()` [no change]
`<code:6> <fence-id:26>`

Streams back the `fence-id` to the server. Replay pauses until
the server streams back the same ID.

## Resources

GAPIR is designed to be run on desktop and Android devices. When replaying on
Android, the communication between GAPIS and GAPIR is usually performed over USB
2, which has a peak throughput of around 60 megabytes per second. It’s not
uncommon for capture files to be hundreds of megabytes in size, and in rare
cases an order of magnitude greater than that.

It is typical for many replay requests to be made for the same capture file -
for example clicking around the draw calls in the client will usually result in
a replay request per click. The bulk of the data in replay requests of the same
capture file is identical - the large assets are typically static textures and
mesh data.

To avoid repeated transmission of these large assets over USB, GAPIR has a
memory cache for storing resource data.

A list of resources used in the replay is included as part of the replay request
payload header. This list consists of all the resource identifiers used by the
replay stream (and their size). Upon receiving the header, GAPIR can check which
of the resources it already has in its cache, and request the resource data for
those that are missing.


[Neko]:                   http://nekovm.org/
[Lua]:                    http://www.lua.org/
[Parrot]:                 http://www.parrot.org/
[glGetString]:            https://www.khronos.org/registry/OpenGL-Refpages/es3/html/glGetString.xhtml
[glMapBufferRange]:       https://www.khronos.org/registry/OpenGL-Refpages/es3/html/glMapBufferRange.xhtml
