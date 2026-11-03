package cli

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"sort"
	"time"

	"github.com/maldous/CircularQuay/internal/deliver"
	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/platform"
	"github.com/maldous/CircularQuay/internal/state"
)

func (a application) initialize(args []string) (int, error) {
	f := flag.NewFlagSet("init", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	out := f.String("output", a.config, "policy output")
	if f.Parse(args) != nil || f.NArg() != 0 {
		return 0, fault.New(fault.Invalid, "init arguments")
	}
	p, err := filepath.Abs(*out)
	if err != nil {
		return 0, fault.New(fault.Invalid, "init path")
	}
	fixture := filepath.Base(p) + ".fixture.json"
	config := model.Policy{Version: 1, Sources: map[string]model.Source{"demo": {Type: "json", Path: fixture}}, Workloads: map[string]model.Workload{"demo": {Mode: "offline", Bindings: []model.Binding{{Name: "DEMO_API_KEY", Source: "demo", Field: "DEMO_API_KEY", AllowOffline: true}}}}}
	b, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return 0, fault.New(fault.Apply, "init encode")
	}
	if _, err = os.Lstat(p); !os.IsNotExist(err) {
		return 0, fault.New(fault.Trust, "existing policy")
	}
	fixturePath := filepath.Join(filepath.Dir(p), fixture)
	if _, err = os.Lstat(fixturePath); !os.IsNotExist(err) {
		return 0, fault.New(fault.Trust, "existing fixture")
	}
	if err = os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return 0, fault.New(fault.Apply, "init directory")
	}
	if err = writeNew(fixturePath, []byte("{\"DEMO_API_KEY\":\"synthetic-only-not-a-credential\"}\n")); err != nil {
		return 0, err
	}
	if err = writeNew(p, b); err != nil {
		return 0, err
	}
	return a.print(map[string]string{"state": "initialized", "policy": p, "credentials": "synthetic fixture only"})
}

func writeNew(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fault.New(fault.Apply, "create file")
	}
	_, w := f.Write(data)
	c := f.Close()
	if w != nil || c != nil {
		return fault.New(fault.Apply, "write file")
	}
	return nil
}

func (a application) doctor() (int, error) {
	tools := map[string]bool{}
	for _, name := range []string{"sops", "git", "systemd-creds", "bws", "op"} {
		_, err := exec.LookPath(name)
		tools[name] = err == nil
	}
	subject := "unavailable"
	if u, err := user.Current(); err == nil {
		subject = "local:" + u.Username
	}
	return a.print(map[string]any{"os": runtime.GOOS, "arch": runtime.GOARCH, "local_subject": subject, "tools": tools, "runtime_required": false, "qualification": "see audit evidence; no native reboot qualification implied"})
}

func (a application) plan() (int, error) {
	p, err := a.load()
	if err != nil {
		return 0, err
	}
	type entry struct {
		ID       string `json:"id"`
		Mode     string `json:"mode"`
		Bindings int    `json:"bindings"`
		Targets  int    `json:"targets"`
	}
	entries := []entry{}
	for id, w := range p.Workloads {
		entries = append(entries, entry{id, w.Mode, len(w.Bindings), len(w.Targets)})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].ID < entries[j].ID })
	return a.print(map[string]any{"valid": true, "value_requests": 0, "workloads": entries})
}

func workloadFlags(command string, args []string) (string, string, []string, error) {
	f := flag.NewFlagSet(command, flag.ContinueOnError)
	f.SetOutput(io.Discard)
	id := f.String("workload", "", "workload ID")
	target := f.String("target", "", "target ID")
	if f.Parse(args) != nil || *id == "" {
		return "", "", nil, fault.New(fault.Invalid, "workload arguments")
	}
	return *id, *target, f.Args(), nil
}

func (a application) status(args []string) (int, error) {
	f := flag.NewFlagSet("status", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	id := f.String("workload", "", "workload ID")
	_ = f.Bool("json", false, "JSON output")
	if f.Parse(args) != nil || *id == "" || f.NArg() != 0 {
		return 0, fault.New(fault.Invalid, "status arguments")
	}
	p, err := a.load()
	if err != nil {
		return 0, err
	}
	if _, ok := p.Workloads[*id]; !ok {
		return 0, fault.New(fault.Invalid, "workload")
	}
	s, err := a.store()
	if err != nil {
		return 0, err
	}
	r, err := s.Read(*id)
	if err != nil {
		return 0, err
	}
	return a.print(r)
}

func (a application) workloadCommand(ctx context.Context, args []string) (int, error) {
	id, target, override, err := workloadFlags(args[0], args[1:])
	if err != nil {
		return 0, err
	}
	if args[0] != "run" && len(override) > 0 {
		return 0, fault.New(fault.Invalid, "unexpected arguments")
	}
	p, err := a.load()
	if err != nil {
		return 0, err
	}
	w, ok := p.Workloads[id]
	if !ok {
		return 0, fault.New(fault.Invalid, "workload")
	}
	e, err := a.engine(p)
	if err != nil {
		return 0, err
	}
	unlock, err := e.State.Lock(id)
	if err != nil {
		return 0, err
	}
	defer unlock()
	var g model.Generation
	if args[0] == "snapshot" || args[0] == "sync" {
		g, err = e.ResolveFresh(ctx, id)
	} else {
		g, err = e.Resolve(ctx, id)
	}
	if err != nil {
		return 0, err
	}
	switch args[0] {
	case "run":
		if err = recordReady(e.State, w, g); err != nil {
			return 0, err
		}
		return platform.RunWithIO(ctx, w, g, override, a.runner.In, a.runner.Out, a.runner.Err)
	case "snapshot":
		err = e.Snapshot(ctx, g)
	case "apply":
		err = apply(ctx, w, target, g)
	case "sync":
		if w.Snapshot.Path != "" && w.Mode != "online" {
			err = e.Snapshot(ctx, g)
		}
	}
	if err != nil {
		return 0, err
	}
	if err = recordReady(e.State, w, g); err != nil {
		return 0, err
	}
	return a.print(map[string]any{"workload": id, "state": "ready", "operation": args[0], "offline": g.Offline, "bindings": len(g.Values)})
}

func recordReady(s state.Store, w model.Workload, g model.Generation) error {
	id := make([]byte, 16)
	if _, err := rand.Read(id); err != nil {
		return fault.New(fault.Apply, "generation identifier")
	}
	return s.Write(state.Record{Workload: g.Workload, Generation: hex.EncodeToString(id), State: "ready", Mode: w.Mode, Offline: g.Offline, LastChecked: time.Now().UTC(), LastSuccess: g.RetrievedAt, Receipts: g.Receipts})
}

func apply(ctx context.Context, w model.Workload, id string, g model.Generation) error {
	t, ok := w.Targets[id]
	if !ok {
		return fault.New(fault.Denied, "target mapping")
	}
	for _, b := range w.Bindings {
		if !b.AllowPublish {
			return fault.New(fault.Denied, "publication permission")
		}
	}
	switch t.Type {
	case "file", "systemd", "windows":
		return platform.Apply(ctx, t, g)
	default:
		return deliver.Apply(ctx, t, g)
	}
}
