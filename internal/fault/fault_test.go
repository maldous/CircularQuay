package fault

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestSanitizedClassification(t *testing.T){
	e:=New(Denied,"fixture")
	if !Is(fmt.Errorf("wrapper: %w",e),Denied)||Category(e)!=Denied||Category(errors.New("untrusted fixture"))!=Apply{t.Fatal("classification")}
	if strings.Contains(e.Error(),"untrusted")||Is(errors.New("access_denied"),Denied){t.Fatal("text-based classification")}
}
