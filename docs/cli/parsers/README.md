# Parsers
   * [openapi3](#openapi3)
      * [Description](#description)
      * [Options](#options)
         * [List of all options](#list-of-all-options)
         * [Example usage in Repose config](#example-usage-in-repose-config)
      * [Extensions](#extensions)
         * [Path](#path)
            * [Fields](#fields)
            * [Example](#example)
         * [Response](#response)
            * [Fields](#fields-1)
            * [Example](#example-1)
         * [Schema](#schema)
            * [Fields](#fields-2)
            * [Example](#example-2)

# openapi3
## Description

This parser supports parsing Open API 3 specifications using [kin-openapi](https://github.com/getkin/kin-openapi).

Currently only one input file is supported, so definitions in local files are not resolved,
however resolving external resources are supported by kin-openapi.

## Options

### List of all options

| Option | Description | Type | Default Value |
|:------:|-------------|:----:|:--------------|	
additionalPropertiesName|Name of the additionalProperties field in structs that have them.|string|<pre lang="yaml">AdditionalProperties</pre>|
extensionName|The name of the extension field.|string|<pre lang="yaml">x-repose</pre>|
resolveReferencesAt|Resolve references at the given URL.|string|<pre lang="yaml">""</pre>|
resolveReferencesIn|Resolve references in a local folder.|string|<pre lang="yaml">""</pre>|
stripExtension|Strip the repose extension from the specification, the spec extension is used for code generation, and in most cases it's useless after that. Removing it for public APIs is also generally a good idea, where the specification will be visible.|bool|<pre lang="yaml">true</pre>|


### Example usage in Repose config

```yaml
openapi3:
    extensionName: x-repose
    additionalPropertiesName: AdditionalProperties
    stripExtension: true
```


## Extensions

The parser supports several [extensions](https://swagger.io/docs/specification/openapi-extensions/)
that can be used in the specification to enhance code generation.

### Path

Extension for Open API 3 [paths](https://swagger.io/docs/specification/paths-and-operations/).

#### Fields

| Field | Description | Type |
|:-----:|-------------|:----:|
name|The name of the path.|*string|


#### Example

```yaml
/gooddogs:
    x-repose:
        name: GetGoodDogs
```


### Response

Extension for Open API 3 [responses](https://swagger.io/docs/specification/describing-responses/).

#### Fields

| Field | Description | Type |
|:-----:|-------------|:----:|
name|The name of the response.|*string|


#### Example

```yaml
"200":
    x-repose:
        name: AllGoodDogs
```


### Schema

Extension for Open API 3 [schemas](https://swagger.io/docs/specification/data-models/).

#### Fields

| Field | Description | Type |
|:-----:|-------------|:----:|
canBeNil|Whether the type can be nil, and should not have a pointer to it (e.g. slices, maps, or interfaces), it is only needed when a custom Go type is set, but create is set to false, so only the type name is known to Repose.|*bool|
create|Whether the type should be created.|*bool|
tags|Additional tags for the field.|map[string][]string|
type|The Go type of the schema.|*string|


#### Example

```yaml
GoodDog:
    x-repose:
        type: petslibrary.GoodDog
        create: false
        canBeNil: true
```


```yaml
GoodDog:
    x-repose:
        type: LocalGoodDog
        create: true
        tags:
            json:
              - localGoodDog
              - omitempty
```



