package crossresolve

const scalaName = "scala"
const cmdGenerate = "generate"
const mavenInstallJsonSimpleExample = `{
	"dependency_tree": {
		"dependencies": [
			{
				"coord": "xml-apis:xml-apis:1.4.01",
				"dependencies": [],
				"directDependencies": [],
				"exclusions": [
					"log4j:log4j"
				],
				"file": "v1/https/repo.maven.apache.org/maven2/xml-apis/xml-apis/1.4.01/xml-apis-1.4.01.jar",
				"mirror_urls": [
					"https://repo.maven.apache.org/maven2/xml-apis/xml-apis/1.4.01/xml-apis-1.4.01.jar",
					"https://omnistac.jfrog.io/artifactory/libs-release/xml-apis/xml-apis/1.4.01/xml-apis-1.4.01.jar"
				],
				"packages": [
					"javax.xml",
					"javax.xml.datatype",
					"javax.xml.namespace",
					"javax.xml.parsers",
					"javax.xml.stream",
					"javax.xml.stream.events",
					"javax.xml.stream.util",
					"javax.xml.transform",
					"javax.xml.transform.dom",
					"javax.xml.transform.sax",
					"javax.xml.transform.stax",
					"javax.xml.transform.stream",
					"javax.xml.validation",
					"javax.xml.xpath",
					"org.apache.xmlcommons",
					"org.w3c.dom",
					"org.w3c.dom.bootstrap",
					"org.w3c.dom.css",
					"org.w3c.dom.events",
					"org.w3c.dom.html",
					"org.w3c.dom.ls",
					"org.w3c.dom.ranges",
					"org.w3c.dom.stylesheets",
					"org.w3c.dom.traversal",
					"org.w3c.dom.views",
					"org.w3c.dom.xpath",
					"org.xml.sax",
					"org.xml.sax.ext",
					"org.xml.sax.helpers"
				],
				"sha256": "a840968176645684bb01aed376e067ab39614885f9eee44abe35a5f20ebe7fad",
				"url": "https://repo.maven.apache.org/maven2/xml-apis/xml-apis/1.4.01/xml-apis-1.4.01.jar"
			}
		],
		"version": "0.1.0"
	}
}`