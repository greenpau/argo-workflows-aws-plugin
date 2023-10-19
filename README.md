# argo-workflows-aws-plugin

<a href="https://github.com/greenpau/argo-workflows-aws-plugin/actions/" target="_blank"><img src="https://github.com/greenpau/argo-workflows-aws-plugin/workflows/build/badge.svg"></a>

Argo Workflows Executor Plugin for AWS Services, e.g. SageMaker Pipelines, Glue, etc.

<!-- begin-markdown-toc -->
## Table of Contents

* [Supported AWS Services](#supported-aws-services)
* [Getting Started](#getting-started)
  * [Enable Executor Plugins](#enable-executor-plugins)
  * [Install the Plugin](#install-the-plugin)
* [References](#references)

<!-- end-markdown-toc -->

## Supported AWS Services

The following tables describe the implementation state for the protocol's RPC
methods and database operations.

| **Service Name** | **Implemented?** |
| --- | --- |
| Amazon SageMaker Pipelines | :construction: |
| AWS Glue | :construction: |


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

### Install the Plugin

Install `argo-workflows-aws-plugin` by creating following the `ConfigMap` in your cluster:

```bash
kubectl apply -f https://raw.githubusercontent.com/greenpau/argo-workflows-aws-plugin/main/assets/plugin.yaml
```

The output follows:

```
serviceaccount/awf-aws-plugin-sa created
clusterrole.rbac.authorization.k8s.io/argo-plugin-addition-role created
clusterrolebinding.rbac.authorization.k8s.io/awf-aws-plugin-addition-binding created
clusterrolebinding.rbac.authorization.k8s.io/awf-aws-plugin-binding created
configmap/awf-aws-plugin created
```

## References

* [Argo Workflows - Plugin Directory](https://argoproj.github.io/argo-workflows/plugin-directory/)
