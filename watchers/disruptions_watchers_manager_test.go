// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.

package watchers_test

import (
	"fmt"
	"net/http"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/mocks"
	"github.com/DataDog/chaos-controller/watchers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Disruptions watchers manager", func() {
	var (
		chaosPodWatcherMock                                              *watchers.WatcherMock
		cacheMock                                                        *CacheMock
		disruption                                                       *chaosv1beta1.Disruption
		disruptionsWatchersManager                                       watchers.DisruptionsWatchersManager
		disruptionTargetWatcherMock                                      *watchers.WatcherMock
		err                                                              error
		expectedDisruptionTargetWatcherName, expectedChaosPodWatcherName string
		twoDisruptions                                                   []*chaosv1beta1.Disruption
		watchersManagerMock                                              *watchers.ManagerMock
		watcherFactoryMock                                               *watchers.FactoryMock
		readerMock                                                       *mocks.ReaderMock
	)

	BeforeEach(func() {
		// Arrange
		cacheMock = &CacheMock{}
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
		controllerMock := mocks.NewRuntimeControllerMock(GinkgoT())

		// Act
		disruptionsWatchersManager = watchers.NewDisruptionsWatchersManager(controllerMock, watcherFactoryMock, readerMock, logger)

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

				expectedDisruptionTargetWatcherName = disSpecHash + string(watchers.DisruptionTargetWatcherName)
				expectedChaosPodWatcherName = disSpecHash + string(watchers.ChaosPodWatcherName)

				chaosPodWatcherMock = watchers.NewWatcherMock(GinkgoT())
				disruptionTargetWatcherMock = watchers.NewWatcherMock(GinkgoT())
				watchersManagerMock = watchers.NewManagerMock(GinkgoT())
			})

			JustBeforeEach(func() {
				// Act
				err = disruptionsWatchersManager.CreateAllWatchers(disruption, watchersManagerMock, cacheMock)
			})

			Context("nominal cases", func() {

				BeforeEach(func() {
					// Arrange / Assert
					By("check once the presence of both watchers")
					watchersManagerMock.EXPECT().GetWatcher(expectedDisruptionTargetWatcherName).Return(nil).Once()
					watchersManagerMock.EXPECT().GetWatcher(expectedChaosPodWatcherName).Return(nil).Once()

					By("create the Disruption Target watcher")
					watcherFactoryMock.EXPECT().NewDisruptionTargetWatcher(expectedDisruptionTargetWatcherName, true, disruption, cacheMock).Return(disruptionTargetWatcherMock, nil).Once()

					By("create the Chaos Pod watcher")
					watcherFactoryMock.EXPECT().NewChaosPodWatcher(expectedChaosPodWatcherName, disruption, cacheMock).Return(chaosPodWatcherMock, nil).Once()

					By("adding both watchers into the watcherManager")
					watchersManagerMock.EXPECT().AddWatcher(disruptionTargetWatcherMock).Return(nil).Once()
					watchersManagerMock.EXPECT().AddWatcher(chaosPodWatcherMock).Return(nil).Once()
				})

				It("should not return an error", func() {
					Expect(err).ShouldNot(HaveOccurred())
				})

				When("CreateAllWatchers method is recalled", func() {
					It("should not recreate watchers", func() {
						// Arrange / Assert
						watchersManagerMock = watchers.NewManagerMock(GinkgoT())

						By("check once the presence of both watchers")
						watchersManagerMock.EXPECT().GetWatcher(expectedDisruptionTargetWatcherName).Return(disruptionTargetWatcherMock).Once()
						watchersManagerMock.EXPECT().GetWatcher(expectedChaosPodWatcherName).Return(chaosPodWatcherMock).Once()

						// Act
						Expect(disruptionsWatchersManager.CreateAllWatchers(disruption, watchersManagerMock, cacheMock)).Should(Succeed())

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
						watchersManagerMock.EXPECT().GetWatcher(expectedDisruptionTargetWatcherName).Return(nil)

						watcherFactoryMock = watchers.NewFactoryMock(GinkgoT())
						watcherFactoryMock.EXPECT().NewDisruptionTargetWatcher(expectedDisruptionTargetWatcherName, true, disruption, cacheMock).Return(disruptionTargetWatcherMock, fmt.Errorf("NewDisruptionTargetWatcher error")).Once()
						watcherFactoryMock.EXPECT().NewChaosPodWatcher(expectedChaosPodWatcherName, disruption, cacheMock).Return(disruptionTargetWatcherMock, nil).Maybe()
					})

					It("should return an error", func() {
						// Act
						Expect(err).To(HaveOccurred())
						Expect(err).To(MatchError("NewDisruptionTargetWatcher error"))
					})
				})

				When("the NewChaosPodWatcher method of the factory return an error", func() {
					BeforeEach(func() {
						// Arrange / Assert
						watchersManagerMock.EXPECT().AddWatcher(mock.Anything).Return(nil).Maybe()
						watchersManagerMock.EXPECT().AddWatcher(disruptionTargetWatcherMock).Return(nil).Maybe()

						watcherFactoryMock = watchers.NewFactoryMock(GinkgoT())
						watcherFactoryMock.EXPECT().NewDisruptionTargetWatcher(expectedDisruptionTargetWatcherName, true, disruption, cacheMock).Return(disruptionTargetWatcherMock, nil).Maybe()
						watcherFactoryMock.EXPECT().NewChaosPodWatcher(expectedChaosPodWatcherName, disruption, cacheMock).Return(disruptionTargetWatcherMock, fmt.Errorf("NewChaosPodWatcher error message")).Once()
					})

					It("should return an error", func() {
						Expect(err).To(HaveOccurred())
						Expect(err).To(MatchError("NewChaosPodWatcher error message"))
					})
				})

				When("the AddWatcher method of the watcherManager return an error with the disruptionTargetWatcher in parameter", func() {
					BeforeEach(func() {
						// Arrange / Assert
						watchersManagerMock.EXPECT().AddWatcher(disruptionTargetWatcherMock).Return(fmt.Errorf("disruptionTargetWatcher message")).Once()
						watchersManagerMock.EXPECT().AddWatcher(mock.Anything).Return(nil).Maybe()

						watcherFactoryMock.EXPECT().NewDisruptionTargetWatcher(expectedDisruptionTargetWatcherName, true, disruption, cacheMock).Return(disruptionTargetWatcherMock, nil).Once()
						watcherFactoryMock.EXPECT().NewChaosPodWatcher(expectedChaosPodWatcherName, disruption, cacheMock).Return(chaosPodWatcherMock, nil).Once().Maybe()
					})

					It("should return an error", func() {
						Expect(err).To(HaveOccurred())
						Expect(err).To(MatchError("failed to create watcher: disruptionTargetWatcher message"))
					})
				})

				When("the AddWatcher method of the watcherManager return an error with the chaosPodWatcher in parameter", func() {
					BeforeEach(func() {
						// Arrange / Assert
						watchersManagerMock.EXPECT().AddWatcher(chaosPodWatcherMock).Return(fmt.Errorf("chaosPodWatcher error"))
						watchersManagerMock.EXPECT().AddWatcher(mock.Anything).Return(nil).Maybe()

						watcherFactoryMock.EXPECT().NewDisruptionTargetWatcher(expectedDisruptionTargetWatcherName, true, disruption, cacheMock).Return(disruptionTargetWatcherMock, nil).Maybe()
						watcherFactoryMock.EXPECT().NewChaosPodWatcher(expectedChaosPodWatcherName, disruption, cacheMock).Return(chaosPodWatcherMock, nil).Once()
					})

					It("should return an error", func() {
						Expect(err).To(HaveOccurred())
						Expect(err).To(MatchError("failed to create watcher: chaosPodWatcher error"))
					})
				})
			})
		})

		Context("with an empty disruption", func() {
			BeforeEach(func() {
				// Arrange
				disruption = &chaosv1beta1.Disruption{}
			})

			It("should return an error", func() {
				// Act
				err := disruptionsWatchersManager.CreateAllWatchers(disruption, watchersManagerMock, cacheMock)

				// Assert
				Expect(err).Should(HaveOccurred())
				Expect(err).To(MatchError("the disruption is not valid. It should contain a name and a namespace"))
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

				expectedDisruptionTargetWatcherName = disSpecHash + string(watchers.DisruptionTargetWatcherName)
				expectedChaosPodWatcherName = disSpecHash + string(watchers.ChaosPodWatcherName)

				chaosPodWatcherMock = watchers.NewWatcherMock(GinkgoT())
				disruptionTargetWatcherMock = watchers.NewWatcherMock(GinkgoT())

				watchersManagerMock = watchers.NewManagerMock(GinkgoT())
				watchersManagerMock.EXPECT().AddWatcher(mock.Anything).Return(nil)
				watchersManagerMock.EXPECT().GetWatcher(mock.Anything).Return(nil)

				watcherFactoryMock = watchers.NewFactoryMock(GinkgoT())
				watcherFactoryMock.EXPECT().NewChaosPodWatcher(expectedChaosPodWatcherName, disruption, cacheMock).Return(watchers.NewWatcherMock(GinkgoT()), nil)
				watcherFactoryMock.EXPECT().NewDisruptionTargetWatcher(expectedDisruptionTargetWatcherName, mock.Anything, disruption, cacheMock).Return(watchers.NewWatcherMock(GinkgoT()), nil)
			})

			JustBeforeEach(func() {
				Expect(disruptionsWatchersManager.CreateAllWatchers(disruption, watchersManagerMock, cacheMock)).Should(Succeed())
			})

			Context("nominal cases", func() {
				BeforeEach(func() {
					// Arrange / Assert
					By("call once the RemoveAllWatchers method of the watcherManager")
					watchersManagerMock.EXPECT().RemoveAllWatchers().Once()
				})

				It("should remove all watchers", func() {
					//	Action
					disruptionsWatchersManager.RemoveAllWatchers(disruption)
				})

				When("the RemoveAllWatchers method is recalled", func() {
					It("should do nothing", func() {
						// Arrange
						disruptionsWatchersManager.RemoveAllWatchers(disruption)

						watchersManagerMock.ExpectedCalls = nil
						watchersManagerMock.Calls = nil

						// Act
						disruptionsWatchersManager.RemoveAllWatchers(disruption)

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

			It("should do nothing", func() {
				disruption = &chaosv1beta1.Disruption{}

				disruptionsWatchersManager.RemoveAllWatchers(disruption)
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

				expectedDisruptionTargetWatcherName = disSpecHash + string(watchers.DisruptionTargetWatcherName)
				expectedChaosPodWatcherName = disSpecHash + string(watchers.ChaosPodWatcherName)

				chaosPodWatcherMock = watchers.NewWatcherMock(GinkgoT())
				disruptionTargetWatcherMock = watchers.NewWatcherMock(GinkgoT())

				watchersManagerMock.EXPECT().AddWatcher(mock.Anything).Return(nil)
				watchersManagerMock.EXPECT().GetWatcher(mock.Anything).Return(nil)
				watcherFactoryMock.EXPECT().NewChaosPodWatcher(expectedChaosPodWatcherName, disruption, cacheMock).Return(watchers.NewWatcherMock(GinkgoT()), nil)
				watcherFactoryMock.EXPECT().NewDisruptionTargetWatcher(expectedDisruptionTargetWatcherName, mock.Anything, disruption, cacheMock).Return(watchers.NewWatcherMock(GinkgoT()), nil)
			}
		})

		JustBeforeEach(func() {
			for _, disruption := range twoDisruptions {
				Expect(disruptionsWatchersManager.CreateAllWatchers(disruption, watchersManagerMock, cacheMock)).To(Succeed())
			}
		})

		Context("with two non orphan disruptions", func() {
			BeforeEach(func() {
				// Arrange / Assert
				By("check if disruptions exist")
				readerMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(nil).Twice()
			})

			It("should do nothing", func() {
				Expect(disruptionsWatchersManager.RemoveAllOrphanWatchers()).Should(Succeed())
			})
		})

		Context("with two not found disruptions", func() {
			BeforeEach(func() {
				// Arrange / Assert
				var errorStatus = errors.StatusError{
					ErrStatus: metav1.Status{
						Message: "Not found",
						Reason:  metav1.StatusReasonNotFound,
						Code:    http.StatusNotFound,
					},
				}
				By("check if the disruptions exists")
				readerMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(&errorStatus).Twice()

				By("call the RemoveAllWatchers method of the watcher instance")
				watchersManagerMock.EXPECT().RemoveAllWatchers().Twice()
			})

			It("should do nothing", func() {
				Expect(disruptionsWatchersManager.RemoveAllOrphanWatchers()).Should(Succeed())
			})

			When("the function RemoveAllOrphanWatchers is recalled", func() {
				It("should have removed the disruption", func() {
					// Arrange
					Expect(disruptionsWatchersManager.RemoveAllOrphanWatchers()).To(Succeed())
					readerMock.ExpectedCalls = nil
					readerMock.Calls = nil
					watchersManagerMock.ExpectedCalls = nil
					watchersManagerMock.Calls = nil

					// Action
					Expect(disruptionsWatchersManager.RemoveAllOrphanWatchers()).To(Succeed())

					// Assert
					By("not call any Get  method of the reader")
					readerMock.AssertNumberOfCalls(GinkgoT(), "Get", 0)

					By("not call any RemoveAllWatcher method of the watchersManager")
					watchersManagerMock.AssertNumberOfCalls(GinkgoT(), "RemoveAllWatchers", 0)
				})
			})
		})

		When("the Get method of the reader return a server error", func() {
			BeforeEach(func() {
				var errorStatus = errors.StatusError{
					ErrStatus: metav1.Status{
						Message: "Unknown error",
						Reason:  metav1.StatusReasonInternalError,
						Code:    http.StatusInternalServerError,
					},
				}
				By("check if the disruptions exists")
				readerMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(&errorStatus).Twice()
			})

			It("should do nothing", func() {
				// Act
				Expect(disruptionsWatchersManager.RemoveAllOrphanWatchers()).To(Succeed())

				// Assert
				By("not call the RemoveAllWatchers method of the watchersManager")
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

					expectedDisruptionTargetWatcherName = disSpecHash + string(watchers.DisruptionTargetWatcherName)
					expectedChaosPodWatcherName = disSpecHash + string(watchers.ChaosPodWatcherName)

					chaosPodWatcherMock = watchers.NewWatcherMock(GinkgoT())
					disruptionTargetWatcherMock = watchers.NewWatcherMock(GinkgoT())

					watchersManagerMock.EXPECT().AddWatcher(mock.Anything).Return(nil)
					watchersManagerMock.EXPECT().GetWatcher(mock.Anything).Return(nil)
					watcherFactoryMock.EXPECT().NewChaosPodWatcher(expectedChaosPodWatcherName, disruption, cacheMock).Return(watchers.NewWatcherMock(GinkgoT()), nil)
					watcherFactoryMock.EXPECT().NewDisruptionTargetWatcher(expectedDisruptionTargetWatcherName, mock.Anything, disruption, cacheMock).Return(watchers.NewWatcherMock(GinkgoT()), nil)
				}
			})

			JustBeforeEach(func() {
				for _, disruption := range twoDisruptions {
					Expect(disruptionsWatchersManager.CreateAllWatchers(disruption, watchersManagerMock, cacheMock)).To(Succeed())
				}
			})

			It("should call twice the RemoveExpiredWatchers method of the watchersManager", func() {
				// Arrange / Assert
				watchersManagerMock.EXPECT().RemoveExpiredWatchers().Twice()

				// Act
				disruptionsWatchersManager.RemoveAllExpiredWatchers()
			})
		})

		Context("without disruption", func() {
			It("should do nothing", func() {
				// Act
				disruptionsWatchersManager.RemoveAllExpiredWatchers()

				// Assert
				By("not call the RemoveExpiredWatchers method of the watchersManager")
				watchersManagerMock.AssertNotCalled(GinkgoT(), "RemoveExpiredWatchers", 0)
			})
		})
	})
})
