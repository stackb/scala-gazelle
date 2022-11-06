"""providers.bzl contains Providers instances that are public.
"""

JarIndexerAspectInfo = provider(
    "a provider for Jar Indexing",
    fields = {
        "info_file": "The index database",
        "jar_index_files": "A list of jar index files",
    },
)
