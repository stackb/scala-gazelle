# pkg/provider

The `TestScalaSourceProviderParseScalaFiles` test parses .scala files in the
`testdata/` directory and compares the output to a similarly-named golden file.

To regenerate the golden files, run the test with the `-update` flag:

```
bazel run //pkg/provider:provider_test -- -update
```
