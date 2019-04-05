#!/bin/bash

# build the image
docker build -t ${CI_PROJECT_NAME}:${TAG} .

for registry in 727006795293 464622532012 ; do
  eval $(aws ecr get-login --no-include-email --region us-east-1 --registry-ids ${registry})
  docker tag ${CI_PROJECT_NAME}:${TAG} ${registry}.dkr.ecr.us-east-1.amazonaws.com/${CI_PROJECT_NAME}:${TAG}
  docker push ${registry}.dkr.ecr.us-east-1.amazonaws.com/${CI_PROJECT_NAME}:${TAG}
done

for environment in staging prod; do
  aws ssm get-parameter --region us-east-1 --name ${environment}.gcr.push.key --with-decryption --query "Parameter.Value" --out text > /tmp/gcloud-credentials.json
  gcloud auth activate-service-account --key-file /tmp/gcloud-credentials.json
  gcloud auth configure-docker
  docker tag ${CI_PROJECT_NAME}:${TAG} eu.gcr.io/datadog-${environment}/${CI_PROJECT_NAME}:${TAG}
  docker push eu.gcr.io/datadog-${environment}/${CI_PROJECT_NAME}:${TAG}
done
