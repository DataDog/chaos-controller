// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package watchers_test

import (
	"context"
	"fmt"
	"sync"

	"github.com/DataDog/chaos-controller/mocks"
	"github.com/DataDog/chaos-controller/watchers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	k8scache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type CacheMock struct {
	mock.Mock
	Wg sync.WaitGroup
}

func (c *CacheMock) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	if opts != nil {
		args := c.Called(ctx, key, obj, opts)
		return args.Error(0)
	}

	args := c.Called(ctx, key, obj)

	return args.Error(0)
}

func (c *CacheMock) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	if opts != nil {
		args := c.Called(ctx, list, opts)
		return args.Error(0)
	}

	args := c.Called(ctx, list)

	return args.Error(0)
}

func (c *CacheMock) GetInformer(ctx context.Context, obj client.Object) (k8scache.Informer, error) {
	args := c.Called(ctx, obj)
	return args.Get(0).(k8scache.Informer), args.Error(1)
}

func (c *CacheMock) GetInformerForKind(ctx context.Context, gvk schema.GroupVersionKind) (k8scache.Informer, error) {
	args := c.Called(ctx, gvk)
	return args.Get(0).(k8scache.Informer), args.Error(1)
}

func (c *CacheMock) Start(ctx context.Context) error {
	args := c.Called(ctx)
	c.Wg.Done()

	return args.Error(0)
}

func (c *CacheMock) WaitForCacheSync(ctx context.Context) bool {
	args := c.Called(ctx)
	return args.Bool(0)
}

func (c *CacheMock) IndexField(ctx context.Context, obj client.Object, field string, extractValue client.IndexerFunc) error {
	args := c.Called(ctx, obj, field, extractValue)
	return args.Error(0)
}

var _ = Describe("watcher", func() {
	const watcherName = "watcher-name"

	var (
		cacheContextFunc func() (ctx context.Context, cancel context.CancelFunc)
		cacheMock        CacheMock
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

		watcher, err = watchers.NewWatcher(config, &cacheMock, cacheContextFunc)
		Expect(err).ShouldNot(HaveOccurred())
	})

	When("Start method is called", func() {
		BeforeEach(func() {
			// Arrange
			handlerMock = mocks.NewResourceEventHandlerMock(GinkgoT())
			informerMock = mocks.NewCacheInformerMock(GinkgoT())
			cacheMock = CacheMock{}
		})

		Context("without error", func() {
			It("should start the watcher", func() {
				// Arrange / Assert
				By("call once the GetInformer method of the Cache")
				cacheMock.On("GetInformer", mock.Anything, targetObjectType).Return(informerMock, nil).Once()

				By("call once the Start method of the watcher")
				cacheMock.On("Start", mock.Anything).Return(nil).Once()

				By("call once AddEventHandler method of the Informer")
				informerMock.EXPECT().AddEventHandler(handlerMock).Return(nil, nil).Once()

				// Act
				cacheMock.Wg.Add(1)
				err = watcher.Start()
				cacheMock.Wg.Wait()
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
				Expect(err.Error()).Should(Equal("error getting informer from cache. Error: get informer error"))
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
				Expect(err.Error()).Should(Equal("error adding event handler to the informer. Error: informer error"))
			})
		})

		When("Start method of Cache returns an error", func() {
			BeforeEach(func() {
				// Arrange
				cacheMock.On("GetInformer", mock.Anything, targetObjectType).Return(informerMock, nil).Once()
				informerMock.EXPECT().AddEventHandler(handlerMock).Return(nil, nil).Once()
				cacheMock.On("Start", mock.Anything).Return(fmt.Errorf("start error")).Once()
			})

			It("should not return an error", func() {
				// Act
				cacheMock.Wg.Add(1)
				err = watcher.Start()
				cacheMock.Wg.Wait()

				// Assert
				Expect(err).ShouldNot(HaveOccurred())
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
				cacheMock = CacheMock{}
				informerMock.EXPECT().AddEventHandler(handlerMock).Return(mock.Anything, nil)
				cacheMock.On("GetInformer", mock.Anything, mock.Anything).Return(informerMock, nil)
				cacheMock.On("Start", mock.Anything).Return(nil)
			})

			JustBeforeEach(func() {
				// Arrange
				cacheMock.Wg.Add(1)
				Expect(watcher.Start()).ShouldNot(HaveOccurred())
				cacheMock.Wg.Wait()

				// Act
				result, err = watcher.GetCacheSource()
				Expect(err).ShouldNot(HaveOccurred())
			})

			It("should return the expected SyncingSource object", func() {
				expectedResult := source.NewKindWithCache(targetObjectType, &cacheMock)
				Expect(result).Should(BeEquivalentTo(expectedResult))
			})
		})

		Context("with an empty cacheSource", func() {
			JustBeforeEach(func() {
				// Act
				result, err = watcher.GetCacheSource()
			})

			It("should return an error", func() {
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).Should(Equal("the watcher should be started with its Start method in order to initialise the cache source"))
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
				Expect(err.Error()).Should(Equal("the watcher should be started with its Start method in order to initialize the context tuple"))
			})
		})

		Context("with an non empty context tuple", func() {
			BeforeEach(func() {
				handlerMock = mocks.NewResourceEventHandlerMock(GinkgoT())
				informerMock = mocks.NewCacheInformerMock(GinkgoT())
				cacheMock = CacheMock{}
				informerMock.EXPECT().AddEventHandler(handlerMock).Return(mock.Anything, nil)
				cacheMock.On("GetInformer", mock.Anything, mock.Anything).Return(informerMock, nil)
				cacheMock.On("Start", mock.Anything).Return(nil)
			})

			JustBeforeEach(func() {
				// Arrange
				cacheMock.Wg.Add(1)
				Expect(watcher.Start()).Should(Succeed())
				cacheMock.Wg.Wait()

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
				Expect(result.DisruptionNamespacedName).Should(Equal(namespaceName))
			})
		})
	})
})
