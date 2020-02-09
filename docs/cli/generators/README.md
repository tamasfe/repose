# Code generators
   * [go-general](#go-general)
      * [Description](#description)
      * [Options](#options)
         * [List of all options](#list-of-all-options)
         * [Example usage in Repose config](#example-usage-in-repose-config)
      * [Targets](#targets)
   * [go-stdlib](#go-stdlib)
      * [Description](#description-1)
      * [Options](#options-1)
         * [List of all options](#list-of-all-options-1)
         * [Example usage in Repose config](#example-usage-in-repose-config-1)
      * [Targets](#targets-1)
   * [go-echo](#go-echo)
      * [Description](#description-2)
      * [Options](#options-2)
         * [List of all options](#list-of-all-options-2)
         * [Example usage in Repose config](#example-usage-in-repose-config-2)
      * [Targets](#targets-2)

# go-general
## Description

This generator generates framework-agnostic Go code.

## Options

### List of all options

| Option | Description | Type | Default Value |
|:------:|-------------|:----:|:--------------|	
expandEnums|Expand enums into const (...) blocks if possible.|bool|<pre lang="yaml">true</pre>|
generateGettersAndSetters|Generate helper methods for getting and setting properties for maps or structs with unknown names (E.g. additional properties).|bool|<pre lang="yaml">true</pre>|
generateMarshalMethods|Generate marshal/unmarshal methods for types that need them.|bool|<pre lang="yaml">true</pre>|
generateTypeHelpers|Generate helper functions and methods for types.|bool|<pre lang="yaml">true</pre>|
typesPackagePath|Package path to already generated types (used internally).|string|<pre lang="yaml">""</pre>|


### Example usage in Repose config

```yaml
go-general:
    generateTypeHelpers: true
    generateGettersAndSetters: true
    generateMarshalMethods: true
    expandEnums: true
```


## Targets

| Target | Description |
|:------:|-------------|
spec|The bytes of the parsed specification file|
types|Go types for the schemas in the specification|


# go-stdlib
## Description

This generator generates code that only relies on the standard library.

## Options

### List of all options

| Option | Description | Type | Default Value |
|:------:|-------------|:----:|:--------------|	
typesPackagePath|Path to the generated types package, if left empty it is assumed that it is in the same package.|string|<pre lang="yaml">""</pre>|


### Example usage in Repose config

```yaml
go-stdlib:
    typesPackagePath: ""
```


## Targets

| Target | Description |
|:------:|-------------|
callbacks|Generate Go HTTP Requests for callbacks|
client|Generate Go HTTP Requests|


# go-echo
## Description

This generator provides code generation for the Go [Echo](https://echo.labstack.com/) server framework.

## Options

### List of all options

| Option | Description | Type | Default Value |
|:------:|-------------|:----:|:--------------|	
allowNoResponse|Add a NoResponse value that indicates that the returned value by a handler should be ignored by the generated wrapper.|bool|<pre lang="yaml">false</pre>|
responsePostfix|Postfix to add for response types, configure it to avoid collisions with actual types.|string|<pre lang="yaml">HandlerResponse</pre>|
serverImplName|Name of the server interface implementation.|string|<pre lang="yaml">ServerImpl</pre>|
serverMiddleware|Enable the ability to add middleware to the individual operations from a method on the server interface.|bool|<pre lang="yaml">true</pre>|
serverName|Name of the server interface.|string|<pre lang="yaml">Server</pre>|
serverPackagePath|Path to the generated server package, used for generating the scaffold, if left empty it is assumed that it is in the same package.|string|<pre lang="yaml">""</pre>|
shortScaffoldComments|Shorter scaffold comments for each method implementation.|bool|<pre lang="yaml">false</pre>|
typesPackagePath|Path to the generated types package, used for generating the server interface, if left empty it is assumed that it is in the same package.|string|<pre lang="yaml">""</pre>|


### Example usage in Repose config

```yaml
go-stdlib:
    serverName: Server
    serverImplName: ServerImpl
    allowNoResponse: false
    serverPackagePath: ""
    typesPackagePath: ""
    responsePostfix: HandlerResponse
    shortScaffoldComments: false
    serverMiddleware: true
```


## Targets

| Target | Description |
|:------:|-------------|
server|The server interface, and the register function|
server-scaffold|Scaffold for a server interface|


