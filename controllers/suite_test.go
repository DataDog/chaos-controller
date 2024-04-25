// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.

package controllers

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
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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
	restConfig *rest.Config
	namespace  string
	lightCfg   lightConfig
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
	namespace = fmt.Sprintf("e2e-test-%d", GinkgoParallelProcess())

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

	bgCtx, cancel := context.WithCancel(context.Background())
	go func() {
		defer GinkgoRecover()
		GinkgoHelper()

		if err := mgr.Start(bgCtx); err != nil {
			Fail(fmt.Sprintf("unable to start manager, test can't be ran: %v", err))
		}
	}()
	DeferCleanup(cancel)

	Eventually(mgr.GetCache().WaitForCacheSync).WithContext(ctx).Within(k8sAPIServerResponseTimeout).ProbeEvery(k8sAPIPotentialChangesEvery).Should(BeTrue())

	k8sClient = mgr.GetClient()

	// Create namespace according to parallelization (and cleanup it on test cleanup)
	namespace := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	Eventually(k8sClient.Create).WithContext(ctx).Within(k8sAPIServerResponseTimeout).ProbeEvery(k8sAPIPotentialChangesEvery).WithArguments(&namespace).Should(WithTransform(client.IgnoreAlreadyExists, Succeed()))
	DeferCleanup(func(ctx SpecContext, nsName corev1.Namespace) {
		// We do not only DELETE the namespace
		Eventually(k8sClient.Delete).WithContext(ctx).Within(k8sAPIServerResponseTimeout).ProbeEvery(k8sAPIPotentialChangesEvery).WithArguments(&nsName, client.GracePeriodSeconds(0), client.PropagationPolicy(metav1.DeletePropagationForeground)).Should(WithTransform(client.IgnoreNotFound, Succeed()))

		strongCleanup(ctx, nsName)
		// But we also WAIT for it's completed deletion to ensure repetitive tests (--until-it-fails) do not face terminated namespace errors
		Eventually(k8sClient.Get).WithContext(ctx).Within(k8sAPIServerResponseTimeout*20).ProbeEvery(k8sAPIPotentialChangesEvery).
			WithArguments(types.NamespacedName{
				Name: nsName.Name,
			}, &nsName).
			Should(WithTransform(logIsNotFound, BeTrue()))
	}, namespace)
}, NodeTimeout(time.Minute*3))

func strongCleanup(ctx SpecContext, nsName corev1.Namespace) {
	log.Infow("Cleaning up namespace", "namespace", namespace, "nsName", nsName.Name)
	err := k8sClient.Delete(ctx, &nsName, client.PropagationPolicy(metav1.DeletePropagationForeground), client.GracePeriodSeconds(0))
	if err != nil {
		log.Infow("error on namespace delete", "err", err)
	}

	ns := corev1.NamespaceList{}
	errTwo := k8sClient.List(ctx, &ns)
	if errTwo != nil {
		log.Errorw("DEBUG LIST ERR", "err", errTwo)
	}
	for _, n := range ns.Items {
		log.Infow("NAMESPACES FOUND DURING DEBUGGING", "ns", n.Name)
		if n.Name == nsName.Name || n.Name == namespace {
			ps := corev1.PodList{}
			err = k8sClient.List(ctx, &ps, client.InNamespace(nsName.Name))
			if err != nil {
				log.Infow("error on namespace list", "err", err)
			}
			log.Infow("checking for pods in namespace", "podsCount", len(ps.Items))
			for _, pod := range ps.Items {
				log.Infow("PODS FOUND IN NAMESPACE", "pod", pod.Name)
			}
		}
	}
}

func logIsNotFound(err error) bool {
	log.Infow("LOGGING CLEANUP ERROR", "err", err)
	ns := corev1.NamespaceList{}
	errTwo := k8sClient.List(context.Background(), &ns)
	if errTwo != nil {
		log.Errorw("DEBUG LIST ERR", "err", errTwo)
	}
	for _, n := range ns.Items {
		log.Infow("NAMESPACE FOUND DURING DEBUGGING", "ns", n.Name)
	}
	return apierrors.IsNotFound(err)
}
