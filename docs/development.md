# Development

[Docs Index](README.md) · [Project README](../README.md)

To improve the validation, you can add validators and translators.

## How to add other file types

Implement a file reader to `reader` package that implements the interface in `reader.go`

There is currently no support for inputting structures, so all input needs to be a linear list of field name and string value pairs.
You could consider separating the hierarchy building step out of normalisation, to enable some file types (CSV) use strucure building and better formats (JSON, XML) bring the structure already in the read phase.

## How to add validators and translators

Implement the interface at `validators/validator.go` or `translators/translators.go`.

Note that because validators are initialised into the validator map `validators.Validators`, you can build a generic validator, like "ValidateAllowedValues" and initialise it with different value sets, so you can use the same code for different validations. Same goes for translators.

Translators have an example of this in `LookupTranslator`.

## Types

The OpenAPI schema in /schemas can be converted to Go with 
```bash
oapi-codegen -generate types -package privxapi -o internal/privx/iam-host-store-api.go schemas/iam-host-store-api.yaml
```