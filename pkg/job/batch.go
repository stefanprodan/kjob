package job

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

func Run(client *kubernetes.Clientset, informers Informers, name string, namespace string, command string, cleanup bool) (string, error) {
	cronjob, err := informers.CronJobInformer.Lister().CronJobs(namespace).Get(name)
	if err != nil {
		return "", err
	}

	spec := cronjob.Spec.JobTemplate.Spec
	if command != "" {
		// TODO: get rid of sh
		spec.Template.Spec.Containers[0].Command = []string{
			"/bin/sh",
			"-c",
			command,
		}
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: cronjob.GetName() + "-",
			Namespace:    namespace,
		},
		Spec: spec,
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

	logs, err := getLogs(client, pods, namespace)
	if err != nil {
		return "", err
	}

	if cleanup {
		err = jobCleanup(client, pods, jobName, namespace)
		if err != nil {
			return "", err
		}
	}

	if failureMessage != "" {
		return logs, fmt.Errorf(failureMessage)
	}
	return logs, nil
}

func jobCleanup(client *kubernetes.Clientset, pods []string, job string, namespace string) error {
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

func getLogs(client *kubernetes.Clientset, pods []string, namespace string) (string, error) {
	buf := new(bytes.Buffer)

	for _, pod := range pods {
		req := client.CoreV1().Pods(namespace).GetLogs(pod, &corev1.PodLogOptions{})
		stream, err := req.Stream()
		if err != nil {
			return "", err
		}

		_, err = io.Copy(buf, stream)
		stream.Close()
		if err != nil {
			return "", err
		}
	}

	return buf.String(), nil
}
