set -e -u -o pipefail

KUBEKITE_IMAGE_BASE=us.gcr.io/sigma-1330/kubekite
TAG=$KUBEKITE_IMAGE_BASE:$VERSION

set +u

if [ "$BASE_VERSION" = "" ]; then
  docker build -t $TAG -f Dockerfile-buildimage .;
else
  KUBEKITE_IMAGE=$KUBEKITE_IMAGE_BASE:$BASE_VERSION
  docker pull $KUBEKITE_IMAGE

  docker build --build-arg KUBEKITE_IMAGE=$KUBEKITE_IMAGE -t $TAG -f Dockerfile-fromkubekitebase .;
fi

docker push $TAG
