package autoscaler

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildPlanUsesExistingNodeCapacityBeforeScaling(t *testing.T) {
	now := time.Now()
	template := Resources{MilliCPU: 2000, Memory: 8 * 1024 * 1024 * 1024, Pods: 10}
	nodes := []corev1.Node{
		node("node-1", map[string]string{"role": "worker"}, Resources{MilliCPU: 2000, Memory: 8 * 1024 * 1024 * 1024, Pods: 10}),
	}
	pods := []corev1.Pod{
		runningPod("used", "node-1", Resources{MilliCPU: 500, Memory: 1024 * 1024 * 1024, Pods: 1}),
		unschedulablePod("pending-a", now, Resources{MilliCPU: 500, Memory: 1024 * 1024 * 1024, Pods: 1}),
		unschedulablePod("pending-b", now, Resources{MilliCPU: 500, Memory: 1024 * 1024 * 1024, Pods: 1}),
	}

	plan := BuildPlan(now, nodes, pods, 1, 0, 10, map[string]string{"role": "worker"}, template, time.Minute)

	if plan.NewNodes != 0 {
		t.Fatalf("NewNodes = %d, want 0", plan.NewNodes)
	}
	if plan.Reason != "no_scale_up_needed" {
		t.Fatalf("Reason = %q, want no_scale_up_needed", plan.Reason)
	}
}

func TestBuildPlanScalesForUnschedulablePods(t *testing.T) {
	now := time.Now()
	template := Resources{MilliCPU: 2000, Memory: 8 * 1024 * 1024 * 1024, Pods: 10}
	pods := []corev1.Pod{
		unschedulablePod("pending-a", now, Resources{MilliCPU: 1500, Memory: 1024 * 1024 * 1024, Pods: 1}),
		unschedulablePod("pending-b", now, Resources{MilliCPU: 1500, Memory: 1024 * 1024 * 1024, Pods: 1}),
	}

	plan := BuildPlan(now, nil, pods, 2, 0, 10, nil, template, time.Minute)

	if plan.NewNodes != 2 {
		t.Fatalf("NewNodes = %d, want 2", plan.NewNodes)
	}
	if plan.DesiredTarget != 4 {
		t.Fatalf("DesiredTarget = %d, want 4", plan.DesiredTarget)
	}
}

func TestBuildPlanCapsAtMaxSize(t *testing.T) {
	now := time.Now()
	template := Resources{MilliCPU: 1000, Memory: 1024 * 1024 * 1024, Pods: 10}
	pods := []corev1.Pod{
		unschedulablePod("pending-a", now, Resources{MilliCPU: 1000, Memory: 128 * 1024 * 1024, Pods: 1}),
		unschedulablePod("pending-b", now, Resources{MilliCPU: 1000, Memory: 128 * 1024 * 1024, Pods: 1}),
	}

	plan := BuildPlan(now, nil, pods, 2, 0, 3, nil, template, time.Minute)

	if plan.NewNodes != 1 {
		t.Fatalf("NewNodes = %d, want 1", plan.NewNodes)
	}
	if plan.DesiredTarget != 3 {
		t.Fatalf("DesiredTarget = %d, want 3", plan.DesiredTarget)
	}
}

func TestBuildPlanReportsScaleDownPotential(t *testing.T) {
	now := time.Now()
	template := Resources{MilliCPU: 2000, Memory: 8 * 1024 * 1024 * 1024, Pods: 10}
	nodes := []corev1.Node{
		node("node-1", map[string]string{"role": "worker"}, template),
		node("node-2", map[string]string{"role": "worker"}, template),
		node("node-3", map[string]string{"role": "worker"}, template),
	}
	pods := []corev1.Pod{
		runningPod("used-a", "node-1", Resources{MilliCPU: 500, Memory: 1024 * 1024 * 1024, Pods: 1}),
		runningPod("used-b", "node-2", Resources{MilliCPU: 500, Memory: 1024 * 1024 * 1024, Pods: 1}),
	}

	plan := BuildPlan(now, nodes, pods, 3, 1, 10, map[string]string{"role": "worker"}, template, time.Minute)

	if plan.ScaleDownPotentialNodes != 2 {
		t.Fatalf("ScaleDownPotentialNodes = %d, want 2", plan.ScaleDownPotentialNodes)
	}
}

func TestBuildPlanDoesNotReportScaleDownPotentialWithPendingPods(t *testing.T) {
	now := time.Now()
	template := Resources{MilliCPU: 2000, Memory: 8 * 1024 * 1024 * 1024, Pods: 10}
	nodes := []corev1.Node{
		node("node-1", map[string]string{"role": "worker"}, template),
		node("node-2", map[string]string{"role": "worker"}, template),
	}
	pods := []corev1.Pod{
		runningPod("used-a", "node-1", Resources{MilliCPU: 500, Memory: 1024 * 1024 * 1024, Pods: 1}),
		unschedulablePod("pending", now, Resources{MilliCPU: 500, Memory: 1024 * 1024 * 1024, Pods: 1}),
	}

	plan := BuildPlan(now, nodes, pods, 2, 0, 10, map[string]string{"role": "worker"}, template, time.Minute)

	if plan.ScaleDownPotentialNodes != 0 {
		t.Fatalf("ScaleDownPotentialNodes = %d, want 0", plan.ScaleDownPotentialNodes)
	}
}

func TestBuildPlanDoesNotReportScaleDownPotentialWhenRunningPodDoesNotFitTemplate(t *testing.T) {
	now := time.Now()
	template := Resources{MilliCPU: 1000, Memory: 1024 * 1024 * 1024, Pods: 10}
	nodes := []corev1.Node{
		node("node-1", map[string]string{"role": "worker"}, Resources{MilliCPU: 4000, Memory: 8 * 1024 * 1024 * 1024, Pods: 10}),
		node("node-2", map[string]string{"role": "worker"}, Resources{MilliCPU: 4000, Memory: 8 * 1024 * 1024 * 1024, Pods: 10}),
	}
	pods := []corev1.Pod{
		runningPod("large", "node-1", Resources{MilliCPU: 2000, Memory: 1024 * 1024 * 1024, Pods: 1}),
	}

	plan := BuildPlan(now, nodes, pods, 2, 0, 10, map[string]string{"role": "worker"}, template, time.Minute)

	if plan.ScaleDownPotentialNodes != 0 {
		t.Fatalf("ScaleDownPotentialNodes = %d, want 0", plan.ScaleDownPotentialNodes)
	}
}

func TestFilterPendingPodsHonorsMinAge(t *testing.T) {
	now := time.Now()
	pods := []corev1.Pod{
		pendingPod("new", now.Add(-10*time.Second), Resources{MilliCPU: 100, Memory: 1, Pods: 1}),
		pendingPod("old", now.Add(-2*time.Minute), Resources{MilliCPU: 100, Memory: 1, Pods: 1}),
		unschedulablePod("unschedulable", now, Resources{MilliCPU: 100, Memory: 1, Pods: 1}),
	}

	filtered := FilterPendingPods(now, pods, time.Minute)

	if len(filtered) != 2 {
		t.Fatalf("len(filtered) = %d, want 2", len(filtered))
	}
}

func node(name string, labels map[string]string, allocatable Resources) corev1.Node {
	return corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: name, Labels: labels},
		Status: corev1.NodeStatus{
			Allocatable: resourceList(allocatable),
			Conditions: []corev1.NodeCondition{{
				Type:   corev1.NodeReady,
				Status: corev1.ConditionTrue,
			}},
		},
	}
}

func runningPod(name string, nodeName string, requests Resources) corev1.Pod {
	pod := pendingPod(name, time.Now(), requests)
	pod.Spec.NodeName = nodeName
	pod.Status.Phase = corev1.PodRunning
	return pod
}

func pendingPod(name string, created time.Time, requests Resources) corev1.Pod {
	return corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         "default",
			Name:              name,
			CreationTimestamp: metav1.NewTime(created),
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name: "app",
				Resources: corev1.ResourceRequirements{
					Requests: resourceList(requests),
				},
			}},
		},
		Status: corev1.PodStatus{Phase: corev1.PodPending},
	}
}

func unschedulablePod(name string, created time.Time, requests Resources) corev1.Pod {
	pod := pendingPod(name, created, requests)
	pod.Status.Conditions = []corev1.PodCondition{{
		Type:   corev1.PodScheduled,
		Status: corev1.ConditionFalse,
		Reason: corev1.PodReasonUnschedulable,
	}}
	return pod
}

func resourceList(resources Resources) corev1.ResourceList {
	return corev1.ResourceList{
		corev1.ResourceCPU:    *resource.NewMilliQuantity(resources.MilliCPU, resource.DecimalSI),
		corev1.ResourceMemory: *resource.NewQuantity(resources.Memory, resource.BinarySI),
		corev1.ResourcePods:   *resource.NewQuantity(resources.Pods, resource.DecimalSI),
	}
}
