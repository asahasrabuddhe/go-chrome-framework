package chrome

import (
	"context"
	"fmt"
	"github.com/mafredri/cdp"
	"github.com/mafredri/cdp/protocol/dom"
	"github.com/mafredri/cdp/protocol/page"
	"github.com/mafredri/cdp/protocol/runtime"
	tgt "github.com/mafredri/cdp/protocol/target"
	"github.com/mafredri/cdp/rpcc"
	"time"
)

type Tab struct {
	// target Id of a single tab
	Id tgt.ID
	// connection to connect with the browser
	conn *rpcc.Conn
	// client to control the browser
	client *cdp.Client
}

func (t *Tab) connect(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var err error
	// connect to chrome
	t.conn, err = rpcc.DialContext(ctx, "ws://127.0.0.1:9222/devtools/page/"+string(t.Id))
	if err != nil {
		return err
	}

	// This cdp client controls the "incognito tab".
	t.client = cdp.NewClient(t.conn)
	return nil
}

func (t *Tab) disconnect() error {
	return t.conn.Close()
}

func (t *Tab) Navigate(url string, timeout time.Duration) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	err := t.connect(timeout)
	if err != nil {
		return false, err
	}

	defer t.disconnect()

	// Open a DOMContentEventFired client to buffer this event.
	domContent, err := t.client.Page.DOMContentEventFired(ctx)
	if err != nil {
		return false, err
	}
	defer domContent.Close()

	// Enable events on the Page domain, it's often preferable to create
	// event clients before enabling events so that we don't miss any.
	if err = t.client.Page.Enable(ctx); err != nil {
		return false, err
	}

	// Create the Navigate arguments with the optional Referrer field set.
	navArgs := page.NewNavigateArgs(url)
	nav, err := t.client.Page.Navigate(ctx, navArgs)
	if err != nil {
		return false, err
	}

	// Wait until we have a DOMContentEventFired event.
	if _, err = domContent.Recv(); err != nil {
		return false, err
	}

	// wait for ajax to render
	time.Sleep(5 * time.Second)

	fmt.Printf("Page loaded with frame ID: %s\n", nav.FrameID)

	return true, nil
}

func (t *Tab) GetHTML(timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	err := t.connect(timeout)
	if err != nil {
		return "", err
	}

	defer t.disconnect()

	// Fetch the document root node. We can pass nil here
	// since this method only takes optional arguments.
	doc, err := t.client.DOM.GetDocument(ctx, nil)
	if err != nil {
		return "", err
	}

	// Get the outer HTML for the page.
	result, err := t.client.DOM.GetOuterHTML(ctx, &dom.GetOuterHTMLArgs{
		NodeID: &doc.Root.NodeID,
	})
	if err != nil {
		return "", err
	}

	return result.OuterHTML, nil
}

func (t *Tab) Exec(javascript string, timeout time.Duration) (*runtime.EvaluateReply, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	err := t.connect(timeout)
	if err != nil {
		return nil, err
	}

	defer t.disconnect()

	evalArgs := runtime.NewEvaluateArgs(javascript).SetAwaitPromise(true).SetReturnByValue(true)
	return t.client.Runtime.Evaluate(ctx, evalArgs)
}

func (t *Tab) GetNewTab(timeout time.Duration) (*Tab, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	newTarget, err := t.client.Target.TargetCreated(ctx)
	if err != nil {
		return nil, err
	} else {
		<-newTarget.Ready()

		res, _ := newTarget.Recv()
		if res.TargetInfo.Type == "page" {
			var tab Tab

			tab.Id = res.TargetInfo.TargetID
			tab.conn = t.conn
			tab.client = t.client

			return &tab, nil
		} else {
			return nil, nil
		}
	}
}
