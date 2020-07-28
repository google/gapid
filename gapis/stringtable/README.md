# String tables

String tables provide a solution for displaying localized messages in the
client(s) with the ability to include graphics API specific details. Given that
the clients are designed to be API-agnostic, these string tables are provided by
the server to the clients via RPCs to form detailed messages.

String tables also provide support for rich-text formatting.

## Interface

The client can query for the list of supported string tables using the
`GetAvailableStringTables` RPC, which returns a list of available
string tables based on culture codes.

The `GetStringTable` RPC retrieves the string table for the requested
localization. Each string table holds a map of entry identifiers to a root
[`Node`](#nodes).

A `Msg` represents a usage of a string table entry. It holds just two
fields, the string table entry identifier and the list of arguments for any
dynamic parameters:

```go
type Msg struct {
  Identifier string                   // String table entry identifier.
  Arguments  map[string]binary.Object // Argument list.
}
```

When displaying a message, the client must lookup the string table entry from
the active string table, and subsitute all [parameter](#parmeters) nodes with
the corresponding argument type.

## Nodes

A string table entry is a tree of `Node`s, each representing a span of the message.

The following types are implementations of `Node`:

### Block

A block node is a sequential container of other nodes. For string table entries
with more than a single line of text, a block will often form the root node of
the string table entry.

### Text

A text node contains a localized text string.

This is a root node type.

### LineBreak

A line break node represents a vertical gap in layout.

This is a root node type.

### Whitespace

A whitespace node represents a horizontal gap in text layout.

This is a root node type.

### Parameter

A parameter node represents a dynamic value parameter. These parameters are
subsituted with the argument with the corresponding identifier.

### Link

A link node represents a dynamic value parameter hyper-link. A link node holds a
node for the link text and another for the link target.

### Bold

A bold node represents a section that should be displayed in bold.

### Underlined

An underlined node represents a section that should be displayed underlined.

### Heading

A heading node represents a section that should be displayed as a heading.

### Code

A code node represents a block of code in a particular language.

### List

A list node represents an unordered list.

## Authoring

String tables are written in a subset of the markdown language, called 'minidown'.

Multiple string table entries can be declared in the same file, with each entry
starting with a H1 header which declares the entry's identifier, followed by the
entry's body:

```markdown
# LOCALIZED_MESSAGE_ID_A

This is the localized body for message A.

# LOCALIZED_MESSAGE_ID_B

This is the localized body for message B.
```

Using the stringgen compiler, a number of `.stb.md` localization files can be compiled into a `.stb` binary localization file and a `.go` containing go helper functions for building messages. The example above would produce a helper `.go` file with the functions:

```go
// LocalizedMessageIdA returns a LOCALIZED_MESSAGE_ID_A message with the
// provided arguments.
func LocalizedMessageIdA() *Msg { ... }

// LocalizedMessageIdB returns a LOCALIZED_MESSAGE_ID_B message with the
// provided arguments.
func LocalizedMessageIdB() *Msg { ... }
```

### Parameters

Each entry's body also supports parameters for dynamic message content:

```markdown
# MESSAGE_WITH_DYNAMIC_CONTENT

Hello {{per﻿son}}! How are you today?
```

These parameters become arguments to the go helper functions:

```go
// MessageWithDynamicContent returns a MESSAGE_WITH_DYNAMIC_CONTENT message with
// the provided arguments.
func LocalizedMessageIdA(person interface{}) *Msg { ... }
```

### Links

Hyperlinks can be declared using the syntax:

```markdown
# MESSAGE_WITH_LINKS

I love [fluffy kittens](https://www.google.co.uk/search?q=fluffy+kitten&tbm=isch)
```

Both the text and the target parts of the links can be parameterized:

```markdown
# MESSAGE_WITH_DYNAMIC_LINK_TEXT

I love [{{fluffies}}](https://www.google.co.uk/search?q=fluffy+kitten&tbm=isch)

# MESSAGE_WITH_DYNAMIC_LINK_TARGET

I love [fluffy kittens]({{fluffy-link}})

# MESSAGE_WITH_DYNAMIC_LINK_TEXT_AND_TARGET

I love [{{fluf﻿fies}}]({{fluffy-link}})

```

### Headers

Because the H1 header is reserved for declaring a new entry, use H2 headers for
an H1 header, H3 for a H2 header. and so on:

```markdown
# MESSAGE_WITH_HEADERS

## This is a H1 header
Blah blah blah

### This is a H2 header
Blah blah blah
```

### Emphasis

\*italic\* displays as *italic*.

\*\*bold\*\* displays as **bold**.

\_underline\_ displays as _underline_.

```markdown
# MESSAGE_WITH_EMPHASIS

I *really*, **really**, _really_ like emphasis!
```
