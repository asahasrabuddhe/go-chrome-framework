package chrome

import (
	"github.com/mafredri/cdp/protocol/target"
	"time"
)

type Browser interface {
	Launch(path string, port *int, arguments []*string) (BrowserTab, error)
	Wait()
	Terminate() error
	OpenTab(targetID target.ID, timeout time.Duration) BrowserTab
	OpenNewTab(timeout time.Duration) (BrowserTab, error)
	OpenNewIncognitoTab(timeout time.Duration) (BrowserTab, error)
	CloseTab(tab BrowserTab, timeout time.Duration) error
}
