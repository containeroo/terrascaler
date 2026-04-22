package autoscaler

import (
	"sort"
	"time"

	corev1 "k8s.io/api/core/v1"
)

type Plan struct {
	CurrentTarget           int
	DesiredTarget           int
	NewNodes                int
	PendingPods             int
	UnscheduledPods         int
	UnschedulablePods       int
	MatchingReadyNodes      int
	ScaleDownPotentialNodes int
	UnfitPods               []string
	Reason                  string
}

func BuildPlan(
	now time.Time,
	nodes []corev1.Node,
	pods []corev1.Pod,
	currentTarget int,
	minSize int,
	maxSize int,
	nodeSelector map[string]string,
	template Resources,
	pendingPodMinAge time.Duration,
) Plan {
	pending := FilterPendingPods(now, pods, pendingPodMinAge)
	nodeSlots := BuildNodeSlots(nodes, pods, nodeSelector)
	newNodes, unfitPods := RequiredNewNodes(nodeSlots, pending, template)
	matchingReadyNodes := CountMatchingReadyNodes(nodes, nodeSelector)
	scaleDownPotential := ScaleDownPotential(nodes, pods, currentTarget, minSize, nodeSelector, template, len(pending))

	desired := currentTarget
	if desired < minSize {
		desired = minSize
	}
	if newNodes > 0 {
		desired = currentTarget + newNodes
	}
	if desired > maxSize {
		desired = maxSize
	}

	reason := "no_scale_up_needed"
	if currentTarget < minSize {
		reason = "min_size_enforced"
	} else if len(pending) == 0 {
		reason = "no_unschedulable_pods"
	} else if newNodes > 0 && desired > currentTarget {
		reason = "scale_up_needed"
	} else if newNodes > 0 && desired == currentTarget {
		reason = "max_size_reached"
	}

	return Plan{
		CurrentTarget:           currentTarget,
		DesiredTarget:           desired,
		NewNodes:                maxInt(0, desired-currentTarget),
		PendingPods:             countPendingUnscheduled(pods),
		UnscheduledPods:         len(pending),
		UnschedulablePods:       countUnschedulable(pending),
		MatchingReadyNodes:      matchingReadyNodes,
		ScaleDownPotentialNodes: scaleDownPotential,
		UnfitPods:               unfitPods,
		Reason:                  reason,
	}
}

func FilterPendingPods(now time.Time, pods []corev1.Pod, pendingPodMinAge time.Duration) []corev1.Pod {
	out := make([]corev1.Pod, 0)
	for _, pod := range pods {
		if pod.DeletionTimestamp != nil || pod.Spec.NodeName != "" || pod.Status.Phase != corev1.PodPending {
			continue
		}
		if isDaemonSetPod(pod) {
			continue
		}
		if IsUnschedulable(pod) || now.Sub(pod.CreationTimestamp.Time) >= pendingPodMinAge {
			out = append(out, pod)
		}
	}
	return out
}

func IsUnschedulable(pod corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodScheduled &&
			condition.Status == corev1.ConditionFalse &&
			condition.Reason == corev1.PodReasonUnschedulable {
			return true
		}
	}
	return false
}

func BuildNodeSlots(nodes []corev1.Node, pods []corev1.Pod, selector map[string]string) []Resources {
	podsByNode := map[string][]corev1.Pod{}
	for _, pod := range pods {
		if pod.Spec.NodeName == "" || pod.DeletionTimestamp != nil || isTerminalPod(pod) {
			continue
		}
		podsByNode[pod.Spec.NodeName] = append(podsByNode[pod.Spec.NodeName], pod)
	}

	slots := make([]Resources, 0, len(nodes))
	for _, node := range nodes {
		if !nodeMatches(node, selector) || !nodeReady(node) || node.Spec.Unschedulable {
			continue
		}

		used := Resources{}
		for _, pod := range podsByNode[node.Name] {
			used = used.Add(PodRequests(pod))
		}
		slots = append(slots, NodeAllocatable(node).Sub(used).Positive())
	}
	return slots
}

func RequiredNewNodes(existingSlots []Resources, pendingPods []corev1.Pod, template Resources) (int, []string) {
	slots := append([]Resources(nil), existingSlots...)
	requests := make([]podRequest, 0, len(pendingPods))
	for _, pod := range pendingPods {
		requests = append(requests, podRequest{
			name:      pod.Namespace + "/" + pod.Name,
			resources: PodRequests(pod),
		})
	}
	sort.SliceStable(requests, func(i int, j int) bool {
		left := requests[i].resources
		right := requests[j].resources
		if left.MilliCPU != right.MilliCPU {
			return left.MilliCPU > right.MilliCPU
		}
		if left.Memory != right.Memory {
			return left.Memory > right.Memory
		}
		return left.Pods > right.Pods
	})

	newNodes := 0
	unfitPods := make([]string, 0)
	for _, request := range requests {
		if !template.Fits(request.resources) {
			unfitPods = append(unfitPods, request.name)
			continue
		}

		if placePod(slots, request.resources) {
			continue
		}

		slots = append(slots, template.Sub(request.resources).Positive())
		newNodes++
	}

	return newNodes, unfitPods
}

func ScaleDownPotential(
	nodes []corev1.Node,
	pods []corev1.Pod,
	currentTarget int,
	minSize int,
	selector map[string]string,
	template Resources,
	eligiblePendingPods int,
) int {
	if eligiblePendingPods > 0 {
		return 0
	}

	matchingReadyNodes := CountMatchingReadyNodes(nodes, selector)
	if matchingReadyNodes < currentTarget {
		return 0
	}

	requiredNodes, allPodsFitTemplate := RequiredNodesForPods(ScheduledPodsOnMatchingNodes(nodes, pods, selector), template)
	if !allPodsFitTemplate {
		return 0
	}
	minRequired := maxInt(minSize, requiredNodes)
	return maxInt(0, currentTarget-minRequired)
}

func RequiredNodesForPods(pods []corev1.Pod, template Resources) (int, bool) {
	slots := []Resources{}
	for _, pod := range pods {
		request := PodRequests(pod)
		if !template.Fits(request) {
			return 0, false
		}
		if placePod(slots, request) {
			continue
		}
		slots = append(slots, template.Sub(request).Positive())
	}
	return len(slots), true
}

func ScheduledPodsOnMatchingNodes(nodes []corev1.Node, pods []corev1.Pod, selector map[string]string) []corev1.Pod {
	matchingNodes := map[string]struct{}{}
	for _, node := range nodes {
		if nodeMatches(node, selector) && nodeReady(node) {
			matchingNodes[node.Name] = struct{}{}
		}
	}

	out := make([]corev1.Pod, 0)
	for _, pod := range pods {
		if pod.Spec.NodeName == "" || pod.DeletionTimestamp != nil || isTerminalPod(pod) || isDaemonSetPod(pod) {
			continue
		}
		if _, ok := matchingNodes[pod.Spec.NodeName]; ok {
			out = append(out, pod)
		}
	}
	return out
}

func CountMatchingReadyNodes(nodes []corev1.Node, selector map[string]string) int {
	count := 0
	for _, node := range nodes {
		if nodeMatches(node, selector) && nodeReady(node) && !node.Spec.Unschedulable {
			count++
		}
	}
	return count
}

func placePod(slots []Resources, request Resources) bool {
	for i, slot := range slots {
		if slot.Fits(request) {
			slots[i] = slot.Sub(request).Positive()
			return true
		}
	}
	return false
}

func nodeMatches(node corev1.Node, selector map[string]string) bool {
	for key, expected := range selector {
		if node.Labels[key] != expected {
			return false
		}
	}
	return true
}

func nodeReady(node corev1.Node) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

func isDaemonSetPod(pod corev1.Pod) bool {
	for _, owner := range pod.OwnerReferences {
		if owner.Kind == "DaemonSet" {
			return true
		}
	}
	return false
}

func isTerminalPod(pod corev1.Pod) bool {
	return pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed
}

func countPendingUnscheduled(pods []corev1.Pod) int {
	count := 0
	for _, pod := range pods {
		if pod.Status.Phase == corev1.PodPending && pod.Spec.NodeName == "" && pod.DeletionTimestamp == nil {
			count++
		}
	}
	return count
}

func countUnschedulable(pods []corev1.Pod) int {
	count := 0
	for _, pod := range pods {
		if IsUnschedulable(pod) {
			count++
		}
	}
	return count
}

func maxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}

type podRequest struct {
	name      string
	resources Resources
}
