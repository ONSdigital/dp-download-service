job "dp-download-service" {
  datacenters = ["eu-west-1"]
  region      = "eu"
  type        = "service" 

  group "web" {
    count = "{{WEB_TASK_COUNT}}"

    constraint {
      attribute = "${node.class}"
      value     = "web"
    }

    task "dp-download-service-web" {
      driver = "exec"

      artifact {
        source = "s3::https://s3-eu-west-1.amazonaws.com/{{BUILD_BUCKET}}/dp-download-service/{{REVISION}}.tar.gz"
      }

      artifact {
        source = "s3::https://s3-eu-west-1.amazonaws.com/{{DEPLOYMENT_BUCKET}}/dp-download-service/{{REVISION}}.tar.gz"
      }

      config {
        command = "${NOMAD_TASK_DIR}/start-task"
        args    = [
          "${NOMAD_TASK_DIR}/dp-download-service",
        ]
      }

      service {
        name = "dp-download-service"
        port = "http"
        tags = ["web"]
        check {
          type     = "http"
          path     = "/healthcheck"
          interval = "10s"
          timeout  = "2s"
        }
      }

      resources {
        cpu    = "{{WEB_RESOURCE_CPU}}"
        memory = "{{WEB_RESOURCE_MEM}}"

        network {
          port "http" {}
        }
      }

      template {
        source      = "${NOMAD_TASK_DIR}/vars-template"
        destination = "${NOMAD_TASK_DIR}/vars"
      }

      vault {
        policies = ["dp-download-service-web"]

        change_mode   = "signal"
        change_signal = "SIGUSR1"
      }
    }
  }

  group "publishing" {
    count = "{{PUBLISHING_TASK_COUNT}}"

    constraint {
      attribute = "${node.class}"
      value     = "publishing"
    }

    task "dp-download-service-publishing" {
      driver = "exec"

      artifact {
        source = "s3::https://s3-eu-west-1.amazonaws.com/{{BUILD_BUCKET}}/dp-download-service/{{REVISION}}.tar.gz"
      }

      artifact {
        source = "s3::https://s3-eu-west-1.amazonaws.com/{{DEPLOYMENT_BUCKET}}/dp-download-service/{{REVISION}}.tar.gz"
      }

      config {
        command = "${NOMAD_TASK_DIR}/start-task"
        args    = [
          "${NOMAD_TASK_DIR}/dp-download-service",
        ]
      }

      service {
        name = "dp-download-service"
        port = "http"
        tags = ["publishing"]
        check {
          type     = "http"
          path     = "/healthcheck"
          interval = "10s"
          timeout  = "2s"
        }
      }

      resources {
        cpu    = "{{PUBLISHING_RESOURCE_CPU}}"
        memory = "{{PUBLISHING_RESOURCE_MEM}}"

        network {
          port "http" {}
        }
      }

      template {
        source      = "${NOMAD_TASK_DIR}/vars-template"
        destination = "${NOMAD_TASK_DIR}/vars"
      }

      vault {
        policies = ["dp-download-service-publishing"]

        change_mode   = "signal"
        change_signal = "SIGUSR1"
      }
    }
  }
}
