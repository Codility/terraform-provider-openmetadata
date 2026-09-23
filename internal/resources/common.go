// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

// Package resources contains shared helpers used across all resource implementations.
package resources

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// --- Common schema attributes reused across resources ---

// IDAttribute returns the computed "id" attribute (UUID assigned by OM).
func IDAttribute() schema.StringAttribute {
	return schema.StringAttribute{
		Description: "UUID of the resource assigned by OpenMetadata.",
		Computed:    true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	}
}

// NameAttribute returns the required "name" attribute.
func NameAttribute() schema.StringAttribute {
	return schema.StringAttribute{
		Description: "Unique name identifying the resource.",
		Required:    true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.RequiresReplace(),
		},
	}
}

// DisplayNameAttribute returns the optional "display_name" attribute.
func DisplayNameAttribute() schema.StringAttribute {
	return schema.StringAttribute{
		Description: "Human-readable display name.",
		Optional:    true,
		Computed:    true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	}
}

// DescriptionAttribute returns a description attribute (required or optional).
func DescriptionAttribute(required bool) schema.StringAttribute {
	attr := schema.StringAttribute{
		Description: "Markdown description of the resource.",
		Required:    required,
		Optional:    !required,
		Computed:    !required,
	}
	if !required {
		attr.PlanModifiers = []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		}
	}
	return attr
}

// FullyQualifiedNameAttribute returns a computed FQN attribute.
func FullyQualifiedNameAttribute() schema.StringAttribute {
	return schema.StringAttribute{
		Description: "Fully qualified name of the resource.",
		Computed:    true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	}
}

// DomainsAttribute returns the optional "domains" attribute.
func DomainsAttribute() schema.ListAttribute {
	return schema.ListAttribute{
		Description: "Fully qualified names of the domains this resource belongs to.",
		Optional:    true,
		ElementType: types.StringType,
	}
}

// OwnersAttribute returns the optional "owners" block.
func OwnersAttribute() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		Description: "Owners of this resource.",
		Optional:    true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"id": schema.StringAttribute{
					Description: "UUID of the owner entity.",
					Required:    true,
				},
				"type": schema.StringAttribute{
					Description: "Type of the owner entity (user or team).",
					Required:    true,
				},
			},
		},
	}
}

// ExpertsAttribute returns the optional "experts" block (same shape as owners).
func ExpertsAttribute() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		Description: "Users who are experts for this resource.",
		Optional:    true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"id": schema.StringAttribute{
					Description: "UUID of the expert user.",
					Required:    true,
				},
				"type": schema.StringAttribute{
					Description: "Type of the expert entity (user).",
					Required:    true,
				},
			},
		},
	}
}

// --- Plan modifiers ---

// NormalizeJSONStringModifier is a plan modifier that normalises a JSON string
// by unmarshalling and re-marshalling it. This produces a canonical,
// alphabetically-key-sorted representation so that the plan value matches
// the API response even when the user writes keys in a different order.
type NormalizeJSONStringModifier struct{}

func (m NormalizeJSONStringModifier) Description(_ context.Context) string {
	return "Normalises the JSON string to canonical (key-sorted) form."
}

func (m NormalizeJSONStringModifier) MarkdownDescription(_ context.Context) string {
	return "Normalises the JSON string to canonical (key-sorted) form."
}

func (m NormalizeJSONStringModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
		return
	}
	var v interface{}
	if err := json.Unmarshal([]byte(req.PlanValue.ValueString()), &v); err != nil {
		return // leave invalid JSON as-is; provider will report the error later
	}
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	resp.PlanValue = types.StringValue(string(b))
}

// NormalizeJSONString returns a plan modifier that canonical-sorts a JSON string.
func NormalizeJSONString() planmodifier.String {
	return NormalizeJSONStringModifier{}
}

// --- JSON helpers ---

// EntityRef is the OM entity reference used in owners, reviewers, etc.
type EntityRef struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// StringVal safely extracts a string from a JSON object field.
func StringVal(data map[string]interface{}, key string) types.String {
	if v, ok := data[key]; ok && v != nil {
		if s, ok := v.(string); ok {
			return types.StringValue(s)
		}
	}
	return types.StringNull()
}

// BoolVal safely extracts a bool from a JSON object field.
func BoolVal(data map[string]interface{}, key string) types.Bool {
	if v, ok := data[key]; ok && v != nil {
		if b, ok := v.(bool); ok {
			return types.BoolValue(b)
		}
	}
	return types.BoolNull()
}

// RawStringList extracts a plain JSON array of strings (e.g., synonyms).
func RawStringList(data map[string]interface{}, key string) []string {
	if v, ok := data[key]; ok && v != nil {
		if arr, ok := v.([]interface{}); ok {
			result := make([]string, 0, len(arr))
			for _, item := range arr {
				if s, ok := item.(string); ok {
					result = append(result, s)
				}
			}
			return result
		}
	}
	return nil
}

// ParseEntityRefs extracts a list of EntityRef from a JSON array.
func ParseEntityRefs(data map[string]interface{}, key string) []EntityRef {
	if v, ok := data[key]; ok && v != nil {
		if arr, ok := v.([]interface{}); ok {
			refs := make([]EntityRef, 0, len(arr))
			for _, item := range arr {
				if m, ok := item.(map[string]interface{}); ok {
					ref := EntityRef{}
					if id, ok := m["id"].(string); ok {
						ref.ID = id
					}
					if t, ok := m["type"].(string); ok {
						ref.Type = t
					}
					refs = append(refs, ref)
				}
			}
			return refs
		}
	}
	return nil
}

// EntityRefNames extracts fullyQualifiedName (or name) from a JSON array of entity references.
func EntityRefNames(data map[string]interface{}, key string) []string {
	if v, ok := data[key]; ok && v != nil {
		if arr, ok := v.([]interface{}); ok {
			names := make([]string, 0, len(arr))
			for _, item := range arr {
				if m, ok := item.(map[string]interface{}); ok {
					if fqn, ok := m["fullyQualifiedName"].(string); ok {
						names = append(names, fqn)
					} else if name, ok := m["name"].(string); ok {
						names = append(names, name)
					}
				}
			}
			return names
		}
	}
	return nil
}

// StringListVal extracts entity ref names from data and returns a typed types.List.
// Returns a null list if the key is absent or has no entries.
func StringListVal(data map[string]interface{}, key string) types.List {
	if names := EntityRefNames(data, key); len(names) > 0 {
		list, _ := types.ListValueFrom(context.Background(), types.StringType, names)
		return list
	}
	return types.ListNull(types.StringType)
}

// StringSliceToList converts a []string (or nil) into a typed types.List of StringType.
func StringSliceToList(vals []string) types.List {
	if len(vals) > 0 {
		list, _ := types.ListValueFrom(context.Background(), types.StringType, vals)
		return list
	}
	return types.ListNull(types.StringType)
}

// OwnersListNull returns a properly typed null list for the owners attribute.
func OwnersListNull() types.List {
	return types.ListNull(types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"id":   types.StringType,
			"type": types.StringType,
		},
	})
}

// Unmarshal parses a json.RawMessage into a map.
func Unmarshal(raw json.RawMessage) (map[string]interface{}, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}
	return data, nil
}

// splitFQN splits a fully qualified name by ".".
func splitFQN(fqn string) []string {
	if fqn == "" {
		return nil
	}
	parts := make([]string, 0)
	for _, p := range strings.Split(fqn, ".") {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}
