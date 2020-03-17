# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [2.2.1]

* Set `Disruption` CRD count field optional ([#91](https://github.com/DataDog/chaos-controller/pull/91))
* Fix injection pod failing when resolving a host returning something else than A records ([#90](https://github.com/DataDog/chaos-controller/pull/90))
* Add NOTICE file ([#89](https://github.com/DataDog/chaos-controller/pull/89))
* Document available `Make` commands ([#84](https://github.com/DataDog/chaos-controller/pull/84))
* Add manifests check in circleci ([#92](https://github.com/DataDog/chaos-controller/pull/92))

## [2.2.0]

* Change Chaos injector image variable name for consistency with k8s-resources config
* Make minikube ISO image public
* Refine CircleCI configuration by using executors and commands instead of templates
* Rename the project to chaos-controller
* Use the new docker-push image format for Gitlab pipelines
* Refine the check for when 3rd-party licenses are missing
* Add Python 'Invoke' for task management
* Use a pod template for generated injection and cleanup pods
* Add configurable metrics sync
* Fix delay not being passed to tc command builder
* Add cmd flag for metrics sink
* Fix typo in gitlab-ci config preventing tag release to be pushed on staging
