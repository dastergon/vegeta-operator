# Vegeta Operator
A Kubernetes Operator for running load testing scenarios with [Vegeta](https://github.com/tsenart/vegeta).

## Status
The Vegeta Operator is currently in **alpha**.

## Description
[Vegeta](https://github.com/tsenart/vegeta) is an HTTP load testing tool and library. The Vegeta [Operator](https://coreos.com/blog/introducing-operators.html) provides an API to make it easy to deploy and run load testing scenarios in Kubernetes.

## Overview

The Operator supports most of the [current](https://github.com/tsenart/vegeta#usage-manual) features of Vegeta, and it also has the ability to store the generated reports to a remote blob storage system (bucket) via [rclone](https://rclone.org/). By default, Vegeta returns the generated report to the `stdout`. Consequently, you need to specify the output (filename) and the blob storage destination explicitly via the Custom Resource. Check the [CRD](deploy/crds/vegeta.dastergon.gr_vegeta_crd.yaml) spec for more options.

Vegeta runs as a [Job](https://kubernetes.io/docs/concepts/workloads/controllers/jobs-run-to-completion/). Therefore, once it's done running the load testing scenario, no more Pods are created but the Pods are not deleted either. They are kept around in case we want to view(`kubectl logs`) the logs of completed pods. However, any local files (reports) inside the container are deleted once the Job is complete. To address this situation, there's support to store the generated report to remote blob storage of a cloud provider.

## Notes

* Currently, the remote storage feature supports AWS S3 only. Feel free to submit pull requests to support more blob storage systems.
* The `rclone` tool is configured to authenticate to the blob storage system via environment variables of a specific format. Check its supported [format](https://rclone.org/docs/#config-file) first.

## Prerequisites

* Go >= v1.13+.
* Kubernetes >= 1.13+.

## Quick Start

Before running the operator, the Custom Resource Definition (CRD) must be registered with the Kubernetes apiserver:

    $ kubectl create -f deploy/crds/vegeta.dastergon.gr_vegeta_crd.yaml

Once this is done, there are two options to run the operator:

1. Locally
2. In the Kubernetes cluster

### Running locally

To run the Operator locally for development or testing, we use the [operator-sdk](https://github.com/operator-framework/operator-sdk).

#### Prerequisites:

* kubectl
* [operator-sdk](https://github.com/operator-framework/operator-sdk) installed
* [kind](https://github.com/kubernetes-sigs/kind) or [minikube](https://github.com/kubernetes/minikube)

Despite the fact that both `kind` and`minikube` automatically set the context, make sure that you are in the desired context and change it if it's not the intended one.

You can use a specific `kubeconfig` via the flag `--kubeconfig=<path/to/kubeconfig>`.

In the terminal we execute:

    $ operator-sdk run --local --watch-namespace=<namespace>

Then, we proceed with the steps as in any other cluster.

###  Running it in the Kubernetes cluster

The Deployment manifest is generated at`deploy/operator.yaml`. If you want to use an image other than the one available in the registry,  make sure to update the image field.

Setup RBAC and deploy the vegeta-operator:

    $ kubectl create -f deploy/service_account.yaml
    $ kubectl create -f deploy/role.yaml
    $ kubectl create -f deploy/role_binding.yaml
    $ kubectl create -f deploy/operator.yaml

To verify that the operator is up and running:

    $ kubectl get deployment

### Custom Resource Examples

The following snippets are some examples of the custom resource. For `target`, put your desired endpoint.

The following snippet shows how to execute a load testing scenario of a duration of 1 second and generate the report in JSON format. By default, it's a binary format.

```yaml
apiVersion: vegeta.dastergon.gr/v1alpha1
kind: Vegeta
metadata:
  name: example-vegeta
spec:
  target: "http://10.96.146.172:9876/info"
  attack:
    duration: 1s
    report:
      type: json
```

The following examples shows how to execute a load testing scenario with 100 requests per second:

```yaml
apiVersion: vegeta.dastergon.gr/v1alpha1
kind: Vegeta
metadata:
  name: example-vegeta
spec:
  target: "http://10.96.146.172:9876/info"
  attack:
    duration: 1s
    rate: 100/1s
    report:
      type: json
```

The following snippet shows how to execute a load testing scenario and store the report in JSON format to an AWS S3 bucket.
For **production** use it's highly recommended to use [Secrets](https://kubernetes.io/docs/concepts/configuration/secret/).

```yaml
apiVersion: vegeta.dastergon.gr/v1alpha1
kind: Vegeta
metadata:
  name: example-vegeta
spec:
  target: "http://10.96.146.172:9876/info"
  attack:
    duration: 1s
    report:
      output: report.json
      type: json
  blobStorage:
    name: bucketname
    provider: aws
    env:
    - name: RCLONE_CONFIG_S3_TYPE
      value: s3
    - name: RCLONE_CONFIG_S3_ACCESS_KEY_ID
      value: XXXX
    - name: RCLONE_CONFIG_S3_SECRET_ACCESS_KEY
      value: YYYY
    - name: RCLONE_CONFIG_S3_REGION
      value: eu-central-1
```

To apply the custom resource, use `kubectl`:

    kubectl apply -f <file>.yaml

To check the progress of the execution run `kubectl get pods --watch`
