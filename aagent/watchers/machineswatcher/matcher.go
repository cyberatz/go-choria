// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package machines

import (
	"fmt"
	"regexp"

	"github.com/antonmedv/expr"
	iu "github.com/choria-io/go-choria/internal/util"
)

func (w *Watcher) identityMatchFunc(re string) bool {
	r, err := regexp.Compile(re)
	if err != nil {
		w.Errorf("Could not process identity match: %s: %s", re, err)
		return false
	}

	return r.MatchString(w.machine.Identity())
}

func (w *Watcher) isNodeMatch(machine *ManagedMachine) (bool, error) {
	if machine.Matcher == "" {
		return true, nil
	}

	env := map[string]interface{}{
		"identity":      w.identityMatchFunc,
		"has_file":      iu.FileExist,
		"has_directory": iu.FileIsDir,
		"has_command":   iu.IsExecutableInPath,
	}

	execEnv := expr.Env(env)
	prog, err := expr.Compile(machine.Matcher, execEnv, expr.AsBool())
	if err != nil {
		return false, err
	}

	res, err := expr.Run(prog, env)
	if err != nil {
		return false, err
	}

	b, ok := res.(bool)
	if !ok {
		return false, fmt.Errorf("match was non boolean")
	}

	return b, nil
}
