package chrome

import (
	"github.com/mafredri/cdp/protocol/target"
	"time"
)

type Browser interface {
	Launch(path string, port *int, arguments []*string) (*Tab, error)
	Wait()
	Terminate() error
	OpenTab(targetID target.ID, timeout time.Duration) *Tab
	OpenNewTab(timeout time.Duration) (*Tab, error)
	OpenNewIncognitoTab(timeout time.Duration) (*Tab, error)
	CloseTab(tab *Tab, timeout time.Duration) error
	connect(timeout time.Duration) (*Tab, error)
}
