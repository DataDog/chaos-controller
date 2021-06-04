# Changelog

## [4.0.0](https://github.com/DataDog/chaos-controller/tree/4.0.0) (2021-06-04)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/3.10.0...4.0.0)

**Closed issues:**

- User Issue: issue with cgroups in EKS [\#317](https://github.com/DataDog/chaos-controller/issues/317)

**Merged pull requests:**

- \[CHAOS-45\] Add Documentation to the ChaosCLI [\#331](https://github.com/DataDog/chaos-controller/pull/331) ([Azoam](https://github.com/Azoam))
- Update the latest supported k8s version to 1.21.1 [\#329](https://github.com/DataDog/chaos-controller/pull/329) ([ptnapoleon](https://github.com/ptnapoleon))
- Fix the Validatecount experience [\#327](https://github.com/DataDog/chaos-controller/pull/327) ([ptnapoleon](https://github.com/ptnapoleon))
- chaosli validators shouldnt require input, survey.Required exists for… [\#326](https://github.com/DataDog/chaos-controller/pull/326) ([ptnapoleon](https://github.com/ptnapoleon))
- CHAOS-85: Create all pods in chaos engineering [\#325](https://github.com/DataDog/chaos-controller/pull/325) ([ptnapoleon](https://github.com/ptnapoleon))
- Improve developer documentation [\#324](https://github.com/DataDog/chaos-controller/pull/324) ([takakonishimura](https://github.com/takakonishimura))
- Generalize cgroup path construction [\#323](https://github.com/DataDog/chaos-controller/pull/323) ([blazebissar](https://github.com/blazebissar))
- Increment a metric when a disruption fails ValidateCreate [\#322](https://github.com/DataDog/chaos-controller/pull/322) ([ptnapoleon](https://github.com/ptnapoleon))
- Temporarily make validateServices conditional, to help CLIs [\#321](https://github.com/DataDog/chaos-controller/pull/321) ([ptnapoleon](https://github.com/ptnapoleon))
- CORE-702: Validation call from CLI PR [\#320](https://github.com/DataDog/chaos-controller/pull/320) ([nathan-tournant](https://github.com/nathan-tournant))
- Tweak configs for Fedora/Podman compatibility [\#319](https://github.com/DataDog/chaos-controller/pull/319) ([blazebissar](https://github.com/blazebissar))
- CHAOS-43: Yaml to English Feature [\#318](https://github.com/DataDog/chaos-controller/pull/318) ([Azoam](https://github.com/Azoam))
- scope netowrkdisruption.port to hosts in error msg [\#316](https://github.com/DataDog/chaos-controller/pull/316) ([takakonishimura](https://github.com/takakonishimura))
- README has emojis [\#315](https://github.com/DataDog/chaos-controller/pull/315) ([takakonishimura](https://github.com/takakonishimura))
- link to external docs on signing GitHub commits with gpg [\#314](https://github.com/DataDog/chaos-controller/pull/314) ([takakonishimura](https://github.com/takakonishimura))
- Readable README [\#313](https://github.com/DataDog/chaos-controller/pull/313) ([takakonishimura](https://github.com/takakonishimura))
- Increase the amount of documentation on how to use the new services f… [\#312](https://github.com/DataDog/chaos-controller/pull/312) ([ptnapoleon](https://github.com/ptnapoleon))
- Add the basic structure for the new CLI [\#311](https://github.com/DataDog/chaos-controller/pull/311) ([ptnapoleon](https://github.com/ptnapoleon))
- Rename master to main for branch ref in release action [\#310](https://github.com/DataDog/chaos-controller/pull/310) ([Devatoria](https://github.com/Devatoria))
- Validate Update: global disruption scope [\#309](https://github.com/DataDog/chaos-controller/pull/309) ([nathan-tournant](https://github.com/nathan-tournant))
- CHAOS-51: Implement the "create" command of chaosli; Help users build a disruption spec [\#307](https://github.com/DataDog/chaos-controller/pull/307) ([ptnapoleon](https://github.com/ptnapoleon))
- Better handle the case where count is set to -1 [\#305](https://github.com/DataDog/chaos-controller/pull/305) ([ptnapoleon](https://github.com/ptnapoleon))
- Upgrade to cobra v1.1.3 from v0.0.5 [\#302](https://github.com/DataDog/chaos-controller/pull/302) ([ptnapoleon](https://github.com/ptnapoleon))

## [3.10.0](https://github.com/DataDog/chaos-controller/tree/3.10.0) (2021-05-20)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/3.9.0...3.10.0)

**Merged pull requests:**

- Switch from kustomize to helm to generate manifests [\#308](https://github.com/DataDog/chaos-controller/pull/308) ([Devatoria](https://github.com/Devatoria))
- Fix CI dockerhub push not logging in [\#306](https://github.com/DataDog/chaos-controller/pull/306) ([Devatoria](https://github.com/Devatoria))
- Fix dead links in README and other docs [\#304](https://github.com/DataDog/chaos-controller/pull/304) ([ptnapoleon](https://github.com/ptnapoleon))
- CORE-84: Prevent config of ingress flow & hosts lists on Network Disruptions [\#303](https://github.com/DataDog/chaos-controller/pull/303) ([takakonishimura](https://github.com/takakonishimura))
- Update container release\_changelog\_202961fc69dceef1d2adc83a7d8f5fa76d9d1a5b [\#301](https://github.com/DataDog/chaos-controller/pull/301) ([github-actions[bot]](https://github.com/apps/github-actions))
- Allow services filtering in the network disruption [\#300](https://github.com/DataDog/chaos-controller/pull/300) ([Devatoria](https://github.com/Devatoria))
- Basic logistics for open-sourcing Chaos Controller [\#299](https://github.com/DataDog/chaos-controller/pull/299) ([takakonishimura](https://github.com/takakonishimura))

## [3.9.0](https://github.com/DataDog/chaos-controller/tree/3.9.0) (2021-05-10)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/3.8.1...3.9.0)

**Merged pull requests:**

- Update container release\_changelog\_0ecdd93d8f7041ae97144d3804c61dd913df3541 [\#296](https://github.com/DataDog/chaos-controller/pull/296) ([github-actions[bot]](https://github.com/apps/github-actions))
- CORE-594: Add Clientset code needing for using client-go with Disruption Resource [\#288](https://github.com/DataDog/chaos-controller/pull/288) ([ptnapoleon](https://github.com/ptnapoleon))

## [3.8.1](https://github.com/DataDog/chaos-controller/tree/3.8.1) (2021-04-30)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/3.8.0...3.8.1)

**Merged pull requests:**

- Log into docker hub on pre-release to avoid pull rate limiting [\#295](https://github.com/DataDog/chaos-controller/pull/295) ([Devatoria](https://github.com/Devatoria))
- Update container release\_changelog\_28c1d25c18f250a0dc42a2b003108fd51484f666 [\#294](https://github.com/DataDog/chaos-controller/pull/294) ([github-actions[bot]](https://github.com/apps/github-actions))

## [3.8.0](https://github.com/DataDog/chaos-controller/tree/3.8.0) (2021-04-29)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/3.7.1...3.8.0)

**Merged pull requests:**

- Allow to specify cert dir for admission webhook [\#293](https://github.com/DataDog/chaos-controller/pull/293) ([Devatoria](https://github.com/Devatoria))
- CORE-540: Bugfix: DNS Disruption on one container within a pod only disrupts targeted container [\#292](https://github.com/DataDog/chaos-controller/pull/292) ([nathan-tournant](https://github.com/nathan-tournant))
- Bootstrap the admission controller [\#291](https://github.com/DataDog/chaos-controller/pull/291) ([Devatoria](https://github.com/Devatoria))
- Exclude export comment rule from linting [\#290](https://github.com/DataDog/chaos-controller/pull/290) ([ptnapoleon](https://github.com/ptnapoleon))
- Update container release\_changelog\_5fcdae27a69f4626fa35fb33f878dbb13b775a3b [\#289](https://github.com/DataDog/chaos-controller/pull/289) ([github-actions[bot]](https://github.com/apps/github-actions))

## [3.7.1](https://github.com/DataDog/chaos-controller/tree/3.7.1) (2021-04-13)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/3.7.0...3.7.1)

**Merged pull requests:**

- Fix spelling error on demo sample [\#287](https://github.com/DataDog/chaos-controller/pull/287) ([clairecng](https://github.com/clairecng))
- Apply tc rules and filters to all interfaces instead of trying to guess the one to choose [\#286](https://github.com/DataDog/chaos-controller/pull/286) ([Devatoria](https://github.com/Devatoria))
- Ignore invalid link index when applying a network disruption [\#285](https://github.com/DataDog/chaos-controller/pull/285) ([Devatoria](https://github.com/Devatoria))
- Improve targeting documentation [\#284](https://github.com/DataDog/chaos-controller/pull/284) ([ptnapoleon](https://github.com/ptnapoleon))
- CORE-623: use zap logger after logger init [\#283](https://github.com/DataDog/chaos-controller/pull/283) ([nathan-tournant](https://github.com/nathan-tournant))
- Fix interface txqlen not being cleared on network disruption apply error [\#282](https://github.com/DataDog/chaos-controller/pull/282) ([Devatoria](https://github.com/Devatoria))
- CORE-586: Fix the local Kubernetes version to 1.19 [\#281](https://github.com/DataDog/chaos-controller/pull/281) ([Devatoria](https://github.com/Devatoria))
- Update container release\_changelog\_12b93b1a36e3872a4a705d28f78f0ba15f3f34f3 [\#280](https://github.com/DataDog/chaos-controller/pull/280) ([github-actions[bot]](https://github.com/apps/github-actions))
- change chaos injector binary [\#278](https://github.com/DataDog/chaos-controller/pull/278) ([Azoam](https://github.com/Azoam))

## [3.7.0](https://github.com/DataDog/chaos-controller/tree/3.7.0) (2021-03-24)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/3.6.0...3.7.0)

**Merged pull requests:**

- CORE-497: add context to injector logger through injector CLI and its chaos-controller controller [\#279](https://github.com/DataDog/chaos-controller/pull/279) ([nathan-tournant](https://github.com/nathan-tournant))
- Explain in the README how to target a specific pod by name [\#277](https://github.com/DataDog/chaos-controller/pull/277) ([ptnapoleon](https://github.com/ptnapoleon))
- CORE-507: Single entry point for injector CLI common args [\#276](https://github.com/DataDog/chaos-controller/pull/276) ([nathan-tournant](https://github.com/nathan-tournant))
- Revamp samples [\#275](https://github.com/DataDog/chaos-controller/pull/275) ([Devatoria](https://github.com/Devatoria))
- CORE-528 - add MetricRestart on chaos-controller [\#274](https://github.com/DataDog/chaos-controller/pull/274) ([nathan-tournant](https://github.com/nathan-tournant))
- Update container release\_changelog\_8bf23cad59208194fdef5c75ef7cb9a96803d0a1 [\#272](https://github.com/DataDog/chaos-controller/pull/272) ([github-actions[bot]](https://github.com/apps/github-actions))
- CORE-371: Network Disruption Documentation Revamp [\#254](https://github.com/DataDog/chaos-controller/pull/254) ([takakonishimura](https://github.com/takakonishimura))

## [3.6.0](https://github.com/DataDog/chaos-controller/tree/3.6.0) (2021-03-11)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/3.5.2...3.6.0)

**Merged pull requests:**

- CORE-467: Emit a disruption count metric per kind [\#271](https://github.com/DataDog/chaos-controller/pull/271) ([ptnapoleon](https://github.com/ptnapoleon))
- CORE-601: Clean up chaos pods if their targets dont exist [\#270](https://github.com/DataDog/chaos-controller/pull/270) ([ptnapoleon](https://github.com/ptnapoleon))
- Change some log text to clarify when ignored targets is too large [\#268](https://github.com/DataDog/chaos-controller/pull/268) ([ptnapoleon](https://github.com/ptnapoleon))
- CORE-458: Gather ongoing duration separately from completed duration … [\#267](https://github.com/DataDog/chaos-controller/pull/267) ([ptnapoleon](https://github.com/ptnapoleon))
- Update container release\_changelog\_54aa80140063186b36a07d28677f191cd414ed36 [\#266](https://github.com/DataDog/chaos-controller/pull/266) ([github-actions[bot]](https://github.com/apps/github-actions))
- CORE-585: Log found chaos pod names when injectors already exist for … [\#259](https://github.com/DataDog/chaos-controller/pull/259) ([ptnapoleon](https://github.com/ptnapoleon))

## [3.5.2](https://github.com/DataDog/chaos-controller/tree/3.5.2) (2021-03-05)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/3.5.1...3.5.2)

**Merged pull requests:**

- Better handling of no targets disruptions [\#265](https://github.com/DataDog/chaos-controller/pull/265) ([Devatoria](https://github.com/Devatoria))

## [3.5.1](https://github.com/DataDog/chaos-controller/tree/3.5.1) (2021-03-04)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/3.5.0...3.5.1)

**Merged pull requests:**

- Subtract ignored targets from targets count [\#263](https://github.com/DataDog/chaos-controller/pull/263) ([Devatoria](https://github.com/Devatoria))

## [3.5.0](https://github.com/DataDog/chaos-controller/tree/3.5.0) (2021-03-04)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/3.4.0...3.5.0)

**Merged pull requests:**

- CORE-467: Collect a duration metric for the entire disruption duration [\#261](https://github.com/DataDog/chaos-controller/pull/261) ([ptnapoleon](https://github.com/ptnapoleon))
- CORE-467: Replace "count" metrics with more accurate gauge names [\#260](https://github.com/DataDog/chaos-controller/pull/260) ([ptnapoleon](https://github.com/ptnapoleon))
- CORE-582: Resolve kubernetes.default before disrupting [\#258](https://github.com/DataDog/chaos-controller/pull/258) ([ptnapoleon](https://github.com/ptnapoleon))
- Handle the chaos pods termination not only on disruption deletion [\#257](https://github.com/DataDog/chaos-controller/pull/257) ([Devatoria](https://github.com/Devatoria))
- Fix expectChaosPods assertion from 16 to 20 [\#256](https://github.com/DataDog/chaos-controller/pull/256) ([ptnapoleon](https://github.com/ptnapoleon))
- \[CORE-423\] Add delete only mode to the chaos controller [\#255](https://github.com/DataDog/chaos-controller/pull/255) ([Azoam](https://github.com/Azoam))
- Update container release\_changelog\_7f93155a1f416dbea9d48bd913a68cf316a6a487 [\#252](https://github.com/DataDog/chaos-controller/pull/252) ([github-actions[bot]](https://github.com/apps/github-actions))
- Instance multiple injectors when targeting multiple containers [\#244](https://github.com/DataDog/chaos-controller/pull/244) ([Devatoria](https://github.com/Devatoria))

## [3.4.0](https://github.com/DataDog/chaos-controller/tree/3.4.0) (2021-02-11)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/3.3.0...3.4.0)

**Merged pull requests:**

- Fix wrong error checking leaking chaos pods [\#251](https://github.com/DataDog/chaos-controller/pull/251) ([Devatoria](https://github.com/Devatoria))
- Fix expectChaosPod assertions after adding DNS spec [\#250](https://github.com/DataDog/chaos-controller/pull/250) ([ptnapoleon](https://github.com/ptnapoleon))
- Generate injector pods from code instead of from template [\#249](https://github.com/DataDog/chaos-controller/pull/249) ([Devatoria](https://github.com/Devatoria))
- CORE-527: Fix blocking delete in test cleanup for the disruption controller [\#248](https://github.com/DataDog/chaos-controller/pull/248) ([ptnapoleon](https://github.com/ptnapoleon))
- Update container release\_changelog\_8432064109352c764e8a6b0375def3d70a2ed4f6 [\#247](https://github.com/DataDog/chaos-controller/pull/247) ([github-actions[bot]](https://github.com/apps/github-actions))

## [3.3.0](https://github.com/DataDog/chaos-controller/tree/3.3.0) (2021-02-08)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/3.2.0...3.3.0)

**Merged pull requests:**

- Send ongoing disruptions count metric [\#246](https://github.com/DataDog/chaos-controller/pull/246) ([Devatoria](https://github.com/Devatoria))
- Updating artifact metadata script to match spec [\#245](https://github.com/DataDog/chaos-controller/pull/245) ([griffin](https://github.com/griffin))
- Use Copyright 2021 for all new license headers [\#241](https://github.com/DataDog/chaos-controller/pull/241) ([ptnapoleon](https://github.com/ptnapoleon))
- Ignore some edge cases failures from the injector on deletion [\#240](https://github.com/DataDog/chaos-controller/pull/240) ([Devatoria](https://github.com/Devatoria))
- Update container release\_changelog\_c04ce2bde80462eb95ae762ceb67f82673bd5641 [\#239](https://github.com/DataDog/chaos-controller/pull/239) ([github-actions[bot]](https://github.com/apps/github-actions))
- CORE-439: DNS Failure Disruptions [\#237](https://github.com/DataDog/chaos-controller/pull/237) ([ptnapoleon](https://github.com/ptnapoleon))

## [3.2.0](https://github.com/DataDog/chaos-controller/tree/3.2.0) (2021-01-25)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/3.1.0...3.2.0)

**Merged pull requests:**

- Rename target\_pod tag to target and remove phase tag from metrics [\#238](https://github.com/DataDog/chaos-controller/pull/238) ([Devatoria](https://github.com/Devatoria))
- Set request context to controller logger [\#236](https://github.com/DataDog/chaos-controller/pull/236) ([Devatoria](https://github.com/Devatoria))
- Ensure duration metrics are sent [\#235](https://github.com/DataDog/chaos-controller/pull/235) ([Devatoria](https://github.com/Devatoria))
- Set a common logger with adapter encoding parameters [\#234](https://github.com/DataDog/chaos-controller/pull/234) ([Devatoria](https://github.com/Devatoria))
- Update container release\_changelog\_c4008f6bac5819725605554fe0fde6b1f299787e [\#233](https://github.com/DataDog/chaos-controller/pull/233) ([github-actions[bot]](https://github.com/apps/github-actions))

## [3.1.0](https://github.com/DataDog/chaos-controller/tree/3.1.0) (2021-01-22)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/3.0.1...3.1.0)

**Merged pull requests:**

- Change controller log format to zap production format [\#232](https://github.com/DataDog/chaos-controller/pull/232) ([Devatoria](https://github.com/Devatoria))
- Ensure DNS resolution response is not nil to avoid a panic [\#231](https://github.com/DataDog/chaos-controller/pull/231) ([Devatoria](https://github.com/Devatoria))
- Update container release\_changelog\_a75ada688ede489cd23b45999c7413f26368db70 [\#230](https://github.com/DataDog/chaos-controller/pull/230) ([github-actions[bot]](https://github.com/apps/github-actions))
- Wait for chaos pod to be created and persisted before returning from the reconcile loop [\#229](https://github.com/DataDog/chaos-controller/pull/229) ([Devatoria](https://github.com/Devatoria))
- Use Packer for minikube iso, upgrade minikube k8s version [\#227](https://github.com/DataDog/chaos-controller/pull/227) ([noqcks](https://github.com/noqcks))
- CORE-443: Log the container ID, Name, and other labels when a chaos pod is injecting into a container [\#221](https://github.com/DataDog/chaos-controller/pull/221) ([ptnapoleon](https://github.com/ptnapoleon))

## [3.0.1](https://github.com/DataDog/chaos-controller/tree/3.0.1) (2021-01-20)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/3.0.0...3.0.1)

**Merged pull requests:**

- Fix disk pressure validator [\#228](https://github.com/DataDog/chaos-controller/pull/228) ([Devatoria](https://github.com/Devatoria))
- Update container release\_changelog\_0e23c15c27dda702c10ea119abff1be84760c8be [\#225](https://github.com/DataDog/chaos-controller/pull/225) ([github-actions[bot]](https://github.com/apps/github-actions))

## [3.0.0](https://github.com/DataDog/chaos-controller/tree/3.0.0) (2021-01-18)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/2.16.1...3.0.0)

**Closed issues:**

- Inaccurate comment in node\_failure? [\#223](https://github.com/DataDog/chaos-controller/issues/223)

**Merged pull requests:**

- Implement a dry-run mode to injectors [\#224](https://github.com/DataDog/chaos-controller/pull/224) ([Devatoria](https://github.com/Devatoria))
- Update README.md [\#222](https://github.com/DataDog/chaos-controller/pull/222) ([noqcks](https://github.com/noqcks))
- Restore computeHash control flow to 911e426583 [\#220](https://github.com/DataDog/chaos-controller/pull/220) ([ptnapoleon](https://github.com/ptnapoleon))
- CORE-450: Add manifests diff check [\#219](https://github.com/DataDog/chaos-controller/pull/219) ([noqcks](https://github.com/noqcks))
- Improve disruption injection status [\#218](https://github.com/DataDog/chaos-controller/pull/218) ([Devatoria](https://github.com/Devatoria))
- Improve reconcile loop by splitting it into smaller functions [\#217](https://github.com/DataDog/chaos-controller/pull/217) ([Devatoria](https://github.com/Devatoria))
- CORE-436: Add disruption event when an invalid networkdisruptionspec is applied [\#216](https://github.com/DataDog/chaos-controller/pull/216) ([ptnapoleon](https://github.com/ptnapoleon))
- Add a label with the disruption name to chaos pods to ease filtering [\#215](https://github.com/DataDog/chaos-controller/pull/215) ([Devatoria](https://github.com/Devatoria))
- Retry to cleanup if it fails [\#214](https://github.com/DataDog/chaos-controller/pull/214) ([Devatoria](https://github.com/Devatoria))
- Update container release\_changelog\_9d064314ebf0e664c506de0fe260275e34c21153 [\#213](https://github.com/DataDog/chaos-controller/pull/213) ([github-actions[bot]](https://github.com/apps/github-actions))
- Use a single long-running injector pod [\#211](https://github.com/DataDog/chaos-controller/pull/211) ([Devatoria](https://github.com/Devatoria))

## [2.16.1](https://github.com/DataDog/chaos-controller/tree/2.16.1) (2021-01-07)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/2.16.0...2.16.1)

**Merged pull requests:**

- Force node failure disruption level to node instead of pod [\#212](https://github.com/DataDog/chaos-controller/pull/212) ([Devatoria](https://github.com/Devatoria))
- Update container release\_changelog\_4370a7b1f671cf5a4589ac2c4fd012f634c6f09c [\#210](https://github.com/DataDog/chaos-controller/pull/210) ([github-actions[bot]](https://github.com/apps/github-actions))

## [2.16.0](https://github.com/DataDog/chaos-controller/tree/2.16.0) (2021-01-05)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/2.15.0...2.16.0)

**Closed issues:**

- count: -1 does not work [\#197](https://github.com/DataDog/chaos-controller/issues/197)

**Merged pull requests:**

- CORE-417: Pass --level flag to injector for node\_failures [\#209](https://github.com/DataDog/chaos-controller/pull/209) ([ptnapoleon](https://github.com/ptnapoleon))
- CORE-414: Validate label selector grammar [\#208](https://github.com/DataDog/chaos-controller/pull/208) ([ptnapoleon](https://github.com/ptnapoleon))
- Ignore already cleaned qdiscs during network disruption cleanup [\#207](https://github.com/DataDog/chaos-controller/pull/207) ([Devatoria](https://github.com/Devatoria))
- Improve logging and dump the disruption selector for debug [\#206](https://github.com/DataDog/chaos-controller/pull/206) ([Devatoria](https://github.com/Devatoria))
- Skip target on injection error instead of stopping the reconcile loop [\#205](https://github.com/DataDog/chaos-controller/pull/205) ([Devatoria](https://github.com/Devatoria))
- CORE-402: Another way to test only selecting Running Pods [\#204](https://github.com/DataDog/chaos-controller/pull/204) ([takakonishimura](https://github.com/takakonishimura))
- \[Doc\] - Update sample documentation for count [\#202](https://github.com/DataDog/chaos-controller/pull/202) ([gaetan-deputier](https://github.com/gaetan-deputier))
- Node level disruptions [\#198](https://github.com/DataDog/chaos-controller/pull/198) ([Devatoria](https://github.com/Devatoria))
- CORE-296: Check pods are Running before Injection [\#196](https://github.com/DataDog/chaos-controller/pull/196) ([takakonishimura](https://github.com/takakonishimura))
- Add jitter for delay to the chaos-controller [\#195](https://github.com/DataDog/chaos-controller/pull/195) ([Azoam](https://github.com/Azoam))
- Delete unused metrics.go file~ [\#194](https://github.com/DataDog/chaos-controller/pull/194) ([takakonishimura](https://github.com/takakonishimura))
- Add event when disruption name is not recognizable [\#193](https://github.com/DataDog/chaos-controller/pull/193) ([Azoam](https://github.com/Azoam))
- Ignore license headers for api auto-generated files [\#192](https://github.com/DataDog/chaos-controller/pull/192) ([Devatoria](https://github.com/Devatoria))
- Update container release\_changelog\_8eb5743aac6412002622254c71390a3d74ba93b7 [\#191](https://github.com/DataDog/chaos-controller/pull/191) ([github-actions[bot]](https://github.com/apps/github-actions))

## [2.15.0](https://github.com/DataDog/chaos-controller/tree/2.15.0) (2020-11-23)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/2.14.2...2.15.0)

**Merged pull requests:**

- Use github repos for spdx because it no longer exists in the pip repo [\#190](https://github.com/DataDog/chaos-controller/pull/190) ([Azoam](https://github.com/Azoam))
- Adding the ability to target specific container in targeted Pod [\#189](https://github.com/DataDog/chaos-controller/pull/189) ([Azoam](https://github.com/Azoam))
- Add duplication to network disruption [\#188](https://github.com/DataDog/chaos-controller/pull/188) ([Azoam](https://github.com/Azoam))
- Add network disruption node level injection workaround [\#187](https://github.com/DataDog/chaos-controller/pull/187) ([Devatoria](https://github.com/Devatoria))
- Update container release\_changelog\_46f81598753b5c6c1a5ce38cdaea16a694927037 [\#186](https://github.com/DataDog/chaos-controller/pull/186) ([github-actions[bot]](https://github.com/apps/github-actions))
- Count percent [\#170](https://github.com/DataDog/chaos-controller/pull/170) ([Azoam](https://github.com/Azoam))

## [2.14.2](https://github.com/DataDog/chaos-controller/tree/2.14.2) (2020-11-03)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/2.14.1...2.14.2)

**Merged pull requests:**

- Re-use existing container net\_cls classid [\#185](https://github.com/DataDog/chaos-controller/pull/185) ([Devatoria](https://github.com/Devatoria))
- Update container release\_changelog\_001ae0f8f2fbed78ffdf12f2268528db67696757 [\#184](https://github.com/DataDog/chaos-controller/pull/184) ([github-actions[bot]](https://github.com/apps/github-actions))

## [2.14.1](https://github.com/DataDog/chaos-controller/tree/2.14.1) (2020-11-02)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/2.14.0...2.14.1)

**Merged pull requests:**

- Skip net\_cls cgroup cleanup if it does not exist [\#183](https://github.com/DataDog/chaos-controller/pull/183) ([Devatoria](https://github.com/Devatoria))
- Update container release\_changelog\_c839b80431c5f9253b2d5d6d5716b8710424e26b [\#182](https://github.com/DataDog/chaos-controller/pull/182) ([github-actions[bot]](https://github.com/apps/github-actions))

## [2.14.0](https://github.com/DataDog/chaos-controller/tree/2.14.0) (2020-11-02)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/2.13.2...2.14.0)

**Merged pull requests:**

- Target only interfaces listed in the routing table when no host is specified [\#181](https://github.com/DataDog/chaos-controller/pull/181) ([Devatoria](https://github.com/Devatoria))
- Host network source ip [\#180](https://github.com/DataDog/chaos-controller/pull/180) ([Devatoria](https://github.com/Devatoria))
- Improve minikube iso building documentation [\#179](https://github.com/DataDog/chaos-controller/pull/179) ([Devatoria](https://github.com/Devatoria))
- Update container release\_changelog\_de418b49d5c362c4d7091d6602d2a7dbe7acc451 [\#178](https://github.com/DataDog/chaos-controller/pull/178) ([github-actions[bot]](https://github.com/apps/github-actions))

## [2.13.2](https://github.com/DataDog/chaos-controller/tree/2.13.2) (2020-10-23)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/2.13.1...2.13.2)

**Merged pull requests:**

- Exclude node IP from node disruptions [\#177](https://github.com/DataDog/chaos-controller/pull/177) ([Devatoria](https://github.com/Devatoria))
- Update container release\_changelog\_a7f35e3936c954addaafcc8d464d91a4244ceae5 [\#176](https://github.com/DataDog/chaos-controller/pull/176) ([github-actions[bot]](https://github.com/apps/github-actions))

## [2.13.1](https://github.com/DataDog/chaos-controller/tree/2.13.1) (2020-10-23)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/2.13.0...2.13.1)

**Merged pull requests:**

- Exclude default route gateway IP on network disruption [\#175](https://github.com/DataDog/chaos-controller/pull/175) ([Devatoria](https://github.com/Devatoria))
- Update container release\_changelog\_fb9ad708ebbe229e86ba83e7d33be08a42f802b2 [\#173](https://github.com/DataDog/chaos-controller/pull/173) ([github-actions[bot]](https://github.com/apps/github-actions))

## [2.13.0](https://github.com/DataDog/chaos-controller/tree/2.13.0) (2020-10-22)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/2.12.0...2.13.0)

**Merged pull requests:**

- Retry cleanup when it fails [\#172](https://github.com/DataDog/chaos-controller/pull/172) ([Devatoria](https://github.com/Devatoria))
- Allow empty string as protocol in network disruption protocol enum list [\#171](https://github.com/DataDog/chaos-controller/pull/171) ([Devatoria](https://github.com/Devatoria))
- Write spec hash in resource status to detect resource changes [\#169](https://github.com/DataDog/chaos-controller/pull/169) ([Devatoria](https://github.com/Devatoria))
- Adds small section on how to deploy [\#168](https://github.com/DataDog/chaos-controller/pull/168) ([brandon-dd](https://github.com/brandon-dd))
- Improve contributing doc [\#167](https://github.com/DataDog/chaos-controller/pull/167) ([Devatoria](https://github.com/Devatoria))
- Improve examples by splitting them in per use-cases examples [\#166](https://github.com/DataDog/chaos-controller/pull/166) ([Devatoria](https://github.com/Devatoria))
- Update container release\_changelog\_8332266aebf78e47a3a34d2ea15d1767d69fb963 [\#165](https://github.com/DataDog/chaos-controller/pull/165) ([github-actions[bot]](https://github.com/apps/github-actions))

## [2.12.0](https://github.com/DataDog/chaos-controller/tree/2.12.0) (2020-09-28)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/2.11.0...2.12.0)

**Merged pull requests:**

- Allow to filter on ingress traffic [\#164](https://github.com/DataDog/chaos-controller/pull/164) ([Devatoria](https://github.com/Devatoria))
- Update container release\_changelog\_b61107ba16290b8046f795d9b8543bceda7417ba [\#163](https://github.com/DataDog/chaos-controller/pull/163) ([github-actions[bot]](https://github.com/apps/github-actions))

## [2.11.0](https://github.com/DataDog/chaos-controller/tree/2.11.0) (2020-09-22)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/2.10.1...2.11.0)

**Merged pull requests:**

- Complete FAQ with a note to retry cleanup phase [\#162](https://github.com/DataDog/chaos-controller/pull/162) ([Devatoria](https://github.com/Devatoria))
- Send an event when no target can be found from the given label selector [\#161](https://github.com/DataDog/chaos-controller/pull/161) ([Devatoria](https://github.com/Devatoria))
- Switch injector resources to zero since we have the priority class now [\#160](https://github.com/DataDog/chaos-controller/pull/160) ([Devatoria](https://github.com/Devatoria))
- Push to ddbuild ecr on release [\#159](https://github.com/DataDog/chaos-controller/pull/159) ([Azoam](https://github.com/Azoam))
- Improve usage documentation and examples comments [\#158](https://github.com/DataDog/chaos-controller/pull/158) ([Devatoria](https://github.com/Devatoria))
- Simplify reconcile loop [\#157](https://github.com/DataDog/chaos-controller/pull/157) ([Devatoria](https://github.com/Devatoria))
- Remove unused vendor [\#156](https://github.com/DataDog/chaos-controller/pull/156) ([Devatoria](https://github.com/Devatoria))
- Update container release\_changelog\_4c18d4fb074ce53c8bb8468c7087d9f8f218dfe9 [\#155](https://github.com/DataDog/chaos-controller/pull/155) ([github-actions[bot]](https://github.com/apps/github-actions))

## [2.10.1](https://github.com/DataDog/chaos-controller/tree/2.10.1) (2020-08-17)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/2.10.0...2.10.1)

**Merged pull requests:**

- Fix release registries [\#154](https://github.com/DataDog/chaos-controller/pull/154) ([Devatoria](https://github.com/Devatoria))
- Update container release\_changelog\_73a3b846f65f4d6f352e81cf49382567dbf4094e [\#153](https://github.com/DataDog/chaos-controller/pull/153) ([github-actions[bot]](https://github.com/apps/github-actions))

## [2.10.0](https://github.com/DataDog/chaos-controller/tree/2.10.0) (2020-08-13)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/2.9.0...2.10.0)

**Merged pull requests:**

- Release images on unified artifact registry [\#152](https://github.com/DataDog/chaos-controller/pull/152) ([Devatoria](https://github.com/Devatoria))
- Add a small comment on known issue with current disk throttling implementation [\#151](https://github.com/DataDog/chaos-controller/pull/151) ([Devatoria](https://github.com/Devatoria))
- Move mocks from test packages to re-use them [\#150](https://github.com/DataDog/chaos-controller/pull/150) ([Devatoria](https://github.com/Devatoria))
- Add retry logic to dns resolution [\#149](https://github.com/DataDog/chaos-controller/pull/149) ([Azoam](https://github.com/Azoam))
- Record events on targeted pods on injection and cleanup [\#148](https://github.com/DataDog/chaos-controller/pull/148) ([Devatoria](https://github.com/Devatoria))
- Injector mounts rework [\#147](https://github.com/DataDog/chaos-controller/pull/147) ([Devatoria](https://github.com/Devatoria))
- Update container release\_changelog\_93dde7cf2cd52f65ba84620dfa48027b17d33b2f [\#146](https://github.com/DataDog/chaos-controller/pull/146) ([github-actions[bot]](https://github.com/apps/github-actions))

## [2.9.0](https://github.com/DataDog/chaos-controller/tree/2.9.0) (2020-07-23)

[Full Changelog](https://github.com/DataDog/chaos-controller/compare/2.8.0...2.9.0)

**Merged pull requests:**

- Network optional fields [\#145](https://github.com/DataDog/chaos-controller/pull/145) ([Devatoria](https://github.com/Devatoria))
- Merge network disruptions under a common field [\#144](https://github.com/DataDog/chaos-controller/pull/144) ([Devatoria](https://github.com/Devatoria))
- Enforce injector pods priority and qos to ensure they are not evicted easily [\#143](https://github.com/DataDog/chaos-controller/pull/143) ([Devatoria](https://github.com/Devatoria))
- Update container release\_changelog\_28fa800ad7c4dbe7407a658c886d70b08d146d74 [\#142](https://github.com/DataDog/chaos-controller/pull/142) ([github-actions[bot]](https://github.com/apps/github-actions))
- Rework network fail [\#137](https://github.com/DataDog/chaos-controller/pull/137) ([Azoam](https://github.com/Azoam))

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
