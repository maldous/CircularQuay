package state

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
)

func TestAtomicMetadataAndProcessLock(t *testing.T){
	s:=Store{Directory:filepath.Join(t.TempDir(),"state")}
	r,err:=s.Read("demo");if err!=nil||r.State!="not_provisioned"{t.Fatal("missing state")}
	if err=s.Write(Record{Workload:"demo",State:"ready",Mode:"offline"});err!=nil{t.Fatal(err)}
	r,err=s.Read("demo");if err!=nil||r.State!="ready"{t.Fatal("metadata roundtrip")}
	unlock,err:=s.Lock("demo");if err!=nil{t.Fatal(err)}
	if _,err=s.Lock("demo");!fault.Is(err,fault.Unavailable){t.Fatal("concurrent reconciliation allowed")}
	if err=unlock();err!=nil{t.Fatal(err)}
	unlock,err=s.Lock("demo");if err!=nil{t.Fatal("OS lock not released")};if err=unlock();err!=nil{t.Fatal(err)}
}

func TestInvalidAndCorruptState(t *testing.T){
	s:=Store{Directory:t.TempDir()}
	for _,name:=range []string{"../escape","",strings.Repeat("x",65)}{if _,err:=s.Read(name);!fault.Is(err,fault.Invalid){t.Fatal("invalid name accepted")}}
	if _,err:=(Store{}).Read("demo");!fault.Is(err,fault.Invalid){t.Fatal("empty state directory")}
	p:=filepath.Join(s.Directory,"demo.json")
	for _,body:=range []string{"not-json",`{"workload":"other"}`,`{"workload":"demo","extra":"field"}`} {if err:=os.WriteFile(p,[]byte(body),0o600);err!=nil{t.Fatal(err)};if _,err:=s.Read("demo");!fault.Is(err,fault.Trust){t.Fatal("corrupt state accepted")}}
	if err:=os.WriteFile(p,make([]byte,(1<<20)+1),0o600);err!=nil{t.Fatal(err)};if _,err:=s.Read("demo");!fault.Is(err,fault.Trust){t.Fatal("oversized state accepted")}
}

func TestUnsafeStateDirectory(t *testing.T){
	d:=t.TempDir();p:=filepath.Join(d,"file");if err:=os.WriteFile(p,[]byte("x"),0o600);err!=nil{t.Fatal(err)}
	if _,err:=(Store{Directory:p}).Read("demo");err==nil{t.Fatal("non-directory accepted")}
	if err:=os.Chmod(d,0o777);err!=nil{t.Fatal(err)};if _,err:=(Store{Directory:d}).Read("demo");!fault.Is(err,fault.Trust){t.Fatal("writable state directory accepted")}
}
