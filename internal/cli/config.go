package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"io"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/maldous/CircularQuay/internal/configuration"
	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/jsondocument"
	"github.com/maldous/CircularQuay/internal/platform"
	"github.com/maldous/CircularQuay/internal/state"
)

const (
	maximumConfigInput = 1 << 20
	flagPointer        = "pointer"
	flagValueJSON      = "value-json"
	flagValueFile      = "value-file"
	flagExpectedSHA256 = "expect-sha256"
	errorConfigPointer = "config pointer"
	errorConfigJSON    = "config JSON"
	errorConfigPath    = "config path"
	errorConfigChanged = "config changed"
)

// configCommand edits root declarations only. It never contacts a provider or
// reads configured values. Validation resolves only explicitly enrolled,
// pinned configuration fragments before an atomic replacement is permitted.
func (a application) configCommand(ctx context.Context, args []string) (int, error) {
	if len(args) == 0 {
		return 0, fault.New(fault.Invalid, "config command")
	}
	handlers := map[string]func([]string) (int, error){
		"init":     func(arguments []string) (int, error) { return a.initializeContext(ctx, arguments) },
		"validate": func(arguments []string) (int, error) { return a.configValidate(ctx, arguments) },
		"get":      a.configGet,
		"list":     a.configList,
		"set":      func(arguments []string) (int, error) { return a.configSet(ctx, arguments) },
		"delete":   func(arguments []string) (int, error) { return a.configDelete(ctx, arguments) },
		"replace":  func(arguments []string) (int, error) { return a.configReplace(ctx, arguments) },
	}
	handler, found := handlers[args[0]]
	if !found {
		return 0, fault.New(fault.Invalid, "config command")
	}
	return handler(args[1:])
}

func (a application) configValidate(ctx context.Context, args []string) (int, error) {
	if len(args) != 0 {
		return 0, fault.New(fault.Invalid, "config validate arguments")
	}
	if _, err := a.loadContext(ctx); err != nil {
		return 0, err
	}
	return a.print(map[string]bool{"valid": true})
}

func configFlags(name string, args []string) (*flag.FlagSet, string, string, string, string, error) {
	f := flag.NewFlagSet(name, flag.ContinueOnError)
	f.SetOutput(io.Discard)
	pointer := f.String(flagPointer, "", "RFC 6901 JSON Pointer")
	valueJSON := f.String(flagValueJSON, "", "JSON literal")
	valueFile := f.String(flagValueFile, "", "JSON value file or - for stdin")
	expected := f.String(flagExpectedSHA256, "", "current configuration SHA-256")
	if f.Parse(args) != nil || f.NArg() != 0 {
		return nil, "", "", "", "", fault.New(fault.Invalid, "config arguments")
	}
	return f, *pointer, *valueJSON, *valueFile, *expected, nil
}

func (a application) configGet(args []string) (int, error) {
	f, pointer, _, _, _, err := configFlags("config get", args)
	if err != nil {
		return 0, err
	}
	if !readFlagsOnly(f) || !flagWasSet(f, flagPointer) {
		return 0, fault.New(fault.Invalid, errorConfigPointer)
	}
	document, digest, err := a.readConfigDocument()
	if err != nil {
		return 0, err
	}
	value, err := jsonPointerGet(document, pointer)
	if err != nil {
		return 0, err
	}
	return a.print(map[string]any{"pointer": pointer, "sha256": digest, "value": value})
}

func (a application) configList(args []string) (int, error) {
	f, pointer, _, _, _, err := configFlags("config list", args)
	if err != nil {
		return 0, err
	}
	if !readFlagsOnly(f) {
		return 0, fault.New(fault.Invalid, "config list arguments")
	}
	document, digest, err := a.readConfigDocument()
	if err != nil {
		return 0, err
	}
	value, err := jsonPointerGet(document, pointer)
	if err != nil {
		return 0, err
	}
	entries, err := configEntries(value)
	if err != nil {
		return 0, err
	}
	return a.print(map[string]any{"entries": entries, "pointer": pointer, "sha256": digest})
}

func (a application) configSet(ctx context.Context, args []string) (int, error) {
	f, pointer, valueJSON, valueFile, expected, err := configFlags("config set", args)
	if err != nil {
		return 0, err
	}
	if !flagWasSet(f, flagPointer) {
		return 0, fault.New(fault.Invalid, errorConfigPointer)
	}
	value, err := a.configValue(valueJSON, valueFile, flagWasSet(f, flagValueJSON), flagWasSet(f, flagValueFile))
	if err != nil {
		return 0, err
	}
	return a.mutateConfig(ctx, expected, false, false, func(document *any) error { return jsonPointerSet(document, pointer, value) })
}

func (a application) configDelete(ctx context.Context, args []string) (int, error) {
	f, pointer, _, _, expected, err := configFlags("config delete", args)
	if err != nil {
		return 0, err
	}
	if flagWasSet(f, flagValueJSON) || flagWasSet(f, flagValueFile) || !flagWasSet(f, flagPointer) {
		return 0, fault.New(fault.Invalid, "config delete arguments")
	}
	return a.mutateConfig(ctx, expected, false, false, func(document *any) error { return jsonPointerDelete(document, pointer) })
}

func (a application) configReplace(ctx context.Context, args []string) (int, error) {
	f := flag.NewFlagSet("config replace", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	file := f.String("file", "", "complete JSON policy file or - for stdin")
	expected := f.String(flagExpectedSHA256, "", "current configuration SHA-256")
	if f.Parse(args) != nil || f.NArg() != 0 || *file == "" {
		return 0, fault.New(fault.Invalid, "config replace arguments")
	}
	data, err := readConfigInput(a.runner.In, *file)
	if err != nil {
		return 0, err
	}
	value, err := strictJSON(data)
	if err != nil {
		return 0, err
	}
	return a.mutateConfig(ctx, *expected, true, true, func(document *any) error { *document = value; return nil })
}

func (a application) configValue(valueJSON, valueFile string, hasJSON, hasFile bool) (any, error) {
	if hasJSON == hasFile {
		return nil, fault.New(fault.Invalid, "one config value required")
	}
	if hasJSON {
		if len(valueJSON) > maximumConfigInput {
			return nil, fault.New(fault.Invalid, "config input size")
		}
		return strictJSON([]byte(valueJSON))
	}
	data, err := readConfigInput(a.runner.In, valueFile)
	if err != nil {
		return nil, err
	}
	return strictJSON(data)
}

func readConfigInput(in io.Reader, path string) ([]byte, error) {
	var reader io.Reader
	if path != "-" {
		abs, err := filepath.Abs(path)
		if err != nil {
			return nil, fault.New(fault.Invalid, "config input")
		}
		data, err := platform.ReadProtected(abs, false)
		if err != nil || len(data) > maximumConfigInput {
			return nil, fault.New(fault.Invalid, "config input")
		}
		return data, nil
	}
	reader = in
	data, err := io.ReadAll(io.LimitReader(reader, maximumConfigInput+1))
	if err != nil || len(data) > maximumConfigInput {
		return nil, fault.New(fault.Invalid, "config input size")
	}
	return data, nil
}

func strictJSON(data []byte) (any, error) {
	if !bytes.HasPrefix(bytes.TrimSpace(data), []byte("{")) && !bytes.HasPrefix(bytes.TrimSpace(data), []byte("[")) && !jsonPrimitive(data) {
		return nil, fault.New(fault.Invalid, errorConfigJSON)
	}
	var checked any
	if jsondocument.Decode(data, &checked) != nil {
		return nil, fault.New(fault.Invalid, errorConfigJSON)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	if decoder.Decode(&value) != nil || decoder.More() {
		return nil, fault.New(fault.Invalid, errorConfigJSON)
	}
	return value, nil
}

func jsonPrimitive(data []byte) bool {
	text := strings.TrimSpace(string(data))
	return text == "true" || text == "false" || text == "null" || strings.HasPrefix(text, `"`) || (text != "" && (text[0] == '-' || (text[0] >= '0' && text[0] <= '9')))
}

func (a application) readConfigDocument() (any, string, error) {
	path, err := filepath.Abs(a.config)
	if err != nil {
		return nil, "", fault.New(fault.Invalid, errorConfigPath)
	}
	data, err := platform.ReadProtected(path, false)
	if err != nil {
		return nil, "", err
	}
	if len(data) > maximumConfigInput {
		return nil, "", fault.New(fault.Invalid, "config size")
	}
	value, err := strictJSON(data)
	if err != nil {
		return nil, "", err
	}
	if _, err = configuration.ReadRoot(data, path); err != nil {
		return nil, "", err
	}
	return value, configDigest(data), nil
}

func (a application) mutateConfig(ctx context.Context, expected string, create, repair bool, mutate func(*any) error) (int, error) {
	path, err := filepath.Abs(a.config)
	if err != nil {
		return 0, fault.New(fault.Invalid, errorConfigPath)
	}
	unlock, err := configLock(path)
	if err != nil {
		return 0, err
	}
	defer unlock()
	document, digest, err := a.mutationDocument(path, create, repair)
	if err != nil {
		return 0, err
	}
	if err = validateExpectedDigest(expected, digest); err != nil {
		return 0, err
	}
	if err = mutate(&document); err != nil {
		return 0, err
	}
	data, err := a.marshalConfigCandidate(ctx, document, path)
	if err != nil {
		return 0, err
	}
	if err = a.ensureConfigUnchanged(path, digest, repair); err != nil {
		return 0, err
	}
	if err = configurationWriteContext(ctx); err != nil {
		return 0, err
	}
	if err = platform.WriteProtected(path, data); err != nil {
		return 0, err
	}
	return a.print(map[string]string{"sha256": configDigest(data), "state": "updated"})
}

func (a application) mutationDocument(path string, create, repair bool) (any, string, error) {
	document, digest, err := a.readConfigDocument()
	if err == nil {
		return document, digest, nil
	}
	if repair {
		digest, rawErr := a.rawConfigDigest(path)
		if rawErr == nil {
			return map[string]any{}, digest, nil
		}
		if create && fault.Is(rawErr, fault.Missing) {
			return map[string]any{}, "", nil
		}
		return nil, "", rawErr
	}
	if create && fault.Is(err, fault.Missing) {
		return map[string]any{}, "", nil
	}
	return nil, "", err
}

func (a application) rawConfigDigest(path string) (string, error) {
	data, err := platform.ReadProtected(path, false)
	if err != nil {
		return "", err
	}
	if len(data) > maximumConfigInput {
		return "", fault.New(fault.Invalid, "config size")
	}
	return configDigest(data), nil
}

func validateExpectedDigest(expected, digest string) error {
	if expected == "" {
		return nil
	}
	if len(expected) != 64 || !validDigest(expected) || expected != digest {
		return fault.New(fault.Denied, errorConfigChanged)
	}
	return nil
}

func (a application) marshalConfigCandidate(ctx context.Context, document any, path string) ([]byte, error) {
	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil || len(data) > maximumConfigInput {
		return nil, fault.New(fault.Invalid, "config document")
	}
	data = append(data, '\n')
	if _, err = a.resolvePolicyBytes(ctx, data, path); err != nil {
		return nil, err
	}
	return data, nil
}

func (a application) ensureConfigUnchanged(path, digest string, repair bool) error {
	if digest == "" {
		return nil
	}
	_, current, err := a.readConfigDocument()
	if repair {
		current, err = a.rawConfigDigest(path)
	}
	if err != nil || current != digest {
		return fault.New(fault.Denied, errorConfigChanged)
	}
	return nil
}

func configLock(path string) (func() error, error) {
	store := state.Store{Directory: filepath.Join(filepath.Dir(path), ".cq-config-lock")}
	return store.Lock("policy")
}

func configDigest(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
func validDigest(value string) bool   { _, err := hex.DecodeString(value); return err == nil }
func readFlagsOnly(flags *flag.FlagSet) bool {
	return !flagWasSet(flags, flagValueJSON) && !flagWasSet(flags, flagValueFile) && !flagWasSet(flags, flagExpectedSHA256)
}

func flagWasSet(flags *flag.FlagSet, name string) bool {
	set := false
	flags.Visit(func(item *flag.Flag) { set = set || item.Name == name })
	return set
}

func configEntries(value any) ([]string, error) {
	switch item := value.(type) {
	case map[string]any:
		entries := make([]string, 0, len(item))
		for key := range item {
			entries = append(entries, key)
		}
		sort.Strings(entries)
		return entries, nil
	case []any:
		entries := make([]string, len(item))
		for index := range item {
			entries[index] = strconv.Itoa(index)
		}
		return entries, nil
	default:
		return nil, fault.New(fault.Invalid, "config list target")
	}
}

func jsonPointerGet(value any, pointer string) (any, error) {
	parts, err := parsePointer(pointer)
	if err != nil {
		return nil, err
	}
	for _, part := range parts {
		switch node := value.(type) {
		case map[string]any:
			var found bool
			value, found = node[part]
			if !found {
				return nil, fault.New(fault.Missing, errorConfigPointer)
			}
		case []any:
			index, err := pointerIndex(part, len(node))
			if err != nil {
				return nil, err
			}
			value = node[index]
		default:
			return nil, fault.New(fault.Invalid, errorConfigPointer)
		}
	}
	return value, nil
}

func jsonPointerSet(document *any, pointer string, replacement any) error {
	parts, err := parsePointer(pointer)
	if err != nil {
		return err
	}
	if len(parts) == 0 {
		*document = replacement
		return nil
	}
	parent, key, err := jsonPointerParent(*document, parts)
	if err != nil {
		return err
	}
	switch node := parent.(type) {
	case map[string]any:
		node[key] = replacement
	case []any:
		index, err := pointerIndex(key, len(node))
		if err != nil {
			return err
		}
		node[index] = replacement
	default:
		return fault.New(fault.Invalid, errorConfigPointer)
	}
	return nil
}

func jsonPointerDelete(document *any, pointer string) error {
	parts, err := parsePointer(pointer)
	if err != nil || len(parts) == 0 {
		return fault.New(fault.Invalid, errorConfigPointer)
	}
	parent, key, err := jsonPointerParent(*document, parts)
	if err != nil {
		return err
	}
	switch node := parent.(type) {
	case map[string]any:
		if _, found := node[key]; !found {
			return fault.New(fault.Missing, errorConfigPointer)
		}
		delete(node, key)
	case []any:
		index, err := pointerIndex(key, len(node))
		if err != nil {
			return err
		}
		copy(node[index:], node[index+1:])
		node[len(node)-1] = nil
		replacement := node[:len(node)-1]
		return replaceArray(document, parts[:len(parts)-1], replacement)
	default:
		return fault.New(fault.Invalid, errorConfigPointer)
	}
	return nil
}

func replaceArray(document *any, parentParts []string, replacement []any) error {
	if len(parentParts) == 0 {
		*document = replacement
		return nil
	}
	parent, key, err := jsonPointerParent(*document, parentParts)
	if err != nil {
		return err
	}
	switch node := parent.(type) {
	case map[string]any:
		node[key] = replacement
	case []any:
		index, err := pointerIndex(key, len(node))
		if err != nil {
			return err
		}
		node[index] = replacement
	default:
		return fault.New(fault.Invalid, errorConfigPointer)
	}
	return nil
}

func jsonPointerParent(document any, parts []string) (any, string, error) {
	parent, err := jsonPointerGet(document, joinPointer(parts[:len(parts)-1]))
	if err != nil {
		return nil, "", err
	}
	return parent, parts[len(parts)-1], nil
}
func joinPointer(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	escaped := make([]string, len(parts))
	for i, part := range parts {
		escaped[i] = strings.ReplaceAll(strings.ReplaceAll(part, "~", "~0"), "/", "~1")
	}
	return "/" + strings.Join(escaped, "/")
}
func parsePointer(pointer string) ([]string, error) {
	if pointer == "" {
		return nil, nil
	}
	if !strings.HasPrefix(pointer, "/") {
		return nil, fault.New(fault.Invalid, errorConfigPointer)
	}
	raw := strings.Split(pointer[1:], "/")
	for i := range raw {
		value, err := unescapePointer(raw[i])
		if err != nil {
			return nil, err
		}
		raw[i] = value
	}
	return raw, nil
}
func unescapePointer(value string) (string, error) {
	var b strings.Builder
	for i := 0; i < len(value); i++ {
		if value[i] != '~' {
			b.WriteByte(value[i])
			continue
		}
		if i+1 == len(value) || (value[i+1] != '0' && value[i+1] != '1') {
			return "", fault.New(fault.Invalid, errorConfigPointer)
		}
		if value[i+1] == '0' {
			b.WriteByte('~')
		} else {
			b.WriteByte('/')
		}
		i++
	}
	return b.String(), nil
}
func pointerIndex(value string, length int) (int, error) {
	if !canonicalPointerIndex(value) {
		return 0, fault.New(fault.Invalid, "config index")
	}
	index, err := strconv.Atoi(value)
	if err != nil || index < 0 || index >= length {
		return 0, fault.New(fault.Missing, "config index")
	}
	return index, nil
}

func canonicalPointerIndex(value string) bool {
	if value == "" || value == "-" || (len(value) > 1 && value[0] == '0') {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

// formatConfig atomically rewrites an already-valid JSON declaration in the
// canonical representation. It refuses YAML, missing, unsafe, and invalid files.
func (a application) formatConfig() (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return a.formatConfigContext(ctx)
}

func (a application) formatConfigContext(ctx context.Context) (bool, error) {
	path, err := filepath.Abs(a.config)
	if err != nil {
		return false, fault.New(fault.Invalid, errorConfigPath)
	}
	unlock, err := configLock(path)
	if err != nil {
		return false, err
	}
	defer unlock()
	document, digest, err := a.readConfigDocument()
	if err != nil {
		return false, err
	}
	data, err := a.marshalConfigCandidate(ctx, document, path)
	if err != nil {
		return false, err
	}
	current, err := platform.ReadProtected(path, false)
	if err != nil || configDigest(current) != digest {
		return false, fault.New(fault.Denied, errorConfigChanged)
	}
	if bytes.Equal(current, data) {
		return false, nil
	}
	if err = configurationWriteContext(ctx); err != nil {
		return false, err
	}
	if err = platform.WriteProtected(path, data); err != nil {
		return false, err
	}
	return true, nil
}

func configurationWriteContext(ctx context.Context) error {
	if ctx.Err() != nil {
		return fault.New(fault.Unavailable, "configuration cancelled")
	}
	return nil
}
