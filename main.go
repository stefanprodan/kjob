package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

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

	stopCh := setupSignalHandler()

	config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		log.Fatalf("Error building kubeconfig: %v", err)
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error building kubernetes client: %v", err)
	}

	informers := startInformers(client, namespace, stopCh)

	logs, err := runJob(client, informers, template, namespace)
	if logs != "" {
		log.Println(logs)
	}
	if err != nil {
		log.Fatalf("Error running job: %v", err)
	}
}

var onlyOneSignalHandler = make(chan struct{})
var shutdownSignals = []os.Signal{os.Interrupt, syscall.SIGTERM}

func setupSignalHandler() (stopCh <-chan struct{}) {
	close(onlyOneSignalHandler)
	stop := make(chan struct{})
	c := make(chan os.Signal, 2)
	signal.Notify(c, shutdownSignals...)
	go func() {
		<-c
		close(stop)
		<-c
		os.Exit(1)
	}()

	return stop
}
