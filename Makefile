
.PHONY: test
test:
	bazel test \
		//cmd/jarindexer/... \
		//cmd/mergeindex/... \
		//language/scala/... \
		//pkg/... \

.PHONY: protos
protos:
	bazel run //:proto_compile_assets

.PHONY: docs
docs:
	bazel build //docs/architecture:all
	cp -f bazel-bin/docs/architecture/components.png docs/architecture
	cp -f bazel-bin/docs/architecture/sequence.png docs/architecture

.PHONY: tidy
gazelle:
	bazel run //:gazelle

.PHONY: tidy
tidy:
	bazel run @go_sdk//:bin/go -- mod tidy
	bazel mod tidy

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

.PHONY: goldens
goldens:
	bazel run //pkg/semanticdb:semanticdb_test -- -update
	bazel run pkg/provider:provider_test -- -update
	bazel run pkg/parser:parser_test -- -update
