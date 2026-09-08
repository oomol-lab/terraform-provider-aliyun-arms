resource "arms_prometheus_alert_rule" "pod_restart" {
  name             = "example-pod-restarts"
  cluster_id       = "c00000000000000000000000000000000"
  promql           = "increase(kube_pod_container_status_restarts_total[5m]) > 3"
  duration_minutes = 5
  level            = "P2"
  message          = "Pod {{$labels.namespace}}/{{$labels.pod}} is restarting frequently; current value: {{$value}}"
  status           = "RUNNING"

  labels = {
    severity = "warning"
  }

  annotations = {
    summary = "Pod restart rate is elevated"
  }
}
