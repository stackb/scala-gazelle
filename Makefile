
.PHONY: parse
parse:
	bazel run //build/stack/gazelle/scala/parse:parse_go_compiled_sources.update
	mv build/stack/gazelle/scala/parse/build/stack/gazelle/scala/parse/*.go build/stack/gazelle/scala/parse/
	rm -rf build/stack/gazelle/scala/parse/build

.PHONY: protos
protos: scalaparse parse
	echo "done"