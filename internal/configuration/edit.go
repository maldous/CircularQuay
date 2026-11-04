package configuration

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/jsondocument"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/policy"
)

func applyEdits(compiled model.Policy, composition *model.Composition, path string) (model.Policy, error) {
	if err := validateEdits(composition.Overrides); err != nil {
		return model.Policy{}, err
	}
	if err := validateEdits(composition.Locks); err != nil {
		return model.Policy{}, err
	}
	data, err := json.Marshal(compiled)
	if err != nil {
		return model.Policy{}, failure(fault.Invalid, "composition_encode")
	}
	var document map[string]any
	if jsondocument.Decode(data, &document) != nil {
		return model.Policy{}, failure(fault.Invalid, "composition_encode")
	}
	for _, edit := range composition.Overrides {
		value, err := editValue(edit.Value)
		if err != nil || applyEdit(document, edit.Pointer, value) != nil {
			return model.Policy{}, failure(fault.Invalid, "composition_override")
		}
	}
	for _, lock := range composition.Locks {
		value, err := editValue(lock.Value)
		if err != nil || !lockedValue(document, lock.Pointer, value) {
			return model.Policy{}, failure(fault.Denied, "composition_lock")
		}
	}
	data, err = json.Marshal(document)
	if err != nil {
		return model.Policy{}, failure(fault.Invalid, "composition_encode")
	}
	resolved, err := policy.DecodeDocument(data, path)
	if err != nil {
		return model.Policy{}, failure(fault.Invalid, "composition_result")
	}
	return withoutComposition(resolved), nil
}

func validateEdits(edits []model.PolicyEdit) error {
	if len(edits) > 128 {
		return failure(fault.Invalid, "composition_edit")
	}
	seen := make([]string, 0, len(edits))
	for _, edit := range edits {
		if len(edit.Pointer) > 4096 || !allowedEditPointer(edit.Pointer) || len(edit.Value) == 0 {
			return failure(fault.Invalid, "composition_edit")
		}
		if _, err := editValue(edit.Value); err != nil || overlaps(seen, edit.Pointer) {
			return failure(fault.Invalid, "composition_edit")
		}
		seen = append(seen, edit.Pointer)
	}
	return nil
}

func allowedEditPointer(pointer string) bool {
	parts, ok := pointerParts(pointer)
	return ok && len(parts) >= 3 && (parts[0] == "sources" || parts[0] == "workloads") && validIdentifier(parts[1])
}

func overlaps(existing []string, candidate string) bool {
	for _, value := range existing {
		if value == candidate || strings.HasPrefix(value, candidate+"/") || strings.HasPrefix(candidate, value+"/") {
			return true
		}
	}
	return false
}

func editValue(raw json.RawMessage) (any, error) {
	if len(raw) == 0 || len(raw) > maximumFragmentBytes || !bytes.Equal(bytes.TrimSpace(raw), raw) {
		return nil, failure(fault.Invalid, "composition_edit")
	}
	var value any
	if jsondocument.Decode(raw, &value) != nil || value == nil {
		return nil, failure(fault.Invalid, "composition_edit")
	}
	return value, nil
}

func applyEdit(document map[string]any, pointer string, value any) error {
	parent, leaf, ok := editParent(document, pointer)
	if !ok {
		return failure(fault.Invalid, "composition_override")
	}
	parent[leaf] = value
	return nil
}

func lockedValue(document map[string]any, pointer string, want any) bool {
	parent, leaf, ok := editParent(document, pointer)
	if !ok {
		return false
	}
	got, found := parent[leaf]
	return found && jsonEqual(got, want)
}

func editParent(document map[string]any, pointer string) (map[string]any, string, bool) {
	parts, ok := pointerParts(pointer)
	if !ok || len(parts) < 2 {
		return nil, "", false
	}
	current := document
	for _, part := range parts[:len(parts)-1] {
		next, found := current[part].(map[string]any)
		if !found {
			return nil, "", false
		}
		current = next
	}
	return current, parts[len(parts)-1], true
}

func pointerParts(pointer string) ([]string, bool) {
	if !strings.HasPrefix(pointer, "/") || strings.Contains(pointer, "//") {
		return nil, false
	}
	parts := strings.Split(pointer[1:], "/")
	for index := range parts {
		decoded, ok := decodePointerToken(parts[index])
		if !ok || decoded == "" {
			return nil, false
		}
		parts[index] = decoded
	}
	return parts, true
}

func decodePointerToken(value string) (string, bool) {
	result := strings.Builder{}
	for index := 0; index < len(value); index++ {
		if value[index] != '~' {
			result.WriteByte(value[index])
			continue
		}
		if index+1 == len(value) || value[index+1] != '0' && value[index+1] != '1' {
			return "", false
		}
		if value[index+1] == '0' {
			result.WriteByte('~')
		} else {
			result.WriteByte('/')
		}
		index++
	}
	return result.String(), true
}

func jsonEqual(first, second any) bool {
	left, leftErr := json.Marshal(first)
	right, rightErr := json.Marshal(second)
	return leftErr == nil && rightErr == nil && bytes.Equal(left, right)
}
