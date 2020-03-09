package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/stefanprodan/kjob/pkg/job"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var runJobCmd = &cobra.Command{
	Use:     `run --template cron-job-template --namespace namespace`,
	Example: `  run --kubeconfig $HOME/.kube/config -t curl -c "curl -sL flagger.app | grep License" --cleanup=false`,
	RunE:    runJob,
}

var (
	masterURL  string
	kubeconfig string
	template   string
	namespace  string
	command    string
	cleanup    bool
)

func init() {
	runJobCmd.Flags().StringVarP(&masterURL, "master", "", "", "The address of the Kubernetes API server.")
	runJobCmd.Flags().StringVarP(&kubeconfig, "kubeconfig", "", "", "Path to a kubeconfig file.")
	runJobCmd.Flags().StringVarP(&template, "template", "t", "", "CronJob name used as template.")
	runJobCmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Namespace of the CronJob used as template.")
	runJobCmd.Flags().StringVarP(&command, "command", "c", "", "Override container command.")
	runJobCmd.Flags().BoolVarP(&cleanup, "cleanup", "", true, "Delete job and pods after completion.")

	rootCmd.AddCommand(runJobCmd)
}

func runJob(cmd *cobra.Command, args []string) error {
	if template == "" {
		return fmt.Errorf("--template is required")
	}
	if namespace == "" {
		return fmt.Errorf("--namespace is required")
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

	informers := job.StartInformers(client, namespace, stopCh)

	logs, err := job.Run(client, informers, template, namespace, command, cleanup)
	if logs != "" {
		log.Print(logs)
	}
	if err != nil {
		log.Fatalf("Error running job: %v", err)
	}

	return nil
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
