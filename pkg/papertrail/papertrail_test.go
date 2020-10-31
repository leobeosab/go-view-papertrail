package papertrail_test

import (
	"testing"

	"github.com/leobeosab/go-view-papertrail/pkg/papertrail"
)

//TestSendPapertrailRequest test to make sure we can get papertrail results
func TestSendPapertrailRequest(t *testing.T) {
	papertrail.Init()

	result := papertrail.GetLogs("")

	t.Log(result)
}
