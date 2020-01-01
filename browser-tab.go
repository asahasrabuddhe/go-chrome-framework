package chrome

import (
	"github.com/mafredri/cdp"
	"github.com/mafredri/cdp/protocol/runtime"
	"github.com/mafredri/cdp/protocol/target"
	"time"
)

type BrowserTab interface {
	Navigate(url string, timeout time.Duration) (bool, error)
	GetHTML(timeout time.Duration) (string, error)
	CaptureScreenshot(opts ScreenshotOpts, timeout time.Duration) (string, error)
	Exec(javascript string, timeout time.Duration) (*runtime.EvaluateReply, error)
	GetClient() *cdp.Client
	GetTargetID() target.ID
	AttachHook(hook ClientHook)
}

type ClientHook func(c *cdp.Client) error

type ClientHooks []ClientHook

type ScreenshotOpts struct {
	Width             int
	Height            int
	DeviceScaleFactor float64
	Mobile            bool
}
