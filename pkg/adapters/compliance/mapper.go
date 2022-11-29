package compliance

import (
	"fmt"

	"github.com/aquasecurity/trivy-operator/pkg/apis/aquasecurity/v1alpha1"
	"github.com/kyverno/kyverno/api/policyreport/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Severity = int

const (
	unknown Severity = iota
	low
	medium
	high
	critical
)

const (
	trivySource  = "Trivy Compliance"
	reportPrefix = "trivy-compliance-cpolr"
)

var (
	reportLabels = map[string]string{
		"app.kubernetes.io/created-by": "trivy-operator-polr-adapter",
		"trivy-operator.source":        "ClusterComplianceDetailReport",
	}
)

func Map(report *v1alpha1.ClusterComplianceDetailReport, polr *v1alpha2.ClusterPolicyReport) (*v1alpha2.ClusterPolicyReport, bool) {
	var updated bool

	if polr == nil {
		polr = CreatePolicyReport(report)
	} else {
		polr.Summary = v1alpha2.PolicyReportSummary{}
		polr.Results = []v1alpha2.PolicyReportResult{}
		updated = true
	}

	for _, check := range report.Report.ControlChecks {
		for _, result := range check.ScannerCheckResult {
			for _, details := range result.Details {
				props := map[string]string{
					"Description": check.Description,
				}

				res := []corev1.ObjectReference{}
				if details.Name != "" {
					res = append(res, corev1.ObjectReference{
						Kind:      result.ObjectType,
						Name:      details.Name,
						Namespace: details.Namespace,
					})
				} else {
					props["objectType"] = result.ObjectType
				}

				if result.ID != "" {
					props["id"] = result.ID
				}

				polr.Results = append(polr.Results, v1alpha2.PolicyReportResult{
					Policy:     check.Name,
					Rule:       result.ID,
					Message:    details.Msg,
					Result:     v1alpha2.StatusFail,
					Severity:   MapServerity(check.Severity),
					Timestamp:  *report.Report.UpdateTimestamp.ProtoTime(),
					Source:     trivySource,
					Resources:  res,
					Properties: props,
				})
			}
		}
	}

	return polr, updated
}

func MapServerity(severity v1alpha1.Severity) v1alpha2.PolicySeverity {
	if severity == v1alpha1.SeverityUnknown {
		return ""
	} else if severity == v1alpha1.SeverityLow {
		return v1alpha2.SeverityLow
	} else if severity == v1alpha1.SeverityMedium {
		return v1alpha2.SeverityMedium
	} else if severity == v1alpha1.SeverityHigh {
		return v1alpha2.SeverityHigh
	} else if severity == v1alpha1.SeverityCritical {
		return v1alpha2.SeverityCritical
	}

	return v1alpha2.SeverityInfo
}

func CreatePolicyReport(report *v1alpha1.ClusterComplianceDetailReport) *v1alpha2.ClusterPolicyReport {
	return &v1alpha2.ClusterPolicyReport{
		ObjectMeta: v1.ObjectMeta{
			Name:   GeneratePolicyReportName(report.Name),
			Labels: reportLabels,
			OwnerReferences: []v1.OwnerReference{
				{
					APIVersion: report.APIVersion,
					Kind:       report.Kind,
					Name:       report.Name,
					UID:        report.UID,
				},
			},
		},
		Summary: v1alpha2.PolicyReportSummary{
			Fail: report.Report.Summary.FailCount,
		},
		Results: []v1alpha2.PolicyReportResult{},
	}
}

func GeneratePolicyReportName(name string) string {
	return fmt.Sprintf("%s-%s", reportPrefix, name)
}
