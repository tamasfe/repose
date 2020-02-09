
# Repose

Repose is a library and CLI tool that generates server and client code stubs for RESTful APIs from specification.

It was designed with only Go code in mind, and uses [Jennifer](github.com/dave/jennifer) extensively for more flexibility. However the code is structured in such a way that doesn't limit it to Go code only, so supporting more languages is a possibility but currently I see no reason for it as there are many template-based generators for various languages.

It is pretty much WIP with probably essential features missing, and there is not a lot of documentation yet along with sparse comments.
There are also no tests and the codebase also needs a lot of refactoring for better maintainability before adding more features.

However even in its current state, it's already in use, and is being slowly developed.

Table of Contents
=================

   * [Installation](#installation)
      * [Command line](#command-line)
      * [Library](#library)
   * [Usage](#usage)
      * [Command Line](#command-line-1)
      * [As a library](#as-a-library)
   * [Features](#features)
   * [Contributing](#contributing)



# Installation

## Command line

Either install latest the binary with Go:

`go get -u github.com/tamasfe/repose/cmd/repose`

Or download a specific version from [releases](https://github.com/tamasfe/repose/releases).

## Library

Get the library with:

`go get github.com/tamasfe/repose`

# Usage

## Command Line

Documentation for the CLI is available [here](github.com/tamasfe/repose/tree/master/docs/cli)

## As a library

TODO: This is yet to be documented, for now you can look at the CLI tool's code.

# Features

The repose CLI works with 3 stages:

- **parsing**: A specific parser parses a specification, and creates Repose's abstraction of it that is designed for code generation.
- **transforming**: One or more transformers process the parsed specification and make changes to it (such as creating new schemas, adding tags, renaming paths, and so on).
- **generation**: One or more generators generate code with one or more targets (e.g. go-echo:server+scaffold, go-general:types).

If used as a library, you can use the parsers, transformers and generators independently, or even create your own ones by implementing the necessary interfaces.

Documentation for the currently available components is generated from code and is available [in the CLI docs](github.com/tamasfe/repose/tree/master/docs/cli).

# Contributing

The project is still in a very early stage, so I believe the most helpful contributions would be opening tickets or discussing the existing ones about missing (and potential future) features rather than actual code.

Tests, documentation and code refactor contributions are welcome but as always, make sure to visit the open issues, or create a new one **before you start working** on a pull request!
