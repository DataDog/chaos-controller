# Changelog

## [2.8.0](https://github.com/DataDog/chaos-controller/tree/2.8.0) (2020-07-08)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/2.7.1...2.8.0)

**Merged pull requests:**

- Update container release\_changelog\_bc5e0460e933309fcbfd63d3cc3403edcef25eec [\#141](https://github.com/DataDog/chaos-controller/pull/141) ([github-actions[bot]](https://github.com/apps/github-actions))
- Add disk pressure feature [\#139](https://github.com/DataDog/chaos-controller/pull/139) ([Devatoria](https://github.com/Devatoria))

## [2.7.1](https://github.com/DataDog/chaos-controller/tree/2.7.1) (2020-07-06)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/2.7.0...2.7.1)

**Merged pull requests:**

- Do not exit on metrics sink initialization error in the injector [\#140](https://github.com/DataDog/chaos-controller/pull/140) ([Devatoria](https://github.com/Devatoria))
- Fix local scripts namespace [\#138](https://github.com/DataDog/chaos-controller/pull/138) ([Devatoria](https://github.com/Devatoria))
- Improve user documentation with use cases [\#136](https://github.com/DataDog/chaos-controller/pull/136) ([Devatoria](https://github.com/Devatoria))
- Update container release\_changelog\_51ed4b107cc31e2ce55ac56f042bf74dd28fb597 [\#135](https://github.com/DataDog/chaos-controller/pull/135) ([github-actions[bot]](https://github.com/apps/github-actions))

## [2.7.0](https://github.com/DataDog/chaos-controller/tree/2.7.0) (2020-06-05)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/2.6.1...2.7.0)

**Merged pull requests:**

- Create a chaos-engineering namespace for local testing [\#134](https://github.com/DataDog/chaos-controller/pull/134) ([Devatoria](https://github.com/Devatoria))
- Factor out networking impl in latency and bandwidth limit disruptions [\#133](https://github.com/DataDog/chaos-controller/pull/133) ([brandon-dd](https://github.com/brandon-dd))
- Adds new network bandwidth limitation disruption [\#132](https://github.com/DataDog/chaos-controller/pull/132) ([brandon-dd](https://github.com/brandon-dd))
- Update docs with custom minikube ISO information [\#131](https://github.com/DataDog/chaos-controller/pull/131) ([brandon-dd](https://github.com/brandon-dd))
- Update container release\_changelog\_732fb8265d8073d88598ec47ec4c3bc11bbe9da4 [\#129](https://github.com/DataDog/chaos-controller/pull/129) ([github-actions[bot]](https://github.com/apps/github-actions))

## [2.6.1](https://github.com/DataDog/chaos-controller/tree/2.6.1) (2020-05-11)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/2.6.0...2.6.1)

**Merged pull requests:**

- Disable cgo on build [\#128](https://github.com/DataDog/chaos-controller/pull/128) ([Devatoria](https://github.com/Devatoria))
- fix kubernetes version at 1.17.0 [\#127](https://github.com/DataDog/chaos-controller/pull/127) ([brandon-dd](https://github.com/brandon-dd))
- Update container release\_changelog\_a878e718a85f72ea2abf23f351af221fea8c0e41 [\#126](https://github.com/DataDog/chaos-controller/pull/126) ([github-actions[bot]](https://github.com/apps/github-actions))

## [2.6.0](https://github.com/DataDog/chaos-controller/tree/2.6.0) (2020-05-11)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/2.5.0...2.6.0)

**Merged pull requests:**

- Change code owners to the new core resilience team [\#125](https://github.com/DataDog/chaos-controller/pull/125) ([Devatoria](https://github.com/Devatoria))
- Add cpu pressure injection [\#124](https://github.com/DataDog/chaos-controller/pull/124) ([Devatoria](https://github.com/Devatoria))
- api: Add a maximum value for networkLatency.delay [\#123](https://github.com/DataDog/chaos-controller/pull/123) ([dd-adn](https://github.com/dd-adn))
- Switch to noop metrics sink by default [\#122](https://github.com/DataDog/chaos-controller/pull/122) ([Devatoria](https://github.com/Devatoria))
- Fix manager bin path in deployment [\#121](https://github.com/DataDog/chaos-controller/pull/121) ([Devatoria](https://github.com/Devatoria))
- Improve images and build process [\#120](https://github.com/DataDog/chaos-controller/pull/120) ([Devatoria](https://github.com/Devatoria))
- Add port configuration to network latency [\#119](https://github.com/DataDog/chaos-controller/pull/119) ([Azoam](https://github.com/Azoam))
- Update container release\_changelog\_df7dfce24de404eb5ba74f91323a58c09dfc9161 [\#118](https://github.com/DataDog/chaos-controller/pull/118) ([github-actions[bot]](https://github.com/apps/github-actions))

## [2.5.0](https://github.com/DataDog/chaos-controller/tree/2.5.0) (2020-04-22)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/2.4.1...2.5.0)

**Merged pull requests:**

- Allow connection establishment before drop in network failure [\#117](https://github.com/DataDog/chaos-controller/pull/117) ([Devatoria](https://github.com/Devatoria))
- Add some more comments in the disruption example [\#116](https://github.com/DataDog/chaos-controller/pull/116) ([Devatoria](https://github.com/Devatoria))
- Update 3rd party licenses to show spdx identifier [\#115](https://github.com/DataDog/chaos-controller/pull/115) ([Devatoria](https://github.com/Devatoria))
- Update container release\_changelog\_e2a5b625599466a40a09665b59090a26d9d8c0c1 [\#114](https://github.com/DataDog/chaos-controller/pull/114) ([github-actions[bot]](https://github.com/apps/github-actions))

## [2.4.1](https://github.com/DataDog/chaos-controller/tree/2.4.1) (2020-04-01)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/2.4.0...2.4.1)

**Merged pull requests:**

- Make optional fields nullable [\#113](https://github.com/DataDog/chaos-controller/pull/113) ([Devatoria](https://github.com/Devatoria))
- Update container release\_changelog\_78c44f3f80dc281ab9fd79cb93f862dd5f84aa37 [\#112](https://github.com/DataDog/chaos-controller/pull/112) ([github-actions[bot]](https://github.com/apps/github-actions))
- Add michelada release [\#97](https://github.com/DataDog/chaos-controller/pull/97) ([Azoam](https://github.com/Azoam))

## [2.4.0](https://github.com/DataDog/chaos-controller/tree/2.4.0) (2020-03-25)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/2.3.0...2.4.0)

**Merged pull requests:**

- Close statsd client connection on injector and controller exit [\#110](https://github.com/DataDog/chaos-controller/pull/110) ([Devatoria](https://github.com/Devatoria))
- Check qdisc hasn't been cleared before trying to clear it [\#109](https://github.com/DataDog/chaos-controller/pull/109) ([Devatoria](https://github.com/Devatoria))
- Add tests for the network package [\#108](https://github.com/DataDog/chaos-controller/pull/108) ([Devatoria](https://github.com/Devatoria))
- Update container release\_changelog\_0b2bd25290da0f1fa63e31ad511625889c22aa90 [\#107](https://github.com/DataDog/chaos-controller/pull/107) ([github-actions[bot]](https://github.com/apps/github-actions))

## [2.3.0](https://github.com/DataDog/chaos-controller/tree/2.3.0) (2020-03-19)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/2.2.1...2.3.0)

**Merged pull requests:**

- Count field is now required and the value to target all pods is -1 [\#106](https://github.com/DataDog/chaos-controller/pull/106) ([Devatoria](https://github.com/Devatoria))
- Add release documentation [\#105](https://github.com/DataDog/chaos-controller/pull/105) ([Devatoria](https://github.com/Devatoria))
- Auto-generate changelog on tag push and open a PR to approve it [\#103](https://github.com/DataDog/chaos-controller/pull/103) ([Devatoria](https://github.com/Devatoria))
- Add missing tag to release pull command [\#99](https://github.com/DataDog/chaos-controller/pull/99) ([Devatoria](https://github.com/Devatoria))
- Add goreleaser GitHub action [\#98](https://github.com/DataDog/chaos-controller/pull/98) ([Devatoria](https://github.com/Devatoria))
- Review the way we push images from the CI [\#96](https://github.com/DataDog/chaos-controller/pull/96) ([Devatoria](https://github.com/Devatoria))
- Add CI job to release images on docker hub [\#95](https://github.com/DataDog/chaos-controller/pull/95) ([Devatoria](https://github.com/Devatoria))
- add targetPod name to logs [\#94](https://github.com/DataDog/chaos-controller/pull/94) ([jvanbrunschot](https://github.com/jvanbrunschot))

## [2.2.1](https://github.com/DataDog/chaos-controller/tree/2.2.1) (2020-03-13)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/2.2.0...2.2.1)

**Merged pull requests:**

- Release 2.2.1 changelog [\#93](https://github.com/DataDog/chaos-controller/pull/93) ([Devatoria](https://github.com/Devatoria))
- Set disruption resource count field optional [\#91](https://github.com/DataDog/chaos-controller/pull/91) ([Devatoria](https://github.com/Devatoria))
- Cast DNS records before appending it to avoid a panic [\#90](https://github.com/DataDog/chaos-controller/pull/90) ([Devatoria](https://github.com/Devatoria))
- Add NOTICE [\#89](https://github.com/DataDog/chaos-controller/pull/89) ([Devatoria](https://github.com/Devatoria))
- document available Make commands [\#84](https://github.com/DataDog/chaos-controller/pull/84) ([jvanbrunschot](https://github.com/jvanbrunschot))

## [2.2.0](https://github.com/DataDog/chaos-controller/tree/2.2.0) (2020-03-12)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/2.1.0...2.2.0)

**Closed issues:**

- Injector pods can be oom killed [\#79](https://github.com/DataDog/chaos-controller/issues/79)

**Merged pull requests:**

- Fix gitlab-ci injector tag release on staging [\#88](https://github.com/DataDog/chaos-controller/pull/88) ([Devatoria](https://github.com/Devatoria))
- Bump CHANGELOG to version 2.2.0 [\#87](https://github.com/DataDog/chaos-controller/pull/87) ([Devatoria](https://github.com/Devatoria))
- add cmd flag for metrics sink [\#86](https://github.com/DataDog/chaos-controller/pull/86) ([jvanbrunschot](https://github.com/jvanbrunschot))
- Pass delay to tc command builder [\#85](https://github.com/DataDog/chaos-controller/pull/85) ([Devatoria](https://github.com/Devatoria))
- Allow to pass a pod template file for generated chaos pods [\#83](https://github.com/DataDog/chaos-controller/pull/83) ([Devatoria](https://github.com/Devatoria))
- Improve task management [\#82](https://github.com/DataDog/chaos-controller/pull/82) ([jvanbrunschot](https://github.com/jvanbrunschot))
- fix typos [\#81](https://github.com/DataDog/chaos-controller/pull/81) ([jvanbrunschot](https://github.com/jvanbrunschot))
- Create configurable metric sink [\#80](https://github.com/DataDog/chaos-controller/pull/80) ([jvanbrunschot](https://github.com/jvanbrunschot))
- error when 3rd-part licenses are out-of-date [\#77](https://github.com/DataDog/chaos-controller/pull/77) ([jvanbrunschot](https://github.com/jvanbrunschot))
- Change naming scheme of injector image to be consistent with k8s config [\#76](https://github.com/DataDog/chaos-controller/pull/76) ([Azoam](https://github.com/Azoam))
- Adapt gitlab configuration to use the new docker-push image [\#75](https://github.com/DataDog/chaos-controller/pull/75) ([Devatoria](https://github.com/Devatoria))
- Replace any occurence of the old name in the project [\#74](https://github.com/DataDog/chaos-controller/pull/74) ([Devatoria](https://github.com/Devatoria))
- Improve CircleCI configuration [\#73](https://github.com/DataDog/chaos-controller/pull/73) ([Devatoria](https://github.com/Devatoria))
- Use a public minikube iso file [\#72](https://github.com/DataDog/chaos-controller/pull/72) ([Devatoria](https://github.com/Devatoria))
- Add changelog [\#71](https://github.com/DataDog/chaos-controller/pull/71) ([Devatoria](https://github.com/Devatoria))
- add 'out' dir to dockerignore [\#70](https://github.com/DataDog/chaos-controller/pull/70) ([jvanbrunschot](https://github.com/jvanbrunschot))
- requirements are documented in the testing docs [\#69](https://github.com/DataDog/chaos-controller/pull/69) ([jvanbrunschot](https://github.com/jvanbrunschot))
- Improve golangci [\#68](https://github.com/DataDog/chaos-controller/pull/68) ([Devatoria](https://github.com/Devatoria))
- Add testing docs [\#67](https://github.com/DataDog/chaos-controller/pull/67) ([jvanbrunschot](https://github.com/jvanbrunschot))
- Remove monkey patching [\#66](https://github.com/DataDog/chaos-controller/pull/66) ([Devatoria](https://github.com/Devatoria))
- Added simple Issues and PR templates [\#65](https://github.com/DataDog/chaos-controller/pull/65) ([Azoam](https://github.com/Azoam))
- Add docker support [\#64](https://github.com/DataDog/chaos-controller/pull/64) ([jvanbrunschot](https://github.com/jvanbrunschot))
- Add a way to run local tests in a container to bypass mprotect syscall issues [\#63](https://github.com/DataDog/chaos-controller/pull/63) ([Devatoria](https://github.com/Devatoria))
- Move CODEOWNERS file [\#62](https://github.com/DataDog/chaos-controller/pull/62) ([Devatoria](https://github.com/Devatoria))
- Add CODEOWNERS file [\#61](https://github.com/DataDog/chaos-controller/pull/61) ([Devatoria](https://github.com/Devatoria))
- Build docker images with the local daemon and scp them into minikube [\#60](https://github.com/DataDog/chaos-controller/pull/60) ([Devatoria](https://github.com/Devatoria))
- Split circleci checks [\#59](https://github.com/DataDog/chaos-controller/pull/59) ([Devatoria](https://github.com/Devatoria))
- Improve CircleCI checks [\#58](https://github.com/DataDog/chaos-controller/pull/58) ([Devatoria](https://github.com/Devatoria))
- Remove any internal references and adapt documentation [\#57](https://github.com/DataDog/chaos-controller/pull/57) ([Devatoria](https://github.com/Devatoria))
- Add license header [\#56](https://github.com/DataDog/chaos-controller/pull/56) ([Devatoria](https://github.com/Devatoria))
- Add LICENSE [\#55](https://github.com/DataDog/chaos-controller/pull/55) ([Devatoria](https://github.com/Devatoria))
- Add directory for circleci [\#54](https://github.com/DataDog/chaos-controller/pull/54) ([Azoam](https://github.com/Azoam))
- Add 3rd party licenses and generation script [\#53](https://github.com/DataDog/chaos-controller/pull/53) ([Devatoria](https://github.com/Devatoria))

## [2.1.0](https://github.com/DataDog/chaos-controller/tree/2.1.0) (2020-01-31)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/2.0.0...2.1.0)

**Merged pull requests:**

- Move logger instance into the CLI and pass it to the injector instance [\#52](https://github.com/DataDog/chaos-controller/pull/52) ([Devatoria](https://github.com/Devatoria))
- Allow to specify a list of hosts in a network failure [\#51](https://github.com/DataDog/chaos-controller/pull/51) ([Devatoria](https://github.com/Devatoria))
- Add requirements for contributing and local development [\#50](https://github.com/DataDog/chaos-controller/pull/50) ([Devatoria](https://github.com/Devatoria))
- Add golangci-lint to the project [\#49](https://github.com/DataDog/chaos-controller/pull/49) ([Devatoria](https://github.com/Devatoria))

## [2.0.0](https://github.com/DataDog/chaos-controller/tree/2.0.0) (2020-01-29)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/1.1.0...2.0.0)

**Merged pull requests:**

- Fix resource version race condition on instance update in controller tests [\#48](https://github.com/DataDog/chaos-controller/pull/48) ([Devatoria](https://github.com/Devatoria))
- Unique CRD and controller for all the failures [\#47](https://github.com/DataDog/chaos-controller/pull/47) ([Devatoria](https://github.com/Devatoria))
- Ignore unneeded files and make better use of build cache [\#46](https://github.com/DataDog/chaos-controller/pull/46) ([Devatoria](https://github.com/Devatoria))

## [1.1.0](https://github.com/DataDog/chaos-controller/tree/1.1.0) (2020-01-23)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/1.0.0...1.1.0)

**Merged pull requests:**

- Generate NetworkLatencyInjection resource [\#45](https://github.com/DataDog/chaos-controller/pull/45) ([Devatoria](https://github.com/Devatoria))

## [1.0.0](https://github.com/DataDog/chaos-controller/tree/1.0.0) (2020-01-09)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/0.6.2...1.0.0)

**Merged pull requests:**

- Upgrade project to kubebuilder v2 [\#44](https://github.com/DataDog/chaos-controller/pull/44) ([Devatoria](https://github.com/Devatoria))

## [0.6.2](https://github.com/DataDog/chaos-controller/tree/0.6.2) (2020-01-06)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/0.6.1...0.6.2)

**Merged pull requests:**

- Review Dockerfile to have smaller images for both manager and injector [\#43](https://github.com/DataDog/chaos-controller/pull/43) ([Devatoria](https://github.com/Devatoria))
- Improve doc and add injector stuff [\#42](https://github.com/DataDog/chaos-controller/pull/42) ([Devatoria](https://github.com/Devatoria))

## [0.6.1](https://github.com/DataDog/chaos-controller/tree/0.6.1) (2020-01-06)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/0.6...0.6.1)

**Merged pull requests:**

- Merge chaos-fi repository into chaos-fi-controller [\#41](https://github.com/DataDog/chaos-controller/pull/41) ([Devatoria](https://github.com/Devatoria))
- Fix minikube driver and docker service start for local testing [\#40](https://github.com/DataDog/chaos-controller/pull/40) ([Devatoria](https://github.com/Devatoria))

## [0.6](https://github.com/DataDog/chaos-controller/tree/0.6) (2019-10-23)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/0.5.1...0.6)

**Merged pull requests:**

- Have pods use local DNSConfig [\#38](https://github.com/DataDog/chaos-controller/pull/38) ([Azoam](https://github.com/Azoam))

## [0.5.1](https://github.com/DataDog/chaos-controller/tree/0.5.1) (2019-10-11)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/0.5.0...0.5.1)

**Merged pull requests:**

- Push to michelada account [\#37](https://github.com/DataDog/chaos-controller/pull/37) ([Devatoria](https://github.com/Devatoria))

## [0.5.0](https://github.com/DataDog/chaos-controller/tree/0.5.0) (2019-08-23)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/0.4.0...0.5.0)

**Merged pull requests:**

- Implement node failure shutdown feature [\#36](https://github.com/DataDog/chaos-controller/pull/36) ([Devatoria](https://github.com/Devatoria))

## [0.4.0](https://github.com/DataDog/chaos-controller/tree/0.4.0) (2019-07-19)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/0.3.0...0.4.0)

**Merged pull requests:**

- Failure targeting Unique Nodes [\#33](https://github.com/DataDog/chaos-controller/pull/33) ([Azoam](https://github.com/Azoam))

## [0.3.0](https://github.com/DataDog/chaos-controller/tree/0.3.0) (2019-07-12)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/0.2.0...0.3.0)

**Merged pull requests:**

- Pass probability field to injection pod [\#35](https://github.com/DataDog/chaos-controller/pull/35) ([Devatoria](https://github.com/Devatoria))

## [0.2.0](https://github.com/DataDog/chaos-controller/tree/0.2.0) (2019-07-08)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/0.1.1...0.2.0)

**Merged pull requests:**

- Improve doc [\#32](https://github.com/DataDog/chaos-controller/pull/32) ([Devatoria](https://github.com/Devatoria))
- Makes host optional in CRD definition [\#31](https://github.com/DataDog/chaos-controller/pull/31) ([Azoam](https://github.com/Azoam))
- Sam/infected node names [\#30](https://github.com/DataDog/chaos-controller/pull/30) ([Azoam](https://github.com/Azoam))

## [0.1.1](https://github.com/DataDog/chaos-controller/tree/0.1.1) (2019-06-25)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/0.1.0...0.1.1)

**Merged pull requests:**

- Fix cleanup pods being deleted before completion [\#29](https://github.com/DataDog/chaos-controller/pull/29) ([Devatoria](https://github.com/Devatoria))
- Improve local testing [\#28](https://github.com/DataDog/chaos-controller/pull/28) ([Devatoria](https://github.com/Devatoria))
- Add stuff to test the controller locally [\#27](https://github.com/DataDog/chaos-controller/pull/27) ([Devatoria](https://github.com/Devatoria))
- Add helpers package tests [\#26](https://github.com/DataDog/chaos-controller/pull/26) ([Devatoria](https://github.com/Devatoria))
- Update README with details about nfis [\#21](https://github.com/DataDog/chaos-controller/pull/21) ([kathy-huang](https://github.com/kathy-huang))

## [0.1.0](https://github.com/DataDog/chaos-controller/tree/0.1.0) (2019-05-02)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/0.0.6...0.1.0)

**Merged pull requests:**

- Add node failure CRD and controller [\#25](https://github.com/DataDog/chaos-controller/pull/25) ([Devatoria](https://github.com/Devatoria))

## [0.0.6](https://github.com/DataDog/chaos-controller/tree/0.0.6) (2019-04-25)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/0.0.5...0.0.6)

**Merged pull requests:**

- Change chaos-fi call due to rework [\#23](https://github.com/DataDog/chaos-controller/pull/23) ([Devatoria](https://github.com/Devatoria))
- update README about updating helm chart for controller [\#20](https://github.com/DataDog/chaos-controller/pull/20) ([kathy-huang](https://github.com/kathy-huang))
- add 'numPodsToTarget' field to crd to allow specifying a random numbe… [\#19](https://github.com/DataDog/chaos-controller/pull/19) ([kathy-huang](https://github.com/kathy-huang))
- Remove injection pod update in each Reconcile call [\#17](https://github.com/DataDog/chaos-controller/pull/17) ([kathy-huang](https://github.com/kathy-huang))

## [0.0.5](https://github.com/DataDog/chaos-controller/tree/0.0.5) (2019-04-12)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/0.0.4...0.0.5)

**Merged pull requests:**

- Remove pull policy from created pods [\#16](https://github.com/DataDog/chaos-controller/pull/16) ([Devatoria](https://github.com/Devatoria))

## [0.0.4](https://github.com/DataDog/chaos-controller/tree/0.0.4) (2019-04-10)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/0.0.3...0.0.4)

**Merged pull requests:**

- Pass instance UID to chaos-fi pods [\#15](https://github.com/DataDog/chaos-controller/pull/15) ([Devatoria](https://github.com/Devatoria))
- Improve CI by using the generic docker-push image [\#14](https://github.com/DataDog/chaos-controller/pull/14) ([Devatoria](https://github.com/Devatoria))

## [0.0.3](https://github.com/DataDog/chaos-controller/tree/0.0.3) (2019-04-10)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/98cd5eedd950b4ddc1db2db55df0529f5e4b0d03...0.0.3)

**Merged pull requests:**

- Add datadog metrics and events [\#13](https://github.com/DataDog/chaos-controller/pull/13) ([Devatoria](https://github.com/Devatoria))
- CI improvement [\#12](https://github.com/DataDog/chaos-controller/pull/12) ([Devatoria](https://github.com/Devatoria))
- Define the chaos-fi image value via an environment variable [\#11](https://github.com/DataDog/chaos-controller/pull/11) ([Devatoria](https://github.com/Devatoria))
- :wrench: set namespace when creating object instead since listoptions… [\#10](https://github.com/DataDog/chaos-controller/pull/10) ([kathy-huang](https://github.com/kathy-huang))
- Match pods to DFI using namespace in addition to label selector [\#9](https://github.com/DataDog/chaos-controller/pull/9) ([kathy-huang](https://github.com/kathy-huang))
- add a check in case label selector is missing from CRD spec to preven… [\#8](https://github.com/DataDog/chaos-controller/pull/8) ([kathy-huang](https://github.com/kathy-huang))
- rename DependencyFailureInjection -\> NetworkFailureInjection [\#7](https://github.com/DataDog/chaos-controller/pull/7) ([kathy-huang](https://github.com/kathy-huang))
- use labels.Selector type instead of just string [\#6](https://github.com/DataDog/chaos-controller/pull/6) ([kathy-huang](https://github.com/kathy-huang))
- Add basic CI [\#5](https://github.com/DataDog/chaos-controller/pull/5) ([Devatoria](https://github.com/Devatoria))
- Move helm chart to the k8s-resources repository [\#4](https://github.com/DataDog/chaos-controller/pull/4) ([Devatoria](https://github.com/Devatoria))
- add standard labels to helm chart [\#3](https://github.com/DataDog/chaos-controller/pull/3) ([kathy-huang](https://github.com/kathy-huang))
- Kathy/add cleanup pod [\#2](https://github.com/DataDog/chaos-controller/pull/2) ([kathy-huang](https://github.com/kathy-huang))
- Add label selector to CRD [\#1](https://github.com/DataDog/chaos-controller/pull/1) ([kathy-huang](https://github.com/kathy-huang))



\* *This Changelog was automatically generated by [github_changelog_generator](https://github.com/github-changelog-generator/github-changelog-generator)*
