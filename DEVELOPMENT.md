# Development

<!-- begin-markdown-toc -->
## Table of Contents

* [Plugin Operations](#plugin-operations)

<!-- end-markdown-toc -->

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
