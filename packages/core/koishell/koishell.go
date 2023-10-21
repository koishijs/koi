package koishell

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/goccy/go-json"
	"github.com/samber/do"
	"gopkg.ilharper.com/koi/app/util"
	"gopkg.ilharper.com/koi/core/koiconfig"
	"gopkg.ilharper.com/koi/core/logger"
	"gopkg.ilharper.com/x/killdren"
)

type KoiShell struct {
	i *do.Injector

	path string
	cwd  string

	// The mutex lock.
	//
	// There's no need to use [sync.RWMutex]
	// because almost all ops are write.
	mutex sync.Mutex
	wg    sync.WaitGroup
	reg   [256]*exec.Cmd
}

func BuildKoiShell(path string) func(i *do.Injector) (*KoiShell, error) {
	return func(i *do.Injector) (*KoiShell, error) {
		return &KoiShell{
			i:    i,
			path: path,
			cwd:  filepath.Dir(path),
		}, nil
	}
}

func (shell *KoiShell) getIndex(cmd *exec.Cmd) uint8 {
	shell.mutex.Lock()
	defer shell.mutex.Unlock()

	var index uint8 = 0
	for {
		if shell.reg[index] == nil {
			shell.reg[index] = cmd

			return index
		}
		index++
	}
}

// exec starts a KoiShell process, waits to exit,
// and returns the execution result.
func (shell *KoiShell) exec(arg any) (map[string]any, error) {
	var err error

	l := do.MustInvoke[*logger.Logger](shell.i)
	cfg := do.MustInvoke[*koiconfig.Config](shell.i)

	argJSON, err := json.Marshal(arg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal arg %#+v: %w", arg, err)
	}
	argB64 := base64.StdEncoding.EncodeToString(argJSON)

	cmd := &exec.Cmd{
		Path: shell.path,
		Args: []string{shell.path, argB64},
		Dir:  cfg.Computed.DirConfig,
	}
	killdren.Set(cmd)

	var outB64 string

	// Setup IO pipes
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	go func(scn *bufio.Scanner) {
		for {
			if !scn.Scan() {
				break
			}
			scnErr := scn.Err()
			if scnErr != nil {
				l.Warn(fmt.Errorf("KoiShell scanner error: %w", scnErr))
			} else {
				outB64 = scn.Text()
			}
		}
	}(bufio.NewScanner(stdoutPipe))

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	go func(scn *bufio.Scanner) {
		for {
			if !scn.Scan() {
				break
			}
			scnErr := scn.Err()
			if scnErr != nil {
				l.Warn(fmt.Errorf("KoiShell scanner error: %w", scnErr))
			} else {
				l.Error(scn.Text())
			}
		}
	}(bufio.NewScanner(stderrPipe))

	// Wait process to stop
	index := shell.getIndex(cmd)
	shell.wg.Add(1)

	l.Debugf("Starting KoiShell process.\narg: %#+v\nargJSON: %s\nargB64: %s", arg, argJSON, argB64)
	err = cmd.Run()

	shell.wg.Done()

	shell.mutex.Lock()
	shell.reg[index] = nil
	shell.mutex.Unlock()

	if err != nil {
		return nil, fmt.Errorf("KoiShell exited with error: %w", err)
	}

	l.Debugf("KoiShell successfully exited.")

	// Parse output
	if outB64 == "" {
		return nil, nil
	}

	outJSON, err := base64.StdEncoding.DecodeString(outB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode KoiShell output %s: %w", outB64, err)
	}

	var out map[string]any
	err = json.Unmarshal(outJSON, &out)
	if err != nil {
		return nil, fmt.Errorf("failed to parse KoiShell output %s: %w", outJSON, err)
	}

	return out, nil
}

// start a [KoiShell] process
// and return its [*exec.Cmd].
//
// One of [*exec.Cmd] and [error] must be nil.
func (shell *KoiShell) start(arg any) (*exec.Cmd, error) {
	var err error

	l := do.MustInvoke[*logger.Logger](shell.i)
	cfg := do.MustInvoke[*koiconfig.Config](shell.i)

	argJson, err := json.Marshal(arg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal arg %#+v: %w", arg, err)
	}
	argB64 := base64.StdEncoding.EncodeToString(argJson)

	cmd := &exec.Cmd{
		Path: shell.path,
		Args: []string{shell.path, argB64},
		Dir:  cfg.Computed.DirConfig,
	}
	killdren.Set(cmd)

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	go func(scn *bufio.Scanner) {
		for {
			if !scn.Scan() {
				break
			}
			scnErr := scn.Err()
			if scnErr != nil {
				l.Warn(fmt.Errorf("KoiShell scanner error: %w", scnErr))
			} else {
				l.Error(scn.Text())
			}
		}
	}(bufio.NewScanner(stderrPipe))

	l.Debugf("Starting KoiShell process.\narg: %#+v\nargJson: %s\nargB64: %s", arg, argJson, argB64)
	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start KoiShell: %w", err)
	}

	go func() {
		waitErr := cmd.Wait()

		if waitErr != nil {
			l.Debugf("KoiShell exited with error: %v", err)
		} else {
			l.Debugf("KoiShell successfully exited.")
		}
	}()

	return cmd, nil
}

func (shell *KoiShell) Shutdown() error {
	shell.mutex.Lock()

	for _, cmd := range shell.reg {
		if cmd != nil {
			err := killdren.Stop(cmd)
			if err != nil {
				_ = killdren.Kill(cmd)
			}
		}
	}

	shell.mutex.Unlock()
	shell.wg.Wait()

	// Do not short other do.Shutdownable
	return nil
}

func (shell *KoiShell) WebView(name, url string) (*exec.Cmd, error) {
	return shell.start(map[string]string{
		"mode": "webview",
		"name": name,
		"url":  url,
	})
}

func (shell *KoiShell) About() {
	l := do.MustInvoke[*logger.Logger](shell.i)

	_, err := shell.exec(map[string]any{
		"mode":        "dialog",
		"title":       "About",
		"style":       "info",
		"text1":       "Koishi Desktop",
		"text2":       fmt.Sprintf("v%s", util.AppVersion),
		"buttonCount": 1,
	})
	if err != nil {
		l.Errorf("About dialog error: %v", err)
	}
}

func (shell *KoiShell) AlreadyRunning() {
	l := do.MustInvoke[*logger.Logger](shell.i)

	_, err := shell.exec(map[string]any{
		"mode":        "dialog",
		"title":       "Already Running",
		"style":       "info",
		"text1":       "Koishi is already running.",
		"text2":       "You can find the Koishi icon from notification area. Tap the Koishi icon to perform action.",
		"buttonCount": 1,
	})
	if err != nil {
		l.Errorf("Already running dialog error: %v", err)
	}
}
