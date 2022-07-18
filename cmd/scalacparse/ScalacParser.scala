package scalaparse

import scala.tools.nsc.{Global, Settings, MainClass}
import scala.util.{Failure, Success}

object ScalacParser{
//   var current = Thread.currentThread().getContextClassLoader
//   val files = collection.mutable.Buffer.empty[java.io.File]
// //   files.appendAll(
// //     System.getProperty("sun.boot.class.path")
// //       .split(":")
// //       .map(new java.io.File(_))
// //   )
//   while (current != null) {
//     current match{
//       case t: java.net.URLClassLoader =>
//         files.appendAll(t.getURLs.map(u => new java.io.File(u.toURI)))
//       case _ =>
//     }
//     current = current.getParent
//   }

  val settings = new Settings()
  settings.usejavacp.value = true
  settings.Xprint.value = List("global")
  settings.printLate.value = true
  settings.Yshowsyms.value = true
  settings.Xshowtrees.value = true
  settings.embeddedDefaults[ScalacParser.type]
  // settings.classpath.append(files.mkString(":"))

  val global = new Global(settings)

  // def checkParseFails(input: String) = this.synchronized{
  //   val run = new global.Run()
  //   var fail = false
  //   import global.syntaxAnalyzer.Offset
  //   val cu = new global.CompilationUnit(global.newSourceFile(input))
  //   val parser = new global.syntaxAnalyzer.UnitParser(cu, Nil){
  //     override def newScanner() = new global.syntaxAnalyzer.UnitScanner(cu, Nil){
  //       override def error(off: Offset, msg: String) = {
  //         println(s"scanner error: $msg (offset=$off)")
  //         fail = true
  //       }
  //       override def syntaxError(off: Offset, msg: String) = {
  //         println(s"scanner syntax error: $msg (offset=$off)")
  //         fail = true
  //       }
  //       override def incompleteInputError(off: Offset, msg: String) = {
  //         println(s"scanner incomplete input error: $msg (offset=$off)")
  //         fail = true
  //       }
  //     }
  //     override def incompleteInputError(msg: String) = {
  //         println(s"parser incomplete input error: $msg")
  //       fail = true
  //     }
  //     override def syntaxError(off: Offset, msg: String) = {
  //         println(s"parser syntax error: $msg (offset=$off)")
  //       fail = true
  //     }
  //   }
  //   parser.parse()
  //   fail
  // }

  def main(args: Array[String]): Unit = {
    val main = new MainClass
    System.exit(if (main.process(args)) 0 else 1)
  }

}