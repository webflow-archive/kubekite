package main

import (
	"context"
	"os"
	"sync"

	"github.com/ProjectSigma/kubekite/pkg/buildkite"
	kube "github.com/ProjectSigma/kubekite/pkg/kubernetes"

	"github.com/namsral/flag"
	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("kubekite")

func main() {

	var debug bool

	var bkAPIToken string
	var bkOrg string
	var bkPipeline string

	var kubeconfig string
	var kubeNamespace string
	var jobTemplateYaml string
	var kubeTimeout int

	var format = logging.MustStringFormatter(
		`%{color}%{time:15:04:05.000} %{shortfile} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}`,
	)

	logBackend := logging.NewLogBackend(os.Stderr, "", 0)
	logBackendFormatter := logging.NewBackendFormatter(logBackend, format)
	logging.SetBackend(logBackendFormatter)

	flag.BoolVar(&debug, "debug", false, "Turn on debugging")

	flag.StringVar(&bkAPIToken, "buildkite-api-token", "", "Buildkite API token")
	flag.StringVar(&bkOrg, "buildkite-org", "", "Your buildkite organization")
	flag.StringVar(&bkPipeline, "buildkite-pipeline", "", "Buildkite pipeline to watch for new jobs")

	flag.StringVar(&kubeconfig, "kube-config", "", "Path to your kubeconfig file")
	flag.StringVar(&kubeNamespace, "kube-namespace", "default", "Kubernetes namespace to run jobs in")
	flag.StringVar(&jobTemplateYaml, "job-template", "job.yaml", "Path to your job template YAML file")
	flag.IntVar(&kubeTimeout, "kube-timeout", 15, "Timeout (in seconds) for Kubernetes API requests. Set to 0 for no timeout.  Default: 15")

	flag.Parse()

	if bkAPIToken == "" {
		log.Fatal("Error: must provide API token via -api-token flag or BUILDKITE_API_TOKEN environment variable")
	}

	if bkOrg == "" {
		log.Fatal("Error: must provide a Buildkite organization via -buildkite-org flag or BUILDKITE_ORG environment variable")
	}

	if bkPipeline == "" {
		log.Fatal("Error: must provide a Buildkite pipeline via -buildkite-pipe flag or BUILDKITE_PIPELINE environment variable")
	}

	if jobTemplateYaml == "" {
		log.Fatal("Error: must provide a Kuberenetes job template filename via -job-template flag or JOB_TEMPLATE environment variable")
	}

	sigs := make(chan os.Signal, 1)
	done := make(chan struct{}, 1)

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	wg := new(sync.WaitGroup)

	j, err := kube.NewKubeJobManager(ctx, wg, jobTemplateYaml, kubeconfig, kubeNamespace, kubeTimeout, bkOrg, bkPipeline)
	if err != nil {
		log.Fatal("Error starting job manager:", err)
	}

	bkc, err := buildkite.NewBuildkiteClient(bkAPIToken, debug)
	if err != nil {
		log.Fatal("Error starting Buildkite API client:", err)
	}

	jobChan := buildkite.StartBuildkiteWatcher(ctx, wg, bkc, bkOrg, bkPipeline)

	go func(cancel context.CancelFunc) {
		// If we get a SIGINT or SIGTERM, cancel the context and unblock 'done'
		// to trigger a program shutdown
		<-sigs
		cancel()
		close(done)
	}(cancel)

	for {
		select {
		case job := <-jobChan:
			err := j.LaunchJob(ctx, job)
			if err != nil {
				log.Error("Error launching job:", err)
			}
		case <-ctx.Done():
			log.Notice("Cancellation request recieved. Cancelling job processor.")
			return
		}
	}

}
