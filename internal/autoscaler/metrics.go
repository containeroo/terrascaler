package autoscaler

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	currentTarget           prometheus.Gauge
	desiredTarget           prometheus.Gauge
	newNodes                prometheus.Gauge
	pendingPods             prometheus.Gauge
	eligiblePendingPods     prometheus.Gauge
	unschedulablePods       prometheus.Gauge
	matchingReadyNodes      prometheus.Gauge
	scaleDownPotentialNodes prometheus.Gauge
	lastCheckTimestamp      prometheus.Gauge
	lastCheckSuccess        prometheus.Gauge
	scaleUpCommits          prometheus.Counter
}

func NewMetrics(registry prometheus.Registerer) *Metrics {
	if registry == nil {
		registry = prometheus.DefaultRegisterer
	}

	metrics := &Metrics{
		currentTarget: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "terrascaler_current_target_nodes",
			Help: "Current Terraform target node count read by Terrascaler.",
		}),
		desiredTarget: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "terrascaler_desired_target_nodes",
			Help: "Desired target node count computed by Terrascaler.",
		}),
		newNodes: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "terrascaler_new_nodes_required",
			Help: "New nodes required for the latest autoscaling plan.",
		}),
		pendingPods: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "terrascaler_pending_pods",
			Help: "Pending unscheduled pods observed by Terrascaler.",
		}),
		eligiblePendingPods: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "terrascaler_eligible_pending_pods",
			Help: "Pending pods eligible for Terrascaler scale-up simulation.",
		}),
		unschedulablePods: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "terrascaler_unschedulable_pods",
			Help: "Eligible pending pods marked Unschedulable by the Kubernetes scheduler.",
		}),
		matchingReadyNodes: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "terrascaler_matching_ready_nodes",
			Help: "Ready schedulable nodes matching Terrascaler's scaling node selector.",
		}),
		scaleDownPotentialNodes: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "terrascaler_scale_down_potential_nodes",
			Help: "Approximate number of nodes that may be removable. Terrascaler only reports this and does not scale down.",
		}),
		lastCheckTimestamp: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "terrascaler_last_check_timestamp_seconds",
			Help: "Unix timestamp of the last completed autoscaling check.",
		}),
		lastCheckSuccess: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "terrascaler_last_check_success",
			Help: "Whether the last autoscaling check succeeded. 1 means success, 0 means failure.",
		}),
		scaleUpCommits: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "terrascaler_scale_up_commits_total",
			Help: "Total number of GitLab target-size update commits requested by Terrascaler.",
		}),
	}

	registry.MustRegister(
		metrics.currentTarget,
		metrics.desiredTarget,
		metrics.newNodes,
		metrics.pendingPods,
		metrics.eligiblePendingPods,
		metrics.unschedulablePods,
		metrics.matchingReadyNodes,
		metrics.scaleDownPotentialNodes,
		metrics.lastCheckTimestamp,
		metrics.lastCheckSuccess,
		metrics.scaleUpCommits,
	)
	return metrics
}

func (m *Metrics) ObservePlan(plan Plan) {
	if m == nil {
		return
	}
	m.currentTarget.Set(float64(plan.CurrentTarget))
	m.desiredTarget.Set(float64(plan.DesiredTarget))
	m.newNodes.Set(float64(plan.NewNodes))
	m.pendingPods.Set(float64(plan.PendingPods))
	m.eligiblePendingPods.Set(float64(plan.UnscheduledPods))
	m.unschedulablePods.Set(float64(plan.UnschedulablePods))
	m.matchingReadyNodes.Set(float64(plan.MatchingReadyNodes))
	m.scaleDownPotentialNodes.Set(float64(plan.ScaleDownPotentialNodes))
}

func (m *Metrics) ObserveCheck(success bool, at time.Time) {
	if m == nil {
		return
	}
	if success {
		m.lastCheckSuccess.Set(1)
	} else {
		m.lastCheckSuccess.Set(0)
	}
	m.lastCheckTimestamp.Set(float64(at.Unix()))
}

func (m *Metrics) IncScaleUpCommit() {
	if m == nil {
		return
	}
	m.scaleUpCommits.Inc()
}
