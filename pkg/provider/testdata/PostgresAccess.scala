package auth.dao

object PostgresAccess extends common.postgres.PostgresAccess {
  override def dbConfigKey: String = "auth"
}
