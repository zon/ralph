package github

import (
	"fmt"
	"time"
)

func WaitForHeadSync(gh GHClient, prBranch string) error {
	for i := 0; i < 30; i++ {
		oid, err := gh.GetPRHeadRefOid(prBranch)
		if err == nil && oid != "" {
			return nil
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("head sync timeout for branch %s", prBranch)
}
