package main

import (
	"bytes"
	"fmt"
	"io"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

func runJob(client *kubernetes.Clientset, informers Informers, name string, namespace string) (string, error) {
	cronjob, err := informers.CronJobInformer.Lister().CronJobs(namespace).Get(name)
	if err != nil {
		return "", err
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: cronjob.GetName() + "-",
			Namespace:    namespace,
		},
		Spec: cronjob.Spec.JobTemplate.Spec,
	}

	job, err = client.BatchV1().Jobs(namespace).Create(job)
	if err != nil {
		return "", err
	}

	failureMessage := ""
	jobName := job.GetName()
	done := false
	for !done {
		for _, condition := range job.Status.Conditions {
			switch condition.Type {
			case batchv1.JobFailed:
				done = true
				failureMessage = condition.Message
			case batchv1.JobComplete:
				done = true
			}
		}
		time.Sleep(1 * time.Second)
		job, err = informers.JobInformer.Lister().Jobs(namespace).Get(jobName)
		if err != nil {
			return "", err
		}
	}

	selector := fmt.Sprintf("job-name=%s", jobName)
	set, _ := labels.ConvertSelectorToLabelsMap(selector)

	jobPods, err := informers.PodInformer.Lister().Pods(namespace).List(labels.SelectorFromSet(set))
	if err != nil {
		return "", err
	}

	pods := make([]string, 0, len(jobPods))
	for _, pod := range jobPods {
		pods = append(pods, pod.GetName())
	}

	if len(pods) < 1 {
		return "", fmt.Errorf("no pods found for job %s", jobName)
	}

	logs, err := logs(client, pods[0], namespace)
	if err != nil {
		return "", err
	}

	err = cleanup(client, pods, jobName, namespace)
	if err != nil {
		return "", err
	}

	if failureMessage != "" {
		return logs, fmt.Errorf(failureMessage)
	}
	return logs, nil
}

func cleanup(client *kubernetes.Clientset, pods []string, job string, namespace string) error {
	err := client.BatchV1().Jobs(namespace).Delete(job, &metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	for _, item := range pods {
		err = client.CoreV1().Pods(namespace).Delete(item, &metav1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func logs(client *kubernetes.Clientset, pod string, namespace string) (string, error) {
	req := client.CoreV1().Pods(namespace).GetLogs(pod, &corev1.PodLogOptions{})
	podLogs, err := req.Stream()
	if err != nil {
		return "", err
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
