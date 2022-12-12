# scalacserver - a gRPC frontend to the scalac compiler

## Run Standalone

```sh
bazel run //cmd/scalacserver
```

## Run Standaline with more logging

Change logging level:

```sh
bazel run //cmd/scalacserver -- --jvm_flag='-Dorg.slf4j.simpleLogger.defaultLogLevel=debug'
```