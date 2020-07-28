# Templates

AGI uses the golang [template] package with its API file parser/compiler to generate Go, C++ and Java code.

The .api files are parsed, and a `.tmpl` file is used to generate one or more output files using the `apic` tool.

AGI's template system exposes a few extensions to the standard golang template system:

## Macros

Macros are similar to templates except they can support more than one single
argument, expressed as key-value pairs. The syntax for calling these can be
either:

```go
{{Macro macro-name single-arg}}
{{Macro macro-name arg0-key arg0-value arg1-key arg1-value...}}
```

For example, the .tmpl code:

```go
{{define "ABC"}}A: '{{$.A}}', B: '{{$.B}}', C: '{{$.C}}'{{end}}
{{Macro "ABC" "A" 1 "B" 2 "C" 3}}
```

Would produce:

```go
A: ‘1’, B: ‘2’, C: ‘3’
```

Macros also support a single argument in exactly the same way that `Template`
would. So these two examples will produce identical results, but with `Template`
being the faster option:

1.  Macro

    ```go
    {{define "Foocakes"}}I was passed: '{{$}}'{{end}}
    {{Macro "Foocakes" "Hello world"}}
    ```

2.  Template

    ```go
    {{define "Foocakes"}}I was passed: '{{$}}'{{end}}
    {{Template "Foocakes" "Hello world"}}
    ```

## Special symbols

To try and help with the formatting of the generated output, AGI's templates support a number of custom unicode symbols:

### `»` Begin indentation of a block

All new-lines following the `»` symbol will be indented one extra level. This is in addition to the automatic indentation between { and } characters.

### `«` End indentation of a block

All new-lines following the `«` symbol will be un-indented one extra level.

### `§` Suppress whitespace, tabs and newlines until the next alpha-numeric character

Use the `§` symbol to break complex template logic across multiple lines without introducing new-lines in the outputted code. For example:

```go
{{define "Parameters"}}
  {{range $i, $p := $.Parameters}}
    {{if $i}},{{end}}§
    {{$p.Name}} {{$p.Type}}§
  {{end}}
{{end}}
```

Would produce a single-lined, comma-separated parameter list. This is far more readable than attempting to define the Parameters macro on a single line.

### `¶` Emit a new-line

Sometimes you want to produce extra new-lines to break up the emitted code. By default, many new-lines are merged by the reflow system into one, so to emit multiple new-lines you can use the `¶` symbol.

### `•` Emit a whitespace

Like new-lines, sometimes you want to emit an extra whitespace for indentation.
By default, whitespaces are consumed at the start of a new-line, so to forcefully emit a whitespace you can use the `•` symbol.

[template]: http://golang.org/pkg/text/template/
