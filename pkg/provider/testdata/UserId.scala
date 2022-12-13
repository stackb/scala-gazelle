package common.types

/** Scala "Value" class for interacting with UserId values
  * Representative of conceptual ActorId/UserId primary key Int values
  *
  * @param value In general it should not be necessary or desirable to interact directly with this property. Exceptions
  *              being Core Libraries, DB interactions, and legacy code (which should be updated opportunistically).
  */
case class UserId(value: Int) extends AnyVal

object UserId {
  implicit val intTypeMapper: scalapb.TypeMapper[Int, UserId] = scalapb.TypeMapper[Int, UserId](UserId(_))(_.value)
}
