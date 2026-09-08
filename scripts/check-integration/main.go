package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		panic("usage: check-integration <go test -json log>")
	}
	f, err := os.Open(os.Args[1])
	if err != nil {
		panic(err)
	}
	defer f.Close()
	d := json.NewDecoder(f)
	required := map[string]bool{"TestReviewNotifyInsertFailureMustRollbackPayment": false, "TestReviewRefundReservationConcurrency": false, "TestNotifyJobWaitsForTenantWork": false, "TestPaymentClientTenantBoundary": false}
	for {
		var e struct{ Action, Test, Package string }
		if err := d.Decode(&e); err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}
		if e.Action == "fail" || (e.Action == "skip" && e.Test != "") {
			panic(fmt.Sprintf("integration acceptance rejected: %s %s %s", e.Action, e.Package, e.Test))
		}
		if e.Action == "pass" {
			if _, ok := required[e.Test]; ok {
				required[e.Test] = true
			}
		}
	}
	for name, passed := range required {
		if !passed {
			panic("required integration test did not pass: " + name)
		}
	}
	fmt.Println("Integration acceptance: no skipped or failed tests; required PostgreSQL/Redis cases passed")
}
