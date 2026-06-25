package cli

const helpText = `pcraft launcher

Usage:
  pcraft run [--port <port>] [--verbose] [--debug]
  pcraft start [--port <port>] [--verbose] [--debug]
  pcraft [--port <port>] [--verbose] [--debug]
  pcraft service install|uninstall|start|stop|restart|status|logs [--system]

Options:
  start            Use local production build.
  run              Use installed runtime bundle (default).
  service          Manage pcraft as an OS service.
  --version, -V    Print CLI version and exit.
  --port           Port for the Go backend. Alias for --backend-port.
  --verbose, -v    Show info logs from backend.
  --debug          Show debug logs + agent message dumps.
  --headless       Skip opening the browser. Used by service units.
  --help, -h       Show help.

Advanced:
  --backend-port         Same as --port.
`

func Help() string {
	return helpText
}
