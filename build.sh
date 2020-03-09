set -e -u pipefail

KUBEKITE_IMAGE_BASE=asia.gcr.io/test-project-buildkite/kubekite
TAG=$KUBEKITE_IMAGE_BASE:$VERSION

set +e

if [ "$BASE_VERSION" = " " ]; then
    docker build -t $TAG -f Dockerfile-buildimage .;
else
    KUBEKITE_IMAGE=$KUBEKITE_IMAGE_BASE:$BASE_VERSION
    docker pull $KUBEKITE_IMAGE

    docker build --build-arg KUBEKITE_IMAGE=$KUBEKITE_IMAGE -t $TAG -f Dockerfile-fromkubekitebase .;
fi

docker push $TAG

