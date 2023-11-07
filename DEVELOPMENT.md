# Development

<!-- begin-markdown-toc -->
## Table of Contents

* [EKS Cluster Configuration](#eks-cluster-configuration)
* [Plugin Operations](#plugin-operations)

<!-- end-markdown-toc -->

## EKS Cluster Configuration

The plugin requires IAM role and policy to execute its operations.

The following CDK code add a role, which is later referenced in `plugin.yaml` manifest.

```ts
    const audClaim = `${cluster.clusterOpenIdConnectIssuer}:aud`;
    const subClaim = `${cluster.clusterOpenIdConnectIssuer}:sub`;

    const k8sConditions = new cdk.CfnJson(this, "KubeOIDCCondition", {
      value: {
        [audClaim]: "sts.amazonaws.com",
        [subClaim]: "system:serviceaccount:kube-system:aws-node",
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

## Plugin Operations

The `assets/sagemaker-pipelines-workflow-template.yaml` uses the plugin in a workflow.
There are two tasks in the workflow: `validate-pipeline` and `execute-pipeline`. Both
tasks use the plugin.

When a user triggers a workflow execution, Argo Workflows creates a **pod**
with two (2) containers:

* `main`: Workflow Executor
* `awf-aws`: AWS Plugin container

The `main` container makes `POST` to the plugin container with the following content:

```json
{
  "workflow": {
    "metadata": {
      "name": "sm-pipelines-r58tg",
      "namespace": "argo",
      "uid": "27c01e7c-9d93-450f-a001-c64d649aac99"
    }
  },
  "template": {
    "name": "validate_pipeline",
    "inputs": {},
    "outputs": {},
    "metadata": {},
    "plugin": {
      "awf-aws-plugin": {
        "account_id": "100000000002",
        "action": "validate",
        "pipeline_name": "MyPipeline",
        "region_name": "us-west-2"
      }
    }
  }
}
```

What distinguishes different tasks to the plugin's container (i.e. `awf-aws`) is
the identifier in `workflow.metadata.uid`. That identifier can be use to map
to a particular execution.
