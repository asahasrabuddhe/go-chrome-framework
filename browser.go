package chrome

import (
	"github.com/mafredri/cdp/protocol/target"
	"time"
)

type Browser interface {
	Launch(LaunchOpts) (BrowserTab, error)
	Wait()
	Terminate() error
	OpenTab(target.ID, time.Duration) BrowserTab
	OpenNewTab(time.Duration) (BrowserTab, error)
	OpenNewIncognitoTab(time.Duration) (BrowserTab, error)
	CloseTab(BrowserTab, time.Duration) error
}
