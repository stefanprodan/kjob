package main

import (
	"log"
	"time"

	kubeInformers "k8s.io/client-go/informers"
	jobInformers "k8s.io/client-go/informers/batch/v1"
	cronJobInformers "k8s.io/client-go/informers/batch/v1beta1"
	podInformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type Informers struct {
	CronJobInformer cronJobInformers.CronJobInformer
	JobInformer     jobInformers.JobInformer
	PodInformer     podInformers.PodInformer
}

func startInformers(client *kubernetes.Clientset, namespace string, stopCh <-chan struct{}) Informers {
	factory := kubeInformers.NewSharedInformerFactoryWithOptions(client, 5*time.Second, kubeInformers.WithNamespace(namespace))

	cronJobsInformer := factory.Batch().V1beta1().CronJobs()
	go cronJobsInformer.Informer().Run(stopCh)
	if ok := cache.WaitForCacheSync(stopCh, cronJobsInformer.Informer().HasSynced); !ok {
		log.Fatalf("failed to wait for CronJobs cache to sync")
	}

	jobsInformer := factory.Batch().V1().Jobs()
	go jobsInformer.Informer().Run(stopCh)
	if ok := cache.WaitForCacheSync(stopCh, jobsInformer.Informer().HasSynced); !ok {
		log.Fatalf("failed to wait for Jobs cache to sync")
	}

	podsInformer := factory.Core().V1().Pods()
	go podsInformer.Informer().Run(stopCh)
	if ok := cache.WaitForCacheSync(stopCh, podsInformer.Informer().HasSynced); !ok {
		log.Fatalf("failed to wait for Pods cache to sync")
	}

	return Informers{
		CronJobInformer: cronJobsInformer,
		JobInformer:     jobsInformer,
		PodInformer:     podsInformer,
	}
}
