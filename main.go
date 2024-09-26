package main

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"

	clientset "github.com/codegen-example/pkg/generated/clientset/versioned"
	informers "github.com/codegen-example/pkg/generated/informers/externalversions/opencanon.io/v1"
)

func main() {
	// Set up the Kubernetes client configuration
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		panic(err)
	}

	// Create the clientset
	clientset, err := clientset.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	// Create the song informer
	songInformer := informers.NewSongInformer(clientset, "", time.Second*30, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})

	// Create a workqueue
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	// Set up the event handlers for the informer
	songInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(newObj)
			if err == nil {
				queue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
	})

	// Start the informer
	stopCh := make(chan struct{})
	defer close(stopCh)
	go songInformer.Run(stopCh)

	// Wait for the caches to sync
	if !cache.WaitForCacheSync(stopCh, songInformer.HasSynced) {
		panic("Failed to sync caches")
	}

	// Process items from the workqueue
	wait.Until(func() {
		for {
			key, shutdown := queue.Get()
			if shutdown {
				return
			}

			// Process the item
			fmt.Printf("Processing key: %s\n", key)

			// Mark the item as done
			queue.Done(key)
		}
	}, time.Second, stopCh)
}
