// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package runner_test

import (
	"iter"
	"strings"
	"testing"

	"google.golang.org/genai"

	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/plugin"
	"google.golang.org/adk/v2/runner"
	"google.golang.org/adk/v2/session"
)

// A panicking plugin must not unwind the goroutine draining Runner.Run. The
// caller iterates `for ev, err := range r.Run(...)`, so a failure has to arrive
// as err; before this, a run-level plugin callback panic went straight past the
// loop and a host application had to wrap every run in a recover to survive a
// third-party plugin.
func TestRun_PluginCallbackPanicArrivesAsAnError(t *testing.T) {
	t.Parallel()

	const panicValue = "plugin-boom"

	tests := []struct {
		name     string
		callback string
		cfg      func(*plugin.Config)
		// wantErr is false for AfterRunCallback, which cannot report a failure:
		// it runs from a defer once the answer has already shipped, so its
		// panic is recorded on the span and the run still succeeds.
		wantErr bool
	}{
		{
			name:     "before run",
			callback: "BeforeRunCallback",
			cfg: func(c *plugin.Config) {
				c.BeforeRunCallback = func(agent.InvocationContext) (*genai.Content, error) {
					panic(panicValue)
				}
			},
			wantErr: true,
		},
		{
			name:     "on user message",
			callback: "OnUserMessageCallback",
			cfg: func(c *plugin.Config) {
				c.OnUserMessageCallback = func(agent.InvocationContext, *genai.Content) (*genai.Content, error) {
					panic(panicValue)
				}
			},
			wantErr: true,
		},
		{
			name:     "on event",
			callback: "OnEventCallback",
			cfg: func(c *plugin.Config) {
				c.OnEventCallback = func(agent.InvocationContext, *session.Event) (*session.Event, error) {
					panic(panicValue)
				}
			},
			wantErr: true,
		},
		{
			name:     "after run",
			callback: "AfterRunCallback",
			cfg: func(c *plugin.Config) {
				c.AfterRunCallback = func(agent.InvocationContext) { panic(panicValue) }
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()

			testAgent, err := agent.New(agent.Config{
				Name: "plugin_panic_agent",
				Run: func(ictx agent.InvocationContext) iter.Seq2[*session.Event, error] {
					return func(yield func(*session.Event, error) bool) {
						ev := session.NewEvent(ictx, ictx.InvocationID())
						ev.LLMResponse.Content = genai.NewContentFromText("answer", genai.RoleModel)
						yield(ev, nil)
					}
				},
			})
			if err != nil {
				t.Fatalf("agent.New() error = %v", err)
			}

			cfg := plugin.Config{Name: "panicker"}
			tc.cfg(&cfg)
			p, err := plugin.New(cfg)
			if err != nil {
				t.Fatalf("plugin.New() error = %v", err)
			}

			svc := session.InMemoryService()
			r, err := runner.New(runner.Config{
				AppName:        "app",
				Agent:          testAgent,
				SessionService: svc,
				PluginConfig:   runner.PluginConfig{Plugins: []*plugin.Plugin{p}},
			})
			if err != nil {
				t.Fatalf("runner.New() error = %v", err)
			}
			if _, err := svc.Create(ctx, &session.CreateRequest{AppName: "app", UserID: "u", SessionID: "s"}); err != nil {
				t.Fatalf("session Create() error = %v", err)
			}

			msg := &genai.Content{Role: genai.RoleUser, Parts: []*genai.Part{genai.NewPartFromText("hi")}}

			var runErr error
			func() {
				defer func() {
					if rec := recover(); rec != nil {
						t.Fatalf("a panicking %s unwound the caller of Run: %v", tc.callback, rec)
					}
				}()
				for _, err := range r.Run(ctx, "u", "s", msg, agent.RunConfig{}) {
					if err != nil && runErr == nil {
						runErr = err
					}
				}
			}()

			if !tc.wantErr {
				if runErr != nil {
					t.Errorf("the run failed, want it to survive a panicking %s: %v", tc.callback, runErr)
				}
				return
			}
			if runErr == nil {
				t.Fatalf("a panicking %s produced no error from Run", tc.callback)
			}
			for _, want := range []string{`plugin "panicker"`, tc.callback, panicValue} {
				if !strings.Contains(runErr.Error(), want) {
					t.Errorf("the error does not mention %q:\n%v", want, runErr)
				}
			}
		})
	}
}
