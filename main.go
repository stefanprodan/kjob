package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const VERSION = "0.0.1"

var (
	masterURL  string
	kubeconfig string
	template   string
	namespace  string
	version    bool
)

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server.")
	flag.StringVar(&template, "template", "", "Job template name.")
	flag.StringVar(&namespace, "namespace", "default", "Job namespace.")
	flag.BoolVar(&version, "version", false, "Prints the version")
}

func main() {
	flag.Parse()
	log.SetFlags(0)

	if version {
		log.Println(VERSION)
		return
	}

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		log.Fatalf("Error building kubeconfig: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("Error building kubernetes clientset: %v", err)
	}

	_, err = runJob(clientset, template, namespace)
	if err != nil {
		log.Fatalf("Error running job: %v", err)
	}
}

func runJob(clientset *kubernetes.Clientset, name string, namespace string) (*batchv1.Job, error) {
	cronjob, err := clientset.BatchV1beta1().CronJobs(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: cronjob.GetName() + "-",
			Namespace:    namespace,
		},
		Spec: cronjob.Spec.JobTemplate.Spec,
	}

	job, err = clientset.BatchV1().Jobs(namespace).Create(job)
	if err != nil {
		return nil, err
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
		job, err = clientset.BatchV1().Jobs(namespace).Get(jobName, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
	}

	selector := fmt.Sprintf("job-name=%s", jobName)
	pods, err := clientset.CoreV1().Pods(namespace).List(metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return nil, err
	}

	if len(pods.Items) < 1 {
		return nil, fmt.Errorf("no pods found for job %s", jobName)
	}

	podName := pods.Items[0].GetName()
	req := clientset.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{})
	podLogs, err := req.Stream()
	if err != nil {
		return nil, err
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return nil, err
	}
	str := buf.String()

	log.Println(str)

	err = clientset.BatchV1().Jobs(namespace).Delete(jobName, metav1.NewDeleteOptions(5000))
	if err != nil {
		return nil, err
	}

	for _, item := range pods.Items {
		err = clientset.CoreV1().Pods(namespace).Delete(item.GetName(), metav1.NewDeleteOptions(5000))
		if err != nil {
			return nil, err
		}
	}

	if failureMessage != "" {
		return job, fmt.Errorf(failureMessage)
	}
	return job, nil
}
