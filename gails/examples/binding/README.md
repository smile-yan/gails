# Bindings Example

This example demonstrates how to generate bindings for your application.

To generate bindings, run `gails3 generate bindings -clean -b -d assets/bindings` in this directory.

See more options by running `gails3 generate bindings --help`.

## Notes
  - The bindings generator is still a work in progress and is subject to change.
  - The generated code uses `gails.CallByID` by default. This is the most robust way to call a function.
