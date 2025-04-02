
.PHONY: test
test:
	bazel test \
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

.PHONY: scalapb_protos
scalapb_protos:
	bazel run //scalapb:scalapb_go_compiled_sources.update
	mv scalapb/scalapb/scalapb.pb.go scalapb/scalapb.pb.go
	rm -rf scalapb/scalapb

.PHONY: semanticdb_protos
semanticdb_protos:
	bazel run //scala/meta/semanticdb:semanticdb_go_compiled_sources.update
	mv scala/meta/semanticdb/scala/meta/semanticdb/semanticdb.pb.go scala/meta/semanticdb/semanticdb.pb.go
	rm -rf scala/meta/semanticdb/scala

.PHONY: autokeep_protos
autokeep_protos:
	bazel run //build/stack/gazelle/scala/autokeep:autokeep_go_compiled_sources.update
	mv build/stack/gazelle/scala/autokeep/build/stack/gazelle/scala/autokeep/autokeep.pb.go build/stack/gazelle/scala/autokeep/autokeep.pb.go
	rm -rf build/stack/gazelle/scala/autokeep/build

.PHONY: worker_protos
worker_protos:
	bazel run //blaze/worker:worker_protocol_go_compiled_sources.update
	mv blaze/worker/blaze/worker/worker_protocol.pb.go blaze/worker/worker_protocol.pb.go
	rm -rf blaze/worker/blaze

.PHONY: protos
protos: jarindex_protos parser_protos scalacache_protos scalapb_protos semanticdb_protos autokeep_protos worker_protos
	echo "Done."

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

.PHONY: goldens
goldens:
	bazel run //pkg/semanticdb:semanticdb_test -- -update
	bazel run pkg/provider:provider_test -- -update
	bazel run pkg/parser:parser_test -- -update
