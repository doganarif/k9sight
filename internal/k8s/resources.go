package k8s

import (
	"context"
	"fmt"
	"sort"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

type ResourceType string

const (
	ResourcePods         ResourceType = "pods"
	ResourceDeployments  ResourceType = "deployments"
	ResourceStatefulSets ResourceType = "statefulsets"
	ResourceDaemonSets   ResourceType = "daemonsets"
	ResourceJobs         ResourceType = "jobs"
	ResourceCronJobs     ResourceType = "cronjobs"
)

var AllResourceTypes = []ResourceType{
	ResourceDeployments,
	ResourceStatefulSets,
	ResourceDaemonSets,
	ResourceJobs,
	ResourceCronJobs,
	ResourcePods,
}

type WorkloadInfo struct {
	Name         string
	Namespace    string
	Type         ResourceType
	Ready        string
	Replicas     int32
	Age          string
	Status       string
	Labels       map[string]string
	RestartCount int32
}

type PodInfo struct {
	Name         string
	Namespace    string
	Node         string
	Status       string
	Ready        string
	Restarts     int32
	Age          string
	IP           string
	Labels       map[string]string
	Containers   []ContainerInfo
	Conditions   []corev1.PodCondition
	Phase        corev1.PodPhase
	OwnerRef     string
	OwnerKind    string
}

type ContainerInfo struct {
	Name         string
	Image        string
	Ready        bool
	RestartCount int32
	State        string
	Reason       string
	Resources    ResourceRequirements
	Ports        []int32
}

type ResourceRequirements struct {
	CPURequest    string
	CPULimit      string
	MemoryRequest string
	MemoryLimit   string
}

func ListNamespaces(ctx context.Context, clientset *kubernetes.Clientset) ([]string, error) {
	nsList, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var namespaces []string
	for _, ns := range nsList.Items {
		namespaces = append(namespaces, ns.Name)
	}
	sort.Strings(namespaces)
	return namespaces, nil
}

func ListWorkloads(ctx context.Context, clientset *kubernetes.Clientset, namespace string, resourceType ResourceType) ([]WorkloadInfo, error) {
	switch resourceType {
	case ResourceDeployments:
		return listDeployments(ctx, clientset, namespace)
	case ResourceStatefulSets:
		return listStatefulSets(ctx, clientset, namespace)
	case ResourceDaemonSets:
		return listDaemonSets(ctx, clientset, namespace)
	case ResourceJobs:
		return listJobs(ctx, clientset, namespace)
	case ResourceCronJobs:
		return listCronJobs(ctx, clientset, namespace)
	case ResourcePods:
		return listPodsAsWorkloads(ctx, clientset, namespace)
	default:
		return nil, fmt.Errorf("unknown resource type: %s", resourceType)
	}
}

func listDeployments(ctx context.Context, clientset *kubernetes.Clientset, namespace string) ([]WorkloadInfo, error) {
	deps, err := clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var workloads []WorkloadInfo
	for _, d := range deps.Items {
		status := "Running"
		if d.Status.ReadyReplicas < d.Status.Replicas {
			status = "Progressing"
		}
		if d.Status.ReadyReplicas == 0 && d.Status.Replicas > 0 {
			status = "NotReady"
		}

		workloads = append(workloads, WorkloadInfo{
			Name:      d.Name,
			Namespace: d.Namespace,
			Type:      ResourceDeployments,
			Ready:     fmt.Sprintf("%d/%d", d.Status.ReadyReplicas, d.Status.Replicas),
			Replicas:  d.Status.Replicas,
			Age:       formatAge(d.CreationTimestamp.Time),
			Status:    status,
			Labels:    d.Spec.Selector.MatchLabels,
		})
	}
	return workloads, nil
}

func listStatefulSets(ctx context.Context, clientset *kubernetes.Clientset, namespace string) ([]WorkloadInfo, error) {
	sts, err := clientset.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var workloads []WorkloadInfo
	for _, s := range sts.Items {
		status := "Running"
		if s.Status.ReadyReplicas < s.Status.Replicas {
			status = "Progressing"
		}

		workloads = append(workloads, WorkloadInfo{
			Name:      s.Name,
			Namespace: s.Namespace,
			Type:      ResourceStatefulSets,
			Ready:     fmt.Sprintf("%d/%d", s.Status.ReadyReplicas, s.Status.Replicas),
			Replicas:  s.Status.Replicas,
			Age:       formatAge(s.CreationTimestamp.Time),
			Status:    status,
			Labels:    s.Spec.Selector.MatchLabels,
		})
	}
	return workloads, nil
}

func listDaemonSets(ctx context.Context, clientset *kubernetes.Clientset, namespace string) ([]WorkloadInfo, error) {
	ds, err := clientset.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var workloads []WorkloadInfo
	for _, d := range ds.Items {
		status := "Running"
		if d.Status.NumberReady < d.Status.DesiredNumberScheduled {
			status = "Progressing"
		}

		workloads = append(workloads, WorkloadInfo{
			Name:      d.Name,
			Namespace: d.Namespace,
			Type:      ResourceDaemonSets,
			Ready:     fmt.Sprintf("%d/%d", d.Status.NumberReady, d.Status.DesiredNumberScheduled),
			Replicas:  d.Status.DesiredNumberScheduled,
			Age:       formatAge(d.CreationTimestamp.Time),
			Status:    status,
			Labels:    d.Spec.Selector.MatchLabels,
		})
	}
	return workloads, nil
}

func listJobs(ctx context.Context, clientset *kubernetes.Clientset, namespace string) ([]WorkloadInfo, error) {
	jobs, err := clientset.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var workloads []WorkloadInfo
	for _, j := range jobs.Items {
		status := "Running"
		if j.Status.Succeeded > 0 {
			status = "Completed"
		} else if j.Status.Failed > 0 {
			status = "Failed"
		}

		workloads = append(workloads, WorkloadInfo{
			Name:      j.Name,
			Namespace: j.Namespace,
			Type:      ResourceJobs,
			Ready:     fmt.Sprintf("%d/%d", j.Status.Succeeded, *j.Spec.Completions),
			Age:       formatAge(j.CreationTimestamp.Time),
			Status:    status,
			Labels:    j.Spec.Selector.MatchLabels,
		})
	}
	return workloads, nil
}

func listCronJobs(ctx context.Context, clientset *kubernetes.Clientset, namespace string) ([]WorkloadInfo, error) {
	cjs, err := clientset.BatchV1().CronJobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var workloads []WorkloadInfo
	for _, cj := range cjs.Items {
		status := "Active"
		if cj.Spec.Suspend != nil && *cj.Spec.Suspend {
			status = "Suspended"
		}

		workloads = append(workloads, WorkloadInfo{
			Name:      cj.Name,
			Namespace: cj.Namespace,
			Type:      ResourceCronJobs,
			Ready:     fmt.Sprintf("%d active", len(cj.Status.Active)),
			Age:       formatAge(cj.CreationTimestamp.Time),
			Status:    status,
		})
	}
	return workloads, nil
}

func listPodsAsWorkloads(ctx context.Context, clientset *kubernetes.Clientset, namespace string) ([]WorkloadInfo, error) {
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var workloads []WorkloadInfo
	for _, p := range pods.Items {
		var restartCount int32
		for _, cs := range p.Status.ContainerStatuses {
			restartCount += cs.RestartCount
		}

		ready := 0
		for _, cs := range p.Status.ContainerStatuses {
			if cs.Ready {
				ready++
			}
		}

		workloads = append(workloads, WorkloadInfo{
			Name:         p.Name,
			Namespace:    p.Namespace,
			Type:         ResourcePods,
			Ready:        fmt.Sprintf("%d/%d", ready, len(p.Spec.Containers)),
			Age:          formatAge(p.CreationTimestamp.Time),
			Status:       string(p.Status.Phase),
			Labels:       p.Labels,
			RestartCount: restartCount,
		})
	}
	return workloads, nil
}

func GetWorkloadPods(ctx context.Context, clientset *kubernetes.Clientset, workload WorkloadInfo) ([]PodInfo, error) {
	if workload.Type == ResourcePods {
		pod, err := clientset.CoreV1().Pods(workload.Namespace).Get(ctx, workload.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return []PodInfo{podToPodInfo(pod)}, nil
	}

	labelSelector := labels.SelectorFromSet(workload.Labels).String()
	pods, err := clientset.CoreV1().Pods(workload.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, err
	}

	var podInfos []PodInfo
	for _, p := range pods.Items {
		podInfos = append(podInfos, podToPodInfo(&p))
	}
	return podInfos, nil
}

func GetPod(ctx context.Context, clientset *kubernetes.Clientset, namespace, name string) (*PodInfo, error) {
	pod, err := clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	info := podToPodInfo(pod)
	return &info, nil
}

func podToPodInfo(p *corev1.Pod) PodInfo {
	var restarts int32
	var containers []ContainerInfo

	for i, c := range p.Spec.Containers {
		ci := ContainerInfo{
			Name:  c.Name,
			Image: c.Image,
			Resources: ResourceRequirements{
				CPURequest:    c.Resources.Requests.Cpu().String(),
				CPULimit:      c.Resources.Limits.Cpu().String(),
				MemoryRequest: c.Resources.Requests.Memory().String(),
				MemoryLimit:   c.Resources.Limits.Memory().String(),
			},
		}

		for _, port := range c.Ports {
			ci.Ports = append(ci.Ports, port.ContainerPort)
		}

		if i < len(p.Status.ContainerStatuses) {
			cs := p.Status.ContainerStatuses[i]
			ci.Ready = cs.Ready
			ci.RestartCount = cs.RestartCount
			restarts += cs.RestartCount

			if cs.State.Running != nil {
				ci.State = "Running"
			} else if cs.State.Waiting != nil {
				ci.State = "Waiting"
				ci.Reason = cs.State.Waiting.Reason
			} else if cs.State.Terminated != nil {
				ci.State = "Terminated"
				ci.Reason = cs.State.Terminated.Reason
			}
		}

		containers = append(containers, ci)
	}

	ready := 0
	for _, cs := range p.Status.ContainerStatuses {
		if cs.Ready {
			ready++
		}
	}

	var ownerRef, ownerKind string
	if len(p.OwnerReferences) > 0 {
		ownerRef = p.OwnerReferences[0].Name
		ownerKind = p.OwnerReferences[0].Kind
	}

	return PodInfo{
		Name:       p.Name,
		Namespace:  p.Namespace,
		Node:       p.Spec.NodeName,
		Status:     getPodStatus(p),
		Ready:      fmt.Sprintf("%d/%d", ready, len(p.Spec.Containers)),
		Restarts:   restarts,
		Age:        formatAge(p.CreationTimestamp.Time),
		IP:         p.Status.PodIP,
		Labels:     p.Labels,
		Containers: containers,
		Conditions: p.Status.Conditions,
		Phase:      p.Status.Phase,
		OwnerRef:   ownerRef,
		OwnerKind:  ownerKind,
	}
}

func getPodStatus(p *corev1.Pod) string {
	if p.DeletionTimestamp != nil {
		return "Terminating"
	}

	for _, cs := range p.Status.ContainerStatuses {
		if cs.State.Waiting != nil {
			if cs.State.Waiting.Reason != "" {
				return cs.State.Waiting.Reason
			}
		}
		if cs.State.Terminated != nil && cs.State.Terminated.Reason != "" {
			return cs.State.Terminated.Reason
		}
	}

	return string(p.Status.Phase)
}

type RelatedResources struct {
	Services   []ServiceInfo
	Ingresses  []IngressInfo
	ConfigMaps []string
	Secrets    []string
	Owner      *OwnerInfo
}

type ServiceInfo struct {
	Name      string
	Type      string
	ClusterIP string
	Ports     string
	Endpoints int
}

type IngressInfo struct {
	Name  string
	Hosts string
	Paths string
}

type OwnerInfo struct {
	Kind string
	Name string
}

func GetRelatedResources(ctx context.Context, clientset *kubernetes.Clientset, pod PodInfo) (*RelatedResources, error) {
	related := &RelatedResources{}

	if pod.OwnerRef != "" {
		related.Owner = &OwnerInfo{
			Kind: pod.OwnerKind,
			Name: pod.OwnerRef,
		}
	}

	svcs, err := clientset.CoreV1().Services(pod.Namespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, svc := range svcs.Items {
			if svc.Spec.Selector == nil {
				continue
			}
			if labelsMatch(svc.Spec.Selector, pod.Labels) {
				var ports []string
				for _, p := range svc.Spec.Ports {
					ports = append(ports, fmt.Sprintf("%d/%s", p.Port, p.Protocol))
				}

				eps, _ := clientset.CoreV1().Endpoints(pod.Namespace).Get(ctx, svc.Name, metav1.GetOptions{})
				endpointCount := 0
				if eps != nil {
					for _, subset := range eps.Subsets {
						endpointCount += len(subset.Addresses)
					}
				}

				related.Services = append(related.Services, ServiceInfo{
					Name:      svc.Name,
					Type:      string(svc.Spec.Type),
					ClusterIP: svc.Spec.ClusterIP,
					Ports:     joinStrings(ports, ", "),
					Endpoints: endpointCount,
				})
			}
		}
	}

	ings, err := clientset.NetworkingV1().Ingresses(pod.Namespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, svc := range related.Services {
			for _, ing := range ings.Items {
				if ingressReferencesService(ing, svc.Name) {
					var hosts, paths []string
					for _, rule := range ing.Spec.Rules {
						hosts = append(hosts, rule.Host)
						if rule.HTTP != nil {
							for _, p := range rule.HTTP.Paths {
								paths = append(paths, p.Path)
							}
						}
					}
					related.Ingresses = append(related.Ingresses, IngressInfo{
						Name:  ing.Name,
						Hosts: joinStrings(hosts, ", "),
						Paths: joinStrings(paths, ", "),
					})
				}
			}
		}
	}

	podObj, err := clientset.CoreV1().Pods(pod.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
	if err == nil {
		for _, vol := range podObj.Spec.Volumes {
			if vol.ConfigMap != nil {
				related.ConfigMaps = append(related.ConfigMaps, vol.ConfigMap.Name)
			}
			if vol.Secret != nil {
				related.Secrets = append(related.Secrets, vol.Secret.SecretName)
			}
		}
		for _, c := range podObj.Spec.Containers {
			for _, env := range c.EnvFrom {
				if env.ConfigMapRef != nil {
					related.ConfigMaps = append(related.ConfigMaps, env.ConfigMapRef.Name)
				}
				if env.SecretRef != nil {
					related.Secrets = append(related.Secrets, env.SecretRef.Name)
				}
			}
		}
	}

	return related, nil
}

func labelsMatch(selector, labels map[string]string) bool {
	for k, v := range selector {
		if labels[k] != v {
			return false
		}
	}
	return true
}

func ingressReferencesService(ing networkingv1.Ingress, svcName string) bool {
	for _, rule := range ing.Spec.Rules {
		if rule.HTTP == nil {
			continue
		}
		for _, path := range rule.HTTP.Paths {
			if path.Backend.Service != nil && path.Backend.Service.Name == svcName {
				return true
			}
		}
	}
	return false
}

func joinStrings(s []string, sep string) string {
	if len(s) == 0 {
		return ""
	}
	result := s[0]
	for i := 1; i < len(s); i++ {
		result += sep + s[i]
	}
	return result
}

func GetDeployment(ctx context.Context, clientset *kubernetes.Clientset, namespace, name string) (*appsv1.Deployment, error) {
	return clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
}

func GetStatefulSet(ctx context.Context, clientset *kubernetes.Clientset, namespace, name string) (*appsv1.StatefulSet, error) {
	return clientset.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
}

func GetDaemonSet(ctx context.Context, clientset *kubernetes.Clientset, namespace, name string) (*appsv1.DaemonSet, error) {
	return clientset.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
}

func GetJob(ctx context.Context, clientset *kubernetes.Clientset, namespace, name string) (*batchv1.Job, error) {
	return clientset.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
}

func DeletePod(ctx context.Context, clientset *kubernetes.Clientset, namespace, name string) error {
	return clientset.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

func ScaleDeployment(ctx context.Context, clientset *kubernetes.Clientset, namespace, name string, replicas int32) error {
	scale, err := clientset.AppsV1().Deployments(namespace).GetScale(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	scale.Spec.Replicas = replicas
	_, err = clientset.AppsV1().Deployments(namespace).UpdateScale(ctx, name, scale, metav1.UpdateOptions{})
	return err
}

func ScaleStatefulSet(ctx context.Context, clientset *kubernetes.Clientset, namespace, name string, replicas int32) error {
	scale, err := clientset.AppsV1().StatefulSets(namespace).GetScale(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	scale.Spec.Replicas = replicas
	_, err = clientset.AppsV1().StatefulSets(namespace).UpdateScale(ctx, name, scale, metav1.UpdateOptions{})
	return err
}

func RestartDeployment(ctx context.Context, clientset *kubernetes.Clientset, namespace, name string) error {
	deploy, err := clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if deploy.Spec.Template.Annotations == nil {
		deploy.Spec.Template.Annotations = make(map[string]string)
	}
	deploy.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = metav1.Now().Format("2006-01-02T15:04:05Z07:00")

	_, err = clientset.AppsV1().Deployments(namespace).Update(ctx, deploy, metav1.UpdateOptions{})
	return err
}

func RestartStatefulSet(ctx context.Context, clientset *kubernetes.Clientset, namespace, name string) error {
	sts, err := clientset.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if sts.Spec.Template.Annotations == nil {
		sts.Spec.Template.Annotations = make(map[string]string)
	}
	sts.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = metav1.Now().Format("2006-01-02T15:04:05Z07:00")

	_, err = clientset.AppsV1().StatefulSets(namespace).Update(ctx, sts, metav1.UpdateOptions{})
	return err
}

func RestartDaemonSet(ctx context.Context, clientset *kubernetes.Clientset, namespace, name string) error {
	ds, err := clientset.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if ds.Spec.Template.Annotations == nil {
		ds.Spec.Template.Annotations = make(map[string]string)
	}
	ds.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = metav1.Now().Format("2006-01-02T15:04:05Z07:00")

	_, err = clientset.AppsV1().DaemonSets(namespace).Update(ctx, ds, metav1.UpdateOptions{})
	return err
}
