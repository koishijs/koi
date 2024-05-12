package koicli

import (
	"fmt"

	"github.com/samber/do"
	"github.com/urfave/cli/v2"
	"golang.org/x/text/message"
	"gopkg.ilharper.com/koi/app/util"
	"gopkg.ilharper.com/koi/core/logger"
)

func NewCli(i *do.Injector) (*cli.App, error) {
	l := do.MustInvoke[*logger.Logger](i)
	p := do.MustInvoke[*message.Printer](i)

	do.ProvideNamed(i, serviceActionPre, newPreAction)
	do.ProvideNamed(i, serviceCommandRun, newRunCommand)
	do.ProvideNamed(i, serviceCommandImport, newImportCommand)
	do.ProvideNamed(i, serviceCommandDaemon, newDaemonCommand)
	do.ProvideNamed(i, serviceCommandPs, newPsCommand)
	do.ProvideNamed(i, serviceCommandOpen, newOpenCommand)
	do.ProvideNamed(i, serviceCommandStart, newStartCommand)
	do.ProvideNamed(i, serviceCommandStop, newStopCommand)
	do.ProvideNamed(i, serviceCommandRestart, newRestartCommand)
	do.ProvideNamed(i, serviceCommandYarn, newYarnCommand)

	return &cli.App{
		Name:    p.Sprintf("Cordis Desktop"),
		Usage:   p.Sprintf("Launch Cordis from your desktop."),
		Version: fmt.Sprintf("v%s", util.AppVersion),
		Authors: []*cli.Author{
			{
				Name:  "Il Harper",
				Email: "hi@ilharper.com",
			},
		},

		UseShortOptionHandling: true,
		EnableBashCompletion:   true,

		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "debug",
				Usage: p.Sprintf("Enable debug mode"),
			},

			&cli.BoolFlag{
				Name:     "no-console",
				Usage:    p.Sprintf("Do not show console window on Windows"),
				Required: false,
				Value:    false,
			},

			&cli.BoolFlag{
				Name:     "no-start",
				Usage:    p.Sprintf("Do not start instance(s)"),
				Required: false,
				Value:    false,
			},

			cli.HelpFlag,
			cli.VersionFlag,
			cli.BashCompletionFlag,
		},

		Commands: []*cli.Command{
			do.MustInvokeNamed[*cli.Command](i, serviceCommandRun),
			do.MustInvokeNamed[*cli.Command](i, serviceCommandImport),
			do.MustInvokeNamed[*cli.Command](i, serviceCommandDaemon),
			do.MustInvokeNamed[*cli.Command](i, serviceCommandPs),
			do.MustInvokeNamed[*cli.Command](i, serviceCommandOpen),
			do.MustInvokeNamed[*cli.Command](i, serviceCommandStart),
			do.MustInvokeNamed[*cli.Command](i, serviceCommandStop),
			do.MustInvokeNamed[*cli.Command](i, serviceCommandRestart),
			do.MustInvokeNamed[*cli.Command](i, serviceCommandYarn),
		},

		Before: do.MustInvokeNamed[cli.BeforeFunc](i, serviceActionPre),
		CommandNotFound: func(context *cli.Context, s string) {
			l.Error(p.Sprintf("Command not found: %s", s))
		},
		ExitErrHandler: func(context *cli.Context, err error) {
			if err == nil {
				return
			}

			l.Error(err)
		},
	}, nil
}
