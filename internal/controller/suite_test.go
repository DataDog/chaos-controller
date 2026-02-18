// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package controller

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/util/uuid"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

const (
	k8sAPIServerResponseTimeout = 30 * time.Second
	k8sAPIPotentialChangesEvery = time.Second
)

var (
	k8sClient  client.Client
	APIReader  client.Reader
	restConfig *rest.Config
	namespace  string
	lightCfg   lightConfig
	suiteCtx   context.Context
)

var (
	clusterName, contextName string
	log                      *zap.SugaredLogger
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func(ctx SpecContext) {
	if envClusterName, envKubeContext := os.Getenv("E2E_TEST_CLUSTER_NAME"), os.Getenv("E2E_TEST_KUBECTL_CONTEXT"); envClusterName != "" && envKubeContext != "" {
		clusterName = envClusterName
		contextName = envKubeContext
	} else {
		Fail("E2E_TEST_CLUSTER_NAME and E2E_TEST_KUBECTL_CONTEXT env vars must be provided")
	}

	log = zaptest.NewLogger(GinkgoT()).Sugar()

	ciValues, err := os.ReadFile("../chart/values/ci.yaml")
	Expect(err).ToNot(HaveOccurred())
	Expect(yaml.Unmarshal(ciValues, &lightCfg)).To(Succeed())
	Expect(lightCfg).ToNot(BeZero())

	// We use ginkgo process identifier to shard our tests among namespaces
	// it enables us to speed up things
	namespace = fmt.Sprintf("e2e-test-%d-%s", GinkgoParallelProcess(), uuid.NewUUID())

	// +kubebuilder:scaffold:scheme
	Expect(chaosv1beta1.AddToScheme(scheme.Scheme)).To(Succeed())

	// We create a kube client to interact with a local kube cluster (by default it expect lima if no CLUSTER_NAME env var is provided)
	restConfig, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{
			CurrentContext: contextName,
		}).ClientConfig()
	Expect(err).ToNot(HaveOccurred())
	Expect(restConfig).ToNot(BeNil())

	// we use manager to create a Kubernetes client in order to benefit from informer pattern
	// we expect only list/watch instead of get
	// this will be more effective than polling CI k8s API server regularly
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme.Scheme,
		Metrics: metricsserver.Options{
			BindAddress: "0",
		},
	})
	Expect(err).ToNot(HaveOccurred())

	var cancel context.CancelFunc
	suiteCtx, cancel = context.WithCancel(context.Background())
	go func() {
		defer GinkgoRecover()
		GinkgoHelper()

		if err := mgr.Start(suiteCtx); err != nil {
			Fail(fmt.Sprintf("unable to start manager, test can't be ran: %v", err))
		}
	}()
	DeferCleanup(cancel)

	Eventually(mgr.GetCache().WaitForCacheSync).WithContext(suiteCtx).Within(k8sAPIServerResponseTimeout).ProbeEvery(k8sAPIPotentialChangesEvery).Should(BeTrue())

	k8sClient = mgr.GetClient()

	APIReader = mgr.GetAPIReader()

	// Create namespace according to parallelization (and cleanup it on test cleanup)
	namespace := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	Eventually(k8sClient.Create).WithContext(ctx).Within(k8sAPIServerResponseTimeout).ProbeEvery(k8sAPIPotentialChangesEvery).WithArguments(&namespace).Should(WithTransform(client.IgnoreAlreadyExists, Succeed()))
	DeferCleanup(func(ctx SpecContext, nsName corev1.Namespace) {
		Eventually(k8sClient.Delete).WithContext(ctx).Within(k8sAPIServerResponseTimeout).ProbeEvery(k8sAPIPotentialChangesEvery).WithArguments(&nsName).Should(WithTransform(client.IgnoreNotFound, Succeed()))
	}, namespace)
}, NodeTimeout(time.Minute))
