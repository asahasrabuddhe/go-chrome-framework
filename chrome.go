package chrome

import (
	"context"
	"fmt"
	"github.com/flowchartsman/retry"
	"github.com/mafredri/cdp"
	"github.com/mafredri/cdp/devtool"
	"github.com/mafredri/cdp/protocol/target"
	"github.com/mafredri/cdp/rpcc"
	"io"
	"log"
	"os/exec"
	"time"
)

type Chrome interface {
	Launch(*LaunchOpts) (Tab, error)
	Wait()
	Terminate() error
	OpenTab(target.ID, time.Duration) Tab
	OpenNewTab(time.Duration) (Tab, error)
	OpenNewIncognitoTab(time.Duration) (Tab, error)
	CloseTab(Tab, time.Duration) error
}

func NewChrome() Chrome {
	return &chrome{}
}


type chrome struct {
	// command object to manage chrome process
	command *exec.Cmd
	// port on which chrome process is listening for dev tools protocol
	port *int
	// rpcc connection to chrome process
	conn *rpcc.Conn
	// browser client
	client *cdp.Client
}

func (c *chrome) Launch(opts *LaunchOpts) (Tab, error) {
	// if port is not specified, default to 9222
	if opts.port == nil {
		c.port = Int(9222)
	} else {
		c.port = opts.port
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
		"--hide-scrollbars",
		"--ignore-certificate-errors",
		"--metrics-recording-only",
		"--mute-audio",
		"--no-first-run",
		"--no-sandbox",
		"--password-store=basic",
		fmt.Sprintf("--remote-debugging-port=%v", IntValue(c.port)),
		"--safebrowsing-disable-auto-update",
		"--use-mock-keychain",
	}

	// if additional arguments are specified, use them alongside the default ones
	if opts.arguments != nil {
		defaultArguments = StringValueSlice(append(StringSlice(defaultArguments), StringSlice(opts.arguments)...))
	}

	// if headless is true, launch in headless mode
	if opts.headless {
		defaultArguments = append(defaultArguments, "--headless")
	}

	// create command with chrome path and arguments
	c.command = exec.Command(opts.path, defaultArguments...)

	// launch chrome process
	err := c.command.Start()
	if err != nil {
		log.Println("go-chrome-framework error: unable to launch chrome", err.Error())
		return nil, err
	}

	// attempt to connect with chrome over dev tools protocol
	tab, err := c.connect(120 * time.Second)
	if err != nil {
		log.Println("go-chrome-framework error: unable to connect to browser devtools protocol", err.Error())
		return nil, err
	}

	return tab, err
}

func (c *chrome) Wait() {
	err := c.command.Wait()
	if err != nil {
		log.Println("go-chrome-framework error: premature exit", err.Error())
	}
}

func (c *chrome) Terminate() error {
	// handle scenario when someone tries to terminate a browser that never launched
	if c.command.Process != nil {
		return c.command.Process.Kill()
	}

	return nil
}

func (c *chrome) OpenTab(targetID target.ID, timeout time.Duration) Tab {
	_, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	// wrap the tab in an object and return
	tab := new(tab)

	tab.id = targetID
	tab.port = c.port

	return tab
}

func (c *chrome) OpenNewTab(timeout time.Duration) (Tab, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// create new target (tab)
	createTarget, err := c.client.Target.CreateTarget(ctx, target.NewCreateTargetArgs("about:blank"))
	if err != nil {
		log.Println("go-chrome-framework error: unable to create new tab", err.Error())
		return nil, err
	}

	// wrap the tab in an object and return
	tab := new(tab)

	tab.id = createTarget.TargetID
	tab.port = c.port

	return tab, nil
}

func (c *chrome) OpenNewIncognitoTab(timeout time.Duration) (Tab, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// create an empty browser context similar to incognito profile
	createCtx, err := c.client.Target.CreateBrowserContext(ctx, target.NewCreateBrowserContextArgs())
	if err != nil {
		log.Println("go-chrome-framework error: unable to create browser context for new incognito tab", err.Error())
		return nil, err
	}

	// create new target (tab) based on above incognito profile
	createTarget, err := c.client.Target.CreateTarget(
		ctx,
		target.NewCreateTargetArgs("about:blank").
			SetBrowserContextID(createCtx.BrowserContextID),
	)

	if err != nil {
		log.Println("go-chrome-framework error: unable to create new incognito tab", err.Error())
		return nil, err
	}

	// wrap the tab in an object and return
	tab := new(tab)

	tab.id = createTarget.TargetID
	tab.port = c.port

	return tab, nil
}

func (c *chrome) CloseTab(tab Tab, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	_, err := c.client.Target.CloseTarget(ctx, target.NewCloseTargetArgs(tab.GetTargetID()))
	return err
}

func (c *chrome) connect(timeout time.Duration) (Tab, error) {
	// prepare timeout context to cancel in case of a timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	tab := new(tab)

	rt := retry.NewRetrier(5, 100*time.Millisecond, time.Second)
	err := rt.RunContext(ctx, func(ctx context.Context) error {
		// use the devtool to create a Page Target
		version, err := devtool.New(fmt.Sprintf("http://localhost:%v", IntValue(c.port))).Version(ctx)
		if err != nil {
			log.Println("go-chrome-framework error: unable to connect to browser over devtools protocol", err.Error())
			return err
		}

		// Initiate a new RPC connection to the chrome DevTools Protocol targetInfo.
		c.conn, err = rpcc.DialContext(ctx, version.WebSocketDebuggerURL)
		if err != nil {
			log.Println("go-chrome-framework error: unable to initiate a new rpc connection to chrome", err.Error())
			return err
		}

		// browser client
		c.client = cdp.NewClient(c.conn)

		// as chrome launches with a new tab already opened, query the browser for a list of available targets to connect to
		targets, err := c.client.Target.GetTargets(ctx)
		if err != nil {
			log.Println("go-chrome-framework error: unable to get list of targets", err.Error())
			return err
		}

		// iterate over all the targets returned
		for _, targetInfo := range targets.TargetInfos {
			// we want to connect to a page and not other target like service worker etc
			if targetInfo.Type == "page" {
				// wrap target in an object
				tab.id = targetInfo.TargetID
				tab.port = c.port

				break
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return tab, err
}

func closeRes(close io.Closer) {
	err := close.Close()
	if err != nil {
		log.Println("error occurred while trying to close resource", err.Error())
	}
}
