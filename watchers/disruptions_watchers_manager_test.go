// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package watchers_test

import (
	"context"
	"fmt"
	"net/http"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/cmd/mocks"
	"github.com/DataDog/chaos-controller/watchers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("Disruptions watchers manager", func() {
	var (
		cacheMock                   *mocks.CacheCacheMock
		disruption                  *chaosv1beta1.Disruption
		disruptionsWatchersManager  watchers.DisruptionsWatchersManager
		disruptionTargetWatcherMock *watchers.WatcherMock
		err                         error
		expectedDTWatcherName       string
		twoDisruptions              []*chaosv1beta1.Disruption
		watchersManagerMock         *watchers.ManagerMock
		watcherFactoryMock          *watchers.FactoryMock
		readerMock                  *mocks.ReaderMock
	)

	BeforeEach(func() {
		// Arrange
		cacheMock = mocks.NewCacheCacheMock(GinkgoT())
		readerMock = mocks.NewReaderMock(GinkgoT())
		twoDisruptions = []*chaosv1beta1.Disruption{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "disruption-name-1",
					Namespace: "namespace-1",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "disruption-name-2",
					Namespace: "namespace-2",
				},
			},
		}
		watcherFactoryMock = watchers.NewFactoryMock(GinkgoT())
		watchersManagerMock = watchers.NewManagerMock(GinkgoT())
	})

	JustBeforeEach(func() {
		// Arrange
		controllerMock := mocks.NewRuntimeControllerMock[reconcile.Request](GinkgoT())

		// Act
		disruptionsWatchersManager = watchers.NewDisruptionsWatchersManager(controllerMock, watcherFactoryMock, readerMock)

		// Assert
		By("return a DisruptionsWatchersManager")
		Expect(disruptionsWatchersManager).ToNot(BeNil())
	})

	When("CreateAllWatchers method is called", func() {
		Context("with a valid disruption", func() {
			BeforeEach(func() {
				// Arrange / Assert
				disruption = &chaosv1beta1.Disruption{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "disruption-name",
						Namespace: "namespace",
					},
				}

				disSpecHash, err := disruption.Spec.HashNoCount()
				Expect(err).ShouldNot(HaveOccurred())

				expectedDTWatcherName = disSpecHash + string(watchers.DisruptionTargetWatcherName)

				disruptionTargetWatcherMock = watchers.NewWatcherMock(GinkgoT())
				watchersManagerMock = watchers.NewManagerMock(GinkgoT())
			})

			JustBeforeEach(func(ctx SpecContext) {
				// Act
				err = disruptionsWatchersManager.CreateAllWatchers(ctx, disruption, watchersManagerMock, cacheMock)
			})

			Context("nominal cases", func() {

				BeforeEach(func() {
					// Arrange / Assert
					By("check once the presence of the DisruptionTarget watcher")
					watchersManagerMock.EXPECT().GetWatcher(expectedDTWatcherName).Return(nil).Once()

					By("create the Disruption Target watcher")
					watcherFactoryMock.EXPECT().NewDisruptionTargetWatcher(expectedDTWatcherName, true, disruption, cacheMock).Return(disruptionTargetWatcherMock, nil).Once()

					By("adding the watcher into the watcherManager")
					watchersManagerMock.EXPECT().AddWatcher(disruptionTargetWatcherMock).Return(nil).Once()
				})

				It("should not return an error", func() {
					Expect(err).ShouldNot(HaveOccurred())
				})

				When("CreateAllWatchers method is recalled", func() {
					It("should not recreate watchers", func(ctx SpecContext) {
						// Arrange / Assert
						watchersManagerMock = watchers.NewManagerMock(GinkgoT())

						By("check once the presence of the DisruptionTarget watcher")
						watchersManagerMock.EXPECT().GetWatcher(expectedDTWatcherName).Return(disruptionTargetWatcherMock).Once()

						// Act
						Expect(disruptionsWatchersManager.CreateAllWatchers(ctx, disruption, watchersManagerMock, cacheMock)).Should(Succeed())

						// Assert
						By("not call any methods of the watcher manager")
						watchersManagerMock.AssertNumberOfCalls(GinkgoT(), "AddWatcher", 0)
					})
				})
			})

			Context("error cases", func() {
				BeforeEach(func() {
					watchersManagerMock = watchers.NewManagerMock(GinkgoT())
					watchersManagerMock.EXPECT().GetWatcher(mock.Anything).Return(nil).Maybe()
				})

				When("the NewDisruptionTargetWatcher method of the factory return an error", func() {
					BeforeEach(func() {
						// Arrange / Assert
						watchersManagerMock.EXPECT().AddWatcher(mock.Anything).Return(nil).Maybe()
						watchersManagerMock.EXPECT().GetWatcher(expectedDTWatcherName).Return(nil)

						watcherFactoryMock = watchers.NewFactoryMock(GinkgoT())
						watcherFactoryMock.EXPECT().NewDisruptionTargetWatcher(expectedDTWatcherName, true, disruption, cacheMock).Return(disruptionTargetWatcherMock, fmt.Errorf("NewDisruptionTargetWatcher error")).Once()
					})

					It("should return an error", func() {
						// Act
						Expect(err).To(HaveOccurred())
						Expect(err).To(MatchError("NewDisruptionTargetWatcher error"))
					})
				})

				When("the AddWatcher method of the watcherManager return an error with the disruptionTargetWatcher in parameter", func() {
					BeforeEach(func() {
						// Arrange / Assert
						watchersManagerMock.EXPECT().AddWatcher(disruptionTargetWatcherMock).Return(fmt.Errorf("disruptionTargetWatcher message")).Once()
						watchersManagerMock.EXPECT().AddWatcher(mock.Anything).Return(nil).Maybe()

						watcherFactoryMock.EXPECT().NewDisruptionTargetWatcher(expectedDTWatcherName, true, disruption, cacheMock).Return(disruptionTargetWatcherMock, nil).Once()
					})

					It("should return an error", func() {
						Expect(err).To(HaveOccurred())
						Expect(err).To(MatchError("failed to create watcher: disruptionTargetWatcher message"))
					})
				})
			})
		})

		Context("with an empty disruption", func() {
			BeforeEach(func() {
				// Arrange
				disruption = &chaosv1beta1.Disruption{}
			})

			It("should return an error", func(ctx SpecContext) {
				// Act
				err := disruptionsWatchersManager.CreateAllWatchers(ctx, disruption, watchersManagerMock, cacheMock)

				// Assert
				Expect(err).Should(HaveOccurred())
				Expect(err).To(MatchError("the disruption is not valid. It should contain a name and a namespace"))
			})
		})
	})

	When("CreateAllWatchers is called with a disruption UID change", func() {
		var (
			oldManagerMock *watchers.ManagerMock
			newManagerMock *watchers.ManagerMock
		)

		BeforeEach(func() {
			disruption = &chaosv1beta1.Disruption{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "disruption-name",
					Namespace: "namespace",
					UID:       types.UID("uid-v1"),
				},
			}
			watcherFactoryMock = watchers.NewFactoryMock(GinkgoT())
			oldManagerMock = watchers.NewManagerMock(GinkgoT())
			newManagerMock = watchers.NewManagerMock(GinkgoT())
		})

		When("the cached manager has a different disruption UID (recreated disruption)", func() {
			It("should evict the stale manager and create fresh watchers", func(ctx SpecContext) {
				disSpecHash, err := disruption.Spec.HashNoCount()
				Expect(err).ShouldNot(HaveOccurred())
				expectedDTWatcher := disSpecHash + string(watchers.DisruptionTargetWatcherName)

				By("seeding the cache via first CreateAllWatchers call with uid-v1")
				oldManagerMock.EXPECT().GetWatcher(mock.Anything).Return(nil).Once()
				oldManagerMock.EXPECT().AddWatcher(mock.Anything).Return(nil).Once()
				watcherFactoryMock.EXPECT().NewDisruptionTargetWatcher(expectedDTWatcher, true, disruption, cacheMock).Return(watchers.NewWatcherMock(GinkgoT()), nil).Once()
				Expect(disruptionsWatchersManager.CreateAllWatchers(ctx, disruption, oldManagerMock, cacheMock)).To(Succeed())

				By("building a recreated disruption with the same namespace/name but a new UID")
				recreatedDisruption := &chaosv1beta1.Disruption{
					ObjectMeta: metav1.ObjectMeta{
						Name:      disruption.Name,
						Namespace: disruption.Namespace,
						UID:       types.UID("uid-v2"),
					},
				}

				By("expecting the stale manager to be evicted (RemoveAllWatchers called)")
				oldManagerMock.EXPECT().RemoveAllWatchers().Once()

				By("expecting fresh watchers to be created for the new disruption")
				newManagerMock.EXPECT().GetWatcher(mock.Anything).Return(nil).Once()
				newManagerMock.EXPECT().AddWatcher(mock.Anything).Return(nil).Once()
				watcherFactoryMock.EXPECT().NewDisruptionTargetWatcher(expectedDTWatcher, true, recreatedDisruption, cacheMock).Return(watchers.NewWatcherMock(GinkgoT()), nil).Once()

				Expect(disruptionsWatchersManager.CreateAllWatchers(ctx, recreatedDisruption, newManagerMock, cacheMock)).To(Succeed())
			})
		})

		When("the cached manager has the same disruption UID", func() {
			It("should reuse the cached manager without eviction", func(ctx SpecContext) {
				disSpecHash, err := disruption.Spec.HashNoCount()
				Expect(err).ShouldNot(HaveOccurred())
				expectedDTWatcher := disSpecHash + string(watchers.DisruptionTargetWatcherName)

				By("seeding the cache via first CreateAllWatchers call with uid-v1")
				oldManagerMock.EXPECT().GetWatcher(mock.Anything).Return(nil).Once()
				oldManagerMock.EXPECT().AddWatcher(mock.Anything).Return(nil).Once()
				watcherFactoryMock.EXPECT().NewDisruptionTargetWatcher(expectedDTWatcher, true, disruption, cacheMock).Return(watchers.NewWatcherMock(GinkgoT()), nil).Once()
				Expect(disruptionsWatchersManager.CreateAllWatchers(ctx, disruption, oldManagerMock, cacheMock)).To(Succeed())

				By("calling CreateAllWatchers again with the same UID — existing watcher already present")
				oldManagerMock.EXPECT().GetWatcher(expectedDTWatcher).Return(watchers.NewWatcherMock(GinkgoT())).Once()
				Expect(disruptionsWatchersManager.CreateAllWatchers(ctx, disruption, oldManagerMock, cacheMock)).To(Succeed())

				By("not calling RemoveAllWatchers on the cached manager")
				oldManagerMock.AssertNumberOfCalls(GinkgoT(), "RemoveAllWatchers", 0)

				By("not calling factory methods on the second pass")
				watcherFactoryMock.AssertNumberOfCalls(GinkgoT(), "NewDisruptionTargetWatcher", 1)
			})
		})
	})

	When("RemoveAllWatchers method is called", func() {
		Context("with an existing disruption", func() {
			BeforeEach(func() {
				// Arrange / Assert
				disruption = &chaosv1beta1.Disruption{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "disruption-name",
						Namespace: "namespace",
					},
				}

				disSpecHash, err := disruption.Spec.HashNoCount()
				Expect(err).ShouldNot(HaveOccurred())

				expectedDTWatcherName = disSpecHash + string(watchers.DisruptionTargetWatcherName)

				disruptionTargetWatcherMock = watchers.NewWatcherMock(GinkgoT())

				watchersManagerMock = watchers.NewManagerMock(GinkgoT())
				watchersManagerMock.EXPECT().AddWatcher(mock.Anything).Return(nil)
				watchersManagerMock.EXPECT().GetWatcher(mock.Anything).Return(nil)

				watcherFactoryMock = watchers.NewFactoryMock(GinkgoT())
				watcherFactoryMock.EXPECT().NewDisruptionTargetWatcher(expectedDTWatcherName, mock.Anything, disruption, cacheMock).Return(watchers.NewWatcherMock(GinkgoT()), nil)
			})

			JustBeforeEach(func(ctx SpecContext) {
				Expect(disruptionsWatchersManager.CreateAllWatchers(ctx, disruption, watchersManagerMock, cacheMock)).Should(Succeed())
			})

			Context("nominal cases", func() {
				BeforeEach(func() {
					// Arrange / Assert
					By("call once the RemoveAllWatchers method of the watcherManager")
					watchersManagerMock.EXPECT().RemoveAllWatchers().Once()
				})

				It("should remove all watchers", func(ctx SpecContext) {
					//	Action
					disruptionsWatchersManager.RemoveAllWatchers(ctx, disruption)
				})

				When("the RemoveAllWatchers method is recalled", func() {
					It("should do nothing", func(ctx SpecContext) {
						// Arrange
						disruptionsWatchersManager.RemoveAllWatchers(ctx, disruption)

						watchersManagerMock.ExpectedCalls = nil
						watchersManagerMock.Calls = nil

						// Act
						disruptionsWatchersManager.RemoveAllWatchers(ctx, disruption)

						// Arrange
						By("not call RemoveAllWatchers method of the watcherManager")
						watchersManagerMock.AssertNumberOfCalls(GinkgoT(), "RemoveAllWatchers", 0)
					})
				})
			})
		})

		Context("with an non existing disruption", func() {
			BeforeEach(func() {
				By("not call the RemoveAllWatchers method of the watcher")
				watchersManagerMock.AssertNumberOfCalls(GinkgoT(), "RemoveAllWatchers", 0)
			})

			It("should do nothing", func(ctx SpecContext) {
				disruption = &chaosv1beta1.Disruption{}

				disruptionsWatchersManager.RemoveAllWatchers(ctx, disruption)
			})
		})
	})

	When("RemoveAllOrphanWatchers method is called", func() {

		BeforeEach(func() {
			// Arrange / Assert
			watcherFactoryMock = watchers.NewFactoryMock(GinkgoT())

			for _, disruption := range twoDisruptions {
				disSpecHash, err := disruption.Spec.HashNoCount()
				Expect(err).ShouldNot(HaveOccurred())

				expectedDTWatcherName = disSpecHash + string(watchers.DisruptionTargetWatcherName)

				disruptionTargetWatcherMock = watchers.NewWatcherMock(GinkgoT())

				watchersManagerMock.EXPECT().AddWatcher(mock.Anything).Return(nil)
				watchersManagerMock.EXPECT().GetWatcher(mock.Anything).Return(nil)
				watcherFactoryMock.EXPECT().NewDisruptionTargetWatcher(expectedDTWatcherName, mock.Anything, disruption, cacheMock).Return(watchers.NewWatcherMock(GinkgoT()), nil)
			}
		})

		JustBeforeEach(func(ctx SpecContext) {
			for _, disruption := range twoDisruptions {
				Expect(disruptionsWatchersManager.CreateAllWatchers(ctx, disruption, watchersManagerMock, cacheMock)).To(Succeed())
			}
		})

		Context("with two non orphan disruptions", func() {
			BeforeEach(func() {
				// Arrange / Assert
				By("list all existing disruptions")
				readerMock.EXPECT().List(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, list client.ObjectList, _ ...client.ListOption) error {
					dl := list.(*chaosv1beta1.DisruptionList)
					dl.Items = []chaosv1beta1.Disruption{*twoDisruptions[0], *twoDisruptions[1]}
					return nil
				}).Once()
			})

			It("should do nothing", func(ctx SpecContext) {
				Expect(disruptionsWatchersManager.RemoveAllOrphanWatchers(ctx)).Should(Succeed())
			})
		})

		Context("with two not found disruptions", func() {
			BeforeEach(func() {
				// Arrange / Assert
				By("list all existing disruptions (returns empty list — both are orphans)")
				readerMock.EXPECT().List(mock.Anything, mock.Anything).Return(nil).Once()

				By("call the RemoveAllWatchers method of the watcher instance")
				watchersManagerMock.EXPECT().RemoveAllWatchers().Twice()
			})

			It("should remove all orphan watchers", func(ctx SpecContext) {
				Expect(disruptionsWatchersManager.RemoveAllOrphanWatchers(ctx)).Should(Succeed())
			})

			When("the function RemoveAllOrphanWatchers is recalled", func() {
				It("should have removed the disruption", func(ctx SpecContext) {
					// Arrange
					Expect(disruptionsWatchersManager.RemoveAllOrphanWatchers(ctx)).To(Succeed())
					readerMock.ExpectedCalls = nil
					readerMock.Calls = nil
					watchersManagerMock.ExpectedCalls = nil
					watchersManagerMock.Calls = nil

					// Action — managers map is now empty, early return kicks in
					Expect(disruptionsWatchersManager.RemoveAllOrphanWatchers(ctx)).To(Succeed())

					// Assert
					By("not call any List method of the reader (early return when managers map is empty)")
					readerMock.AssertNumberOfCalls(GinkgoT(), "List", 0)

					By("not call any RemoveAllWatcher method of the watchersManager")
					watchersManagerMock.AssertNumberOfCalls(GinkgoT(), "RemoveAllWatchers", 0)
				})
			})
		})

		When("the List method of the reader returns a server error", func() {
			BeforeEach(func() {
				var errorStatus = errors.StatusError{
					ErrStatus: metav1.Status{
						Message: "Unknown error",
						Reason:  metav1.StatusReasonInternalError,
						Code:    http.StatusInternalServerError,
					},
				}
				By("list all existing disruptions (returns server error)")
				readerMock.EXPECT().List(mock.Anything, mock.Anything).Return(&errorStatus).Once()
			})

			It("should return the error and not remove any watchers", func(ctx SpecContext) {
				// Act
				Expect(disruptionsWatchersManager.RemoveAllOrphanWatchers(ctx)).NotTo(Succeed())

				// Assert
				By("not call the RemoveAllWatchers method of the watchersManager")
				watchersManagerMock.AssertNumberOfCalls(GinkgoT(), "RemoveAllWatchers", 0)
			})
		})
	})

	When("RemoveWatchersForDisruption method is called", func() {
		BeforeEach(func() {
			// Arrange / Assert
			watcherFactoryMock = watchers.NewFactoryMock(GinkgoT())

			for _, disruption := range twoDisruptions {
				disSpecHash, err := disruption.Spec.HashNoCount()
				Expect(err).ShouldNot(HaveOccurred())

				expectedDTWatcherName = disSpecHash + string(watchers.DisruptionTargetWatcherName)

				watchersManagerMock.EXPECT().AddWatcher(mock.Anything).Return(nil)
				watchersManagerMock.EXPECT().GetWatcher(mock.Anything).Return(nil)
				watcherFactoryMock.EXPECT().NewDisruptionTargetWatcher(expectedDTWatcherName, mock.Anything, disruption, cacheMock).Return(watchers.NewWatcherMock(GinkgoT()), nil)
			}
		})

		JustBeforeEach(func(ctx SpecContext) {
			for _, disruption := range twoDisruptions {
				Expect(disruptionsWatchersManager.CreateAllWatchers(ctx, disruption, watchersManagerMock, cacheMock)).To(Succeed())
			}
		})

		Context("with an existing disruption", func() {
			BeforeEach(func() {
				By("call RemoveAllWatchers only for the targeted disruption")
				watchersManagerMock.EXPECT().RemoveAllWatchers().Once()
			})

			It("should remove watchers for that disruption only, leaving the other untouched", func(ctx SpecContext) {
				namespacedName := types.NamespacedName{
					Name:      twoDisruptions[0].Name,
					Namespace: twoDisruptions[0].Namespace,
				}

				disruptionsWatchersManager.RemoveWatchersForDisruption(ctx, namespacedName)

				By("not calling RemoveAllWatchers again when invoked a second time (entry already removed)")
				disruptionsWatchersManager.RemoveWatchersForDisruption(ctx, namespacedName)

				By("not affecting the second disruption's manager")
				watchersManagerMock.EXPECT().RemoveExpiredWatchers().Once()
				disruptionsWatchersManager.RemoveAllExpiredWatchers(ctx)
			})
		})

		Context("with a non-existing disruption", func() {
			It("should do nothing", func(ctx SpecContext) {
				namespacedName := types.NamespacedName{
					Name:      "does-not-exist",
					Namespace: "nowhere",
				}

				disruptionsWatchersManager.RemoveWatchersForDisruption(ctx, namespacedName)

				By("not calling RemoveAllWatchers on any manager")
				watchersManagerMock.AssertNumberOfCalls(GinkgoT(), "RemoveAllWatchers", 0)
			})
		})
	})

	When("RemoveAllExpiredWatchers method is called", func() {
		Context("with two disruptions", func() {
			BeforeEach(func() {
				// Arrange / Assert
				watchersManagerMock = watchers.NewManagerMock(GinkgoT())
				watcherFactoryMock = watchers.NewFactoryMock(GinkgoT())

				for _, disruption := range twoDisruptions {
					disSpecHash, err := disruption.Spec.HashNoCount()
					Expect(err).ShouldNot(HaveOccurred())

					expectedDTWatcherName = disSpecHash + string(watchers.DisruptionTargetWatcherName)

					disruptionTargetWatcherMock = watchers.NewWatcherMock(GinkgoT())

					watchersManagerMock.EXPECT().AddWatcher(mock.Anything).Return(nil)
					watchersManagerMock.EXPECT().GetWatcher(mock.Anything).Return(nil)
					watcherFactoryMock.EXPECT().NewDisruptionTargetWatcher(expectedDTWatcherName, mock.Anything, disruption, cacheMock).Return(watchers.NewWatcherMock(GinkgoT()), nil)
				}
			})

			JustBeforeEach(func(ctx SpecContext) {
				for _, disruption := range twoDisruptions {
					Expect(disruptionsWatchersManager.CreateAllWatchers(ctx, disruption, watchersManagerMock, cacheMock)).To(Succeed())
				}
			})

			It("should call twice the RemoveExpiredWatchers method of the watchersManager", func(ctx SpecContext) {
				// Arrange / Assert
				watchersManagerMock.EXPECT().RemoveExpiredWatchers().Twice()

				// Act
				disruptionsWatchersManager.RemoveAllExpiredWatchers(ctx)
			})
		})

		Context("without disruption", func() {
			It("should do nothing", func(ctx SpecContext) {
				// Act
				disruptionsWatchersManager.RemoveAllExpiredWatchers(ctx)

				// Assert
				By("not call the RemoveExpiredWatchers method of the watchersManager")
				watchersManagerMock.AssertNotCalled(GinkgoT(), "RemoveExpiredWatchers", 0)
			})
		})
	})
})
