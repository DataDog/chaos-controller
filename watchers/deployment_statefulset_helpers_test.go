// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package watchers_test

import (
	"encoding/hex"
	"encoding/json"
	"reflect"
	"strings"
	"testing/quick"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	"github.com/DataDog/chaos-controller/watchers"
)

func makeContainer(name, image string) corev1.Container {
	return corev1.Container{Name: name, Image: image}
}

func makePodSpec(initContainers, containers []corev1.Container) *corev1.PodSpec {
	return &corev1.PodSpec{
		InitContainers: initContainers,
		Containers:     containers,
	}
}

var _ = Describe("Hash", func() {
	It("returns a fixed-size digest without serialized environment values", func() {
		c := makeContainer("app", "nginx:latest")
		c.Env = []corev1.EnvVar{{Name: "TOKEN", Value: "synthetic-sensitive-value"}}
		h, err := watchers.Hash(&c)
		Expect(err).NotTo(HaveOccurred())
		Expect(h).To(MatchRegexp(`^[0-9a-f]{32}$`))
		decoded, err := hex.DecodeString(h)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(decoded)).NotTo(ContainSubstring(c.Env[0].Value))
	})

	It("detects arbitrary environment value changes", func() {
		Expect(quick.Check(func(before, after string) bool {
			c := makeContainer("app", "nginx:latest")
			c.Env = []corev1.EnvVar{{Name: "TOKEN", Value: before}}
			oldHash, err := watchers.Hash(&c)
			if err != nil {
				return false
			}
			c.Env[0].Value = after
			newHash, err := watchers.Hash(&c)
			// JSON normalizes invalid UTF-8, so compare the serialized inputs.
			c.Env[0].Value = before
			oldJSON, _ := json.Marshal(c)
			c.Env[0].Value = after
			newJSON, _ := json.Marshal(c)
			return err == nil && (oldHash == newHash) == (string(oldJSON) == string(newJSON))
		}, &quick.Config{MaxCount: 1000})).To(Succeed())
	})

	It("includes every serialized container field in change detection", func() {
		// Each fixture changes one top-level field. Keep coverage aligned with the API.
		fields := map[string]string{
			"name": `"other"`, "image": `"other:v2"`,
			"command": `["run"]`, "args": `["--flag"]`, "workingDir": `"/work"`,
			"ports":                  `[{"containerPort":8080}]`,
			"envFrom":                `[{"configMapRef":{"name":"settings"}}]`,
			"env":                    `[{"name":"TOKEN","value":"synthetic-value"}]`,
			"resources":              `{"limits":{"cpu":"1"}}`,
			"resizePolicy":           `[{"resourceName":"cpu","restartPolicy":"RestartContainer"}]`,
			"restartPolicy":          `"Always"`,
			"restartPolicyRules":     `[{"action":"Restart","exitCodes":{"operator":"In","values":[1]}}]`,
			"volumeMounts":           `[{"name":"data","mountPath":"/data"}]`,
			"volumeDevices":          `[{"name":"data","devicePath":"/dev/data"}]`,
			"livenessProbe":          `{"exec":{"command":["true"]}}`,
			"readinessProbe":         `{"exec":{"command":["true"]}}`,
			"startupProbe":           `{"exec":{"command":["true"]}}`,
			"lifecycle":              `{"postStart":{"exec":{"command":["true"]}}}`,
			"terminationMessagePath": `"/termination"`, "terminationMessagePolicy": `"FallbackToLogsOnError"`,
			"imagePullPolicy": `"Always"`, "securityContext": `{"runAsNonRoot":true}`,
			"stdin": `true`, "stdinOnce": `true`, "tty": `true`,
		}
		containerType := reflect.TypeOf(corev1.Container{})
		for i := 0; i < containerType.NumField(); i++ {
			name := strings.Split(containerType.Field(i).Tag.Get("json"), ",")[0]
			Expect(fields).To(HaveKey(name), "missing fixture for %s", name)
		}
		original := corev1.Container{}
		originalHash, err := watchers.Hash(&original)
		Expect(err).NotTo(HaveOccurred())
		for name, value := range fields {
			var changed corev1.Container
			Expect(json.Unmarshal([]byte(`{"`+name+`":`+value+`}`), &changed)).To(Succeed(), name)
			changedHash, err := watchers.Hash(&changed)
			Expect(err).NotTo(HaveOccurred(), name)
			Expect(changedHash).NotTo(Equal(originalHash), name)
		}
	})

	It("hashes large specs completely and deterministically", func() {
		c := makeContainer("app", "nginx:latest")
		c.Env = []corev1.EnvVar{{Name: "LARGE", Value: strings.Repeat("synthetic", 128*1024)}}
		first, err := watchers.Hash(&c)
		Expect(err).NotTo(HaveOccurred())
		Expect(first).To(HaveLen(32))
		copyHash, err := watchers.Hash(c.DeepCopy())
		Expect(err).NotTo(HaveOccurred())
		Expect(copyHash).To(Equal(first))
		c.Env[0].Value += "changed-at-the-end"
		changed, err := watchers.Hash(&c)
		Expect(err).NotTo(HaveOccurred())
		Expect(changed).NotTo(Equal(first))
	})

	It("returns non-empty hash for a container", func() {
		c := makeContainer("app", "nginx:latest")
		h, err := watchers.Hash(&c)
		Expect(err).NotTo(HaveOccurred())
		Expect(h).NotTo(BeEmpty())
	})

	It("returns same hash for identical containers", func() {
		c := makeContainer("app", "nginx:latest")
		h1, _ := watchers.Hash(&c)
		h2, _ := watchers.Hash(&c)
		Expect(h1).To(Equal(h2))
	})

	It("returns different hashes for different containers", func() {
		c1 := makeContainer("app", "nginx:1.0")
		c2 := makeContainer("app", "nginx:2.0")
		h1, _ := watchers.Hash(&c1)
		h2, _ := watchers.Hash(&c2)
		Expect(h1).NotTo(Equal(h2))
	})
})

var _ = Describe("HashContainerList", func() {
	It("returns empty map for empty list", func() {
		hashes, err := watchers.HashContainerList(&[]corev1.Container{})
		Expect(err).NotTo(HaveOccurred())
		Expect(hashes).To(BeEmpty())
	})

	It("returns hash per container name", func() {
		containers := []corev1.Container{
			makeContainer("app", "nginx:latest"),
			makeContainer("sidecar", "envoy:latest"),
		}
		hashes, err := watchers.HashContainerList(&containers)
		Expect(err).NotTo(HaveOccurred())
		Expect(hashes).To(HaveLen(2))
		Expect(hashes).To(HaveKey("app"))
		Expect(hashes).To(HaveKey("sidecar"))
	})
})

var _ = Describe("HashPodSpec", func() {
	It("returns hashes for both init containers and containers", func() {
		ps := makePodSpec(
			[]corev1.Container{makeContainer("init", "busybox")},
			[]corev1.Container{makeContainer("app", "nginx")},
		)
		initHashes, containerHashes, err := watchers.HashPodSpec(ps)
		Expect(err).NotTo(HaveOccurred())
		Expect(initHashes).To(HaveKey("init"))
		Expect(containerHashes).To(HaveKey("app"))
	})

	It("handles pod spec with no containers", func() {
		ps := makePodSpec(nil, nil)
		initHashes, containerHashes, err := watchers.HashPodSpec(ps)
		Expect(err).NotTo(HaveOccurred())
		Expect(initHashes).To(BeEmpty())
		Expect(containerHashes).To(BeEmpty())
	})
})

var _ = Describe("HashesChanged", func() {
	It("returns false when hashes are equal", func() {
		h := map[string]string{"app": "abc123"}
		Expect(watchers.HashesChanged(h, h, logger)).To(BeFalse())
	})

	It("returns true when a hash value changed", func() {
		old := map[string]string{"app": "abc123"}
		new := map[string]string{"app": "xyz789"}
		Expect(watchers.HashesChanged(old, new, logger)).To(BeTrue())
	})

	It("returns true when a container is missing from new", func() {
		old := map[string]string{"app": "abc", "sidecar": "def"}
		new := map[string]string{"app": "abc"}
		Expect(watchers.HashesChanged(old, new, logger)).To(BeTrue())
	})

	It("returns true when a container is added in new", func() {
		old := map[string]string{"app": "abc"}
		new := map[string]string{"app": "abc", "sidecar": "def"}
		Expect(watchers.HashesChanged(old, new, logger)).To(BeTrue())
	})

	It("returns false for two empty maps", func() {
		Expect(watchers.HashesChanged(map[string]string{}, map[string]string{}, logger)).To(BeFalse())
	})
})

var _ = Describe("ContainersChanged", func() {
	It("returns false when pod specs are identical", func() {
		ps := makePodSpec(nil, []corev1.Container{makeContainer("app", "nginx")})
		changed, _, _, err := watchers.ContainersChanged(ps, ps, logger)
		Expect(err).NotTo(HaveOccurred())
		Expect(changed).To(BeFalse())
	})

	It("returns true when a container image changes", func() {
		old := makePodSpec(nil, []corev1.Container{makeContainer("app", "nginx:1.0")})
		new := makePodSpec(nil, []corev1.Container{makeContainer("app", "nginx:2.0")})
		changed, _, _, err := watchers.ContainersChanged(old, new, logger)
		Expect(err).NotTo(HaveOccurred())
		Expect(changed).To(BeTrue())
	})

	It("returns true when a container is added", func() {
		old := makePodSpec(nil, []corev1.Container{makeContainer("app", "nginx")})
		new := makePodSpec(nil, []corev1.Container{makeContainer("app", "nginx"), makeContainer("sidecar", "envoy")})
		changed, _, _, err := watchers.ContainersChanged(old, new, logger)
		Expect(err).NotTo(HaveOccurred())
		Expect(changed).To(BeTrue())
	})

	It("returns true when an init container changes", func() {
		old := makePodSpec([]corev1.Container{makeContainer("init", "busybox:1.0")}, nil)
		new := makePodSpec([]corev1.Container{makeContainer("init", "busybox:2.0")}, nil)
		changed, _, _, err := watchers.ContainersChanged(old, new, logger)
		Expect(err).NotTo(HaveOccurred())
		Expect(changed).To(BeTrue())
	})

	It("returns new hashes when changed", func() {
		old := makePodSpec(nil, []corev1.Container{makeContainer("app", "nginx:1.0")})
		new := makePodSpec(nil, []corev1.Container{makeContainer("app", "nginx:2.0")})
		_, _, newHashes, err := watchers.ContainersChanged(old, new, logger)
		Expect(err).NotTo(HaveOccurred())
		Expect(newHashes).To(HaveKey("app"))
	})
})
