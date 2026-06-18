package core

import (
	"testing"
)

func TestDispatcherGetTools(t *testing.T) {
	d := NewDispatcher(nil)
	tools := d.GetTools()

	found := false
	for _, tool := range tools {
		if tool.Name == "export_report" {
			found = true
			// Check parameter definition
			if tool.Parameters == nil {
				t.Error("expected parameters to be defined")
			}
			props, ok := tool.Parameters["properties"].(map[string]interface{})
			if !ok {
				t.Error("expected parameters properties map")
			}
			if _, formatOk := props["format"]; !formatOk {
				t.Error("expected format parameter")
			}
			if _, titleOk := props["title"]; !titleOk {
				t.Error("expected title parameter")
			}
		}
	}

	if !found {
		t.Error("expected export_report tool to be registered")
	}
}
