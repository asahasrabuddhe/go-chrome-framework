package chrome

import (
	"context"
	"fmt"
	"github.com/mafredri/cdp"
	dt "github.com/mafredri/cdp/devtool"
	tgt "github.com/mafredri/cdp/protocol/target"
	"github.com/mafredri/cdp/rpcc"
	"log"
	"os/exec"
	"time"
)

type Chrome struct {
	// command object to manage chrome process
	command *exec.Cmd
	// port on which chrome process is listening for dev tools protocol
	port *int
	// rpcc connection to chrome process
	conn *rpcc.Conn
	// browser client
	client *cdp.Client
}

func (c *Chrome) Launch(path string, port *int, arguments []*string) (*Tab, error) {
	// if port is not specified, default to 9222
	if port == nil {
		c.port = Int(9222)
	} else {
		c.port = port
	}

	// prepare default arguments
	defaultArguments := []string{
		"--disable-background-networking",
		"--disable-backgrounding-occluded-windows",
		"--disable-background-timer-throttling",
		"--disable-breakpad",
		"--disable-client-side-phishing-detection",
		"--disable-default-apps",
		"--disable-dev-shm-usage",
		"--disable-extensions",
		"--disable-features=site-per-process,TranslateUI",
		"--disable-gpu",
		"--disable-hang-monitor",
		"--disable-infobars",
		"--disable-ipc-flooding-protection",
		"--disable-popup-blocking",
		"--disable-prompt-on-repost",
		"--disable-renderer-backgrounding",
		"--disable-sync",
		"--disable-translate",
		"--enable-features=NetworkService,NetworkServiceInProcess",
		"--enable-automation",
		"--force-color-profile=srgb",
		"--headless",
		"--hide-scrollbars",
		"--ignore-certificate-errors",
		"--metrics-recording-only",
		"--mute-audio",
		"--no-first-run",
		"--password-store=basic",
		fmt.Sprintf("--remote-debugging-port=%v", IntValue(c.port)),
		"--safebrowsing-disable-auto-update",
		"--use-mock-keychain",
	}

	// if additional arguments are specified, use them alongside the default ones
	if arguments != nil {
		defaultArguments = StringValueSlice(append(StringSlice(defaultArguments), arguments...))
	}

	// create command with chrome path and arguments
	c.command = exec.Command(path, defaultArguments...)

	// launch chrome process
	err := c.command.Start()
	if err != nil {
		log.Println("go-chrome-framework error: unable to launch chrome", err.Error())
		return nil, err
	}

	// wait for process to launch
	time.Sleep(3 * time.Second)

	// attempt to connect with chrome over dev tools protocol
	tab, err := c.connect(120 * time.Second)
	if err != nil {
		log.Println("go-chrome-framework error: unable to connect to browser devtools protocol", err.Error())
		return nil, err
	}

	return tab, err
}

func (c *Chrome) Wait() {
	err := c.command.Wait()
	if err != nil {
		log.Println("go-chrome-framework error: premature exit", err.Error())
	}
}

func (c *Chrome) Terminate() error {
	return c.command.Process.Kill()
}

func (c *Chrome) OpenTab(targetID tgt.ID, timeout time.Duration) *Tab {
	_, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	// wrap the tab in an object and return
	tab := new(Tab)

	tab.id = targetID
	tab.port = c.port

	return tab
}

func (c *Chrome) OpenNewTab(timeout time.Duration) (*Tab, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// create new target (tab)
	createTarget, err := c.client.Target.CreateTarget(ctx, tgt.NewCreateTargetArgs("about:blank"))
	if err != nil {
		log.Println("go-chrome-framework error: unable to create new tab", err.Error())
		return nil, err
	}

	// wrap the tab in an object and return
	tab := new(Tab)

	tab.id = createTarget.TargetID
	tab.port = c.port

	return tab, nil
}

func (c *Chrome) OpenNewIncognitoTab(timeout time.Duration) (*Tab, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// create an empty browser context similar to incognito profile
	createCtx, err := c.client.Target.CreateBrowserContext(ctx)
	if err != nil {
		log.Println("go-chrome-framework error: unable to create browser context for new incognito tab", err.Error())
		return nil, err
	}

	// create new target (tab) based on above incognito profile
	createTarget, err := c.client.Target.CreateTarget(
		ctx,
		tgt.NewCreateTargetArgs("about:blank").
			SetBrowserContextID(createCtx.BrowserContextID),
	)

	if err != nil {
		log.Println("go-chrome-framework error: unable to create new incognito tab", err.Error())
		return nil, err
	}

	// wrap the tab in an object and return
	tab := new(Tab)

	tab.id = createTarget.TargetID
	tab.port = c.port

	return tab, nil
}

func (c *Chrome) CloseTab(tab *Tab, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	_, err := c.client.Target.CloseTarget(ctx, tgt.NewCloseTargetArgs(tab.id))
	return err
}

func (c *Chrome) connect(timeout time.Duration) (*Tab, error) {
	// prepare timeout context to cancel in case of a timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	version, err := dt.New(fmt.Sprintf("http://127.0.0.1:%v", IntValue(c.port))).Version(ctx)
	if err != nil {
		log.Println("go-chrome-framework error: unable to connect to browser over devtools protocol", err.Error())
		return nil, err
	}

	// Initiate a new RPC connection to the Chrome DevTools Protocol targetInfo.
	c.conn, err = rpcc.DialContext(ctx, version.WebSocketDebuggerURL)
	if err != nil {
		log.Println("go-chrome-framework error: unable to initiate a new rpc connection to chrome", err.Error())
		return nil, err
	}

	// browser client
	c.client = cdp.NewClient(c.conn)

	// as chrome launches with a new tab already opened, query the browser for a list of available targets to connect to
	targets, err := c.client.Target.GetTargets(ctx)
	if err != nil {
		log.Println("go-chrome-framework error: unable to get list of targets", err.Error())
		return nil, err
	}

	tab := new(Tab)

	// iterate over all the targets returned
	for _, targetInfo := range targets.TargetInfos {
		// we want to connect to a page and not other target like service worker etc
		if targetInfo.Type == "page" {
			// wrap target in an object
			tab.id = targetInfo.TargetID
			tab.port = c.port
		}
	}

	return tab, err
}
