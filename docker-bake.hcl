// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

variable "DOCKER_BAKE_REGISTRY" {
  default = "ghcr.io/slinkyproject"
}

variable "DOCKER_BAKE_SUFFIX" {}

version = "0.3.0"

function "format_tag" {
  params = [registry, stage, version, suffix]
  result = format("%s:%s", join("/", compact([registry, stage])), join("-", compact([version, suffix])))
}

group "default" {
  targets = ["exporter"]
}

target "exporter" {
  tags = [
    format_tag("${DOCKER_BAKE_REGISTRY}", "slurm-exporter", "${version}", "${DOCKER_BAKE_SUFFIX}"),
  ]
}
