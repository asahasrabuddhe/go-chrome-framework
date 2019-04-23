package chrome

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/mafredri/cdp"
	"github.com/mafredri/cdp/protocol/dom"
	"github.com/mafredri/cdp/protocol/emulation"
	"github.com/mafredri/cdp/protocol/page"
	"github.com/mafredri/cdp/protocol/runtime"
	tgt "github.com/mafredri/cdp/protocol/target"
	"github.com/mafredri/cdp/rpcc"
	"log"
	"time"
)

type Tab struct {
	// target id of a single tab
	id tgt.ID
	// port on which chrome process is listening for dev tools protocol
	port *int
	// connection to connect with the browser
	conn *rpcc.Conn
	// client to control the browser
	client *cdp.Client
	// hooks to attach additional functionality to client, enable domains etc
	hooks ClientHooks
}

type ClientHook func(c *cdp.Client) error

type ClientHooks []ClientHook

func (t *Tab) connect(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var err error
	// connect to chrome
	t.conn, err = rpcc.DialContext(
		ctx,
		fmt.Sprintf("ws://127.0.0.1:%v/devtools/page/%v", IntValue(t.port), t.id),
	)
	if err != nil {
		log.Println("go-chrome-framework error: unable to connect to target", err.Error())
		return err
	}

	// This cdp Client controls the tab.
	t.client = cdp.NewClient(t.conn)

	// execute hooks for current target
	for _, hook := range t.hooks {
		err := hook(t.client)
		if err != nil {
			log.Println("go-chrome-framework error: unable to execute hook", err.Error())
			return err
		}
	}

	return nil
}

func (t *Tab) disconnect() error {
	return t.conn.Close()
}

func (t *Tab) Navigate(url string, timeout time.Duration) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if t.conn == nil {
		err := t.connect(timeout)
		if err != nil {
			return false, err
		}
	}

	// Open a DOMContentEventFired Client to buffer this event.
	domContent, err := t.client.Page.DOMContentEventFired(ctx)
	if err != nil {
		log.Println("go-chrome-framework error: unable to open dom content event fired client", err.Error())
		return false, err
	}
	defer domContent.Close()

	// Enable events on the Page domain, it's often preferable to create
	// event clients before enabling events so that we don't miss any.
	if err = t.client.Page.Enable(ctx); err != nil {
		log.Println("go-chrome-framework error: unable to enable page domain", err.Error())
		return false, err
	}

	// Create the Navigate arguments with the optional Referrer field set.
	navArgs := page.NewNavigateArgs(url)
	nav, err := t.client.Page.Navigate(ctx, navArgs)
	if err != nil {
		log.Println("go-chrome-framework error: unable to navigate to given url", err.Error())
		return false, err
	}

	// Wait until we have a DOMContentEventFired event.
	if _, err = domContent.Recv(); err != nil {
		log.Println("go-chrome-framework error: unable to get dom content event", err.Error())
		return false, err
	}

	// wait for ajax to render
	time.Sleep(5 * time.Second)

	log.Printf("go-chrome-framework: page loaded with frame ID: %s\n", nav.FrameID)

	return true, nil
}

func (t *Tab) GetHTML(timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if t.conn == nil {
		err := t.connect(timeout)
		if err != nil {
			return "", err
		}
	}

	// Fetch the document root node. We can pass nil here
	// since this method only takes optional arguments.
	doc, err := t.client.DOM.GetDocument(ctx, nil)
	if err != nil {
		log.Println("go-chrome-framework error: unable to get DOM root node", err.Error())
		return "", err
	}

	// Get the outer HTML for the page.
	result, err := t.client.DOM.GetOuterHTML(ctx, &dom.GetOuterHTMLArgs{
		NodeID: &doc.Root.NodeID,
	})
	if err != nil {
		log.Println("go-chrome-framework error: unable to get outer html", err.Error())
		return "", err
	}

	return result.OuterHTML, nil
}

func (t *Tab) CaptureScreenshot(timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if t.conn == nil {
		err := t.connect(timeout)
		if err != nil {
			return "", err
		}
	}

	// Fetch the document root node. We can pass nil here
	// since this method only takes optional arguments.
	doc, err := t.client.DOM.GetDocument(ctx, nil)
	if err != nil {
		log.Println("go-chrome-framework error: unable to get DOM root node", err.Error())
		return "", err
	}

	querySelectorArgs := dom.NewQuerySelectorArgs(doc.Root.NodeID, "body")
	bodyNode, err := t.client.DOM.QuerySelector(ctx, querySelectorArgs)
	if err != nil {
		log.Println("go-chrome-framework error: unable to get DOM root node", err.Error())
		return "", err
	}

	getBoxModelArgs := dom.NewGetBoxModelArgs().SetNodeID(bodyNode.NodeID)
	bodyBoxModel, err := t.client.DOM.GetBoxModel(ctx, getBoxModelArgs)
	if err != nil {
		log.Println("go-chrome-framework error: unable to get DOM root node", err.Error())
		return "", err
	}

	deviceMetricsOverrideArgs := emulation.NewSetDeviceMetricsOverrideArgs(800, bodyBoxModel.Model.Height, 1.0, false)
	err = t.client.Emulation.SetDeviceMetricsOverride(ctx, deviceMetricsOverrideArgs)

	screenshotArgs := page.NewCaptureScreenshotArgs().SetFormat("png").SetQuality(80)
	screenshot, err := t.client.Page.CaptureScreenshot(ctx, screenshotArgs)
	if err != nil {
		// error
		return "", err
	}

	image := fmt.Sprintf("data:image/png;base64,%v", base64.StdEncoding.EncodeToString(screenshot.Data))

	return image, nil
}

func (t *Tab) Exec(javascript string, timeout time.Duration) (*runtime.EvaluateReply, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if t.conn == nil {
		err := t.connect(timeout)
		if err != nil {
			return nil, err
		}
	}

	evalArgs := runtime.NewEvaluateArgs(javascript).SetAwaitPromise(true).SetReturnByValue(true)
	return t.client.Runtime.Evaluate(ctx, evalArgs)
}

func (t *Tab) GetClient() *cdp.Client {
	return t.client
}

func (t *Tab) GetTargetID() tgt.ID {
	return t.id
}

func (t *Tab) AttachHook(hook ClientHook) {
	t.hooks = append(t.hooks, hook)
}
