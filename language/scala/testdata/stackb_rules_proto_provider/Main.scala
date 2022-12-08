package a.b.c

// proto.Customer should require //proto:proto_proto_scala_library
import proto.Customer
// not.Exists should be annotated as unresolved
import not.Exists

object Main {
  def main(args: Array[String]): Unit = {
  }
}
