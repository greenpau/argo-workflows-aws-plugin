# argo-workflows-aws-plugin

<a href="https://github.com/greenpau/argo-workflows-aws-plugin/actions/" target="_blank"><img src="https://github.com/greenpau/argo-workflows-aws-plugin/workflows/build/badge.svg"></a>

Argo Workflows Executor Plugin for AWS Services, e.g. SageMaker Pipelines, Glue, etc.

<!-- begin-markdown-toc -->
## Table of Contents

* [Supported AWS Services](#supported-aws-services)
* [Getting Started](#getting-started)
  * [Add IAM Role and Policy](#add-iam-role-and-policy)
  * [Enable Executor Plugins](#enable-executor-plugins)
  * [Installation](#installation)
  * [Add Workflow Template](#add-workflow-template)
  * [Trigger Workflow](#trigger-workflow)
  * [Uninstall Plugin](#uninstall-plugin)
* [References](#references)

<!-- end-markdown-toc -->

## Supported AWS Services

The following tables describe the implementation state for the protocol's RPC
methods and database operations.

| **Service Name** | **Implemented?** |
| --- | --- |
| Amazon SageMaker Pipelines | :heavy_check_mark: |
| AWS Glue | :construction: |
| AWS Step Functions | :construction: |


## Getting Started

### Add IAM Role and Policy

The plugin requires IAM role and policy to execute its operations.

The following CDK code add a role, which is later referenced in `plugin.yaml` manifest.

```ts
    const audClaim = `${cluster.clusterOpenIdConnectIssuer}:aud`;
    const subClaim = `${cluster.clusterOpenIdConnectIssuer}:sub`;

    const k8sConditions = new cdk.CfnJson(this, "KubeOIDCCondition", {
      value: {
        [audClaim]: "sts.amazonaws.com",
        // [subClaim]: "system:serviceaccount:kube-system:aws-node",
        [subClaim]: "system:serviceaccount:argo:awf-aws-executor-plugin",
      },
    });

    const awfPluginRole = new cdk.aws_iam.Role(this, "ArgoWorkflowsExecutorPluginRole", {
      roleName: `${stack.stackName}-awf-aws-executor-plugin`,
      assumedBy: new cdk.aws_iam.WebIdentityPrincipal(
        `arn:aws:iam::${cdk.Aws.ACCOUNT_ID}:oidc-provider/${cluster.clusterOpenIdConnectIssuer}`
      ).withConditions({
        StringEquals: k8sConditions,
      }),
    });

    awfPluginRole.addToPolicy(new cdk.aws_iam.PolicyStatement({
      effect: cdk.aws_iam.Effect.ALLOW,
      resources: ["arn:aws:sagemaker:*:*:pipeline/*"],
      actions: [
        "sagemaker:DescribePipeline",
        "sagemaker:StartPipelineExecution",
        "sagemaker:ListPipelineExecutionSteps",
        "sagemaker:DescribePipelineExecution",
        "sagemaker:ListPipelineExecutions",
        "sagemaker:ListPipelines"
      ]
    }));
```

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

Next, restart:

```bash
kubectl -n argo set env deployment/workflow-controller ARGO_EXECUTOR_PLUGINS=true
kubectl rollout restart -n argo deployment workflow-controller
```

### Installation

Download the plugin manifest:

```bash
wget https://raw.githubusercontent.com/greenpau/argo-workflows-aws-plugin/main/assets/plugin.yaml
```

Edit `metadata.annotations.eks.amazonaws.com/role-arn` in the `ServiceAccount`. (see `DEVELOPMENT.md` for
more information about associated IAM role and policy)

Next, install the plugin:

```bash
kubectl apply -f plugin.yaml
```

The output follows:

```
serviceaccount/awf-aws-plugin-sa unchanged
clusterrole.rbac.authorization.k8s.io/argo-plugin-addition-role unchanged
clusterrolebinding.rbac.authorization.k8s.io/awf-aws-plugin-addition-binding unchanged
clusterrolebinding.rbac.authorization.k8s.io/awf-aws-plugin-binding unchanged
configmap/awf-aws-plugin created
```

List Argo Workflows Executor Plugins again:

```
$ kubectl get cm -l workflows.argoproj.io/configmap-type=ExecutorPlugin -n argo

NAME             DATA   AGE
awf-aws          2      34s
```

Get details about the plugins:

```bash
kubectl describe cm -l workflows.argoproj.io/configmap-type=ExecutorPlugin -n argo
```

### Add Workflow Template

Create a workflow template:

```bash
kubectl apply -f https://raw.githubusercontent.com/greenpau/argo-workflows-aws-plugin/main/assets/amz-sagemaker-pipelines-workflow-template.yaml
```

### Trigger Workflow

Start new workflow:

```bash
kubectl create -f https://raw.githubusercontent.com/greenpau/argo-workflows-aws-plugin/main/assets/amz-sagemaker-pipelines-workflow.yaml
```

The output follows:

```
workflow.argoproj.io/sm-pipelines-tswbm created
```

Review the status of the workflow by the its name, e.g. `sm-pipelines-tswbm`:

```bash
kubectl describe pod -n argo sm-pipelines-tswbm-1340600742-agent
```

Review logs of the containers (`main`, `awf-aws`) inside the pod:

```bash
kubectl logs -n argo -c main sm-pipelines-tswbm-1340600742-agent
kubectl logs -n argo -c awf-aws sm-pipelines-tswbm-1340600742-agent
```

### Uninstall Plugin

If necessary, run the following commands to uninstall the plugin:

```bash
kubectl delete -f https://raw.githubusercontent.com/greenpau/argo-workflows-aws-plugin/main/assets/plugin.yaml
```

## References

* [Argo Workflows - Plugin Directory](https://argoproj.github.io/argo-workflows/plugin-directory/)
