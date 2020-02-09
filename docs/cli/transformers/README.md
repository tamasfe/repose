# Specification transformers
   * [default](#default)
      * [Description](#description)
      * [Options](#options)
         * [List of all options](#list-of-all-options)
         * [Example usage in Repose config](#example-usage-in-repose-config)
            * [Tag template values](#tag-template-values)

# default
## Description

This transformer is part the core of Repose Go code generation.
It doesn't have a lot of options yet, but almost everything relies on it.

## Options

### List of all options

| Option | Description | Type | Default Value |
|:------:|-------------|:----:|:--------------|	
tags|Add additional tags to struct fields. Supports Go templating with sprig functions.|map[string][]string|<pre lang="yaml">json:<br>  - '{{ .FieldName }}'<br>  - omitempty</pre>|


### Example usage in Repose config

```yaml
go-general:
    tags:
        json:
          - '{{ .FieldName }}'
          - omitempty
```


#### Tag template values

| Value | Description |
|:-----:|-------------|
Description|Description of the schema|
FieldName|Name of the field|
Type|Type of the field|



