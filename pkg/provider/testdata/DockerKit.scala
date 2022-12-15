package container.testing

import com.github.dockerjava.api.DockerClient
import com.github.dockerjava.api.command.CreateContainerResponse
import com.github.dockerjava.api.exception.NotFoundException
import com.github.dockerjava.api.model.Frame
import com.github.dockerjava.api.async.ResultCallback
import com.github.dockerjava.core.{DefaultDockerClientConfig, DockerClientBuilder}
import com.typesafe.scalalogging.LazyLogging

import scala.collection.JavaConverters._
import scala.util.Try

object DockerKit extends TestConfig with LazyLogging {
  lazy val dockerConfig = DefaultDockerClientConfig.createDefaultConfigBuilder
    .withDockerTlsVerify(tlsEnabled)
    .withDockerHost(dockerUrl)
    .build()
  lazy val dockerClient = {
    logger.info("Building docker client with dockerUrl=" + dockerUrl)
    DockerClientBuilder.getInstance(dockerConfig).build()
  }
}

trait DockerKit extends TestConfig with LazyLogging {

  def containerName: String

  def createContainer: CreateContainerResponse

  def dockerClient: DockerClient = DockerKit.dockerClient

  private def containers = dockerClient.listContainersCmd().withShowAll(true).exec().asScala.toList

  def containerId: String = {
    containers
      .filter(_.getNames.contains(s"/$containerName"))
      .find { container =>
        dockerClient.inspectContainerCmd(container.getId).exec.getState.getRunning
      }
      .map(_.getId)
      .getOrElse {
        cleanupContainer()
        val container = createContainer
        dockerClient.startContainerCmd(container.getId).exec()
        container.getId
      }
  }

  def cleanupContainer(): Unit = {
    containers.filter(_.getNames.contains(s"/$containerName")).foreach { c =>
      Try(dockerClient.killContainerCmd(c.getId).exec())
      Try(dockerClient.removeContainerCmd(c.getId).withForce(true).withRemoveVolumes(true).exec())
      (0 to 300).exists { _ =>
        if (!containers.exists(_.getId == c.getId)) {
          true
        } else {
          Thread.sleep(100)
          false
        }
      }
    }
  }

  def isContainerRunning: Boolean = containers.filter(_.getNames.contains(s"/$containerName")).exists { container =>
    dockerClient.inspectContainerCmd(container.getId).exec.getState.getRunning
  }

  def runContainer(): String = containerId

  private def containerLog(id: String): String = {
    dockerClient
      .logContainerCmd(id)
      .withStdOut(true)
      .withStdErr(true)
      .withTail(10)
      .exec(LoggingCallback())
      .awaitCompletion()
      .toString
  }

  def waitForStringInLogs(stringToWaitFor: String, milliSecondsToWait: Int): Boolean =
    try {
      val startTime = System.currentTimeMillis
      var found = false // scalastyle:ignore var.local
      var count = 0 // scalastyle:ignore var.local
      val id = containerId
      while (!found && System.currentTimeMillis - startTime < milliSecondsToWait) {
        if (count < 100) {
          logger.debug("checking>>> " + containerLog(id))
        } else if (count < 600) {
          logger.warn(containerLog(id))
        } else {
          logger.error(containerLog(id))
        }
        found = containerLog(id).contains(stringToWaitFor)
        if (!found) {
          count += 1
          Thread.sleep(100)
        }
      }
      found
    } catch {
      case _: NotFoundException =>
        false
    }

  def restart(): Unit = {
    dockerClient.restartContainerCmd(containerId).exec()
  }

  def kill(): Unit = {
    dockerClient.killContainerCmd(containerId).exec()
  }

  def stop(): Unit = {
    containers.filter(_.getNames.contains(s"/$containerName")).foreach { c =>
      try {
        logger.debug(s"Tests completed. Stopping containerName=$containerName -- $c")
        dockerClient.removeContainerCmd(c.getId).withForce(true).withRemoveVolumes(true).exec()
        logger.debug("Stopped " + containerName)
      } catch {
        case _: Throwable =>
          logger.info("Unable to stop container: " + containerName)
      }
    }
  }
}

case class LoggingCallback() extends ResultCallback.Adapter[Frame] {
  val log = new StringBuffer()

  override def onNext(frame: Frame): Unit = {
    log.append(new String(frame.getPayload()))
    super.onNext(frame)
  }

  override def toString(): String = log.toString
}
