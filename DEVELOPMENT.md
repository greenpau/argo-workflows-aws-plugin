# Development

<!-- begin-markdown-toc -->

## Table of Contents

- [Plugin Operations](#plugin-operations)
- [Troubleshooting](#troubleshooting)
  - [WebIdentityErr Access Denied](#webidentityerr-access-denied)

<!-- end-markdown-toc -->

## Plugin Operations

The `assets/sagemaker-pipelines-workflow-template.yaml` uses the plugin in a workflow.
There are two tasks in the workflow: `validate-pipeline` and `execute-pipeline`. Both
tasks use the plugin.

When a user triggers a workflow execution, Argo Workflows creates a **pod**
with two (2) containers:

- `main`: Workflow Executor
- `awf-aws`: AWS Plugin container

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

## Troubleshooting

### WebIdentityErr Access Denied

The plugin may err with the following message.

```
failed to describe amazon sagemaker pipeline: WebIdentityErr: failed to retrieve credentials caused by: AccessDenied: Not authorized to perform sts:AssumeRoleWithWebIdentity status code: 403, request id: a3ead691-6855-45ed-aa48-b3714ddcff1f
```

The CloudTrail event follows:

```json
{
    "eventVersion": "1.08",
    "userIdentity": {
        "type": "WebIdentityUser",
        "principalId": "arn:aws:iam::100000000002:oidc-provider/oidc.eks.us-west-2.amazonaws.com/id/A8242F25BFBB8F98F5321B2AE63C8B5C:sts.amazonaws.com:system:serviceaccount:argo:awf-aws-executor-plugin",
        "userName": "system:serviceaccount:argo:awf-aws-executor-plugin",
        "identityProvider": "arn:aws:iam::100000000002:oidc-provider/oidc.eks.us-west-2.amazonaws.com/id/A8242F25BFBB8F98F5321B2AE63C8B5C"
    },
    "eventTime": "2023-11-08T15:09:08Z",
    "eventSource": "sts.amazonaws.com",
    "eventName": "AssumeRoleWithWebIdentity",
    "awsRegion": "us-west-2",
    "sourceIPAddress": "44.238.229.20",
    "userAgent": "aws-sdk-go/1.45.1 (go1.21.4; linux; amd64)",
    "errorCode": "AccessDenied",
    "errorMessage": "An unknown error occurred",
    "requestParameters": {
        "roleArn": "arn:aws:iam::100000000002:role/sm-pipelines-k8s-awf-aws-executor-plugin",
        "roleSessionName": "1699456147945229070"
    },
    "responseElements": null,
    "requestID": "a3ead691-6855-45ed-aa48-b3714ddcff1f",
    "eventID": "899f0a00-4e04-4fab-bb35-0297ed266f62",
    "readOnly": true,
    "resources": [
        {
            "accountId": "100000000002",
            "type": "AWS::IAM::Role",
            "ARN": "arn:aws:iam::100000000002:role/sm-pipelines-k8s-awf-aws-executor-plugin"
        }
    ],
    "eventType": "AwsApiCall",
    "managementEvent": true,
    "recipientAccountId": "100000000002",
    "eventCategory": "Management",
    "tlsDetails": {
        "tlsVersion": "TLSv1.2",
        "cipherSuite": "ECDHE-RSA-AES128-GCM-SHA256",
        "clientProvidedHostHeader": "sts.us-west-2.amazonaws.com"
    }
}
```

The trust policy associated with the `arn:aws:iam::100000000002:role/awf-aws-executor-plugin` role should be:

```json
{
	"Version": "2012-10-17",
	"Statement": [
    {
			"Effect": "Allow",
			"Principal": {
				"Federated": "arn:aws:iam::100000000002:oidc-provider/oidc.eks.us-west-2.amazonaws.com/id/A8242F25BFBB8F98F5321B2AE63C8B5C"
			},
			"Action": "sts:AssumeRoleWithWebIdentity",
			"Condition": {
				"StringEquals": {
					"oidc.eks.us-west-2.amazonaws.com/id/A8242F25BFBB8F98F5321B2AE63C8B5C:aud": "sts.amazonaws.com",
					"oidc.eks.us-west-2.amazonaws.com/id/A8242F25BFBB8F98F5321B2AE63C8B5C:sub": "system:serviceaccount:argo:awf-aws-executor-plugin"
				}
			}
		}
	]
}
```
