package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"sync"
	"time"

	"github.com/buildkite/go-buildkite/buildkite"
	"github.com/ghodss/yaml"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	resource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// KubeJobManager holds all Kubernetes job resources managed by buildkite-job-manager
type KubeJobManager struct {
	jobTemplate []byte
	Client      *kubernetes.Clientset
	Jobs        map[string]*batchv1.Job
	JobsMutex   sync.RWMutex
	namespace   string
	org         string
	pipeline    string
}

// NewKubeJobManager creates a new KubeJobManager object for managing jobs
func NewKubeJobManager(ctx context.Context, wg *sync.WaitGroup, templateFilename string, kubeconfig string, kubeNamespace string, kubeTimeout int, org string, pipeline string) (*KubeJobManager, error) {
	var err error

	k := new(KubeJobManager)

	k.namespace = kubeNamespace
	k.org = org
	k.pipeline = pipeline

	k.Jobs = make(map[string]*batchv1.Job)

	if kubeconfig == "" {
		log.Info("No kubeconfig was provided; using in-cluster config.")
		log.Info("If you're not running kubekite within Kubernetes, please run with -kubeconfig flag or set KUBECONFIG environment variable")
	}

	k.Client, err = NewKubeClientSet(kubeconfig, kubeTimeout)
	if err != nil {
		return nil, err
	}

	jobTemplate, err := ioutil.ReadFile(templateFilename)
	if err != nil {
		return nil, fmt.Errorf("could not open job template: %v", err)
	}

	// Marshalling a YAML pod template into a PodTemplateSpec is problematic,
	// so we marshall it to JSON and then into the struct.
	k.jobTemplate, err = yaml.YAMLToJSON(jobTemplate)
	if err != nil {
		log.Fatal(err)
	}

	k.StartJobCleaner(ctx, wg)

	return k, nil
}

// StartJobCleaner starts a monitor that watches for completed build jobs and cleans up the dangling Kube job resources
func (k *KubeJobManager) StartJobCleaner(ctx context.Context, wg *sync.WaitGroup) {
	go k.cleanCompletedJobs(ctx, wg)

	log.Info("Kube job cleaner started.")
}

// LaunchJob launches a Kubernetes job for a given Buildkite job ID
func (k *KubeJobManager) LaunchJob(job *buildkite.Job) error {
	var err error
	var t batchv1.Job
	uuid := *job.ID

	jobLabels := make(map[string]string)
	jobLabels["kubekite-managed"] = "true"
	jobLabels["kubekite-org"] = k.org
	jobLabels["kubekite-pipeline"] = k.pipeline

	err = json.Unmarshal(k.jobTemplate, &t)
	if err != nil {
		log.Fatal(err)
	}

	// Set our labels on both the job and the pod that it generates
	t.SetLabels(jobLabels)
	t.Spec.Template.SetLabels(jobLabels)

	container := t.Spec.Template.Spec.Containers[0]

	tags_var := corev1.EnvVar{"BUILDKITE_AGENT_TAGS", strings.Join(job.AgentQueryRules, ","), nil}
	container.Env = append(container.Env, tags_var)
	for _, v := range job.AgentQueryRules {
		split_rule := strings.Split(v, "=")
		if len(split_rule) != 2 {
			continue
		}
		rule_value := split_rule[1]
		switch rule_key := split_rule[0]; rule_key {
		case "image":
			container.Image = rule_value
		case "cpu":
			quantity, err := resource.ParseQuantity(rule_value)
			if err == nil {
				container.Resources.Requests["cpu"] = quantity
			}
		case "memory":
			quantity, err := resource.ParseQuantity(rule_value)
			if err == nil {
				container.Resources.Requests["memory"] = quantity
			}
		default:
		}
	}
	t.Spec.Template.Spec.Containers[0] = container

	t.Name = "buildkite-agent-" + uuid

	runningJob, err := k.Client.BatchV1().Jobs(k.namespace).Get(t.Name, metav1.GetOptions{})
	if err == nil {
		log.Infof("Job %v already exists, not launching.\n", runningJob.Name)
		return nil
	}

	k.JobsMutex.Lock()
	defer k.JobsMutex.Unlock()
	k.Jobs[uuid] = new(batchv1.Job)
	k.Jobs[uuid] = &t

	k.Jobs[uuid], err = k.Client.BatchV1().Jobs(k.namespace).Create(k.Jobs[uuid])
	if err != nil {
		return fmt.Errorf("could not launch job: %v", err)
	}

	log.Infof("Launched job: %v", k.Jobs[uuid].Name)
	return nil
}

func (k *KubeJobManager) cleanCompletedJobs(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	selector := fmt.Sprintf("kubekite-managed=true,kubekite-org=%v,kubekite-pipeline=%v", k.org, k.pipeline)

	for {

		log.Info("Cleaning completed jobs...")

		pods, err := k.Client.CoreV1().Pods(k.namespace).List(metav1.ListOptions{LabelSelector: selector})
		if err != nil {
			log.Errorf("Could not list pods: %v", err)
		}

		for _, pod := range pods.Items {
			for _, container := range pod.Status.ContainerStatuses {
				if container.State.Terminated != nil && container.Name == "buildkite-agent" {
					jobName := pod.Labels["job-name"]
					log.Infof("Deleting job: %v", jobName)

					policy := metav1.DeletePropagationForeground

					err := k.Client.BatchV1().Jobs(k.namespace).Delete(jobName, &metav1.DeleteOptions{PropagationPolicy: &policy})
					if err != nil {
						log.Error("Error deleting job:", err)
					}
				}
			}
		}

		time.Sleep(15 * time.Second)

	}

}
