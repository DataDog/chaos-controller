// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package watchers_test

import (
	"context"
	"errors"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/mocks"
	"github.com/DataDog/chaos-controller/watchers"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"
	"k8s.io/apimachinery/pkg/runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// fakeSubResourceWriter implements client.SubResourceWriter for testing.
type fakeSubResourceWriter struct {
	updateErr error
}

func (f *fakeSubResourceWriter) Create(_ context.Context, _ client.Object, _ client.Object, _ ...client.SubResourceCreateOption) error {
	return nil
}

func (f *fakeSubResourceWriter) Update(_ context.Context, _ client.Object, _ ...client.SubResourceUpdateOption) error {
	return f.updateErr
}

func (f *fakeSubResourceWriter) Patch(_ context.Context, _ client.Object, _ client.Patch, _ ...client.SubResourcePatchOption) error {
	return nil
}

func makeDeployment(name, ns string, containers []corev1.Container) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{Containers: containers},
			},
		},
	}
}

var _ = Describe("DeploymentHandler", func() {
	var (
		k8sMock *mocks.K8SClientMock
		handler watchers.DeploymentHandler
	)

	BeforeEach(func() {
		k8sMock = mocks.NewK8SClientMock(GinkgoT())
		handler = watchers.NewDeploymentHandler(k8sMock, zaptest.NewLogger(GinkgoT()).Sugar())
	})

	Describe("NewDeploymentHandler", func() {
		It("creates handler", func() {
			Expect(handler.Client).NotTo(BeNil())
		})
	})

	Describe("OnDelete", func() {
		It("does nothing (empty body)", func() {
			Expect(func() { handler.OnDelete(&appsv1.Deployment{}) }).NotTo(Panic())
		})
	})

	Describe("OnAdd", func() {
		It("does nothing when object is not a deployment", func() {
			Expect(func() { handler.OnAdd("not-a-deployment") }).NotTo(Panic())
		})

		It("does nothing when List fails", func() {
			dep := makeDeployment("dep", "ns", nil)
			k8sMock.EXPECT().List(mock.Anything, mock.AnythingOfType("*v1beta1.DisruptionRolloutList"), mock.Anything).
				Return(errors.New("list error"))
			Expect(func() { handler.OnAdd(dep) }).NotTo(Panic())
		})

		It("does nothing when no disruption rollouts associated", func() {
			dep := makeDeployment("dep", "ns", nil)
			k8sMock.EXPECT().List(mock.Anything, mock.AnythingOfType("*v1beta1.DisruptionRolloutList"), mock.Anything).
				Run(func(_ context.Context, list client.ObjectList, _ ...client.ListOption) {
					list.(*v1beta1.DisruptionRolloutList).Items = []v1beta1.DisruptionRollout{}
				}).Return(nil)
			Expect(func() { handler.OnAdd(dep) }).NotTo(Panic())
		})

		It("calls UpdateDisruptionRolloutStatus when rollout exists", func() {
			dep := makeDeployment("dep", "ns", []corev1.Container{{Name: "app", Image: "nginx"}})
			dr := v1beta1.DisruptionRollout{ObjectMeta: metav1.ObjectMeta{Name: "dr", Namespace: "ns"}}
			statusWriter := &fakeSubResourceWriter{}

			k8sMock.EXPECT().List(mock.Anything, mock.AnythingOfType("*v1beta1.DisruptionRolloutList"), mock.Anything).
				Run(func(_ context.Context, list client.ObjectList, _ ...client.ListOption) {
					list.(*v1beta1.DisruptionRolloutList).Items = []v1beta1.DisruptionRollout{dr}
				}).Return(nil).Times(2)
			k8sMock.EXPECT().Status().Return(statusWriter)
			k8sMock.EXPECT().Status().Return(statusWriter)

			Expect(func() { handler.OnAdd(dep) }).NotTo(Panic())
		})
	})

	Describe("OnUpdate", func() {
		It("does nothing when objects are not deployments", func() {
			Expect(func() { handler.OnUpdate("old", "new") }).NotTo(Panic())
		})

		It("does nothing when no disruption rollouts associated", func() {
			old := makeDeployment("dep", "ns", []corev1.Container{{Name: "app", Image: "nginx:1"}})
			new := makeDeployment("dep", "ns", []corev1.Container{{Name: "app", Image: "nginx:2"}})
			k8sMock.EXPECT().List(mock.Anything, mock.AnythingOfType("*v1beta1.DisruptionRolloutList"), mock.Anything).
				Run(func(_ context.Context, list client.ObjectList, _ ...client.ListOption) {
					list.(*v1beta1.DisruptionRolloutList).Items = []v1beta1.DisruptionRollout{}
				}).Return(nil)
			Expect(func() { handler.OnUpdate(old, new) }).NotTo(Panic())
		})

		It("does nothing when containers have not changed", func() {
			dep := makeDeployment("dep", "ns", []corev1.Container{{Name: "app", Image: "nginx:1"}})
			dr := v1beta1.DisruptionRollout{ObjectMeta: metav1.ObjectMeta{Name: "dr", Namespace: "ns"}}
			k8sMock.EXPECT().List(mock.Anything, mock.AnythingOfType("*v1beta1.DisruptionRolloutList"), mock.Anything).
				Run(func(_ context.Context, list client.ObjectList, _ ...client.ListOption) {
					list.(*v1beta1.DisruptionRolloutList).Items = []v1beta1.DisruptionRollout{dr}
				}).Return(nil)
			// Same deployment → no container changes
			Expect(func() { handler.OnUpdate(dep, dep) }).NotTo(Panic())
		})

		It("updates status when containers changed and rollout exists", func() {
			old := makeDeployment("dep", "ns", []corev1.Container{{Name: "app", Image: "nginx:1"}})
			new := makeDeployment("dep", "ns", []corev1.Container{{Name: "app", Image: "nginx:2"}})
			dr := v1beta1.DisruptionRollout{ObjectMeta: metav1.ObjectMeta{Name: "dr", Namespace: "ns"}}
			statusWriter := &fakeSubResourceWriter{}

			k8sMock.EXPECT().List(mock.Anything, mock.AnythingOfType("*v1beta1.DisruptionRolloutList"), mock.Anything).
				Run(func(_ context.Context, list client.ObjectList, _ ...client.ListOption) {
					list.(*v1beta1.DisruptionRolloutList).Items = []v1beta1.DisruptionRollout{dr}
				}).Return(nil).Times(2)
			k8sMock.EXPECT().Status().Return(statusWriter)
			k8sMock.EXPECT().Status().Return(statusWriter)

			Expect(func() { handler.OnUpdate(old, new) }).NotTo(Panic())
		})
	})

	Describe("FetchAssociatedDisruptionRollouts", func() {
		It("returns rollouts from List", func() {
			dep := makeDeployment("dep", "ns", nil)
			dr := v1beta1.DisruptionRollout{ObjectMeta: metav1.ObjectMeta{Name: "dr"}}
			k8sMock.EXPECT().List(mock.Anything, mock.AnythingOfType("*v1beta1.DisruptionRolloutList"), mock.Anything).
				Run(func(_ context.Context, list client.ObjectList, _ ...client.ListOption) {
					list.(*v1beta1.DisruptionRolloutList).Items = []v1beta1.DisruptionRollout{dr}
				}).Return(nil)

			rollouts, err := handler.FetchAssociatedDisruptionRollouts(dep)
			Expect(err).NotTo(HaveOccurred())
			Expect(rollouts.Items).To(HaveLen(1))
		})

		It("returns error when List fails", func() {
			dep := makeDeployment("dep", "ns", nil)
			k8sMock.EXPECT().List(mock.Anything, mock.AnythingOfType("*v1beta1.DisruptionRolloutList"), mock.Anything).
				Return(errors.New("list error"))

			_, err := handler.FetchAssociatedDisruptionRollouts(dep)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("HasAssociatedDisruptionRollout", func() {
		It("returns true when rollouts exist", func() {
			dep := makeDeployment("dep", "ns", nil)
			dr := v1beta1.DisruptionRollout{ObjectMeta: metav1.ObjectMeta{Name: "dr"}}
			k8sMock.EXPECT().List(mock.Anything, mock.AnythingOfType("*v1beta1.DisruptionRolloutList"), mock.Anything).
				Run(func(_ context.Context, list client.ObjectList, _ ...client.ListOption) {
					list.(*v1beta1.DisruptionRolloutList).Items = []v1beta1.DisruptionRollout{dr}
				}).Return(nil)

			has, err := handler.HasAssociatedDisruptionRollout(dep)
			Expect(err).NotTo(HaveOccurred())
			Expect(has).To(BeTrue())
		})

		It("returns false when no rollouts", func() {
			dep := makeDeployment("dep", "ns", nil)
			k8sMock.EXPECT().List(mock.Anything, mock.AnythingOfType("*v1beta1.DisruptionRolloutList"), mock.Anything).
				Run(func(_ context.Context, list client.ObjectList, _ ...client.ListOption) {
					list.(*v1beta1.DisruptionRolloutList).Items = []v1beta1.DisruptionRollout{}
				}).Return(nil)

			has, err := handler.HasAssociatedDisruptionRollout(dep)
			Expect(err).NotTo(HaveOccurred())
			Expect(has).To(BeFalse())
		})
	})

	Describe("UpdateDisruptionRolloutStatus", func() {
		It("updates status for each rollout", func() {
			dep := makeDeployment("dep", "ns", nil)
			dr := v1beta1.DisruptionRollout{ObjectMeta: metav1.ObjectMeta{Name: "dr", Namespace: "ns"}}
			statusWriter := &fakeSubResourceWriter{}

			k8sMock.EXPECT().List(mock.Anything, mock.AnythingOfType("*v1beta1.DisruptionRolloutList"), mock.Anything).
				Run(func(_ context.Context, list client.ObjectList, _ ...client.ListOption) {
					list.(*v1beta1.DisruptionRolloutList).Items = []v1beta1.DisruptionRollout{dr}
				}).Return(nil)
			k8sMock.EXPECT().Status().Return(statusWriter)
			k8sMock.EXPECT().Status().Return(statusWriter)

			err := handler.UpdateDisruptionRolloutStatus(dep, map[string]string{}, map[string]string{"app": "abc"})
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns error when Status().Update fails", func() {
			dep := makeDeployment("dep", "ns", nil)
			dr := v1beta1.DisruptionRollout{ObjectMeta: metav1.ObjectMeta{Name: "dr"}}
			statusWriter := &fakeSubResourceWriter{updateErr: errors.New("update failed")}

			k8sMock.EXPECT().List(mock.Anything, mock.AnythingOfType("*v1beta1.DisruptionRolloutList"), mock.Anything).
				Run(func(_ context.Context, list client.ObjectList, _ ...client.ListOption) {
					list.(*v1beta1.DisruptionRolloutList).Items = []v1beta1.DisruptionRollout{dr}
				}).Return(nil)
			k8sMock.EXPECT().Status().Return(statusWriter)
			k8sMock.EXPECT().Status().Return(statusWriter)

			err := handler.UpdateDisruptionRolloutStatus(dep, nil, nil)
			Expect(err).To(HaveOccurred())
		})
	})
})

var _ = Describe("SharedChaosPodHandler", func() {
	It("NewSharedChaosPodHandler creates a non-nil handler", func() {
		k8sMock := mocks.NewK8SClientMock(GinkgoT())
		recorder := &fakeEventRecorder{}
		handler := watchers.NewSharedChaosPodHandler(k8sMock, recorder, zaptest.NewLogger(GinkgoT()).Sugar(), watchers.NewWatcherMetricsAdapterMock(GinkgoT()), nil)
		Expect(handler).NotTo(BeNil())
	})
})

// fakeEventRecorder implements record.EventRecorder for testing.
type fakeEventRecorder struct{}

func (f *fakeEventRecorder) Event(_ runtime.Object, _, _, _ string)            {}
func (f *fakeEventRecorder) Eventf(_ runtime.Object, _, _, _ string, _ ...any) {}
func (f *fakeEventRecorder) AnnotatedEventf(_ runtime.Object, _ map[string]string, _, _, _ string, _ ...any) {
}
