// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package watchers_test

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	k8scache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/DataDog/chaos-controller/mocks"
	"github.com/DataDog/chaos-controller/watchers"
	"github.com/stretchr/testify/mock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("watcher", func() {
	const watcherName = "watcher-name"

	var (
		cacheContextFunc func() (ctx context.Context, cancel context.CancelFunc)
		cacheMock        *mocks.CacheCacheMock
		err              error
		handlerMock      *mocks.ResourceEventHandlerMock
		informerMock     *mocks.CacheInformerMock
		targetObjectType client.Object
		watcher          watchers.Watcher
	)

	// Arrange
	namespaceName := types.NamespacedName{Name: "namespace-name", Namespace: "namespace"}
	targetObjectType = &v1.Pod{}

	JustBeforeEach(func() {
		var config = watchers.WatcherConfig{
			Name:           watcherName,
			Handler:        handlerMock,
			ObjectType:     targetObjectType,
			NamespacedName: namespaceName,
			CacheOptions:   k8scache.Options{},
			Log:            logger,
		}

		watcher, err = watchers.NewWatcher(config, cacheMock, cacheContextFunc)
		Expect(err).ShouldNot(HaveOccurred())
	})

	When("Start method is called", func() {
		BeforeEach(func() {
			// Arrange
			handlerMock = mocks.NewResourceEventHandlerMock(GinkgoT())
			informerMock = mocks.NewCacheInformerMock(GinkgoT())
			cacheMock = mocks.NewCacheCacheMock(GinkgoT())
		})

		Context("without error", func() {
			It("should start the watcher", func() {
				// Arrange / Assert
				By("call once the GetInformer method of the Cache")
				cacheMock.EXPECT().GetInformer(mock.Anything, targetObjectType).Return(informerMock, nil).Once()

				By("call once AddEventHandler method of the Informer")
				informerMock.EXPECT().AddEventHandler(handlerMock).Return(nil, nil).Once()

				By("call once the Start method of the watcher")
				cacheMock.EXPECT().Start(mock.Anything).Return(nil).Maybe() // Maybe because it is called in the background

				// Act
				err = watcher.Start()

				// Assert
				ctxTuple, err := watcher.GetContextTuple()
				Expect(err).ShouldNot(HaveOccurred())
				Expect(ctxTuple.NamespacedName.Name).Should(Equal(namespaceName.Name))
				Expect(ctxTuple.NamespacedName.Namespace).Should(Equal(namespaceName.Namespace))
			})
		})

		When("GetInformer method of Cache returns an error", func() {
			BeforeEach(func() {
				// Arrange
				cacheMock.On("GetInformer", mock.Anything, targetObjectType).Return(mocks.NewCacheInformerMock(GinkgoT()), fmt.Errorf("get informer error"))
			})

			It("should return an error", func() {
				//  Act
				err := watcher.Start()

				// Assert
				Expect(err).Should(HaveOccurred())
				Expect(err).To(MatchError("error getting informer from cache: get informer error"))
			})
		})

		When("AddEventHandler method of Informer returns an error", func() {
			JustBeforeEach(func() {
				// Arrange
				cacheMock.On("GetInformer", mock.Anything, targetObjectType).Return(informerMock, nil).Once()
				informerMock.EXPECT().AddEventHandler(handlerMock).Return(nil, fmt.Errorf("informer error")).Once()
			})

			It("should return an error", func() {
				// Act
				err = watcher.Start()

				Expect(err).Should(HaveOccurred())
				Expect(err).To(MatchError("error adding event handler to the informer: informer error"))
			})
		})

		When("Start method of Cache returns an error", func() {
			BeforeEach(func() {
				// Arrange
				cacheMock.EXPECT().GetInformer(mock.Anything, targetObjectType).Return(informerMock, nil).Once()
				informerMock.EXPECT().AddEventHandler(handlerMock).Return(nil, nil).Once()
				cacheMock.EXPECT().Start(mock.Anything).Return(fmt.Errorf("start error")).Maybe()
			})

			It("should not return an error", func() {
				Expect(watcher.Start()).To(Succeed())
			})
		})
	})

	When("GetName method is called", func() {
		It("should return the expected watcher name", func() {
			// Act & Assert
			Expect(watcher.GetName()).Should(Equal(watcherName))
		})
	})

	When("Clean method is called", func() {
		var IsCacheCancelFuncCalled bool

		BeforeEach(func() {
			// Arrange
			IsCacheCancelFuncCalled = false
			cacheContextFunc = func() (ctx context.Context, cancel context.CancelFunc) {
				return context.Background(), func() {
					IsCacheCancelFuncCalled = true
				}
			}
		})

		It("should cancel the context", func() {
			// Act
			watcher.Clean()

			// Assert
			Expect(IsCacheCancelFuncCalled).To(BeTrue())
		})
	})

	When("IsExpired method is called", func() {
		var contextMock context.Context
		var result bool

		BeforeEach(func() {
			// Arrange
			contextMock = context.Background()

			By("call once the Err method of the context")
			cacheContextFunc = func() (ctx context.Context, cancel context.CancelFunc) {
				return contextMock, func() {}
			}
		})

		JustBeforeEach(func() {
			// Act
			result = watcher.IsExpired()
		})

		Context("nominal case", func() {
			It("should return false", func() {
				Expect(result).To(BeFalse())
			})
		})
	})

	When("GetCacheSource method is called", func() {
		var result source.SyncingSource

		Context("with a non empty cacheSource", func() {
			BeforeEach(func() {
				// Arrange
				handlerMock = mocks.NewResourceEventHandlerMock(GinkgoT())
				informerMock = mocks.NewCacheInformerMock(GinkgoT())
				cacheMock = mocks.NewCacheCacheMock(GinkgoT())
				informerMock.EXPECT().AddEventHandler(handlerMock).Return(informerMock, nil).Maybe()
				cacheMock.EXPECT().GetInformer(mock.Anything, mock.Anything).Return(informerMock, nil).Maybe()
				cacheMock.EXPECT().Start(mock.Anything).Return(nil).Maybe()
			})

			JustBeforeEach(func() {
				// Arrange
				Expect(watcher.Start()).ShouldNot(HaveOccurred())

				// Act
				result, err = watcher.GetCacheSource()
				Expect(err).ShouldNot(HaveOccurred())
			})

			It("should return an object", func() {
				Expect(result).ShouldNot(BeNil())
			})
		})

		Context("with an empty cacheSource", func() {
			JustBeforeEach(func() {
				// Act
				result, err = watcher.GetCacheSource()
			})

			It("should return an error", func() {
				Expect(err).Should(HaveOccurred())
				Expect(err).To(MatchError("the watcher should be started with its Start method in order to initialise the cache source"))
			})

			It("should return nil", func() {
				Expect(result).Should(BeNil())
			})
		})
	})

	When("GetContextTuple method is called", func() {
		var result watchers.CtxTuple

		BeforeEach(func() {
			// Arrange
			cacheContextFunc = nil
		})

		Context("with an empty context tuple", func() {
			It("should return an error", func() {
				// Act
				_, err = watcher.GetContextTuple()

				// Assert
				Expect(err).Should(HaveOccurred())
				Expect(err).To(MatchError("the watcher should be started with its Start method in order to initialize the context tuple"))
			})
		})

		Context("with an non empty context tuple", func() {
			BeforeEach(func() {
				handlerMock = mocks.NewResourceEventHandlerMock(GinkgoT())
				informerMock = mocks.NewCacheInformerMock(GinkgoT())
				cacheMock = mocks.NewCacheCacheMock(GinkgoT())
				informerMock.EXPECT().AddEventHandler(handlerMock).Return(informerMock, nil)
				cacheMock.On("GetInformer", mock.Anything, mock.Anything).Return(informerMock, nil)
				cacheMock.On("Start", mock.Anything).Return(nil)
			})

			JustBeforeEach(func() {
				// Arrange
				Expect(watcher.Start()).Should(Succeed())

				// Act
				result, err = watcher.GetContextTuple()
			})

			It("should not return an error", func() {
				Expect(err).ShouldNot(HaveOccurred())
			})

			It("should return the expected context tuple", func() {
				cacheCtx, cacheCancelFunc := context.WithCancel(context.Background())
				Expect(result.Ctx).Should(BeEquivalentTo(cacheCtx))
				Expect(result.CancelFunc).Should(BeAssignableToTypeOf(cacheCancelFunc))
				Expect(result.NamespacedName).Should(Equal(namespaceName))
			})
		})
	})
})
