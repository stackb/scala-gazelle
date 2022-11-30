.PHONY: scalaparse
scalaparse:
	bazel run //api/scalaparse:scalaparse_go_compiled_sources.update
	mv api/scalaparse/api/scalaparse/scalaparse*.go api/scalaparse
	rm -rf api/scalaparse/api/

.PHONY: sourceindex
sourceindex:
	bazel run //build/stack/gazelle/scala/sourceindex:sourceindex_go_compiled_sources.update
	mv build/stack/gazelle/scala/sourceindex/build/stack/gazelle/scala/sourceindex/sourceindex.pb.go build/stack/gazelle/scala/sourceindex/
	rm -rf build/stack/gazelle/scala/sourceindex/build

.PHONY: protos
protos: scalaparse sourceindex
	echo "done"