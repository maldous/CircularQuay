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
	if err := validateCompositionEdits(composition); err != nil {
		return model.Policy{}, err
	}
	document, err := compositionDocument(compiled)
	if err != nil {
		return model.Policy{}, err
	}
	if err = applyOverrides(document, composition.Overrides); err != nil {
		return model.Policy{}, err
	}
	if err = enforceLocks(document, composition.Locks); err != nil {
		return model.Policy{}, err
	}
	return editedPolicy(document, path)
}

func validateCompositionEdits(composition *model.Composition) error {
	if err := validateEdits(composition.Overrides); err != nil {
		return err
	}
	return validateEdits(composition.Locks)
}

func compositionDocument(compiled model.Policy) (map[string]any, error) {
	data, err := json.Marshal(compiled)
	if err != nil {
		return nil, failure(fault.Invalid, "composition_encode")
	}
	var document map[string]any
	if jsondocument.Decode(data, &document) != nil {
		return nil, failure(fault.Invalid, "composition_encode")
	}
	return document, nil
}

func applyOverrides(document map[string]any, edits []model.PolicyEdit) error {
	for _, edit := range edits {
		value, err := editValue(edit.Value)
		if err != nil || applyEdit(document, edit.Pointer, value) != nil {
			return failure(fault.Invalid, "composition_override")
		}
	}
	return nil
}

func enforceLocks(document map[string]any, locks []model.PolicyEdit) error {
	for _, lock := range locks {
		value, err := editValue(lock.Value)
		if err != nil || !lockedValue(document, lock.Pointer, value) {
			return failure(fault.Denied, "composition_lock")
		}
	}
	return nil
}

func editedPolicy(document map[string]any, path string) (model.Policy, error) {
	data, err := json.Marshal(document)
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
