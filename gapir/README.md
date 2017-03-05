# Graphics API Replay (GAPIR)

GAPIR is a stack-based virtual machine that can execute programs formed from a
very small instruction set.

## Evaluation of existing VMs

Before embarking on building a new virtual machine from scratch, we evaluated
our needs, and compared it to a number of existing, lightweight, open-source VMs
([Lua], [Parrot], [Neko], etc).

We opted for building a custom VM because:

* Our [required instruction set](#vm-instruction-set) was significantly
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
* Our speed requirements are very high, we do profiling based on the vm
  playback, we need as little vm overhead as possible per draw call.


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
[glGetString.xml] or [`glMapBufferRange`][glMapBufferRange.xhtml] are examples
of memory that’s not allocated by the replay system, but may need to be
accessed.


## Data types

The GAPID virtual-machine supports the following primitive data types:

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


## VM instruction set

The entire replay VM instruction set consists of the following opcodes:

<table>
<tr><td>CALL</td><td>PUSH_I</td><td>LOAD_C</td><td>LOAD_V</td><td>LOAD</td></tr>
<tr><td>POP</td><td>STORE_V</td><td>STORE</td><td>RESOURCE</td><td>POST</td></tr>
<tr><td>COPY</td><td>CLONE</td><td>STRCPY</td><td>EXTEND</td><td>LABEL</td></tr>
</table>

Full descriptions on each opcode is described in the **{{TODO}}** [document][interpreter_doc]
accompanying the source code


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