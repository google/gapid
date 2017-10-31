# Proto-Pack format Version 2.0

## Header

 name   | type       | description
------- | ---------- | ------------
 magic  | `byte[16]` | `"ProtoPack\r\n2.0\n\0"`

The header contains both types of new-lines, which is common in file
headers to detect corruption caused by automatic new-line conversions.
The header is followed by arbitrary number of variable-sized chunks.
Chunks can be either object instance or type definition depending on the
sign of the `size` field (encoded as protobuf's variable-length zigzag).

## Object instance chunk (size>0)

 name     | type      | description
--------- | --------- | ------------
 `size`   | `sint32`  | Total size of the chunk excluding this size field.
 `parent` | `sint32`  | If >= 0: New root object instance. <br /> If < 0: Relative index of chunk which is the parent.
 `type`   | `sint32`  | If > 0: Type index. The object has no children. <br /> If < 0: Negated type index if it may have children. <br /> If == 0: Terminates children list of the `parent`.
 `data`   | `byte[]`  | Proto message.

Objects may form a tree. Parent object is specified as back-reference to previous
chunk (-1 is the previous chunk). The referenced chunk must also be an object and
it must allow addition of children. Values of `parent>=0` define a root object.
Values of `parent>0` may be used in the future to add information to the root object.

Negated `type` index denotes object which may have children.
The children list must be terminated by chunk with `type==0`.
Object with positive `type` can not have children and does not need terminator.

As an optimization, if `size` is small enough to cover only the `parent`
field, `type` field is implicitly set to 0 (i.e. it is list terminator).

## Type definition chunk (size<0)

 name    | type     | description
-------- | -------- | ------------
 `size`  | `sint32` | Negated total size of the chunk excluding this size field.
 `name`  | `string` | Fully qualified type name as proto string.
 `desc`  | `byte[]` | Proto descriptor of one message type of the given `name`.

The format is self-describing. All objects are stored as typed proto messages,
where the type must be first described by type definition chunk.
Types are assigned indices based on the order in the file (starting with 1).
