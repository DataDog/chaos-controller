// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package injector

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func Test_standardPythonRunner_RunPython(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		dryRun  bool
		expect  func(*testing.T, *mockCommand, *observer.ObservedLogs)
		wantErr string
	}{
		{
			name:   "dry run does not call start or wait",
			dryRun: true,
		},
		{
			name: "when starts returns an error, error is returned to the caller",
			expect: func(t *testing.T, mockCommand *mockCommand, logs *observer.ObservedLogs) {
				mockCommand.EXPECT().Start().Return(errors.New("start some error"))
			},
			wantErr: "unable to start command, encountered error (start some error) using args ([/usr/local/bin/some_python.py -v -i]): ",
		},
		{
			name: "when wait error immediatly, error is returned to the caller",
			expect: func(t *testing.T, mockCommand *mockCommand, logs *observer.ObservedLogs) {
				mockCommand.EXPECT().Start().Return(nil)
				mockCommand.EXPECT().Wait().Return(errors.New("wait immediate error"))

				t.Cleanup(func() {
					logEntries := logs.All()

					require.Len(t, logEntries, 1, "a single log entry is expected")
				})
			},
			wantErr: "unable to wait command, exited early error (wait immediate error) using args ([/usr/local/bin/some_python.py -v -i]): ",
		},
		{
			name: "when wait 100ms then return an error, error is returned to the caller",
			expect: func(t *testing.T, mockCommand *mockCommand, logs *observer.ObservedLogs) {
				mockCommand.EXPECT().Start().Return(nil)
				mockCommand.EXPECT().Wait().Call.Return(func() error {
					return fmt.Errorf("wait early error")
				}).WaitUntil(time.After(100 * time.Millisecond))

				t.Cleanup(func() {
					// wait at least more than wait time
					time.Sleep(500 * time.Millisecond)

					logEntries := logs.All()

					require.Len(t, logEntries, 1, "a single log entry is expected")
				})
			},
			wantErr: "unable to wait command, exited early error (wait early error) using args ([/usr/local/bin/some_python.py -v -i]): ",
		},
		{
			name: "when wait 500ms then return no error want no error and no additional logs",
			expect: func(t *testing.T, mockCommand *mockCommand, logs *observer.ObservedLogs) {
				mockCommand.EXPECT().Start().Return(nil)
				mockCommand.EXPECT().Wait().Call.Return(func() error {
					return nil
				}).WaitUntil(time.After(500 * time.Millisecond))

				t.Cleanup(func() {
					// wait at least more than wait time
					time.Sleep(1 * time.Second)

					logEntries := logs.All()

					require.Len(t, logEntries, 1, "a single log entry is expected")
				})
			},
		},
		{
			name: "when wait 500ms then return an error want no error returned and an error log (later)",
			expect: func(t *testing.T, mockCommand *mockCommand, logs *observer.ObservedLogs) {
				mockCommand.EXPECT().Start().Return(nil)
				mockCommand.EXPECT().Wait().Call.Return(func() error {
					return errors.New("wait late error")
				}).WaitUntil(time.After(500 * time.Millisecond))

				t.Cleanup(func() {
					// wait at least more than wait time
					time.Sleep(1 * time.Second)

					logEntries := logs.All()

					require.Len(t, logEntries, 2, "two log entries are expected, received %d", len(logEntries))
					require.Equal(t, "command late exit with error (wait late error) using args ([/usr/local/bin/some_python.py -v -i]): ", logEntries[1].Message, "blabla expected")
				})
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			observer, logs := observer.New(zap.InfoLevel)
			z := zap.New(observer)
			logger := z.Sugar()

			mockCommand := newMockCommand(t)

			args := []string{"/usr/local/bin/some_python.py", "-v", "-i"}

			p := &standardPythonRunner{
				dryRun: tt.dryRun,
				log:    logger,
				newCmd: func(out, err io.Writer, args ...string) command {
					mockCommand.EXPECT().String().Return(strings.Join(args, " "))

					if tt.expect != nil {
						tt.expect(t, mockCommand, logs)
					}

					return mockCommand
				},
				maxErrorLines:  1,
				maxWaitCommand: 250 * time.Millisecond,
			}

			err := p.RunPython(args...)

			if err != nil || tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				return
			}

			logEntries := logs.All()

			require.GreaterOrEqual(t, len(logEntries), 1, "at least one log entry is expected")
			require.Equal(t, "running python3 command: /usr/local/bin/some_python.py -v -i", logEntries[0].Message)
		})
	}
}

const (
	lines = `first line
second line
third line
fourth line
fifth line`
)

func Test_lastLines(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		lineCount int
		want      string
	}{
		{
			name:      "lineCount greater than all lines return all lines",
			input:     lines,
			lineCount: 10,
			want:      lines,
		},
		{
			name:      "lineCount=1 return single last line",
			input:     lines,
			lineCount: 1,
			want:      "fifth line",
		},
		{
			name:      "empty return empty",
			input:     "",
			lineCount: 10,
			want:      "",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := lastLines(tt.input, tt.lineCount)

			require.Equal(t, tt.want, got)
		})
	}
}
