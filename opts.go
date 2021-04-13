package chrome

type LaunchOpts struct {
	path string
	port *int
	arguments []string
	headless bool
}

func NewLaunchOpts() *LaunchOpts {
	return &LaunchOpts{
		headless: true,
	}
}

func (l *LaunchOpts) SetPath(path string) {
	l.path = path
}

func (l *LaunchOpts) SetPort(port int) {
	l.port = &port
}

func (l *LaunchOpts) SetArguments(arguments ...string) {
	l.arguments = append(l.arguments, arguments...)
}

func (l *LaunchOpts) SetHeadless(headless bool) {
	l.headless = headless
}

type ScreenshotOpts struct {
	Width             int
	Height            int
	DeviceScaleFactor float64
	Mobile            bool
}