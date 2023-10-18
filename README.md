# argo-workflows-aws-plugin

<a href="https://github.com/greenpau/argo-workflows-aws-plugin/actions/" target="_blank"><img src="https://github.com/greenpau/argo-workflows-aws-plugin/workflows/build/badge.svg?branch=main"></a>

Argo Workflows Executor Plugin for AWS Services, e.g. SageMaker Pipelines, Glue, etc.

<!-- begin-markdown-toc -->
## Table of Contents

* [Supported AWS Services](#supported-aws-services)
* [Getting Started](#getting-started)
  * [Enable Executor Plugins](#enable-executor-plugins)

<!-- end-markdown-toc -->

## Supported AWS Services

The following tables describe the implementation state for the protocol's RPC
methods and database operations.

| **Service Name** | **Implemented?** |
| --- | --- |
| SageMaker Pipelines | :white_check_mark: |

## Getting Started

### Enable Executor Plugins

First, enable Executor Plugins:

```bash
kubectl patch deployment \
  workflow-controller \
  --namespace argo \
  --type='json' \
  -p='[{"op": "add", "path": "/spec/template/spec/containers/0/env/0", "value": {
    "name": "ARGO_EXECUTOR_PLUGINS",
    "value": "true",
}}]'
```
