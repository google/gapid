# Proto-Pack format

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

### Type

 name   | type              | description
------- | ----------------- | ------------
 tag    | `varint`          | `0`
 name   | `varstr`          | Proto type name
 desc   | `DescriptorProto` | Proto type descriptor
 
### Object
 
 name   | type             | description
------- | ---------------- | ------------
 tag    | `varint`         | Type index `- 1`
 data   | `byte[]`         | Proto data
