package armsapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	arms "github.com/alibabacloud-go/arms-20190808/v8/client"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyun/credentials-go/credentials"
	credentialproviders "github.com/aliyun/credentials-go/credentials/providers"
)

const (
	alertType      = "PROMETHEUS_MONITORING_ALERT_RULE"
	alertCheckType = "CUSTOM"
	alertGroup     = int64(-1)
	notifyMode     = "NORMAL_MODE"
	// The generated SDK models the create response's AlertId as float32.
	// Integers above this value cannot all be represented exactly.
	maxExactFloat32Integer = int64(1 << 24)
)

// AlertRule is the provider-facing representation of a custom PromQL alert rule.
type AlertRule struct {
	ID              int64
	Name            string
	ClusterID       string
	PromQL          string
	DurationMinutes int64
	Level           string
	Message         string
	Status          string
	Labels          map[string]string
	Annotations     map[string]string
	Tags            map[string]string
	NoDataRevision  int64
}

// AlertRuleService is the API surface used by the Terraform resource.
type AlertRuleService interface {
	CreateOrUpdate(rule AlertRule) (int64, error)
	Get(id int64) (*AlertRule, error)
	Delete(id int64) error
}

type sdkClient interface {
	CreateOrUpdateAlertRule(request *arms.CreateOrUpdateAlertRuleRequest) (*arms.CreateOrUpdateAlertRuleResponse, error)
	GetAlertRules(request *arms.GetAlertRulesRequest) (*arms.GetAlertRulesResponse, error)
	DeleteAlertRule(request *arms.DeleteAlertRuleRequest) (*arms.DeleteAlertRuleResponse, error)
}

// Client calls the ARMS 2019-08-08 alert management API.
type Client struct {
	sdk      sdkClient
	regionID string
}

// NewClient creates an ARMS client using the Alibaba Cloud default credential
// chain or a named aliyun CLI profile.
func NewClient(regionID, profile, endpoint string) (*Client, error) {
	credential, err := resolveCredential(profile)
	if err != nil {
		return nil, err
	}

	config := &openapi.Config{
		Credential: credential,
		RegionId:   tea.String(regionID),
	}
	if endpoint != "" {
		config.Endpoint = tea.String(endpoint)
	}

	sdk, err := arms.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("create ARMS client: %w", err)
	}

	return &Client{sdk: sdk, regionID: regionID}, nil
}

func resolveCredential(profile string) (credentials.Credential, error) {
	if profile == "" {
		credential, err := credentials.NewCredential(nil)
		if err != nil {
			return nil, fmt.Errorf("resolve Alibaba Cloud credentials: %w", err)
		}
		return credential, nil
	}

	profileProvider, err := credentialproviders.NewCLIProfileCredentialsProviderBuilder().
		WithProfileName(profile).
		Build()
	if err != nil {
		return nil, fmt.Errorf("configure aliyun CLI profile %q: %w", profile, err)
	}

	return credentials.FromCredentialsProvider("cli_profile", profileProvider), nil
}

// CreateOrUpdate calls the new unified alert API. An ID selects update mode;
// omitting it creates a new rule.
func (c *Client) CreateOrUpdate(rule AlertRule) (int64, error) {
	request, err := buildCreateOrUpdateRequest(c.regionID, rule)
	if err != nil {
		return 0, err
	}

	response, err := c.sdk.CreateOrUpdateAlertRule(request)
	if err != nil {
		return 0, fmt.Errorf("CreateOrUpdateAlertRule: %w", err)
	}
	if rule.ID != 0 {
		return rule.ID, nil
	}
	if response == nil || response.Body == nil || response.Body.AlertRule == nil || response.Body.AlertRule.AlertId == nil {
		return 0, errors.New("CreateOrUpdateAlertRule returned no alert rule ID")
	}

	// Use the generated response only while its float32 representation is exact.
	// For larger IDs, query by the unique new-version rule name instead.
	alertID := int64(*response.Body.AlertRule.AlertId)
	if alertID <= 0 || float32(alertID) != *response.Body.AlertRule.AlertId {
		return 0, fmt.Errorf("CreateOrUpdateAlertRule returned invalid alert rule ID %v", *response.Body.AlertRule.AlertId)
	}
	if alertID <= maxExactFloat32Integer {
		return alertID, nil
	}
	return c.getIDByName(rule.Name, rule.ClusterID)
}

func buildCreateOrUpdateRequest(regionID string, rule AlertRule) (*arms.CreateOrUpdateAlertRuleRequest, error) {
	labels, err := marshalNameValueMap(rule.Labels)
	if err != nil {
		return nil, fmt.Errorf("encode labels: %w", err)
	}
	annotations, err := marshalNameValueMap(rule.Annotations)
	if err != nil {
		return nil, fmt.Errorf("encode annotations: %w", err)
	}
	dataConfig, err := json.Marshal(map[string]int64{"dataRevision": rule.NoDataRevision})
	if err != nil {
		return nil, fmt.Errorf("encode data configuration: %w", err)
	}

	request := (&arms.CreateOrUpdateAlertRuleRequest{}).
		SetAlertName(rule.Name).
		SetRegionId(regionID).
		SetAlertType(alertType).
		SetAlertStatus(rule.Status).
		SetAlertCheckType(alertCheckType).
		SetAlertGroup(alertGroup).
		SetClusterId(rule.ClusterID).
		SetPromQL(rule.PromQL).
		SetDuration(rule.DurationMinutes).
		SetLevel(rule.Level).
		SetMessage(rule.Message).
		SetLabels(labels).
		SetAnnotations(annotations).
		SetDataConfig(string(dataConfig)).
		SetNotifyMode(notifyMode).
		SetTags(toRequestTags(rule.Tags))
	if rule.ID != 0 {
		request.SetAlertId(rule.ID)
	}
	return request, nil
}

// Get returns nil when the alert rule no longer exists.
func (c *Client) Get(id int64) (*AlertRule, error) {
	alertIDs, err := json.Marshal([]string{strconv.FormatInt(id, 10)})
	if err != nil {
		return nil, fmt.Errorf("encode alert rule ID: %w", err)
	}

	request := (&arms.GetAlertRulesRequest{}).
		SetAlertIds(string(alertIDs)).
		SetAlertType(alertType).
		SetRegionId(c.regionID).
		SetPage(1).
		SetSize(20)
	response, err := c.sdk.GetAlertRules(request)
	if err != nil {
		return nil, fmt.Errorf("GetAlertRules: %w", err)
	}
	if response == nil || response.Body == nil || response.Body.PageBean == nil {
		return nil, errors.New("GetAlertRules returned an empty response")
	}

	for _, apiRule := range response.Body.PageBean.AlertRules {
		if apiRule != nil && apiRule.AlertId != nil && *apiRule.AlertId == id {
			return alertRuleFromAPI(apiRule)
		}
	}
	return nil, nil
}

func (c *Client) getIDByName(name, clusterID string) (int64, error) {
	alertNames, err := json.Marshal([]string{name})
	if err != nil {
		return 0, fmt.Errorf("encode alert rule name: %w", err)
	}
	request := (&arms.GetAlertRulesRequest{}).
		SetAlertNames(string(alertNames)).
		SetAlertType(alertType).
		SetClusterId(clusterID).
		SetRegionId(c.regionID).
		SetPage(1).
		SetSize(20)
	response, err := c.sdk.GetAlertRules(request)
	if err != nil {
		return 0, fmt.Errorf("GetAlertRules after create: %w", err)
	}
	if response == nil || response.Body == nil || response.Body.PageBean == nil {
		return 0, errors.New("GetAlertRules after create returned an empty response")
	}
	for _, apiRule := range response.Body.PageBean.AlertRules {
		if apiRule != nil && apiRule.AlertId != nil && tea.StringValue(apiRule.AlertName) == name && tea.StringValue(apiRule.ClusterId) == clusterID {
			return *apiRule.AlertId, nil
		}
	}
	return 0, fmt.Errorf("new alert rule %q was not returned by GetAlertRules", name)
}

// Delete removes a new-version alert rule.
func (c *Client) Delete(id int64) error {
	response, err := c.sdk.DeleteAlertRule((&arms.DeleteAlertRuleRequest{}).SetAlertId(id))
	if err != nil {
		return fmt.Errorf("DeleteAlertRule: %w", err)
	}
	if response == nil || response.Body == nil || response.Body.IsSuccess == nil {
		return errors.New("DeleteAlertRule returned an empty response")
	}
	if !*response.Body.IsSuccess {
		return errors.New("DeleteAlertRule reported failure")
	}
	return nil
}

func alertRuleFromAPI(apiRule *arms.GetAlertRulesResponseBodyPageBeanAlertRules) (*AlertRule, error) {
	duration, err := parseDuration(apiRule.Duration)
	if err != nil {
		return nil, err
	}

	rule := &AlertRule{
		ID:              tea.Int64Value(apiRule.AlertId),
		Name:            tea.StringValue(apiRule.AlertName),
		ClusterID:       tea.StringValue(apiRule.ClusterId),
		PromQL:          tea.StringValue(apiRule.PromQL),
		DurationMinutes: duration,
		Level:           tea.StringValue(apiRule.Level),
		Message:         tea.StringValue(apiRule.Message),
		Status:          tea.StringValue(apiRule.AlertStatus),
		Labels:          labelsFromAPI(apiRule.Labels),
		Annotations:     annotationsFromAPI(apiRule.Annotations),
		Tags:            tagsFromAPI(apiRule.Tags),
	}
	return rule, nil
}

func parseDuration(value *string) (int64, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return 0, errors.New("GetAlertRules returned no duration")
	}
	normalized := strings.TrimSuffix(strings.TrimSpace(*value), "m")
	duration, err := strconv.ParseInt(normalized, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse alert duration %q: %w", *value, err)
	}
	return duration, nil
}

type nameValue struct {
	Name  string `json:"Name"`
	Value string `json:"Value"`
}

func marshalNameValueMap(values map[string]string) (string, error) {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	items := make([]nameValue, 0, len(keys))
	for _, key := range keys {
		items = append(items, nameValue{Name: key, Value: values[key]})
	}
	encoded, err := json.Marshal(items)
	return string(encoded), err
}

func toRequestTags(values map[string]string) []*arms.CreateOrUpdateAlertRuleRequestTags {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	result := make([]*arms.CreateOrUpdateAlertRuleRequestTags, 0, len(keys))
	for _, key := range keys {
		result = append(result, (&arms.CreateOrUpdateAlertRuleRequestTags{}).SetKey(key).SetValue(values[key]))
	}
	return result
}

func labelsFromAPI(values []*arms.GetAlertRulesResponseBodyPageBeanAlertRulesLabels) map[string]string {
	result := make(map[string]string, len(values))
	for _, value := range values {
		if value != nil && value.Name != nil {
			result[*value.Name] = tea.StringValue(value.Value)
		}
	}
	return result
}

func annotationsFromAPI(values []*arms.GetAlertRulesResponseBodyPageBeanAlertRulesAnnotations) map[string]string {
	result := make(map[string]string, len(values))
	for _, value := range values {
		if value != nil && value.Name != nil {
			result[*value.Name] = tea.StringValue(value.Value)
		}
	}
	return result
}

func tagsFromAPI(values []*arms.GetAlertRulesResponseBodyPageBeanAlertRulesTags) map[string]string {
	result := make(map[string]string, len(values))
	for _, value := range values {
		if value != nil && value.Key != nil {
			result[*value.Key] = tea.StringValue(value.Value)
		}
	}
	return result
}
