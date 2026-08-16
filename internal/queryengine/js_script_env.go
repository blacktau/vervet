package queryengine

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dop251/goja"
)

// scriptLocation resolves the script's own path and the directory that
// relative paths resolve against. An unsaved tab has no path, so it falls
// back to the process working directory — the same base the runtime used
// before scripts could be saved to files.
func scriptLocation(scriptPath string) (path, baseDir string) {
	if scriptPath != "" {
		abs, err := filepath.Abs(scriptPath)
		if err != nil {
			abs = scriptPath
		}
		return abs, filepath.Dir(abs)
	}

	wd, err := os.Getwd()
	if err != nil {
		return "", ""
	}
	return "", wd
}

// registerScriptEnv gives the script the file-location globals mongosh
// provides — __filename, __dirname, process.cwd() and load().
func registerScriptEnv(rt *goja.Runtime, scriptPath, baseDir string) error {
	if err := rt.Set("__filename", scriptPath); err != nil {
		return fmt.Errorf("failed to set __filename global: %w", err)
	}

	if err := rt.Set("__dirname", baseDir); err != nil {
		return fmt.Errorf("failed to set __dirname global: %w", err)
	}

	// goja_nodejs's process module exposes env and nothing else, so cwd is
	// added here rather than upstream.
	if proc, ok := rt.Get("process").(*goja.Object); ok {
		if err := proc.Set("cwd", func() string { return baseDir }); err != nil {
			return fmt.Errorf("failed to set process.cwd: %w", err)
		}
	}

	return rt.Set("load", loadFn(rt, baseDir))
}

// loadFn implements mongosh's load(): it runs another script in the current
// runtime, so anything the loaded file declares at the top level is visible
// to the caller afterwards. It returns true, as mongosh's does.
func loadFn(rt *goja.Runtime, baseDir string) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		path := call.Argument(0).String()
		if !filepath.IsAbs(path) && baseDir != "" {
			path = filepath.Join(baseDir, path)
		}

		src, err := os.ReadFile(path)
		if err != nil {
			panic(rt.NewGoError(fmt.Errorf("could not open file: %s", path)))
		}

		// The same top-level rewrite the main script gets, so a loaded file's
		// const/let declarations become globals the caller can see.
		if _, err := rt.RunScript(path, rewriteTopLevelDeclarations(string(src))); err != nil {
			var exc *goja.Exception
			if errors.As(err, &exc) {
				panic(exc.Value())
			}
			panic(rt.NewGoError(fmt.Errorf("failed to load %s: %w", path, err)))
		}

		return rt.ToValue(true)
	}
}
