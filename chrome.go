package chrome

import (
	"context"
	"fmt"
	"github.com/mafredri/cdp"
	dt "github.com/mafredri/cdp/devtool"
	tgt "github.com/mafredri/cdp/protocol/target"
	"github.com/mafredri/cdp/rpcc"
	"os/exec"
	"time"
)

type Chrome struct {
	// command object to manage chrome process
	command *exec.Cmd
	// port on which chrome process is listening for dev tools protocol
	port *int
	// devtools protocol version
	Version *dt.Version
}

func (c *Chrome) Launch(path string, port *int, arguments []*string) error {
	// if port is not specified, default to 9222
	if port == nil {
		c.port = Int(9222)
	} else {
		c.port = port
	}

	// prepare default arguments
	defaultArguments := []string{
		"--headless",
		fmt.Sprintf("--remote-debugging-port=%v", IntValue(c.port)),
		"--no-sandbox",
		"--disable-gpu",
		"--disable-sync",
		"--disable-translate",
		"--disable-extensions",
		"--disable-default-apps",
		"--disable-background-networking",
		"--disable-popup-blocking",
		"--safebrowsing-disable-auto-update",
		"--mute-audio",
		"--no-first-run",
		"--hide-scrollbars",
		"--metrics-recording-only",
		"--ignore-certificate-error",
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
		// error occurred
	}
	// wait for chrome to launch
	time.Sleep(5 * time.Second)
	// attempt to connect with chrome over dev tools protocol
	return c.connect(120 * time.Second)
}

func (c *Chrome) Wait() {
	err := c.command.Wait()
	if err != nil {
		// error
	}
}

func (c *Chrome) Terminate() error {
	return c.command.Process.Kill()
}

func (c *Chrome) OpenTab(timeout time.Duration) (*Tab, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Initiate a new RPC connection to the Chrome DevTools Protocol target.
	conn, err := rpcc.DialContext(ctx, c.Version.WebSocketDebuggerURL)
	if err != nil {
		return nil, err
	}
	defer conn.Close() // Leaving connections open will leak memory.

	client := cdp.NewClient(conn)

	createCtx, err := client.Target.GetBrowserContexts(ctx)
	if err != nil {
		return nil, err
	}

	createTargetArgs := tgt.NewCreateTargetArgs("about:blank").
		SetBrowserContextID(createCtx.BrowserContextIDs[0])

	var tab *Tab
	createTarget, err := client.Target.CreateTarget(ctx, createTargetArgs)
	if err != nil {
		return nil, err
	}

	tab = &Tab{}
	tab.Id = createTarget.TargetID

	return tab, nil
}

func (c *Chrome) CloseTab(tab *Tab, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Initiate a new RPC connection to the Chrome DevTools Protocol target.
	conn, err := rpcc.DialContext(ctx, c.Version.WebSocketDebuggerURL)
	if err != nil {
		return err
	}
	defer conn.Close() // Leaving connections open will leak memory.

	client := cdp.NewClient(conn)

	_, err = client.Target.CloseTarget(ctx, tgt.NewCloseTargetArgs(tab.Id))
	return err
}

func (c *Chrome) connect(timeout time.Duration) (err error) {
	// prepare timeout context to cancel in case of a timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// get version for chrome instance to access debugger URL
	c.Version, err = dt.New(fmt.Sprintf("http://127.0.0.1:%v", IntValue(c.port))).Version(ctx)
	if err != nil {
		return
	}

	return
}
