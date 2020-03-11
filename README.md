# kubekite
**kubekite** is a manager for buildkite-agent jobs in Kubernetes.  It watches the [Buildkite](https://buildkite.com) API for new build jobs and when one is detected, it launches a Kubernetes job resource to run a single-user pod of [buildkite-agent](https://github.com/buildkite/agent).  When the agent is finished, kubekite cleans up the job and the associated pod.

## Usage
Kubekite is designed to be run within Kubernetes as a single-replica deployment.  An example deployment spec [can be found here](https://github.com/webflow/kubekite/blob/master/kube-deploy/your-cluster-name.yourdomain.com/deployment.yaml).  You can build and deploy kubekite from within Buildkite using the [included pipeline](https://github.com/webflow/kubekite/tree/master/.buildkite).  

**Note that you will have to modify the deployment spec, these scripts, and the `pipeline.yml` to suit your infrastructure and preferred Docker registry.**

# Test to run new pods on k8s