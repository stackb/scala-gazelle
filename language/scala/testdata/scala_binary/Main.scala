package a.b.c

// java.lang.Boolean is in the platformclassspath.jar.  Should not require a dep.
import java.lang.Boolean
// com.google.gson.Gson should require @maven//:com_google_code_gson_gson
import com.google.gson.Gson
// slick.jdbc.PostgresProfile.api._ does not exist in classes.json.  This
// demonstrates that the resolver will try [slick.jdbc.PostgresProfile.api._,
// slick.jdbc.PostgresProfile.api, slick.jdbc.PostgresProfile] until it gets a
// match.
import slick.jdbc.PostgresProfile.api._

object Main {
  def main(args: Array[String]): Unit = {
  }
}
