// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.
package injector

import (
	"context"
	"fmt"
	"strconv"
	"time"

	chaosapi "github.com/DataDog/chaos-controller/api"
	"github.com/DataDog/chaos-controller/command"
	"github.com/DataDog/chaos-controller/process"
	"github.com/DataDog/chaos-controller/types"
	"go.uber.org/zap"
)

type flag string

func (f flag) String() string {
	return fmt.Sprintf("--%s", string(f))
}

const (
	ChaosInjectorBinaryLocation = "/usr/local/bin/chaos-injector"

	ParentPIDFlag flag = "parent-pid"
	DeadlineFlag  flag = "deadline"
)

type InjectorCmdFactory interface {
	NewInjectorBackgroundCmd(deadline time.Time, disruptionArgs chaosapi.DisruptionArgs, target string, args []string) (command.BackgroundCmd, context.CancelFunc, error)
}

type injectorCmdFactory struct {
	log            *zap.SugaredLogger
	processManager process.Manager
	cmdBuilder     command.Factory
}

func NewInjectorCmdFactory(log *zap.SugaredLogger, processManager process.Manager, cmdBuilder command.Factory) InjectorCmdFactory {
	return injectorCmdFactory{
		log,
		processManager,
		cmdBuilder,
	}
}

// NewInjectorBackgroundCmd creates a command to run in background with the exact same configuration as the one provided by disruptionArgs
// except it will be restricted to a single TargetContainers (a container if targeting a pod, or the node if targeting a node)
// - command will last at most until provided deadline
// - command will be considered a "child" command
// - returns associated context cancelled func to use to release resources
func (i injectorCmdFactory) NewInjectorBackgroundCmd(deadline time.Time, disruptionArgs chaosapi.DisruptionArgs, target string, args []string) (command.BackgroundCmd, context.CancelFunc, error) {
	// when we target a node, we have a single value in TargetContainers, nothing should be changed
	if disruptionArgs.Level == types.DisruptionLevelPod {
		targetContainerID, ok := disruptionArgs.TargetContainers[target]
		if !ok {
			return nil, nil, fmt.Errorf("targeted container does not exists: %s", target)
		}

		disruptionArgs.TargetContainers = map[string]string{target: targetContainerID}
	}

	args = append(
		args,
		ParentPIDFlag.String(), strconv.Itoa(i.processManager.ProcessID()),
		DeadlineFlag.String(), deadline.Format(time.RFC3339),
	)
	args = disruptionArgs.CreateCmdArgs(args)

	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	cmd := i.cmdBuilder.NewCmd(ctx, ChaosInjectorBinaryLocation, args)

	backgroundCmd := command.NewBackgroundCmd(cmd, i.log, i.processManager)

	return backgroundCmd, cancel, nil
}
