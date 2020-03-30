# kubekite
**kubekite** is a manager for buildkite-agent jobs in Kubernetes.  It watches the [Buildkite](https://buildkite.com) API for new build jobs and when one is detected, it launches a Kubernetes job resource to run a single-user pod of [buildkite-agent](https://github.com/buildkite/agent).  When the agent is finished, kubekite cleans up the job and the associated pod.

## Usage

### How to build a new version of the container
- Build the binary docker image first: `docker build . -f Dockerfile-buildbinary`
- Run the binary docker image: `docker run $IMAGE_HASH`
- Copy the kubekite binary from the container into the root of this repo: `docker container cp $CONTAINER_HASH:/go/src/github.com/ProjectSigma/kubekite/cmd/kubekite/kubekite .`
  - You can get the container hash by running `docker ps -ql`
- To build and push to GCR in one go run `VERSION=[YOUR VERSION] ./build.sh`
- Alternatively
  - Build the full docker image, and tag it with the GCR resource: `docker build . -f Dockerfile-buildimage -t us.gcr.io/sigma-1330/kubekite:$VERSION`
  - Push the kubekite image to GCR: `docker push us.gcr.io/sigma-1330/kubekite:$VERSION`

Note:
If you only want to change `job.yaml` on already built $BASE_VERSION use `VERSION=[YOUR VERSION] BASE_VERSION=[BASE VERSION] ./build.sh `

Kubekite is designed to be run within Kubernetes as a single-replica deployment.  An example deployment spec [can be found here](https://github.com/ProjectSigma/kubekite/blob/master/kube-deploy/sigma-1330/deployment.yaml).  You can build and deploy kubekite from within Buildkite using the [included pipeline](https://github.com/ProjectSigma/kubekite/tree/master/.buildkite).

**Note that you will have to modify the deployment spec, these scripts, and the `pipeline.yml` to suit your infrastructure and preferred Docker registry.**

