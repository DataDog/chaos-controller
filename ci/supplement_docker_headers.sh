#!/bin/sh

DOCKER_CONFIG_DIR="${HOME}/.docker";
DOCKER_CONFIG="${DOCKER_CONFIG_DIR}/config.json";
JQ_QUERY=".HttpHeaders[\"X-Meta-OCI-CI-commit_ref\"] = \"${CI_COMMIT_SHA}\" | .HttpHeaders[\"X-Meta-OCI-CI-branch\"] = \"${CI_COMMIT_BRANCH}\" | .HttpHeaders[\"X-Meta-OCI-CI-pipeline_id\"] = \"${CI_PIPELINE_ID}\""

if [ -e ${DOCKER_CONFIG} ]; then
    TMP_CONFIG=$(mktemp);
    jq "${JQ_QUERY}" ${DOCKER_CONFIG} > ${TMP_CONFIG};
    cat ${TMP_CONFIG};
    mv ${TMP_CONFIG} ${DOCKER_CONFIG};
else
    mkdir -p ${DOCKER_CONFIG_DIR}
    echo '{}' | jq "${JQ_QUERY}" > ${DOCKER_CONFIG};
fi
