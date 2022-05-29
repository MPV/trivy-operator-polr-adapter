package kubebench

import (
	"fmt"

	"github.com/aquasecurity/trivy-operator/pkg/apis/aquasecurity/v1alpha1"
	"github.com/kyverno/kyverno/api/policyreport/v1alpha2"
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
	StatusFail = "FAIL"
	StatusWarn = "WARN"
	StatusInfo = "INFO"
	StatusPass = "PASS"
)

const (
	source       = "Trivy CIS Kube Bench"
	reportPrefix = "trivy-cis-cpolr"
)

var (
	reportLabels = map[string]string{
		"managed-by":            "trivy-operator-polr-adapter",
		"trivy-operator.source": "CISKubeBenchReport",
	}
)

func Map(report *v1alpha1.CISKubeBenchReport, polr *v1alpha2.ClusterPolicyReport) (*v1alpha2.ClusterPolicyReport, bool) {
	var updated bool

	if polr == nil {
		polr = CreatePolicyReport(report)
	} else {
		polr.Summary = v1alpha2.PolicyReportSummary{}
		polr.Results = []*v1alpha2.PolicyReportResult{}
		updated = true
	}

	for _, section := range report.Report.Sections {
		for _, test := range section.Tests {
			for _, result := range test.Results {
				switch result.Status {
				case StatusFail:
					polr.Summary.Fail++
				case StatusPass:
					polr.Summary.Pass++
				case StatusWarn:
					polr.Summary.Warn++
				case StatusInfo:
					polr.Summary.Skip++
				}

				polr.Results = append(polr.Results, &v1alpha2.PolicyReportResult{
					Policy:    fmt.Sprintf("%s %s", test.Section, test.Desc),
					Rule:      fmt.Sprintf("%s %s", result.TestNumber, result.TestDesc),
					Message:   result.Remediation,
					Scored:    result.Scored,
					Result:    MapResult(result.Status),
					Category:  section.Text,
					Timestamp: *report.CreationTimestamp.ProtoTime(),
					Source:    source,
				})
			}
		}
	}

	return polr, updated
}

func MapResult(status string) v1alpha2.PolicyResult {
	switch status {
	case StatusFail:
		return v1alpha2.StatusFail
	case StatusPass:
		return v1alpha2.StatusPass
	case StatusWarn:
		return v1alpha2.StatusWarn
	}

	return v1alpha2.StatusSkip
}

func MapServerity(severity v1alpha1.Severity) v1alpha2.PolicySeverity {
	if severity == v1alpha1.SeverityUnknown || severity == v1alpha1.SeverityNone {
		return ""
	} else if severity == v1alpha1.SeverityLow {
		return v1alpha2.SeverityLow
	} else if severity == v1alpha1.SeverityMedium {
		return v1alpha2.SeverityMedium
	}

	return v1alpha2.SeverityHigh
}

func CreatePolicyReport(report *v1alpha1.CISKubeBenchReport) *v1alpha2.ClusterPolicyReport {
	return &v1alpha2.ClusterPolicyReport{
		ObjectMeta: v1.ObjectMeta{
			Name:      GeneratePolicyReportName(report.Name),
			Namespace: report.Namespace,
			Labels:    reportLabels,
			OwnerReferences: []v1.OwnerReference{
				{
					APIVersion: report.APIVersion,
					Kind:       report.Kind,
					Name:       report.Name,
					UID:        report.UID,
				},
			},
		},
		Summary: v1alpha2.PolicyReportSummary{},
		Results: []*v1alpha2.PolicyReportResult{},
	}
}

func GeneratePolicyReportName(name string) string {
	return fmt.Sprintf("%s-%s", reportPrefix, name)
}
