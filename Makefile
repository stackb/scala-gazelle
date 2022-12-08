
.PHONY: test
test:
	bazel test //pkg/... //language/scala:scala_test

.PHONY: jarindex_protos
jarindex_protos:
	bazel run //build/stack/gazelle/scala/jarindex:jarindex_go_compiled_sources.update
	mv build/stack/gazelle/scala/jarindex/build/stack/gazelle/scala/jarindex/*.go build/stack/gazelle/scala/jarindex/
	rm -rf build/stack/gazelle/scala/jarindex/build

.PHONY: scalaparse_protos
scalaparse_protos:
	bazel run //build/stack/gazelle/scala/parse:parse_go_compiled_sources.update
	mv build/stack/gazelle/scala/parse/build/stack/gazelle/scala/parse/*.go build/stack/gazelle/scala/parse/
	rm -rf build/stack/gazelle/scala/parse/build

.PHONY: scalacache_protos
scalacache_protos:
	bazel run //build/stack/gazelle/scala/cache:cache_go_compiled_sources.update
	mv build/stack/gazelle/scala/cache/build/stack/gazelle/scala/cache/*.go build/stack/gazelle/scala/cache/
	rm -rf build/stack/gazelle/scala/cache/build

.PHONY: protos
protos: jarindex_protos scalaparse_protos scalacache_protosgit 
	echo "Done."

.PHONY: tidy
gazelle:
	bazel run //:gazelle

.PHONY: tidy
tidy:
	bazel run @go_sdk//:bin/go -- mod tidy
	bazel run //:update_go_repositories

.PHONY: tools
tools:
	go install github.com/vektra/mockery/v2@latest

.PHONY: mocks
mocks:
	mockery --testonly --output pkg/resolver/mocks --dir=pkg/resolver --name=ImportResolver 
	mockery --testonly --output pkg/resolver/mocks --dir=pkg/resolver --name=KnownImportProvider
	mockery --testonly --output pkg/resolver/mocks --dir=pkg/resolver --name=KnownImportRegistry
	mockery --testonly --output pkg/resolver/mocks --dir=pkg/resolver --name=KnownImportResolver
