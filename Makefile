
.PHONY: test
test:
	bazel test \
		--keep_going \
		//cmd/jarindexer/... \
		//cmd/mergeindex/... \
		//language/scala/... \
		//pkg/... \

.PHONY: jarindex_protos
jarindex_protos:
	bazel run //build/stack/gazelle/scala/jarindex:jarindex_go_compiled_sources.update
	mv build/stack/gazelle/scala/jarindex/build/stack/gazelle/scala/jarindex/*.go build/stack/gazelle/scala/jarindex/
	rm -rf build/stack/gazelle/scala/jarindex/build

.PHONY: parser_protos
parser_protos:
	bazel run //build/stack/gazelle/scala/parse:parse_go_compiled_sources.update
	mv build/stack/gazelle/scala/parse/build/stack/gazelle/scala/parse/*.go build/stack/gazelle/scala/parse/
	rm -rf build/stack/gazelle/scala/parse/build

.PHONY: scalacache_protos
scalacache_protos:
	bazel run //build/stack/gazelle/scala/cache:cache_go_compiled_sources.update
	mv build/stack/gazelle/scala/cache/build/stack/gazelle/scala/cache/*.go build/stack/gazelle/scala/cache/
	rm -rf build/stack/gazelle/scala/cache/build

.PHONY: autokeep_protos
autokeep_protos:
	bazel run //build/stack/gazelle/scala/autokeep:autokeep_go_compiled_sources.update
	mv build/stack/gazelle/scala/autokeep/build/stack/gazelle/scala/autokeep/*.go build/stack/gazelle/scala/autokeep/
	rm -rf build/stack/gazelle/scala/autokeep/build

.PHONY: protos
protos: jarindex_protos parser_protos scalacache_protos autokeep_protos
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
	mockery --output pkg/resolver/mocks --dir=pkg/resolver --name=Universe 
	mockery --output pkg/resolver/mocks --dir=pkg/resolver --name=Scope
	mockery --output pkg/resolver/mocks --dir=pkg/resolver --name=SymbolProvider
	mockery --output pkg/resolver/mocks --dir=pkg/resolver --name=SymbolResolver
	mockery --output pkg/resolver/mocks --dir=pkg/resolver --name=ConflictResolver
	mockery --output pkg/parser/mocks --dir=pkg/parser --name=Parser
	mockery --output pkg/scalarule/mocks --dir=pkg/scalarule --name=ProviderRegistry

.PHONY: gen
gen: mocks protos

update_provider_test_golden_files:
	bazel run pkg/provider:provider_test -- -update
