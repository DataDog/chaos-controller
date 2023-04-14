// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package watchers_test

import (
	"context"
	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/mocks"
	. "github.com/DataDog/chaos-controller/watchers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"net/http"
)

var _ = Describe("Manager of watchers", func() {
	// Watcher methods
	const methodWatcherClean string = "Clean"
	const methodWatcherStart string = "Start"
	const methodWatcherIsExpired string = "IsExpired"
	const methodWatcherGetContextTuple string = "GetContextTuple"

	var (
		controllerMock              *mocks.ControllerMock
		ctxTuple                    CtxTuple
		err                         error
		readerMock                  *mocks.ReaderMock
		watcherManager              Manager
		watcherMock                 *WatcherMock
		watcherMock1                *WatcherMock
		watcherMock2                *WatcherMock
		watcherMock2IsExpired       *WatcherMock_IsExpired_Call
		watcherMock1GetContextTuple *WatcherMock_GetContextTuple_Call
		watcherMock2GetContextTuple *WatcherMock_GetContextTuple_Call
		syncingSourceMock           *mocks.SyncingSourceMock
	)

	// Arrange
	cacheCtx, cacheCancelFunc := context.WithCancel(context.Background())
	ctxTuple = CtxTuple{
		Ctx:        cacheCtx,
		CancelFunc: cacheCancelFunc,
		DisruptionNamespacedName: types.NamespacedName{
			Namespace: "namespace",
			Name:      "namespace-name",
		},
	}

	BeforeEach(func() {
		syncingSourceMock = mocks.NewSyncingSourceMock(GinkgoT())
		watcherMock1 = NewWatcherMock(GinkgoT())
		watcherMock2 = NewWatcherMock(GinkgoT())
		readerMock = mocks.NewReaderMock(GinkgoT())
		controllerMock = mocks.NewControllerMock(GinkgoT())

		watcherManager = NewManager(readerMock, controllerMock)
	})

	When("AddWatcher function is called", func() {
		Context("the watcher is not present", func() {
			It("should add the watcher and start it", func() {
				// Arrange / Assert
				watcherMock = NewWatcherMock(GinkgoT())

				By("call once the GetName method of watcher")
				watcherMockGetName := watcherMock.EXPECT().GetName().Return("1")
				watcherMockGetName.Once()

				By("call once the Start method of the watcher")
				watcherMockStart := watcherMock.EXPECT().Start().Return(nil)
				watcherMockStart.NotBefore(watcherMockGetName.Call).Once()

				By("call once the GetCacheSource method of the watcher")
				watcherMockGetCacheSource := watcherMock.EXPECT().GetCacheSource().Return(syncingSourceMock, nil)
				watcherMockGetCacheSource.Once().NotBefore(watcherMockStart.Call)

				By("call once the GetContextTuple method of the watcher")
				watcherMockGetContextTuple := watcherMock.EXPECT().GetContextTuple().Return(ctxTuple, nil)
				watcherMockGetContextTuple.Once().NotBefore(watcherMockGetCacheSource.Call)

				By("link the watcher to the controller")
				controllerMock.EXPECT().Watch(mock.Anything, mock.Anything, mock.Anything).Return(nil).Once().NotBefore(watcherMockGetContextTuple.Call)

				// Act
				err = watcherManager.AddWatcher(watcherMock)
			})
		})

		Context("the watcher is already present", func() {
			It("should not start the watcher", func() {
				// Arrange
				watcherMock = createSingleWatcher("1", watcherManager, controllerMock, ctxTuple, syncingSourceMock)

				By("call once the GetName method of the watcher")
				watcherMock.EXPECT().GetName().Return("1").Once()

				// Act
				err = watcherManager.AddWatcher(watcherMock)

				By("not return an error")
				Expect(err).ShouldNot(HaveOccurred())

				By("not call the Start method of the watcher")
				watcherMock.AssertNotCalled(GinkgoT(), methodWatcherStart)
			})
		})
	})

	When("GetWatcher function is called", func() {
		var (
			watcherName   string
			watcherResult Watcher
		)

		JustBeforeEach(func() {
			// Act
			watcherResult = watcherManager.GetWatcher(watcherName)
		})

		Context("with two watchers", func() {
			BeforeEach(func() {
				// Arrange
				watcherMock1, watcherMock2 = createTwoWatchers(watcherManager, controllerMock, ctxTuple, ctxTuple, syncingSourceMock, syncingSourceMock)

				watcherName = "watcher1"
			})

			It("should return the watcher1", func() {
				Expect(watcherResult).Should(Equal(watcherMock1))
			})

			It("should not return the watcher2", func() {
				Expect(watcherResult).ShouldNot(Equal(watcherMock2))
			})

			When("the watcher does not exist", func() {
				BeforeEach(func() {
					watcherName = ""
				})

				It("should return nil", func() {
					Expect(watcherResult).To(BeNil())
				})
			})
		})

		Context("without watcher", func() {
			BeforeEach(func() {
				// Arrange
				watcherName = "watcher1"
			})

			It("should return nil", func() {
				Expect(watcherResult).To(BeNil())
			})
		})
	})

	When("RemoveWatcher function is called", func() {
		Context("with two watchers", func() {
			BeforeEach(func() {
				// Arrange

				watcherMock1, watcherMock2 = createTwoWatchers(watcherManager, controllerMock, ctxTuple, ctxTuple, syncingSourceMock, syncingSourceMock)

				// Assert
				By("call once the GetName method of watcherMock1")
				watcherMock1.EXPECT().GetName().Return("watcher1").Once()

				By("call once the Clean function of watcherMock1")
				watcherMock1.EXPECT().Clean().Once()

				// Act
				err = watcherManager.RemoveWatcher(watcherMock1)
			})

			It("should not call the Clean method of Watcher2", func() {
				watcherMock2.AssertNotCalled(GinkgoT(), methodWatcherClean)
			})

			It("should not return an error", func() {
				Expect(err).ShouldNot(HaveOccurred())
			})

			When("the RemoveWatcher is called a second time for the same watcher", func() {
				BeforeEach(func() {
					// Arrange
					watcherMock1 = NewWatcherMock(GinkgoT())

					// Assert
					By("call once the GetName method of the watcher1")
					watcherMock1.EXPECT().GetName().Return("watcher1").Once()

					// Act
					err = watcherManager.RemoveWatcher(watcherMock1)
				})

				It("not call the Clean method of the watcher1", func() {
					watcherMock1.AssertNotCalled(GinkgoT(), methodWatcherStart)
				})

				It("should return an error", func() {
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).Should(HavePrefix("the watcher watcher1 does not exist"))
				})
			})
		})

		Context("without watcher", func() {
			BeforeEach(func() {
				// Arrange

				// Remove the expected calls of the controller mock to avoid assert errors.
				// In this context watchers does not exist so the asserts of the controller will never be called.
				controllerMock.ExpectedCalls = nil

				watcherMock = NewWatcherMock(GinkgoT())
				watcherMock.EXPECT().GetName().Return("1").Once()

				// Act
				err = watcherManager.RemoveWatcher(watcherMock)
			})

			It("should return an error", func() {
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).Should(Equal("the watcher 1 does not exist"))
			})
		})
	})

	When("RemoveExpiredWatchers function is called", func() {
		Context("with a watcher1 expired and a watcher2 not expired", func() {
			BeforeEach(func() {
				// Arrange
				watcherMock1, watcherMock2 = createTwoWatchers(watcherManager, controllerMock, ctxTuple, ctxTuple, syncingSourceMock, syncingSourceMock)

				// Assert
				By("call once the IsExpired function for all managed watchers")
				watcherMock1IsExpired := watcherMock1.EXPECT().IsExpired()
				watcherMock1IsExpired.Return(true).Once()
				watcherMock2IsExpired = watcherMock2.EXPECT().IsExpired().Return(false)
				watcherMock2IsExpired.Once()

				By("call once the Clean method of watcher1")
				watcherMock1.EXPECT().Clean().Once().NotBefore(watcherMock1IsExpired.Call)

				// Act
				watcherManager.RemoveExpiredWatchers()
			})

			It("should not call the Clean method of watcher2", func() {
				watcherMock2.AssertNotCalled(GinkgoT(), methodWatcherClean)
			})

			When("the RemoveExpiredWatchers is called a second time", func() {
				BeforeEach(func() {
					// Arrange: remove watcher1 expectations to avoid asserts errors
					watcherMock1 = NewWatcherMock(GinkgoT())

					// Assert
					By("call once the IsExpired method of the watcher2")
					watcherMock2IsExpired.Return(false).Once()

					// Act
					watcherManager.RemoveExpiredWatchers()
				})

				It("should not call the IsExpired method of the watcher1", func() {
					watcherMock1.AssertNotCalled(GinkgoT(), methodWatcherIsExpired)
				})

				It("should not call the Clean method of both watchers", func() {
					watcherMock1.AssertNotCalled(GinkgoT(), methodWatcherClean)
					watcherMock2.AssertNotCalled(GinkgoT(), methodWatcherClean)
				})
			})
		})
	})

	When("RemoveOrphanWatchers function is called", func() {
		Context("with two non orphan watchers", func() {
			BeforeEach(func() {
				// Arrange
				watcherMock1, watcherMock2 = createTwoWatchers(watcherManager, controllerMock, ctxTuple, ctxTuple, syncingSourceMock, syncingSourceMock)

				// Assert
				By("call twice the Get method of the reader")
				readerMock.EXPECT().Get(ctxTuple.Ctx, ctxTuple.DisruptionNamespacedName, &v1beta1.Disruption{}).Return(nil).Twice()

				By("call once the GetContextTuple method of both watchers")
				watcherMock1.EXPECT().GetContextTuple().Return(ctxTuple, nil).Once()
				watcherMock2.EXPECT().GetContextTuple().Return(ctxTuple, nil).Once()

				// Act
				watcherManager.RemoveOrphanWatchers()
			})

			It("should not call the Clean method of watchers", func() {
				watcherMock1.AssertNotCalled(GinkgoT(), methodWatcherClean)
				watcherMock2.AssertNotCalled(GinkgoT(), methodWatcherClean)
			})
		})

		Context("with two watchers", func() {
			cacheCtx2, cacheCancelFunc2 := context.WithCancel(context.Background())
			ctxTupleWatcher2 := CtxTuple{
				Ctx:        cacheCtx2,
				CancelFunc: cacheCancelFunc2,
				DisruptionNamespacedName: types.NamespacedName{
					Namespace: "namespace-2",
					Name:      "namespace-name-2",
				},
			}

			BeforeEach(func() {
				// Arrange
				watcherMock1, watcherMock2 = createTwoWatchers(watcherManager, controllerMock, ctxTuple, ctxTuple, syncingSourceMock, syncingSourceMock)

				// Assert
				By("call once the GetContextTuple method of both watchers")
				watcherMock1GetContextTuple = watcherMock1.EXPECT().GetContextTuple()
				watcherMock1GetContextTuple.Return(ctxTuple, nil).Once()
				watcherMock2GetContextTuple = watcherMock2.EXPECT().GetContextTuple()
				watcherMock2GetContextTuple.Return(ctxTuple, nil).Once()
			})

			When("the Get function of the reader return nil for both watchers", func() {
				BeforeEach(func() {
					// Arrange
					readerMock.EXPECT().Get(ctxTuple.Ctx, ctxTuple.DisruptionNamespacedName, &v1beta1.Disruption{}).Return(nil).Twice()

					// Act
					watcherManager.RemoveOrphanWatchers()
				})

				It("should not call the Clean method of watcher1", func() {
					watcherMock1.AssertNotCalled(GinkgoT(), methodWatcherClean)
				})
			})

			When("the Get method of the reader return a not found error for the watcher2", func() {
				BeforeEach(func() {
					// Arrange/Assert
					By("call once the GetContextTuple of the watcher2")
					watcherMock2GetContextTuple.Return(ctxTupleWatcher2, nil).Once()

					By("call once the Get method of the reader with the watcher1 context")
					readerMock.EXPECT().Get(ctxTuple.Ctx, ctxTuple.DisruptionNamespacedName, &v1beta1.Disruption{}).Return(nil).Once()

					By("call once the Get method of the reader with the watcher2 context and return a not found error")
					errorStatus := errors.StatusError{
						ErrStatus: metav1.Status{
							Message: "Not found",
							Reason:  metav1.StatusReasonNotFound,
							Code:    http.StatusNotFound,
						},
					}
					readerMock.EXPECT().Get(ctxTupleWatcher2.Ctx, ctxTupleWatcher2.DisruptionNamespacedName, &v1beta1.Disruption{}).Return(&errorStatus).Once()

					By("call once the Clean function of the watcher2")
					watcherMock2.EXPECT().Clean().Once()

					// Act
					watcherManager.RemoveOrphanWatchers()
				})

				When("the RemoveOrphanWatchers method is called a second time", func() {
					JustBeforeEach(func() {
						// Arrange: Reset watchers
						watcherMock1 = NewWatcherMock(GinkgoT())
						watcherMock2 = NewWatcherMock(GinkgoT())

						// Arrange/Assert
						By("call once the GetContextTuple method of watcher1")
						watcherMock1GetContextTuple.Return(ctxTuple, nil).Once()

						By("call once the Get method of the reader ")
						readerMock.EXPECT().Get(ctxTuple.Ctx, ctxTuple.DisruptionNamespacedName, &v1beta1.Disruption{}).Return(nil).Once()

						// Act
						watcherManager.RemoveOrphanWatchers()
					})

					It("should have removed the watcher2 from its cache", func() {
						By("not call the GetContextTuple method of the watcher2")
						watcherMock2.AssertNotCalled(GinkgoT(), methodWatcherGetContextTuple)
					})
				})
			})

			When("the Get method of the reader return an unexpected error for the watcher2", func() {
				BeforeEach(func() {
					// Arrange/Assert
					By("call once the GetContextTuple method of watcher2")
					watcherMock2GetContextTuple.Return(ctxTupleWatcher2, nil).Once()

					By("call once the Get method of the reader with the watcher1 context")
					readerMock.EXPECT().Get(ctxTuple.Ctx, ctxTuple.DisruptionNamespacedName, &v1beta1.Disruption{}).Return(nil).Once()

					By("call once the Get method of the reader with the watcher2 context and return an unexpected error")
					errorStatus := errors.StatusError{
						ErrStatus: metav1.Status{
							Message: "Unknown error",
							Reason:  metav1.StatusFailure,
							Code:    http.StatusInternalServerError,
						},
					}
					readerMock.EXPECT().Get(ctxTupleWatcher2.Ctx, ctxTupleWatcher2.DisruptionNamespacedName, &v1beta1.Disruption{}).Return(&errorStatus).Once()

					// Act
					watcherManager.RemoveOrphanWatchers()
				})

				It("should not call the clean method of the watcher2", func() {
					watcherMock2.AssertNotCalled(GinkgoT(), methodWatcherClean)
				})

				When("the RemoveOrphanWatchers method is called a second time", func() {
					JustBeforeEach(func() {
						// Arrange/Assert
						By("call once the GetContextTuple method of both watchers")
						watcherMock1GetContextTuple.Return(ctxTuple, nil).Once()
						watcherMock2GetContextTuple.Return(ctxTuple, nil).Once()

						By("call twice the Get method of the reader")
						readerMock.EXPECT().Get(ctxTuple.Ctx, ctxTuple.DisruptionNamespacedName, &v1beta1.Disruption{}).Return(nil).Twice()

						// Act
						watcherManager.RemoveOrphanWatchers()
					})

					It("should not call the Clean method of both watchers", func() {
						watcherMock1.AssertNotCalled(GinkgoT(), methodWatcherClean)
						watcherMock2.AssertNotCalled(GinkgoT(), methodWatcherClean)
					})
				})
			})
		})
	})

	When("RemoveAllWatchers function is called", func() {
		Context("with two watchers", func() {
			BeforeEach(func() {
				// Arrange
				watcherMock1, watcherMock2 = createTwoWatchers(watcherManager, controllerMock, ctxTuple, ctxTuple, syncingSourceMock, syncingSourceMock)

				By("call once the Clean function of both watchers")
				watcherMock1.EXPECT().Clean().Once()
				watcherMock2.EXPECT().Clean().Once()
			})

			It("should remove all watchers", func() {
				// Act
				watcherManager.RemoveAllWatchers()
			})

			When("the RemoveAllWatchers method is recalled", func() {
				It("should do nothing", func() {
					// Arrange / reset mocks
					watcherManager.RemoveAllWatchers()

					watcherMock1.ExpectedCalls = nil
					watcherMock1.Calls = nil
					watcherMock2.ExpectedCalls = nil
					watcherMock2.Calls = nil

					// Act
					watcherManager.RemoveAllWatchers()

					// Assert
					watcherMock2.AssertNumberOfCalls(GinkgoT(), methodWatcherClean, 0)
				})
			})
		})
	})
})

func createSingleWatcher(name string, manager Manager, controllerMock *mocks.ControllerMock, tuple CtxTuple, source *mocks.SyncingSourceMock) (watcherMock *WatcherMock) {
	controllerMock.EXPECT().Watch(mock.Anything, mock.Anything, mock.Anything).Return(nil)

	watcherMock = NewWatcherMock(GinkgoT())

	watcherMockGetName := watcherMock.EXPECT().GetName().Return(name)
	watcherMockGetName.Once()

	watcherMockStart := watcherMock.EXPECT().Start().Return(nil)
	watcherMockStart.Once().NotBefore(watcherMockGetName.Call)

	watcherMockGetCacheSource := watcherMock.EXPECT().GetCacheSource().Return(source, nil)
	watcherMockGetCacheSource.Once().NotBefore(watcherMockStart.Call)

	watcherGetContextTuple := watcherMock.EXPECT().GetContextTuple().Return(tuple, nil)
	watcherGetContextTuple.Once().NotBefore(watcherMockGetCacheSource.Call)

	Expect(manager.AddWatcher(watcherMock)).Should(Succeed())

	watcherMock.Calls = nil

	return
}

func createTwoWatchers(manager Manager, controllerMock *mocks.ControllerMock, watcher1CtxTuple, watcher2CtxTuple CtxTuple, watcher1SyncSource, watcher2SyncSource *mocks.SyncingSourceMock) (watcherMock1, watcherMock2 *WatcherMock) {
	watcherMock1 = createSingleWatcher("watcher1", manager, controllerMock, watcher1CtxTuple, watcher1SyncSource)
	watcherMock2 = createSingleWatcher("watcher2", manager, controllerMock, watcher2CtxTuple, watcher2SyncSource)
	return
}
