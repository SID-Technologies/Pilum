// Stub for the gcp-image-publish fixture. Pilum's `build binary` step
// expects a Go source to compile (or at least a directory to chdir into);
// this file makes the directory a valid Go package even though the test
// recipe mocks the actual build invocation.
package main

func main() {}
