package armsapi

import (
	"errors"
	"reflect"
	"testing"

	arms "github.com/alibabacloud-go/arms-20190808/v8/client"
	"github.com/alibabacloud-go/tea/tea"
)

type fakeSDKClient struct {
	createRequest  *arms.CreateOrUpdateAlertRuleRequest
	createResponse *arms.CreateOrUpdateAlertRuleResponse
	createError    error
	getRequest     *arms.GetAlertRulesRequest
	getResponse    *arms.GetAlertRulesResponse
	getError       error
	deleteRequest  *arms.DeleteAlertRuleRequest
	deleteResponse *arms.DeleteAlertRuleResponse
	deleteError    error
}

func (f *fakeSDKClient) CreateOrUpdateAlertRule(request *arms.CreateOrUpdateAlertRuleRequest) (*arms.CreateOrUpdateAlertRuleResponse, error) {
	f.createRequest = request
	return f.createResponse, f.createError
}

func (f *fakeSDKClient) GetAlertRules(request *arms.GetAlertRulesRequest) (*arms.GetAlertRulesResponse, error) {
	f.getRequest = request
	return f.getResponse, f.getError
}

func (f *fakeSDKClient) DeleteAlertRule(request *arms.DeleteAlertRuleRequest) (*arms.DeleteAlertRuleResponse, error) {
	f.deleteRequest = request
	return f.deleteResponse, f.deleteError
}

func TestBuildCreateOrUpdateRequest(t *testing.T) {
	t.Parallel()

	request, err := buildCreateOrUpdateRequest("ap-southeast-1", AlertRule{
		ID:              42,
		Name:            "prod-pod-restarts",
		ClusterID:       "cluster-id",
		PromQL:          "increase(restarts[1m]) > 1",
		DurationMinutes: 5,
		Level:           "P2",
		Message:         "Pod restarted",
		Status:          "RUNNING",
		Labels:          map[string]string{"severity": "critical", "team": "platform"},
		Annotations:     map[string]string{"summary": "restart"},
		Tags:            map[string]string{"CreatedBy": "Terraform", "Environment": "prod"},
		NoDataRevision:  2,
	})
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	checks := map[string]bool{
		"alert ID":          request.AlertId != nil && *request.AlertId == 42,
		"new alert type":    tea.StringValue(request.AlertType) == alertType,
		"custom check type": tea.StringValue(request.AlertCheckType) == alertCheckType,
		"custom group":      tea.Int64Value(request.AlertGroup) == -1,
		"duration":          tea.Int64Value(request.Duration) == 5,
		"normal mode":       tea.StringValue(request.NotifyMode) == notifyMode,
		"data config":       tea.StringValue(request.DataConfig) == `{"dataRevision":2}`,
		"labels sorted":     tea.StringValue(request.Labels) == `[{"Name":"severity","Value":"critical"},{"Name":"team","Value":"platform"}]`,
		"annotations":       tea.StringValue(request.Annotations) == `[{"Name":"summary","Value":"restart"}]`,
	}
	for name, ok := range checks {
		if !ok {
			t.Errorf("check %q failed: %#v", name, request)
		}
	}
	if len(request.Tags) != 2 || tea.StringValue(request.Tags[0].Key) != "CreatedBy" || tea.StringValue(request.Tags[1].Key) != "Environment" {
		t.Fatalf("tags are not deterministic: %#v", request.Tags)
	}
}

func TestAlertRuleFromAPI(t *testing.T) {
	t.Parallel()

	rule, err := alertRuleFromAPI(&arms.GetAlertRulesResponseBodyPageBeanAlertRules{
		AlertId:     tea.Int64(123),
		AlertName:   tea.String("rule"),
		ClusterId:   tea.String("cluster"),
		PromQL:      tea.String("up == 0"),
		Duration:    tea.String("10m"),
		Level:       tea.String("P1"),
		Message:     tea.String("down"),
		AlertStatus: tea.String("RUNNING"),
		Labels: []*arms.GetAlertRulesResponseBodyPageBeanAlertRulesLabels{
			{Name: tea.String("severity"), Value: tea.String("critical")},
		},
		Annotations: []*arms.GetAlertRulesResponseBodyPageBeanAlertRulesAnnotations{
			{Name: tea.String("summary"), Value: tea.String("down")},
		},
		Tags: []*arms.GetAlertRulesResponseBodyPageBeanAlertRulesTags{
			{Key: tea.String("CreatedBy"), Value: tea.String("Terraform")},
		},
	})
	if err != nil {
		t.Fatalf("convert response: %v", err)
	}
	if rule.DurationMinutes != 10 {
		t.Fatalf("duration = %d, want 10", rule.DurationMinutes)
	}
	if !reflect.DeepEqual(rule.Labels, map[string]string{"severity": "critical"}) {
		t.Fatalf("labels = %#v", rule.Labels)
	}
	if !reflect.DeepEqual(rule.Annotations, map[string]string{"summary": "down"}) {
		t.Fatalf("annotations = %#v", rule.Annotations)
	}
}

func TestParseDurationRejectsInvalidValue(t *testing.T) {
	t.Parallel()
	if _, err := parseDuration(tea.String("forever")); err == nil {
		t.Fatal("expected invalid duration to fail")
	}
}

func TestClientCreateUsesUnifiedAlertAPI(t *testing.T) {
	t.Parallel()

	fake := &fakeSDKClient{
		createResponse: &arms.CreateOrUpdateAlertRuleResponse{
			Body: &arms.CreateOrUpdateAlertRuleResponseBody{
				AlertRule: &arms.CreateOrUpdateAlertRuleResponseBodyAlertRule{AlertId: tea.Float32(123)},
			},
		},
	}
	client := &Client{sdk: fake, regionID: "ap-southeast-1"}
	id, err := client.CreateOrUpdate(AlertRule{
		Name:            "rule",
		ClusterID:       "cluster",
		PromQL:          "up == 0",
		DurationMinutes: 5,
		Level:           "P2",
		Message:         "down",
		Status:          "RUNNING",
		NoDataRevision:  2,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if id != 123 {
		t.Fatalf("id = %d, want 123", id)
	}
	if fake.createRequest == nil || fake.createRequest.AlertId != nil {
		t.Fatalf("create request unexpectedly contains an ID: %#v", fake.createRequest)
	}
}

func TestClientUpdateKeepsExistingID(t *testing.T) {
	t.Parallel()

	fake := &fakeSDKClient{createResponse: &arms.CreateOrUpdateAlertRuleResponse{}}
	client := &Client{sdk: fake, regionID: "ap-southeast-1"}
	id, err := client.CreateOrUpdate(AlertRule{
		ID:              456,
		Name:            "rule",
		ClusterID:       "cluster",
		PromQL:          "up == 0",
		DurationMinutes: 5,
		Level:           "P2",
		Message:         "down",
		Status:          "RUNNING",
		NoDataRevision:  2,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if id != 456 || tea.Int64Value(fake.createRequest.AlertId) != 456 {
		t.Fatalf("update ID mismatch: result=%d request=%v", id, fake.createRequest.AlertId)
	}
}

func TestClientGetReturnsNilWhenMissing(t *testing.T) {
	t.Parallel()

	fake := &fakeSDKClient{
		getResponse: &arms.GetAlertRulesResponse{
			Body: &arms.GetAlertRulesResponseBody{
				PageBean: &arms.GetAlertRulesResponseBodyPageBean{},
			},
		},
	}
	client := &Client{sdk: fake, regionID: "ap-southeast-1"}
	rule, err := client.Get(789)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if rule != nil {
		t.Fatalf("rule = %#v, want nil", rule)
	}
	if tea.StringValue(fake.getRequest.AlertIds) != `["789"]` {
		t.Fatalf("alert IDs = %q", tea.StringValue(fake.getRequest.AlertIds))
	}
}

func TestClientDeleteChecksAPIResult(t *testing.T) {
	t.Parallel()

	fake := &fakeSDKClient{
		deleteResponse: &arms.DeleteAlertRuleResponse{
			Body: &arms.DeleteAlertRuleResponseBody{IsSuccess: tea.Bool(false)},
		},
	}
	client := &Client{sdk: fake, regionID: "ap-southeast-1"}
	if err := client.Delete(123); err == nil {
		t.Fatal("expected API failure to be returned")
	}
	if tea.Int64Value(fake.deleteRequest.AlertId) != 123 {
		t.Fatalf("delete ID = %d", tea.Int64Value(fake.deleteRequest.AlertId))
	}
}

func TestClientReturnsSDKError(t *testing.T) {
	t.Parallel()

	fake := &fakeSDKClient{createError: errors.New("throttled")}
	client := &Client{sdk: fake, regionID: "ap-southeast-1"}
	_, err := client.CreateOrUpdate(AlertRule{})
	if err == nil || err.Error() != "CreateOrUpdateAlertRule: throttled" {
		t.Fatalf("error = %v", err)
	}
}

func TestClientLooksUpLargeFloat32AlertIDByName(t *testing.T) {
	t.Parallel()

	largeID := int64(16777218)
	fake := &fakeSDKClient{
		createResponse: &arms.CreateOrUpdateAlertRuleResponse{
			Body: &arms.CreateOrUpdateAlertRuleResponseBody{
				AlertRule: &arms.CreateOrUpdateAlertRuleResponseBodyAlertRule{AlertId: tea.Float32(float32(largeID))},
			},
		},
		getResponse: &arms.GetAlertRulesResponse{
			Body: &arms.GetAlertRulesResponseBody{
				PageBean: &arms.GetAlertRulesResponseBodyPageBean{
					AlertRules: []*arms.GetAlertRulesResponseBodyPageBeanAlertRules{
						{AlertId: tea.Int64(largeID), AlertName: tea.String("rule"), ClusterId: tea.String("cluster")},
					},
				},
			},
		},
	}
	client := &Client{sdk: fake, regionID: "ap-southeast-1"}
	id, err := client.CreateOrUpdate(AlertRule{Name: "rule", ClusterID: "cluster"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if id != largeID {
		t.Fatalf("id = %d, want %d", id, largeID)
	}
	if tea.StringValue(fake.getRequest.AlertNames) != `["rule"]` {
		t.Fatalf("alert names = %q", tea.StringValue(fake.getRequest.AlertNames))
	}
}
