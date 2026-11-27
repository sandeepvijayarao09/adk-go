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

package plugininternal

import (
	"iter"
	"strings"
	"testing"

	"google.golang.org/genai"

	"google.golang.org/adk/v2/agent"
	icontext "google.golang.org/adk/v2/internal/context"
	"google.golang.org/adk/v2/plugin"
	"google.golang.org/adk/v2/session"
)

const pluginPanicValue = "plugin-boom"

func panicContexts(t *testing.T) (agent.InvocationContext, agent.Context) {
	t.Helper()
	a, err := agent.New(agent.Config{
		Name: "agent_1",
		Run: func(agent.InvocationContext) iter.Seq2[*session.Event, error] {
			return func(func(*session.Event, error) bool) {}
		},
	})
	if err != nil {
		t.Fatalf("agent.New() error = %v", err)
	}
	ictx := icontext.NewInvocationContext(t.Context(), icontext.InvocationContextParams{
		InvocationID: "inv_1",
		Agent:        a,
	})
	return ictx, icontext.NewCallbackContext(ictx)
}

func managerWith(t *testing.T, cfgs ...plugin.Config) *PluginManager {
	t.Helper()
	var plugins []*plugin.Plugin
	for _, cfg := range cfgs {
		p, err := plugin.New(cfg)
		if err != nil {
			t.Fatalf("plugin.New(%q) error = %v", cfg.Name, err)
		}
		plugins = append(plugins, p)
	}
	pm, err := NewPluginManager(PluginConfig{Plugins: plugins})
	if err != nil {
		t.Fatalf("NewPluginManager() error = %v", err)
	}
	return pm
}

// A plugin is third-party code. The run-level callbacks have no recovery above
// them: a panic in one used to unwind whichever goroutine was draining
// Runner.Run, so the caller's `for ev, err := range` never saw an error and a
// host application had to wrap every run in a recover. The tool callbacks
// reach their caller through recoverToolPanic and the model callbacks through
// the node scheduler, so those already end as errors and are left alone.
func TestPluginManager_CallbackPanicBecomesAnError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		callback string
		cfg      func(*plugin.Config)
		run      func(*PluginManager, agent.InvocationContext, agent.Context) error
	}{
		{
			name:     "on user message",
			callback: "OnUserMessageCallback",
			cfg: func(c *plugin.Config) {
				c.OnUserMessageCallback = func(agent.InvocationContext, *genai.Content) (*genai.Content, error) {
					panic(pluginPanicValue)
				}
			},
			run: func(pm *PluginManager, ictx agent.InvocationContext, _ agent.Context) error {
				_, err := pm.RunOnUserMessageCallback(ictx, genai.NewContentFromText("hi", genai.RoleUser))
				return err
			},
		},
		{
			name:     "before run",
			callback: "BeforeRunCallback",
			cfg: func(c *plugin.Config) {
				c.BeforeRunCallback = func(agent.InvocationContext) (*genai.Content, error) {
					panic(pluginPanicValue)
				}
			},
			run: func(pm *PluginManager, ictx agent.InvocationContext, _ agent.Context) error {
				_, err := pm.RunBeforeRunCallback(ictx)
				return err
			},
		},
		{
			name:     "on event",
			callback: "OnEventCallback",
			cfg: func(c *plugin.Config) {
				c.OnEventCallback = func(agent.InvocationContext, *session.Event) (*session.Event, error) {
					panic(pluginPanicValue)
				}
			},
			run: func(pm *PluginManager, ictx agent.InvocationContext, _ agent.Context) error {
				_, err := pm.RunOnEventCallback(ictx, session.NewEvent(ictx, ictx.InvocationID()))
				return err
			},
		},
		{
			name:     "before agent",
			callback: "BeforeAgentCallback",
			cfg: func(c *plugin.Config) {
				c.BeforeAgentCallback = func(agent.Context) (*genai.Content, error) {
					panic(pluginPanicValue)
				}
			},
			run: func(pm *PluginManager, _ agent.InvocationContext, cctx agent.Context) error {
				_, err := pm.RunBeforeAgentCallback(cctx)
				return err
			},
		},
		{
			name:     "after agent",
			callback: "AfterAgentCallback",
			cfg: func(c *plugin.Config) {
				c.AfterAgentCallback = func(agent.Context) (*genai.Content, error) {
					panic(pluginPanicValue)
				}
			},
			run: func(pm *PluginManager, _ agent.InvocationContext, cctx agent.Context) error {
				_, err := pm.RunAfterAgentCallback(cctx)
				return err
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cfg := plugin.Config{Name: "panicker"}
			tc.cfg(&cfg)
			pm := managerWith(t, cfg)
			ictx, cctx := panicContexts(t)

			var err error
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("the panic escaped the plugin manager: %v", r)
					}
				}()
				err = tc.run(pm, ictx, cctx)
			}()

			if err == nil {
				t.Fatalf("a panicking %s produced no error", tc.callback)
			}
			// The plugin and the callback are both named, because a deployment
			// with several plugins cannot otherwise tell which one failed.
			// adk-python's _run_callbacks carries both in its RuntimeError.
			for _, want := range []string{`plugin "panicker"`, tc.callback, pluginPanicValue} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("the error does not mention %q:\n%v", want, err)
				}
			}
			if !strings.Contains(err.Error(), "\nstack: ") {
				t.Errorf("the error carries no stack:\n%v", err)
			}
		})
	}
}

// AfterRunCallback has no way to report a failure, and every call site reaches
// it from a defer after the run's answer has shipped. A panic there is recorded
// on the span instead, and the plugins behind the one that panicked still run.
func TestPluginManager_AfterRunCallbackPanicIsIsolated(t *testing.T) {
	t.Parallel()

	laterRan := false
	pm := managerWith(t,
		plugin.Config{
			Name:             "panicker",
			AfterRunCallback: func(agent.InvocationContext) { panic(pluginPanicValue) },
		},
		plugin.Config{
			Name:             "later",
			AfterRunCallback: func(agent.InvocationContext) { laterRan = true },
		},
	)
	ictx, _ := panicContexts(t)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("the panic escaped RunAfterRunCallback: %v", r)
		}
	}()
	pm.RunAfterRunCallback(ictx)

	if !laterRan {
		t.Error("the plugin registered after the one that panicked did not run")
	}
}
