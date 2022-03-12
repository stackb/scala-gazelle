package main

import (
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/google/go-cmp/cmp"
	"github.com/stackb/scala-gazelle/pkg/index"
)

func TestParseJarFile(t *testing.T) {
	for name, tc := range map[string]struct {
		filename     string
		wantPackages []string
		wantClasses  []string
		wantExtends  map[string]string
		wantErr      error
	}{
		// representative java jar
		"javax.activation": {
			filename:     "cmd/jarindexer/javax.activation-api-1.2.0.jar",
			wantPackages: []string{"javax.activation"},
			wantClasses: []string{
				"javax.activation.MimetypesFileTypeMap",
				"javax.activation.DataHandlerDataSource",
				"javax.activation.UnsupportedDataTypeException",
				"javax.activation.SecuritySupport",
				"javax.activation.FileTypeMap",
				"javax.activation.CommandObject",
				"javax.activation.ObjectDataContentHandler",
				"javax.activation.URLDataSource",
				"javax.activation.DataSource",
				"javax.activation.MimeTypeParseException",
				"javax.activation.CommandInfo",
				"javax.activation.ActivationDataFlavor",
				"javax.activation.CommandMap",
				"javax.activation.FileDataSource",
				"javax.activation.MailcapCommandMap",
				"javax.activation.MimeTypeParameterList",
				"javax.activation.DataHandler",
				"javax.activation.DataContentHandlerFactory",
				"javax.activation.DataSourceDataContentHandler",
				"javax.activation.DataContentHandler",
				"javax.activation.CommandInfo$Beans",
				"javax.activation.MimeType",
			},
			wantExtends: map[string]string{
				"javax.activation.ActivationDataFlavor":         "java.awt.datatransfer.DataFlavor",
				"javax.activation.MailcapCommandMap":            "javax.activation.CommandMap",
				"javax.activation.MimeTypeParseException":       "java.lang.Exception",
				"javax.activation.MimetypesFileTypeMap":         "javax.activation.FileTypeMap",
				"javax.activation.UnsupportedDataTypeException": "java.io.IOException",
			},
		},
		// representative scala jar
		"akka-stream-testkit": {
			filename: "cmd/jarindexer/akka-stream-testkit_2.12-2.6.10.jar",
			wantPackages: []string{
				"akka.stream.testkit",
				"akka.stream.testkit.javadsl",
				"akka.stream.testkit.scaladsl",
			},
			wantClasses: []string{
				"akka.stream.testkit.GraphStageMessages$Failure",
				"akka.stream.testkit.GraphStageMessages$StageFailure",
				"akka.stream.testkit.GraphStageMessages$StageMessage",
				"akka.stream.testkit.GraphStageMessages",
				"akka.stream.testkit.StreamTestKit$CompletedSubscription",
				"akka.stream.testkit.StreamTestKit$FailedSubscription",
				"akka.stream.testkit.StreamTestKit$ProbeSink",
				"akka.stream.testkit.StreamTestKit$ProbeSource",
				"akka.stream.testkit.StreamTestKit$PublisherProbeSubscription",
				"akka.stream.testkit.StreamTestKit",
				"akka.stream.testkit.TestPublisher$CancelSubscription",
				"akka.stream.testkit.TestPublisher$ManualProbe",
				"akka.stream.testkit.TestPublisher$Probe",
				"akka.stream.testkit.TestPublisher$PublisherEvent",
				"akka.stream.testkit.TestPublisher$RequestMore",
				"akka.stream.testkit.TestPublisher$Subscribe",
				"akka.stream.testkit.TestPublisher",
				"akka.stream.testkit.TestSinkStage",
				"akka.stream.testkit.TestSourceStage",
				"akka.stream.testkit.TestSubscriber$ManualProbe",
				"akka.stream.testkit.TestSubscriber$OnError",
				"akka.stream.testkit.TestSubscriber$OnNext",
				"akka.stream.testkit.TestSubscriber$OnSubscribe",
				"akka.stream.testkit.TestSubscriber$Probe",
				"akka.stream.testkit.TestSubscriber$SubscriberEvent",
				"akka.stream.testkit.TestSubscriber",
				"akka.stream.testkit.javadsl.StreamTestKit",
				"akka.stream.testkit.javadsl.TestSink",
				"akka.stream.testkit.javadsl.TestSource",
				"akka.stream.testkit.scaladsl.StreamTestKit",
				"akka.stream.testkit.scaladsl.TestSink",
				"akka.stream.testkit.scaladsl.TestSource",
			},
			wantExtends: map[string]string{
				"akka.stream.testkit.StreamTestKit$ProbeSink":   "akka.stream.impl.SinkModule",
				"akka.stream.testkit.StreamTestKit$ProbeSource": "akka.stream.impl.SourceModule",
				"akka.stream.testkit.TestPublisher$Probe":       "akka.stream.testkit.TestPublisher$ManualProbe",
				"akka.stream.testkit.TestSinkStage":             "akka.stream.stage.GraphStageWithMaterializedValue",
				"akka.stream.testkit.TestSourceStage":           "akka.stream.stage.GraphStageWithMaterializedValue",
				"akka.stream.testkit.TestSubscriber$Probe":      "akka.stream.testkit.TestSubscriber$ManualProbe",
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			jar, err := bazel.Runfile(tc.filename)
			if err != nil {
				t.Fatal(err)
			}
			var got index.JarSpec
			err = parseJarFile(jar, &got)
			if err != nil {
				if tc.wantErr == nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if diff := cmp.Diff(tc.wantErr.Error(), err.Error()); diff != "" {
					t.Errorf("error (-want +got):\n%s", diff)
				}
			} else {
				if diff := cmp.Diff(tc.wantClasses, got.Classes); diff != "" {
					t.Errorf("classes (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(tc.wantPackages, got.Packages); diff != "" {
					t.Errorf("packages (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(tc.wantExtends, got.Extends); diff != "" {
					t.Errorf("extends (-want +got):\n%s", diff)
				}
			}
		})
	}
}
