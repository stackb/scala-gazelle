package a.b.c

// java.lang.Boolean is in the platformclassspath.jar.  Should not require a dep.
import java.lang.Boolean
// javax.xml._ should require @maven//:xml
import javax.xml._

object Main {
  def main(args: Array[String]): Unit = {
  }
}
