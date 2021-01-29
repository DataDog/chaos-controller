#!/bin/sh
# https://datadoghq.atlassian.net/wiki/spaces/EEE/pages/890437939/OCI+Artifact+Publishing

DOCKER_CONFIG_DIR="${HOME}/.docker";
DOCKER_CONFIG="${DOCKER_CONFIG_DIR}/config.json";
JQ_QUERY=".HttpHeaders[\"X-Meta-GIT_REPOSITORY\"] = \"${CI_PROJECT_NAME}\" | .HttpHeaders[\"X-Meta-GIT_COMMIT_SHA\"] = \"${CI_COMMIT_SHA}\" | .HttpHeaders[\"X-Meta-GIT_COMMIT_BRANCH\"] = \"${CI_COMMIT_BRANCH}\" | .HttpHeaders[\"X-Meta-GITLAB_CI_PIPELINE_ID\"] = \"${CI_PIPELINE_ID}\""

if [ -e ${DOCKER_CONFIG} ]; then
    TMP_CONFIG=$(mktemp);
    jq "${JQ_QUERY}" ${DOCKER_CONFIG} > ${TMP_CONFIG};
    cat ${TMP_CONFIG};
    mv ${TMP_CONFIG} ${DOCKER_CONFIG};
else
    mkdir -p ${DOCKER_CONFIG_DIR}
    echo '{}' | jq "${JQ_QUERY}" > ${DOCKER_CONFIG};
fi
