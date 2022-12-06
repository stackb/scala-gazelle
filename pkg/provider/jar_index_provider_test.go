package provider

import (
	"flag"
	"os"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/testtools"
	"github.com/google/go-cmp/cmp"
	"github.com/stackb/scala-gazelle/pkg/testutil"
)

func ExampleJarIndexProvider_RegisterFlags_printdefaults() {
	os.Stderr = os.Stdout
	cr := NewJarIndexProvider()
	got := flag.NewFlagSet(scalaName, flag.ExitOnError)
	c := &config.Config{}
	cr.RegisterFlags(got, "update", c)
	got.PrintDefaults()
	// output:
	//	-jarindex_file value
	//     	path to jarindex.pb or jarindex.json file
}

func TestJarIndexProvider(t *testing.T) {
	for name, tc := range map[string]struct {
		args  []string
		files []testtools.FileSpec
		want  []string
	}{
		"empty file": {
			args: []string{
				"-jarindex_file=./jarindex.json",
			},
			files: []testtools.FileSpec{
				{
					Path:    "jarindex.json",
					Content: "{}",
				},
			},
			want: []string{},
		},
		"example jarindex file": {
			args: []string{
				"-jarindex_file=./testdata/jarindex.json",
			},
			files: []testtools.FileSpec{
				{
					Path: "testdata/jarindex.json",
				},
			},
			want: []string{
				"PACKAGE com.google.gson (@maven//:com_google_code_gson_gson)",
				"PACKAGE com.google.gson.annotations (@maven//:com_google_code_gson_gson)",
				"PACKAGE com.google.gson.internal (@maven//:com_google_code_gson_gson)",
				"PACKAGE com.google.gson.internal.bind (@maven//:com_google_code_gson_gson)",
				"PACKAGE com.google.gson.internal.bind.util (@maven//:com_google_code_gson_gson)",
				"PACKAGE com.google.gson.internal.reflect (@maven//:com_google_code_gson_gson)",
				"PACKAGE com.google.gson.internal.sql (@maven//:com_google_code_gson_gson)",
				"PACKAGE com.google.gson.reflect (@maven//:com_google_code_gson_gson)",
				"PACKAGE com.google.gson.stream (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.ExclusionStrategy (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.FieldAttributes (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.FieldNamingPolicy (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.FieldNamingPolicy$1 (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.FieldNamingPolicy$2 (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.FieldNamingPolicy$3 (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.FieldNamingPolicy$4 (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.FieldNamingPolicy$5 (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.FieldNamingPolicy$6 (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.FieldNamingStrategy (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.Gson (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.Gson$1 (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.Gson$2 (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.Gson$3 (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.Gson$4 (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.Gson$5 (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.Gson$FutureTypeAdapter (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.GsonBuilder (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.InstanceCreator (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.JsonArray (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.JsonDeserializationContext (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.JsonDeserializer (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.JsonElement (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.JsonIOException (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.JsonNull (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.JsonObject (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.JsonParseException (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.JsonParser (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.JsonPrimitive (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.JsonSerializationContext (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.JsonSerializer (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.JsonStreamParser (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.JsonSyntaxException (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.LongSerializationPolicy (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.LongSerializationPolicy$1 (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.LongSerializationPolicy$2 (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.ToNumberPolicy (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.ToNumberPolicy$1 (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.ToNumberPolicy$2 (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.ToNumberPolicy$3 (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.ToNumberPolicy$4 (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.ToNumberStrategy (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.TypeAdapter (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.TypeAdapter$1 (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.TypeAdapterFactory (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.annotations.Expose (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.annotations.JsonAdapter (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.annotations.SerializedName (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.annotations.Since (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.annotations.Until (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.internal.$Gson$Preconditions (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.internal.$Gson$Types (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.internal.$Gson$Types$GenericArrayTypeImpl (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.internal.$Gson$Types$ParameterizedTypeImpl (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.internal.$Gson$Types$WildcardTypeImpl (@maven//:com_google_code_gson_gson)",
				"INTERFACE com.google.gson.ExclusionStrategy (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.FieldAttributes (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.FieldNamingPolicy (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.FieldNamingPolicy$1 (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.FieldNamingPolicy$2 (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.FieldNamingPolicy$3 (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.FieldNamingPolicy$4 (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.FieldNamingPolicy$5 (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.FieldNamingPolicy$6 (@maven//:com_google_code_gson_gson)",
				"INTERFACE com.google.gson.FieldNamingStrategy (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.Gson (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.Gson$1 (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.Gson$2 (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.Gson$3 (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.Gson$4 (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.Gson$5 (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.Gson$FutureTypeAdapter (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.GsonBuilder (@maven//:com_google_code_gson_gson)",
				"INTERFACE com.google.gson.InstanceCreator (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.JsonArray (@maven//:com_google_code_gson_gson)",
				"INTERFACE com.google.gson.JsonDeserializationContext (@maven//:com_google_code_gson_gson)",
				"INTERFACE com.google.gson.JsonDeserializer (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.JsonElement (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.JsonIOException (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.JsonNull (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.JsonObject (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.JsonParseException (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.JsonParser (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.JsonPrimitive (@maven//:com_google_code_gson_gson)",
				"INTERFACE com.google.gson.JsonSerializationContext (@maven//:com_google_code_gson_gson)",
				"INTERFACE com.google.gson.JsonSerializer (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.JsonStreamParser (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.JsonSyntaxException (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.LongSerializationPolicy (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.LongSerializationPolicy$1 (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.LongSerializationPolicy$2 (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.ToNumberPolicy (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.ToNumberPolicy$1 (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.ToNumberPolicy$2 (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.ToNumberPolicy$3 (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.ToNumberPolicy$4 (@maven//:com_google_code_gson_gson)",
				"INTERFACE com.google.gson.ToNumberStrategy (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.TypeAdapter (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.TypeAdapter$1 (@maven//:com_google_code_gson_gson)",
				"INTERFACE com.google.gson.TypeAdapterFactory (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.annotations.Expose (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.annotations.JsonAdapter (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.annotations.SerializedName (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.annotations.Since (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.annotations.Until (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.internal.$Gson$Preconditions (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.internal.$Gson$Types (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.internal.$Gson$Types$GenericArrayTypeImpl (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.internal.$Gson$Types$ParameterizedTypeImpl (@maven//:com_google_code_gson_gson)",
				"CLASS com.google.gson.internal.$Gson$Types$WildcardTypeImpl (@maven//:com_google_code_gson_gson)",
				"PACKAGE java.util (//:)",
				"CLASS java.util.Map (//:)",
				"CLASS java.util.Map$Entry (//:)",
				"CLASS java.util.MissingFormatArgumentException (//:)",
				"INTERFACE java.util.Map (//:)",
				"INTERFACE java.util.Map$Entry (//:)",
				"CLASS java.util.MissingFormatArgumentException (//:)",
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			tmpDir, _, cleanup := testutil.MustReadAndPrepareTestFiles(t, tc.files)
			defer cleanup()

			p := NewJarIndexProvider()
			fs := flag.NewFlagSet(scalaName, flag.ExitOnError)
			c := &config.Config{
				WorkDir: tmpDir,
			}
			p.RegisterFlags(fs, "update", c)
			if err := fs.Parse(tc.args); err != nil {
				t.Fatal(err)
			}

			importRegistry := &mockKnownImportRegistry{}

			if err := p.CheckFlags(fs, c, importRegistry); err != nil {
				t.Fatal(err)
			}

			got := make([]string, len(importRegistry.got))
			for i, known := range importRegistry.got {
				got[i] = known.String()
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}

// func TestJarIndexProviderCanProvide(t *testing.T) {
// 	for name, tc := range map[string]struct {
// 		mavenInstallJsonContent string
// 		lang                    string
// 		from                    label.Label
// 		want                    bool
// 	}{
// 		"degenerate case": {
// 			mavenInstallJsonContent: mavenInstallJsonExample,
// 			lang:                    scalaName,
// 			from:                    label.NoLabel,
// 			want:                    false,
// 		},
// 		"managed xml_apis_xml_apis": {
// 			mavenInstallJsonContent: mavenInstallJsonExample,
// 			lang:                    scalaName,
// 			from:                    label.New("maven", "", "xml_apis_xml_apis"),
// 			want:                    true,
// 		},
// 		"managed generic maven dependency": {
// 			mavenInstallJsonContent: mavenInstallJsonExample,
// 			lang:                    scalaName,
// 			from:                    label.New("maven", "", "com_guava_guava"),
// 			want:                    true,
// 		},
// 		"unmanaged non-maven dependency": {
// 			mavenInstallJsonContent: mavenInstallJsonExample,
// 			lang:                    scalaName,
// 			from:                    label.New("artifactory", "", "xml_apis_xml_apis"),
// 			want:                    false,
// 		},
// 	} {
// 		t.Run(name, func(t *testing.T) {
// 			tmpDir, _, cleanup := testutil.MustPrepareTestFiles(t, []testtools.FileSpec{
// 				{
// 					Path:    "jarindex.json",
// 					Content: tc.mavenInstallJsonContent,
// 				},
// 			})
// 			defer cleanup()

// 			p := NewJarIndexProvider(scalaName)
// 			fs := flag.NewFlagSet(scalaName, flag.ExitOnError)
// 			c := &config.Config{WorkDir: tmpDir}
// 			p.RegisterFlags(fs, "update", c)
// 			if err := fs.Parse([]string{
// 				"-jarindex_file=./jarindex.json",
// 			}); err != nil {
// 				t.Fatal(err)
// 			}

// 			importRegistry := &mockKnownImportRegistry{}

// 			if err := p.CheckFlags(fs, c, importRegistry); err != nil {
// 				t.Fatal(err)
// 			}

// 			got := p.CanProvide(tc.from, func(from label.Label) (*rule.Rule, bool) {
// 				return nil, false
// 			})

// 			if diff := cmp.Diff(tc.want, got); diff != "" {
// 				t.Errorf(".CanProvide (-want +got):\n%s", diff)
// 			}
// 		})
// 	}
// }
