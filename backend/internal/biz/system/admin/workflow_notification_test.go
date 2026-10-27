package biz

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/liujitcn/kratos-admin/backend/pkg/notification"
)

func TestWorkflowNotificationUsesPersistedI18nMarkers(t *testing.T) {
	message := (&WorkflowInstanceCase{}).workflowNotification(context.Background(), "system.workflow.notification.title.pending", "system.workflow.notification.content.pending")
	if message.title != "__I18N__:system.workflow.notification.title.pending" || message.content != "__I18N__:system.workflow.notification.content.pending" || message.senderName != "__I18N__:system.notification.sender.system" {
		t.Fatalf("unexpected workflow notification markers: %#v", message)
	}
}

func TestWorkflowMessageActionParamsBindsTaskToPendingList(t *testing.T) {
	raw := workflowMessageActionParams("project", 123, 0)
	var params struct {
		Admin struct {
			Route      string `json:"route"`
			Source     string `json:"source"`
			TaskID     string `json:"task_id"`
			InstanceID string `json:"instance_id"`
			ListType   string `json:"list_type"`
		} `json:"admin"`
	}
	if err := json.Unmarshal([]byte(raw), &params); err != nil {
		t.Fatal(err)
	}
	if params.Admin.Route != "WorkflowApproval" || params.Admin.Source != "project" || params.Admin.TaskID != "123" || params.Admin.InstanceID != "" || params.Admin.ListType != "PENDING" {
		t.Fatalf("unexpected task action params: %s", raw)
	}
}

func TestWorkflowMessageActionParamsBindsInstanceToSubmittedList(t *testing.T) {
	raw := workflowMessageActionParams("tenant", 0, 456)
	var params struct {
		Admin struct {
			Route      string `json:"route"`
			Source     string `json:"source"`
			TaskID     string `json:"task_id"`
			InstanceID string `json:"instance_id"`
			ListType   string `json:"list_type"`
		} `json:"admin"`
	}
	if err := json.Unmarshal([]byte(raw), &params); err != nil {
		t.Fatal(err)
	}
	if params.Admin.Route != "WorkflowApproval" || params.Admin.Source != "tenant" || params.Admin.TaskID != "" || params.Admin.InstanceID != "456" || params.Admin.ListType != "SUBMITTED" {
		t.Fatalf("unexpected instance action params: %s", raw)
	}
}

func TestWorkflowMessageSourceByBusinessType(t *testing.T) {
	tests := map[string]string{
		"BASE_TENANT_PROJECT_CREATE": "project",
		"BASE_TENANT_CREATE":         "tenant",
		"OTHER":                      "workflow",
	}
	for businessType, want := range tests {
		if got := workflowMessageSourceByBusinessType(businessType); got != want {
			t.Fatalf("business type %q source = %q, want %q", businessType, got, want)
		}
	}
}

func TestPublishWorkflowNotificationBindsOneRecipientTask(t *testing.T) {
	var messages []notification.Message
	notification.SetDefaultPublisher(notification.PublisherFunc(func(_ context.Context, message notification.Message) (int64, error) {
		messages = append(messages, message)
		return 1, nil
	}))
	t.Cleanup(func() { notification.SetDefaultPublisher(nil) })

	caseValue := &WorkflowInstanceCase{}
	message := workflowNotificationMessage{title: "待审批", content: "请处理", senderName: "系统"}
	recipients := []workflowNotificationRecipient{
		{userID: 11, taskID: 101, instanceID: 201, businessType: baseTenantProjectCreateBusinessType},
		{userID: 12, taskID: 102, instanceID: 201, businessType: baseTenantProjectCreateBusinessType},
	}
	for _, recipient := range recipients {
		if err := caseValue.publishWorkflowNotification(context.Background(), 1, recipient, message, ""); err != nil {
			t.Fatal(err)
		}
	}
	if len(messages) != 2 {
		t.Fatalf("published %d messages, want 2", len(messages))
	}
	for index, published := range messages {
		if len(published.Audiences) != 1 || published.Audiences[0].ID != recipients[index].userID {
			t.Fatalf("message %d audience = %#v, want user %d", index, published.Audiences, recipients[index].userID)
		}
		if published.CategoryCode != workflowMessageCategory || published.Source != "project" || published.ActionTarget != workflowMessageActionTarget {
			t.Fatalf("message %d metadata = %#v", index, published)
		}
		var params struct {
			Admin struct {
				TaskID string `json:"task_id"`
			} `json:"admin"`
		}
		if err := json.Unmarshal([]byte(published.ActionParams), &params); err != nil {
			t.Fatal(err)
		}
		if params.Admin.TaskID != []string{"101", "102"}[index] {
			t.Fatalf("message %d task_id = %q", index, params.Admin.TaskID)
		}
	}
}
