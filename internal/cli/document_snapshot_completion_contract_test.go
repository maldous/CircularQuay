package cli

import (
	"bytes"
	"context"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/maldous/CircularQuay/internal/bootstrap"
 "github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/observability"
 "github.com/maldous/CircularQuay/internal/platform"
	"github.com/maldous/CircularQuay/internal/resolve"
	"github.com/maldous/CircularQuay/internal/state"
)

type cliCompletionSnapshot struct { t *testing.T; count int }
func (s *cliCompletionSnapshot) Encrypt(_ context.Context, snapshot model.Snapshot, values map[string]string) error {
	s.count++
	if snapshot.Path == "" || len(values) != 1 || values["cq_snapshot"] == "" { s.t.Fatal("snapshot envelope changed") }
	return nil
}
func (s *cliCompletionSnapshot) Decrypt(context.Context, model.Snapshot) (map[string]string,error) {
	s.t.Fatal("snapshot creation selected restore")
	return nil, nil
}

func TestCLICompletionActualSnapshotAndApplyDispatcherRetainState(t *testing.T) {
 app:=application{directory:filepath.Join(privateCLIDirectory(t),"state"),runner:Runner{Out:io.Discard}}
 workload:=model.Workload{Mode:"offline",Snapshot:model.Snapshot{Path:"owned-envelope"}}
 backend:=&cliCompletionSnapshot{t:t}
 engine:=&resolve.Engine{Policy:model.Policy{Version:1,Workloads:map[string]model.Workload{"demo":workload}},Snapshots:backend,State:state.Store{Directory:app.directory}}
 generation:=cliCompletionGeneration()
 code,err:=app.completeWorkloadCommand(context.Background(),workloadRequest{command:"snapshot",id:"demo"},workload,engine,generation)
 cliCompletionSnapshotDispatchResult(t,code,err,backend.count)
 record:=cliCompletionStoredPhase(t,engine.State,"snapshotted")
 cliCompletionGenerationIdentifier(t,record.Generation)
 // Empty targets persist metadata without any native destination/activation.
 code,err=app.completeWorkloadCommand(context.Background(),workloadRequest{command:"apply",id:"demo"},workload,engine,generation)
 cliCompletionSnapshotDispatchResult(t,code,err,backend.count)
 cliCompletionStoredPhase(t,engine.State,"resolved")
}

func TestCLICompletionActualSnapshotFailurePreventsStateReceipt(t *testing.T) {
	app := application{runner:Runner{Out:io.Discard}}
	engine := &resolve.Engine{Policy:model.Policy{}, State:state.Store{Directory:filepath.Join(privateCLIDirectory(t),"unused-state")}}
	code, err := app.completeWorkloadCommand(context.Background(),workloadRequest{command:"snapshot",id:"demo"},model.Workload{},engine,cliCompletionGeneration())
	cliCompletionFault(t,err,fault.Denied,"snapshot policy")
	if code != 0 { t.Fatal("snapshot refusal changed inner code") }
	if _, err := os.Lstat(engine.State.Directory); !os.IsNotExist(err) { t.Fatal("refused snapshot created receipt") }
}

func TestCLICompletionActualSnapshotSuccessCannotHideStateWriteFailure(t *testing.T) {
	root := privateCLIDirectory(t)
	occupied := filepath.Join(root,"occupied")
	if err := platform.WriteProtected(occupied,[]byte("owned")); err != nil { t.Fatal(err) }
	workload := model.Workload{Mode:"offline",Snapshot:model.Snapshot{Path:"owned-envelope"}}
	backend := &cliCompletionSnapshot{t:t}
	engine := &resolve.Engine{Policy:model.Policy{Version:1,Workloads:map[string]model.Workload{"demo":workload}},Snapshots:backend,State:state.Store{Directory:occupied}}
	code, err := (application{runner:Runner{Out:io.Discard}}).storeSnapshot(context.Background(),workloadRequest{command:"snapshot",id:"demo"},workload,engine,cliCompletionGeneration())
	if err == nil || code != 0 || backend.count != 1 { t.Fatal("state write failure hid actual prior snapshot callback") }
	data, readErr := os.ReadFile(occupied)
	if readErr != nil || string(data) != "owned" { t.Fatal("state write failure changed occupied parent") }
}

func TestCLICompletionConfigurationArrayDeletionAndRepairCreation(t *testing.T) {
	for _, test := range []struct { original any; pointer string; expected any }{
		{[]any{"a","b"},"/0",[]any{"b"}},
		{[]any{[]any{"a","b"}},"/0/1",[]any{[]any{"a"}}},
		{map[string]any{"a":[]any{"a","b"}},"/a/0",map[string]any{"a":[]any{"b"}}},
	} {
		document := test.original
		if err := jsonPointerDelete(&document,test.pointer); err != nil || !reflect.DeepEqual(document,test.expected) { t.Fatalf("array deletion failed: %v",err) }
	}
	path := filepath.Join(privateCLIDirectory(t),"new.json")
	app := application{config:path}
	document,digest,err := app.mutationDocument(path,true,false)
	if err != nil || digest != "" || !reflect.DeepEqual(document,map[string]any{}) { t.Fatal("new configuration did not begin as empty document") }
	if _, err := os.Lstat(path); !os.IsNotExist(err) { t.Fatal("document preparation materialized file") }
}

func TestCLICompletionBootstrapReadAndOutputFailuresAreBounded(t *testing.T) {
	app, original := cliCompletionConfig(t)
	app.runner.Out = cliCompletionFailWriter{}
	code, err := app.validateLocalBootstrap(context.Background(),app.config)
	cliCompletionFault(t,err,fault.Apply,"output")
	data, readErr := os.ReadFile(app.config)
	if code != 0 || readErr != nil || !bytes.Equal(data,original) { t.Fatal("refused receipt changed validated declaration") }
	if err := apply(context.Background(),model.Workload{},"missing",model.Generation{}); !fault.Is(err,fault.Denied) { t.Fatal("unmapped target was admitted") }
	// An unsupported declared delivery kind refuses without a transport.
	workload := model.Workload{Targets:map[string]model.Target{"a":{Type:"unsupported"}}}
	if err := apply(context.Background(),workload,"a",model.Generation{}); err == nil { t.Fatal("unsupported delivery target was admitted") }
}

func TestCLICompletionBootstrapLateCollisionPreservesExactOwnedFile(t *testing.T) {
	root := privateCLIDirectory(t)
	output := filepath.Join(root,"output.json")
	app := application{config:output, runner:Runner{Out:io.Discard}}
	fetches := 0
	code,err := app.bootstrapWithFetcher(context.Background(),[]string{"--config-source","https://github.com/acme/cq/tree/"+bootstrapRef,"--config-ref",bootstrapRef,"--config-sha256",bootstrapDigest},func(context.Context,bootstrap.Request)(bootstrap.Result,error){
		fetches++
		if err:=platform.CreateProtectedExclusive(output,bootstrapPolicy());err!=nil {t.Fatal(err)}
		return bootstrap.Result{Bytes:bootstrapPolicy()},nil
	})
	if code!=0 || err==nil || fetches!=1 {t.Fatal("exclusive output collision was not refused")}
	data,readErr:=os.ReadFile(output)
	if readErr!=nil || !bytes.Equal(data,bootstrapPolicy()) {t.Fatal("collision overwrote owned declaration")}
}

func TestCLICompletionOperationalInitializationRefusalsStopDispatch(t *testing.T) {
	for _,phase:=range []string{"runtime","updates"} {
		cliCompletionOperationalRefusal(t,phase)
	}
}
func cliCompletionOperationalRefusal(t *testing.T,phase string) {
	t.Helper()
	refusal:=fault.New(fault.Apply,"owned "+phase)
	var calls []string
	app:=application{runner:Runner{Out:io.Discard,Err:io.Discard}}
	code,err:=app.dispatchLoadedOperationalPolicyWithOperations(context.Background(),[]string{"unknown"},model.Policy{},func(context.Context,model.Policy)(*observability.Runtime,error){
		calls=append(calls,"runtime")
		if phase=="runtime" {return nil,refusal}
		return observability.Disabled(),nil
	},func(context.Context,string,*model.Updates)error{calls=append(calls,"updates");return refusal})
	want:=[]string{"runtime"}
	if phase=="updates" {want=append(want,"updates")}
	if code!=0 || err!=refusal || !reflect.DeepEqual(calls,want) {t.Fatal("initialization failure reached runtime command")}
}

func cliCompletionSnapshotDispatchResult(t *testing.T,code int,err error,encryptions int) {
 t.Helper()
 if err!=nil || code!=0 || encryptions!=1 {t.Fatalf("snapshot/apply dispatch failed: %v",err)}
}
func cliCompletionStoredPhase(t *testing.T,store state.Store,phase string) state.Record {
 t.Helper()
 record,err:=store.Read("demo")
 if err!=nil || record.State!=phase || record.Workload!="demo" {t.Fatal("exact owned phase was not durable")}
 return record
}
func cliCompletionGenerationIdentifier(t *testing.T,value string) {
 t.Helper()
 decoded,err:=hex.DecodeString(value)
 if err!=nil || len(decoded)!=16 {t.Fatal("real generation identifier changed")}
}
