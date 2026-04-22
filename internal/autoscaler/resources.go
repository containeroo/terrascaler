package autoscaler

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

type Resources struct {
	MilliCPU int64
	Memory   int64
	Pods     int64
}

func TemplateResources(cpu string, memory string, pods int64) (Resources, error) {
	cpuQuantity, err := resource.ParseQuantity(cpu)
	if err != nil {
		return Resources{}, err
	}
	memoryQuantity, err := resource.ParseQuantity(memory)
	if err != nil {
		return Resources{}, err
	}
	return Resources{
		MilliCPU: cpuQuantity.MilliValue(),
		Memory:   memoryQuantity.Value(),
		Pods:     pods,
	}, nil
}

func PodRequests(pod corev1.Pod) Resources {
	appRequests := Resources{Pods: 1}
	for _, container := range pod.Spec.Containers {
		appRequests = appRequests.Add(containerRequests(container))
	}

	initRequests := Resources{}
	for _, container := range pod.Spec.InitContainers {
		initRequests = initRequests.Max(containerRequests(container))
	}

	requests := appRequests.Max(initRequests)
	if pod.Spec.Overhead != nil {
		requests.MilliCPU += pod.Spec.Overhead.Cpu().MilliValue()
		requests.Memory += pod.Spec.Overhead.Memory().Value()
	}
	return requests
}

func NodeAllocatable(node corev1.Node) Resources {
	return Resources{
		MilliCPU: node.Status.Allocatable.Cpu().MilliValue(),
		Memory:   node.Status.Allocatable.Memory().Value(),
		Pods:     node.Status.Allocatable.Pods().Value(),
	}
}

func containerRequests(container corev1.Container) Resources {
	return Resources{
		MilliCPU: container.Resources.Requests.Cpu().MilliValue(),
		Memory:   container.Resources.Requests.Memory().Value(),
	}
}

func (r Resources) Add(other Resources) Resources {
	return Resources{
		MilliCPU: r.MilliCPU + other.MilliCPU,
		Memory:   r.Memory + other.Memory,
		Pods:     r.Pods + other.Pods,
	}
}

func (r Resources) Sub(other Resources) Resources {
	return Resources{
		MilliCPU: r.MilliCPU - other.MilliCPU,
		Memory:   r.Memory - other.Memory,
		Pods:     r.Pods - other.Pods,
	}
}

func (r Resources) Max(other Resources) Resources {
	return Resources{
		MilliCPU: maxInt64(r.MilliCPU, other.MilliCPU),
		Memory:   maxInt64(r.Memory, other.Memory),
		Pods:     maxInt64(r.Pods, other.Pods),
	}
}

func (r Resources) Fits(request Resources) bool {
	return r.MilliCPU >= request.MilliCPU && r.Memory >= request.Memory && r.Pods >= request.Pods
}

func (r Resources) Positive() Resources {
	return Resources{
		MilliCPU: maxInt64(r.MilliCPU, 0),
		Memory:   maxInt64(r.Memory, 0),
		Pods:     maxInt64(r.Pods, 0),
	}
}

func maxInt64(left int64, right int64) int64 {
	if left > right {
		return left
	}
	return right
}
