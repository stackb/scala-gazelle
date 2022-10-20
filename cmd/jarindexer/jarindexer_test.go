package main

import (
	"strings"
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
		wantFiles    []*index.ClassFileSpec
		wantExtends  map[string]string
		wantErr      error
	}{
		// representative java jar
		"skip javax.activation": {
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
		"skip akka-stream-testkit": {
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
		"akka-grpc-runtime_2.12-1.0.1.jar": {
			filename: "cmd/jarindexer/akka-grpc-runtime_2.12-1.0.1.jar",
			wantPackages: []string{
				"akka.grpc",
				"akka.grpc.internal",
				"akka.grpc.javadsl",
				"akka.grpc.scaladsl",
				"akka.grpc.scaladsl.headers",
				"grpc.reflection.v1alpha.reflection",
			},
			wantClasses: []string{
				"akka.grpc.GrpcClientSettings",
				"akka.grpc.GrpcProtocol$DataFrame",
				"akka.grpc.GrpcProtocol$Frame",
				"akka.grpc.GrpcProtocol$GrpcProtocolReader",
				"akka.grpc.GrpcProtocol$GrpcProtocolWriter",
				"akka.grpc.GrpcProtocol$TrailerFrame",
				"akka.grpc.GrpcProtocol",
				"akka.grpc.GrpcResponseMetadata",
				"akka.grpc.GrpcServiceException",
				"akka.grpc.GrpcSingleResponse",
				"akka.grpc.ProtobufSerializer",
				"akka.grpc.SSLContextUtils",
				"akka.grpc.ServiceDescription",
				"akka.grpc.Trailers",
				"akka.grpc.internal.AbstractGrpcProtocol$GrpcFramingDecoderStage$$anon$1$ReadFrame",
				"akka.grpc.internal.AbstractGrpcProtocol$GrpcFramingDecoderStage$$anon$1$Step",
				"akka.grpc.internal.AbstractGrpcProtocol$GrpcFramingDecoderStage",
				"akka.grpc.internal.AbstractGrpcProtocol",
				"akka.grpc.internal.AkkaDiscoveryNameResolver",
				"akka.grpc.internal.AkkaDiscoveryNameResolverProvider",
				"akka.grpc.internal.AkkaNettyGrpcClientGraphStage$Closed",
				"akka.grpc.internal.AkkaNettyGrpcClientGraphStage$ControlMessage",
				"akka.grpc.internal.AkkaNettyGrpcClientGraphStage",
				"akka.grpc.internal.CancellationBarrierGraphStage",
				"akka.grpc.internal.ChannelUtils",
				"akka.grpc.internal.ClientClosedException",
				"akka.grpc.internal.ClientConnectionException",
				"akka.grpc.internal.ClientState",
				"akka.grpc.internal.Codec",
				"akka.grpc.internal.Codecs",
				"akka.grpc.internal.EntryMetadataImpl",
				"akka.grpc.internal.GrpcEntityHelpers",
				"akka.grpc.internal.GrpcMetadataImpl",
				"akka.grpc.internal.GrpcProtocolNative",
				"akka.grpc.internal.GrpcProtocolWeb",
				"akka.grpc.internal.GrpcProtocolWebBase",
				"akka.grpc.internal.GrpcProtocolWebText",
				"akka.grpc.internal.GrpcRequestHelpers",
				"akka.grpc.internal.GrpcResponseHelpers",
				"akka.grpc.internal.Gzip",
				"akka.grpc.internal.HardcodedServiceDiscovery",
				"akka.grpc.internal.HeaderMetadataImpl",
				"akka.grpc.internal.Identity",
				"akka.grpc.internal.InternalChannel",
				"akka.grpc.internal.JavaBidirectionalStreamingRequestBuilder",
				"akka.grpc.internal.JavaClientStreamingRequestBuilder",
				"akka.grpc.internal.JavaMetadataImpl",
				"akka.grpc.internal.JavaServerStreamingRequestBuilder",
				"akka.grpc.internal.JavaUnaryRequestBuilder",
				"akka.grpc.internal.Marshaller",
				"akka.grpc.internal.MetadataImpl",
				"akka.grpc.internal.MetadataOperations",
				"akka.grpc.internal.MissingParameterException",
				"akka.grpc.internal.NettyClientUtils",
				"akka.grpc.internal.ProtoMarshaller",
				"akka.grpc.internal.ScalaBidirectionalStreamingRequestBuilder",
				"akka.grpc.internal.ScalaClientStreamingRequestBuilder",
				"akka.grpc.internal.ScalaServerStreamingRequestBuilder",
				"akka.grpc.internal.ScalaUnaryRequestBuilder",
				"akka.grpc.internal.ServerReflectionImpl",
				"akka.grpc.internal.ServiceDescriptionImpl",
				"akka.grpc.internal.UnaryCallAdapter",
				"akka.grpc.internal.UnaryCallWithMetadataAdapter",
				"akka.grpc.javadsl.AkkaGrpcClient",
				"akka.grpc.javadsl.BytesEntry",
				"akka.grpc.javadsl.GoogleProtobufSerializer",
				"akka.grpc.javadsl.GrpcExceptionHandler",
				"akka.grpc.javadsl.GrpcMarshalling",
				"akka.grpc.javadsl.Metadata",
				"akka.grpc.javadsl.MetadataBuilder",
				"akka.grpc.javadsl.MetadataEntry",
				"akka.grpc.javadsl.RouteUtils",
				"akka.grpc.javadsl.ServerReflection",
				"akka.grpc.javadsl.ServiceHandler",
				"akka.grpc.javadsl.SingleResponseRequestBuilder",
				"akka.grpc.javadsl.StreamResponseRequestBuilder",
				"akka.grpc.javadsl.StringEntry",
				"akka.grpc.javadsl.WebHandler",
				"akka.grpc.scaladsl.AkkaGrpcClient",
				"akka.grpc.scaladsl.BytesEntry",
				"akka.grpc.scaladsl.GrpcExceptionHandler",
				"akka.grpc.scaladsl.GrpcMarshalling",
				"akka.grpc.scaladsl.Metadata",
				"akka.grpc.scaladsl.MetadataBuilder",
				"akka.grpc.scaladsl.MetadataEntry",
				"akka.grpc.scaladsl.ScalapbProtobufSerializer",
				"akka.grpc.scaladsl.ServerReflection",
				"akka.grpc.scaladsl.ServiceHandler",
				"akka.grpc.scaladsl.SingleResponseRequestBuilder",
				"akka.grpc.scaladsl.StreamResponseRequestBuilder",
				"akka.grpc.scaladsl.StringEntry",
				"akka.grpc.scaladsl.WebHandler",
				"akka.grpc.scaladsl.headers.Message$minusAccept$minusEncoding",
				"akka.grpc.scaladsl.headers.Message$minusEncoding",
				"akka.grpc.scaladsl.headers.Status$minusMessage",
				"akka.grpc.scaladsl.headers.Status",
				"grpc.reflection.v1alpha.reflection.DefaultServerReflectionClient",
				"grpc.reflection.v1alpha.reflection.ErrorResponse$ErrorResponseLens",
				"grpc.reflection.v1alpha.reflection.ErrorResponse",
				"grpc.reflection.v1alpha.reflection.ExtensionNumberResponse$ExtensionNumberResponseLens",
				"grpc.reflection.v1alpha.reflection.ExtensionNumberResponse",
				"grpc.reflection.v1alpha.reflection.ExtensionRequest$ExtensionRequestLens",
				"grpc.reflection.v1alpha.reflection.ExtensionRequest",
				"grpc.reflection.v1alpha.reflection.FileDescriptorResponse$FileDescriptorResponseLens",
				"grpc.reflection.v1alpha.reflection.FileDescriptorResponse",
				"grpc.reflection.v1alpha.reflection.ListServiceResponse$ListServiceResponseLens",
				"grpc.reflection.v1alpha.reflection.ListServiceResponse",
				"grpc.reflection.v1alpha.reflection.ReflectionProto",
				"grpc.reflection.v1alpha.reflection.ServerReflection",
				"grpc.reflection.v1alpha.reflection.ServerReflectionClient",
				"grpc.reflection.v1alpha.reflection.ServerReflectionClientPowerApi",
				"grpc.reflection.v1alpha.reflection.ServerReflectionHandler",
				"grpc.reflection.v1alpha.reflection.ServerReflectionMarshallers",
				"grpc.reflection.v1alpha.reflection.ServerReflectionRequest$MessageRequest$AllExtensionNumbersOfType",
				"grpc.reflection.v1alpha.reflection.ServerReflectionRequest$MessageRequest$FileByFilename",
				"grpc.reflection.v1alpha.reflection.ServerReflectionRequest$MessageRequest$FileContainingExtension",
				"grpc.reflection.v1alpha.reflection.ServerReflectionRequest$MessageRequest$FileContainingSymbol",
				"grpc.reflection.v1alpha.reflection.ServerReflectionRequest$MessageRequest$ListServices",
				"grpc.reflection.v1alpha.reflection.ServerReflectionRequest$MessageRequest",
				"grpc.reflection.v1alpha.reflection.ServerReflectionRequest$ServerReflectionRequestLens",
				"grpc.reflection.v1alpha.reflection.ServerReflectionRequest",
				"grpc.reflection.v1alpha.reflection.ServerReflectionResponse$MessageResponse$AllExtensionNumbersResponse",
				"grpc.reflection.v1alpha.reflection.ServerReflectionResponse$MessageResponse$ErrorResponse",
				"grpc.reflection.v1alpha.reflection.ServerReflectionResponse$MessageResponse$FileDescriptorResponse",
				"grpc.reflection.v1alpha.reflection.ServerReflectionResponse$MessageResponse$ListServicesResponse",
				"grpc.reflection.v1alpha.reflection.ServerReflectionResponse$MessageResponse",
				"grpc.reflection.v1alpha.reflection.ServerReflectionResponse$ServerReflectionResponseLens",
				"grpc.reflection.v1alpha.reflection.ServerReflectionResponse",
				"grpc.reflection.v1alpha.reflection.ServiceResponse$ServiceResponseLens",
				"grpc.reflection.v1alpha.reflection.ServiceResponse",
			},
			wantExtends: map[string]string{
				"akka.grpc.GrpcServiceException":                                                           "java.lang.RuntimeException",
				"akka.grpc.internal.AbstractGrpcProtocol$GrpcFramingDecoderStage":                          "akka.stream.impl.io.ByteStringParser",
				"akka.grpc.internal.AkkaDiscoveryNameResolver":                                             "io.grpc.NameResolver",
				"akka.grpc.internal.AkkaDiscoveryNameResolverProvider":                                     "io.grpc.NameResolverProvider",
				"akka.grpc.internal.AkkaNettyGrpcClientGraphStage":                                         "akka.stream.stage.GraphStageWithMaterializedValue",
				"akka.grpc.internal.CancellationBarrierGraphStage":                                         "akka.stream.stage.GraphStage",
				"akka.grpc.internal.ClientClosedException":                                                 "java.lang.RuntimeException",
				"akka.grpc.internal.ClientConnectionException":                                             "java.lang.RuntimeException",
				"akka.grpc.internal.GrpcProtocolWebBase":                                                   "akka.grpc.internal.AbstractGrpcProtocol",
				"akka.grpc.internal.HardcodedServiceDiscovery":                                             "akka.discovery.ServiceDiscovery",
				"akka.grpc.internal.MissingParameterException":                                             "java.lang.Throwable",
				"akka.grpc.internal.UnaryCallAdapter":                                                      "io.grpc.ClientCall$Listener",
				"akka.grpc.internal.UnaryCallWithMetadataAdapter":                                          "io.grpc.ClientCall$Listener",
				"akka.grpc.scaladsl.headers.Message$minusAccept$minusEncoding":                             "akka.http.scaladsl.model.headers.ModeledCustomHeader",
				"akka.grpc.scaladsl.headers.Message$minusEncoding":                                         "akka.http.scaladsl.model.headers.ModeledCustomHeader",
				"akka.grpc.scaladsl.headers.Status":                                                        "akka.http.scaladsl.model.headers.ModeledCustomHeader",
				"akka.grpc.scaladsl.headers.Status$minusMessage":                                           "akka.http.scaladsl.model.headers.ModeledCustomHeader",
				"grpc.reflection.v1alpha.reflection.ErrorResponse$ErrorResponseLens":                       "scalapb.lenses.ObjectLens",
				"grpc.reflection.v1alpha.reflection.ExtensionNumberResponse$ExtensionNumberResponseLens":   "scalapb.lenses.ObjectLens",
				"grpc.reflection.v1alpha.reflection.ExtensionRequest$ExtensionRequestLens":                 "scalapb.lenses.ObjectLens",
				"grpc.reflection.v1alpha.reflection.FileDescriptorResponse$FileDescriptorResponseLens":     "scalapb.lenses.ObjectLens",
				"grpc.reflection.v1alpha.reflection.ListServiceResponse$ListServiceResponseLens":           "scalapb.lenses.ObjectLens",
				"grpc.reflection.v1alpha.reflection.ServerReflectionRequest$ServerReflectionRequestLens":   "scalapb.lenses.ObjectLens",
				"grpc.reflection.v1alpha.reflection.ServerReflectionResponse$ServerReflectionResponseLens": "scalapb.lenses.ObjectLens",
				"grpc.reflection.v1alpha.reflection.ServiceResponse$ServiceResponseLens":                   "scalapb.lenses.ObjectLens",
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			if strings.HasPrefix(name, "skip") {
				return
			}
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
				if diff := cmp.Diff(tc.wantFiles, got.Files); diff != "" {
					t.Errorf("extends (-want +got):\n%s", diff)
				}
			}
		})
	}
}
