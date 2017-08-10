# Proto-Pack format Version 1.1

## File

 name   | type               | description
------- | ------------------ | ------------
 magic  | `byte[9]`          | `'p'`, `'r'`, `'o'`, `'t'`, `'o'`, `'p'`, `'a'`, `'c'`, `'k'`
 header | `Chunk<Header>`    | Header chunk
 data   | `Chunk<Section>[]` | File content

## Chunk\<T\>

 name  | type             | description
------ | ---------------- | ------------
 size  | `varint`         | Size of the chunk data
 data  | `byte[size]`     | `T` data

## Section

A section is either a type or a object:

### Group object (`tag < 0`)

Encoding of an object instance.
May be a child object of a previously encoded object.

 name   | type             | description
------- | ---------------- | ------------
 tag    | `zig-zag varint` | Type index `-tag - 1`
 parent | `varint`         | Relative parent group index `+ 1`. 0 indicates no parent
 data   | `byte[]`         | Proto data

### Group object finalizer (`tag = 0`)

Indicates that the specified object has no more children.
Once a group has been encoded, the global group counter is incremented. This counter is used to reference parent groups.

 name   | type             | description
------- | ---------------- | ------------
 tag    | `zig-zag varint` | `0`
 object | `varint`         | Relative group index that's being finalized

### Type (`tag = 1`)

Declaration of a new type. The type is described using a `DescriptorProto`.
Types are referenced by index of their declaration order in the stream.

 name   | type              | description
------- | ----------------- | ------------
 tag    | `zig-zag varint`  | `1`
 name   | `varstr`          | Proto type name
 desc   | `DescriptorProto` | Proto type descriptor

### Object (`tag >= 2`)

Encoding of an object instance.
May be a child object of a previously encoded object.

 name   | type             | description
------- | ---------------- | ------------
 tag    | `zig-zag varint` | Type index `tag - 2`
 parent | `varint`         | Relative parent group index `+ 1`. 0 indicates no parent
 data   | `byte[]`         | Proto data
