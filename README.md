# terraform-provider-go

This is an experimental OpenTofu function provider based on terraform-plugin-go.

It allows you to write Go helper functions next to your Tofu code, so that you can use them in your Tofu configuration, in a completely type-safe way. The provider is based on [Yaegi](https://github.com/traefik/yaegi), and most of the Go standard library is available.

In OpenTofu 1.7.0-beta1 and upwards you can configure the provider and pass it a Go file to load.
- The package name should be `lib`
- Exported functions need to start with upper-case letters.
- The Tofu-facing name of the function **will be lower-cased**.
- It supports simple types, like strings, integers, floats, and booleans.
- It also supports complex type, like maps, slices, nullable pointers, and structures.

This feature is an experimental preview and is subject to change before the OpenTofu 1.7.0 release.

> :warning: When writing Go, the features available depend on the version of
> Go used to build the relevant OpenTofu release you're using.  For example,
> OpenTofu 1.7.0 was built with Go 1.21, so features only available in 1.22+
> will not work and may result in obscure error messages.

## Example

```hcl
// main.tf
provider "go" {
  go = file("./lib.go")
}

output "test" {
  value = provider::go::hello("papaya")
}
```
```go
// lib.go
package lib

func Hello(name string) string {
	return "Hello, " + name + "!"
}
```
Output excerpt:
```
Changes to Outputs:
  + test = "Hello, papaya!"
```

## More involved example

```hcl
// main.tf
provider "go" {
  go = file("./lib.go")
}

output "test" {
  value = provider::go::hello({
    name = "papaya",
    surname = "bacon",
  })
}
```
```go
// lib.go
package lib

import (
	"fmt"
)

type Person struct {
	// We can let it default to the un-capitalized field name.
	Name string
	// Or use a struct tag to specify the object field name explicitly.
	Surname string `tf:"surname"`
}

func Hello(person Person) string {
	return fmt.Sprintf("Hello, %s %s!", person.Name, person.Surname)
}
```
Output excerpt:
```
Changes to Outputs:
  + test = "Hello, papaya bacon!"
```

Moreover, all of this is type-safe and mistakes will be caught by tofu. So passing a number to the function will fail with `object required`, while forgetting e.g. the surname will fail with `attribute "surname" is required`.

## Importing
Here's a snippet to require the provider in your OpenTofu configuration:
```hcl
terraform {
  required_providers {
    go = {
      source  = "registry.opentofu.org/opentofu/go"
      version = "0.0.1"
    }
  }
}
```
