package chrome

import (
	"github.com/mafredri/cdp"
	"github.com/mafredri/cdp/protocol/runtime"
	"github.com/mafredri/cdp/protocol/target"
	"time"
)

type BrowserTab interface {
	connect(timeout time.Duration) error
	disconnect() error
	Navigate(url string, timeout time.Duration) (bool, error)
	GetHTML(timeout time.Duration) (string, error)
	CaptureScreenshot(timeout time.Duration) (string, error)
	Exec(javascript string, timeout time.Duration) (*runtime.EvaluateReply, error)
	GetClient() *cdp.Client
	GetTargetID() target.ID
	AttachHook(hook ClientHook)
}
