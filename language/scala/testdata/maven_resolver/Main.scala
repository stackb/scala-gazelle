package a.b.c

// javax.xml._ should require @maven//:xml
import javax.xml._
// TemporaryFolder requires @atlassian-public//:junit_junit, but 
// we didn't specify atlassian-public_install.json for the 
// -pinned_maven_install_json_files, so it should not be removed 
// from the deps.
import org.junit.rules.TemporaryFolder

object Main {
  def main(args: Array[String]): Unit = {
  }
}
