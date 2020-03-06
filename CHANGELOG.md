# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

* Change Chaos injector image variable name for consistency with k8s-resources config
* Make minikube ISO image public
* Refine CircleCI configuration by using executors and commands instead of templates
* Rename the project to chaos-controller
* Use the new docker-push image format for Gitlab pipelines
* Refine the check for when 3rd-party licenses are missing
* Add Python 'Invoke' for task management
* Use a pod template for generated injection and cleanup pods
