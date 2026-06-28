package autotest

import (
	"encoding/json"
	"os"
	"testing"
)

func TestFeatureReportIsMachineReadable(t *testing.T) {
	data, err := os.ReadFile("reports/features.json")
	if err != nil {
		t.Fatalf("read feature report: %v", err)
	}
	var report struct {
		SchemaVersion int `json:"schemaVersion"`
		Features      []struct {
			Area    string `json:"area"`
			Feature string `json:"feature"`
			Status  string `json:"status"`
			Test    string `json:"test"`
		} `json:"features"`
	}
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("decode feature report: %v", err)
	}
	if report.SchemaVersion != 1 {
		t.Fatalf("schemaVersion = %d, want 1", report.SchemaVersion)
	}
	if len(report.Features) == 0 {
		t.Fatal("feature report has no features")
	}
	for _, feature := range report.Features {
		if feature.Area == "" || feature.Feature == "" || feature.Status == "" || feature.Test == "" {
			t.Fatalf("incomplete feature report entry: %+v", feature)
		}
	}
}
