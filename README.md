# terraform-provider-go

This is an experimental OpenTofu function provider based on terraform-plugin-go.

It allows you to write Go helper functions next to your Tofu code, so that you can use them in your Tofu configuration, in a completely type-safe way. The provider is based on [Yaegi](https://github.com/traefik/yaegi), and most of the Go standard library is available.

In OpenTofu 1.7.0-beta1 and upwards you can configure the provider and pass it a Go file to load.
- The package name should be `lib`
- Exported functions need to start with upper-case letters.
- The Tofu-facing name of the function **will be lower-cased**.
- It supports simple types, like strings, integers, floats, and booleans.
- It also supports complex type, like maps, slices, and nullable pointers (structs coming soon).

This feature is an experimental preview and is subject to change before the OpenTofu 1.7.0 release.

```hcl
// main.tf
provider "go" {
  go = file("./fixtures/lib.go")
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
