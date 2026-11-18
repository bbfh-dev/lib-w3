# Lib W3

[![brainmade.org](https://img.shields.io/badge/brainmade.org-000000?style=for-the-badge&logo=data%3Aimage%2Fsvg%2Bxml%3Bbase64%2CPHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHdpZHRoPSIxZW0iIGhlaWdodD0iNzkiIHZpZXdCb3g9IjAgMCA2NyA3OSIgZmlsbD0ibm9uZSI%2BPHBhdGggZmlsbD0iI2ZmZiIgZD0iTTUyLjYxMiA3OC43ODJIMjMuMzNhMi41NTkgMi41NTkgMCAwIDEtMi41Ni0yLjU1OHYtNy42NzdoLTcuOTczYTIuNTYgMi41NiAwIDAgMS0yLjU2LTIuNTZWNTUuMzE1bC04LjgyLTQuMzk3YTIuNTU5IDIuNTU5IDAgMCAxLS45ODYtMy43MWw5LjgwNy0xNC43MTR2LTQuMzVDMTAuMjQgMTIuNTk5IDIyLjg0MyAwIDM4LjM4OCAwIDUzLjkzMiAwIDY2LjUzNCAxMi42IDY2LjUzOCAyOC4xNDNjLS42MzIgMjcuODI0LTEwLjc2IDIzLjUxNi0xMS4xOCAzNC4wNDVsLS4xODcgMTQuMDM1YTIuNTkgMi41OSAwIDAgMS0uNzUgMS44MSAyLjU1IDIuNTUgMCAwIDEtMS44MDkuNzVabS0yNi43MjMtNS4xMTdoMjQuMTY0bC4yODYtMTQuNTQyYy0uMjYzLTYuNjU2IDExLjcxNi04LjI0MyAxMS4wOC0zMC43MzQtLjM1OC0xMi43MTMtMTAuMzEzLTIzLjI3MS0yMy4wMzEtMjMuMjcxLTEyLjcxOCAwLTIzLjAyOSAxMC4zMDctMjMuMDMyIDIzLjAyNXY1LjExN2MwIC41MDYtLjE1IDEtLjQzIDEuNDJsLTguNjMgMTIuOTQxIDcuNjQ1IDMuODJhMi41NTkgMi41NTkgMCAwIDEgMS40MTUgMi4yOTF2OS42OTdoNy45NzRhMi41NTkgMi41NTkgMCAwIDEgMi41NiAyLjU1OXY3LjY3N1oiLz48cGF0aCBmaWxsPSIjZmZmIiBkPSJNNDAuMzcyIDU4LjIyMlYzOC45MzRjLjExOCAwIC4yMzcuMDE4LjM1NS4wMTggOS43NjktLjAxMiAxNy4wNS05LjAxMiAxNS4wMjItMTguNTY3YTIuMzY2IDIuMzY2IDAgMCAwLTEuODIxLTEuODIyYy04LjEwNi0xLjczLTE2LjEyMSAzLjI5Mi0xOC4wOTggMTEuMzQxLS4wMjQtLjAyNC0uMDQzLS4wNS0uMDY2LS4wNzNhMTUuMzIzIDE1LjMyMyAwIDAgMC0xNC4wNi00LjE3IDIuMzY1IDIuMzY1IDAgMCAwLTEuODIxIDEuODJjLTIuMDI4IDkuNTU1IDUuMjUyIDE4LjU1NCAxNS4wMiAxOC41NjguMjM2IDAgLjQ5Mi0uMDI4LjczOC0uMDR2MTIuMjEzaDQuNzMxWm0yLjgzOS0zMi4xNDNhMTAuNjQ2IDEwLjY0NiAwIDAgMSA4LjEyNC0zLjEwNmMuMzUgNi4zNC00Ljg4OCAxMS41NzctMTEuMjI4IDExLjIzYTEwLjU4IDEwLjU4IDAgMCAxIDMuMTA0LTguMTI0Wk0yNy40MDMgMzguMTkzYTEwLjYwNyAxMC42MDcgMCAwIDEtMy4xMTgtOC4xMjNjNi4zNDQtLjM1OCAxMS41ODcgNC44ODYgMTEuMjI4IDExLjIzLTMuMDIzLjE2OS01Ljk3My0uOTYxLTguMTEtMy4xMDdaIi8%2BPC9zdmc%2B)](https://brainmade.org/)

Go library for creating efficient web backends powered by [HttpRouter](https://github.com/bbfh-dev/tree-sitter-html).

Check out the [📁 **examples**](./examples/) directory for working project examples.

> [!IMPORTANT]
> Documentation of Go code can be found on the auto-generated [API Reference](https://pkg.go.dev/codeberg.org/bbfh/lib-w3) website.

<!-- vim-markdown-toc GFM -->

* [1 🏅 Features](#1--features)
* [2 📃 Templates](#2--templates)
    * [2.1 Introduction](#21-introduction)
    * [2.2 Tag arguments](#22-tag-arguments)
        * [2.2.1 Primitive arguments](#221-primitive-arguments)
        * [2.2.2 Paths](#222-paths)
        * [2.2.3 Compile-time paths](#223-compile-time-paths)
    * [2.3 Built-in Tags](#23-built-in-tags)
        * [2.3.1 Include](#231-include)
        * [2.3.2 Extend](#232-extend)
        * [2.3.3 Place](#233-place)
        * [2.3.4 Slot](#234-slot)
        * [2.3.5 Variable](#235-variable)
        * [2.3.6 Range](#236-range)
        * [2.3.7 If](#237-if)
        * [2.3.7 Static](#237-static)
    * [2.4 Custom tags](#24-custom-tags)

<!-- vim-markdown-toc -->

# 1 🏅 Features

- High-performance request handling using object pools & pre-parsed documents.
- Type-safe template parsing to minimize rendering errors during requests.
- Automatic checksums and serving for static files.

# 2 📃 Templates

> [!TIP]
> There's a [tree-sitter-html](https://github.com/bbfh-dev/tree-sitter-html) fork that adds support for W3 tags, which can be used for syntax highlighting in your editor of choice.

This section describes the syntax and features of the [w3html](https://pkg.go.dev/codeberg.org/bbfh/lib-w3/w3html) templating language.

## 2.1 Introduction

The HTML document is left as-is (without any parsing, syntax validation, indentation, etc.) other than parts inside or surrounded by W3 Tags.

## 2.2 Tag arguments

W3 tags always query **named arguments**, but fall back to them in order (e.g. `extend hello="world" path="hello.svg"` is equivalent to `extend "hello.svg" hello="world"` as for `extend` tag definition).

### 2.2.1 Primitive arguments

| Type           | Example                          |
| -------------- | -------------------------------- |
| String         | `"Hello World"`, `'Hello World!` |
| FormatString   | `` `Hello {Path.To.Value()}` ``  |
| Operation      | `==`, `>=`, `<`                  |
| LogicOperation | `&&`, `\|\|`                     |
| Integer        | `51`, `-9185`                    |
| Float          | `-.5`, `31.8`                    |
| Bool           | `true`, `false`                  |

### 2.2.2 Paths

Path must begin and be separated with a period `.`. Suffix a path segment with `[]` to iterate each element, `()` to call a method.
All paths are validated and baked during parsing using the provided datatype.

Example: `.Path.To().Each[].Value()`

If a path begins with a double period `..`, then the current scope will be appended (i.e. variable local to the parent tags).

### 2.2.3 Compile-time paths

If a path begins with a lowercase character it is instead replaced with the corresponding value provided in the parameters map (usually in `extend` or `include` tags).

Example: `<span>{{ ._filename.base }}</span>` could turn into `<span>my_file.html</span>` during parsing.

Built-in parameters include:

| Key               | Value                                                                                                                                                                  |
| ----------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `._filename`      | The path to the root file as provided to `os.OpenFile(...)`. A root file is the one that initiated `NewParser(...).Parse()` rather than any subsequent nested document |
| `._filename.base` | Same as `._filename`, but with `filepath.Base(...)` applied to it, stripping the path and only keeping the filename                                                    |

## 2.3 Built-in Tags

### 2.3.1 Include

```
{{ include <relative path to file> [key=value parameters...] }}
```

Provided path can ommit `.html` extension. HTML files are parsed and included by replacing the tag with nested document's contents. Provided `key=value` pairs can be accessed by using [Compile-time Paths](#223-compile-time-paths) inside of the nested document.

### 2.3.2 Extend

```
{{ extend <relative path to file> [key=value parameters...] }}
    ...
{{ /extend }}
```

Similar to [Include tag](#231-include), but injects HTML inside of the nested document.

Refer to [Slot](#234-slot) and [Place](#233-place) tags for injection documentation.

### 2.3.3 Place

```
{{ place [slot id] }}
```

Indicates where the [Extend](#232-extend) tag may inject HTML using [Slot](#234-slot) tag.
Use `{{ place "default" }}` to allow for unnamed slot injection.

### 2.3.4 Slot

```
{{ slot [slot_id] }}
    ...
{{ /slot }}
```

Place inside of an [Extend](#232-extend) tag to specify a hunk of HTML to be injected in place of a [Place tag](#233-place).

### 2.3.5 Variable

```
{{ .Path.To[].Here() }}
```

Compute the path and replace the tag with the contents.

### 2.3.6 Range

```
{{ range .Values[] }}
    <span>{{ ..ItemValue }}</span>
{{ /range }}
```

### 2.3.7 If

```
{{ if .Value == 123 }}
    <div>Hello World!</div>
{{ /if }}
```

Render contents only if the predicate is evaluated to true.

### 2.3.7 Static

```html
<head>
	<link rel="stylesheet" href="{{ static 'styles/main.css' }}" />
</head>
```

Evaluates into `/static/<path>?checksum=<checksum>` with a computed checksum. Used to force clients to re-download the files (cache invalidation on file change).

## 2.4 Custom tags

Tags are data-driven, meaning that it is possible to implement your own. Refer to [TagEnum API Reference](https://pkg.go.dev/codeberg.org/bbfh/lib-w3#TagEnum) documentation.

```go
var MyTag = TagEnum{
    // Your properties here...
}

func init() {
    w3html.RegisteredTags = append(w3html.RegisteredTags, MyTag)
}
```
