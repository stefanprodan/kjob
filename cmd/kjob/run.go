package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/stefanprodan/kjob/pkg/job"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"
)

var runJobCmd = &cobra.Command{
	Use:     `run --template cron-job-template --namespace namespace`,
	Example: `  run --kubeconfig $HOME/.kube/config -t curl -c "curl -sL flagger.app | grep License" --cleanup=false`,
	RunE:    runJob,
}

var (
	kubeconfig string
	template   string
	namespace  string
	command    string
	cleanup    bool
	timeout    time.Duration
)

func init() {
	if home := homeDir(); home != "" {
		runJobCmd.Flags().StringVarP(&kubeconfig, "kubeconfig", "", filepath.Join(home, ".kube", "config"), "path to the kubeconfig file")
	} else {
		runJobCmd.Flags().StringVarP(&kubeconfig, "kubeconfig", "", "", "absolute path to the kubeconfig file")
	}
	runJobCmd.Flags().StringVarP(&template, "template", "t", "", "CronJob name used as template")
	runJobCmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "namespace of the CronJob used as template")
	runJobCmd.Flags().StringVarP(&command, "command", "c", "", "override container command")
	runJobCmd.Flags().BoolVarP(&cleanup, "cleanup", "", true, "delete job and pods after completion")
	runJobCmd.Flags().DurationVarP(&timeout, "timeout", "", time.Minute, "timeout for Kubernetes operations")
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

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatalf("Error building kubeconfig: %v", err)
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error building kubernetes client: %v", err)
	}

	informers := job.StartInformers(client, namespace, stopCh)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	logs, err := job.Run(ctx, client, informers, template, namespace, command, cleanup)
	if logs != "" {
		log.Print(logs)
	}
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	return nil
}

func setupSignalHandler() <-chan struct{} {
	stop := make(chan struct{})
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		close(stop)
		os.Exit(1)
	}()

	return stop
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
