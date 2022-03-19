package examples.helloworld.greeter

import akka.actor.ActorSystem
import akka.grpc.GrpcClientSettings
import akka.stream.Materializer
import akka.stream.scaladsl.Source
import akka.{Done, NotUsed}
import com.typesafe.scalalogging.LazyLogging
import examples.helloworld.greeter.proto._

import scala.concurrent.Future
import scala.concurrent.duration._
import scala.util.{Failure, Success}

import proto.ApiRequest

object GreeterClient extends LazyLogging {
  implicit val sys = ActorSystem("HelloWorldClient")
  implicit val mat: Materializer = Materializer(sys)
  implicit val ec = sys.dispatcher

  def main(args: Array[String]): Unit = {
    // TODO: JBrucker, Anthony, Parker - Update this to reuse the server config/use service discovery.
    val client = GreeterServiceClient(GrpcClientSettings.fromConfig("helloworld.greeter"))
    if (args.length >= 1 && args(0) == "Monitor") monitorHellos(client, args)
    else if (args.length >= 1 && args(0) == "Broadcast") broadcastHellos(client)
    else if (args.length >= 1 && args(0) == "Snapshot") monitorHellosWithSnapshot(client, args)
    else makeHelloRequests(client, args)
  }

  def monitorHellos(client: GreeterServiceClient, args: Array[String]): Unit = {
    if (args.length == 2) {
      client.monitorHellos(MonitorHelloRequest(Some(args(1)))).runForeach(msg => logger.info(msg.toString))
    } else {
      client.monitorHellos(MonitorHelloRequest()).runForeach(msg => logger.info(msg.toString))
    }
  }

  def monitorHellosWithSnapshot(client: GreeterServiceClient, args: Array[String]): Unit = {
    if (args.length == 2) {
      client.monitorHellosWithSnapshot(MonitorHelloRequest(Some(args(1)))).runForeach(msg => logger.info(msg.toString))
    } else {
      client.monitorHellosWithSnapshot(MonitorHelloRequest()).runForeach(msg => logger.info(msg.toString))
    }
  }

  def broadcastHellos(client: GreeterServiceClient): Unit = {
    val reply = client.broadcastHellos(Source((1 to 10).map(i => HelloRequest(s"Test-$i"))))
    reply.onComplete {
      case Success(msg) =>
        logger.info(msg.toString)
      case Failure(e) =>
        logger.error(s"Error: $e")
    }
  }

  def makeHelloRequests(client: GreeterServiceClient, args: Array[String]) {
    val names =
      if (args.isEmpty) List("Alice", "Bob")
      else args.toList

    names.foreach(singleRequestReply)

    if (args.nonEmpty)
      names.foreach(streamingBroadcast)

    def singleRequestReply(name: String): Unit = {
      logger.info(s"Performing request: $name")
      val reply = client.sayHello(HelloRequest(name))

      reply.onComplete {
        case Success(msg) =>
          logger.info(s"Success: $msg")
        case Failure(e) =>
          logger.error(s"Error: $e")
      }
    }

    def streamingBroadcast(name: String): Unit = {
      logger.info(s"Performing streaming requests: $name")

      val requestStream: Source[HelloRequest, NotUsed] =
        Source
          .tick(1.second, 1.second, "tick")
          .zipWithIndex
          .map { case (_, i) => i }
          .map(i => HelloRequest(s"$name-$i"))
          .mapMaterializedValue(_ => NotUsed)

      val responseStream: Source[HelloReplyMessage, NotUsed] = client.sayHelloToAll(requestStream)
      val done: Future[Done] =
        responseStream.runForeach(message => logger.info(s"$name got streaming reply: $message"))

      done.onComplete {
        case Success(_) =>
          logger.info("streamingBroadcast done")
        case Failure(e) =>
          logger.error(s"Error streamingBroadcast: $e")
      }
    }
  }
}
