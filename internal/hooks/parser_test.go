package hooks

import (
	"strings"
	"testing"
)

func TestParseEnvOverrides(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *EnvOverrides
		wantErr bool
	}{
		{
			name:  "empty input",
			input: "",
			want: &EnvOverrides{
				Global:   map[string]string{},
				Services: map[string]map[string]string{},
			},
			wantErr: false,
		},
		{
			name:  "global override only",
			input: "GLOBAL:DATABASE_URL=postgres://localhost/db",
			want: &EnvOverrides{
				Global: map[string]string{
					"DATABASE_URL": "postgres://localhost/db",
				},
				Services: map[string]map[string]string{},
			},
			wantErr: false,
		},
		{
			name:  "service-specific override only",
			input: "api:PORT=4201",
			want: &EnvOverrides{
				Global: map[string]string{},
				Services: map[string]map[string]string{
					"api": {
						"PORT": "4201",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "multiple overrides mixed",
			input: `GLOBAL:DATABASE_URL=postgres://localhost/db
GLOBAL:DEBUG=true
api:PORT=4201
web:PORT=4202
api:API_KEY=secret123`,
			want: &EnvOverrides{
				Global: map[string]string{
					"DATABASE_URL": "postgres://localhost/db",
					"DEBUG":        "true",
				},
				Services: map[string]map[string]string{
					"api": {
						"PORT":    "4201",
						"API_KEY": "secret123",
					},
					"web": {
						"PORT": "4202",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "mixed with non-override lines",
			input: `Setting up environment...
GLOBAL:DATABASE_URL=postgres://localhost/db
Calculating ports...
api:PORT=4201
web:PORT=4202
Done!`,
			want: &EnvOverrides{
				Global: map[string]string{
					"DATABASE_URL": "postgres://localhost/db",
				},
				Services: map[string]map[string]string{
					"api": {
						"PORT": "4201",
					},
					"web": {
						"PORT": "4202",
					},
				},
			},
			wantErr: false,
		},
		{
			name:  "lowercase global scope",
			input: "global:DATABASE_URL=postgres://localhost/db",
			want: &EnvOverrides{
				Global: map[string]string{
					"DATABASE_URL": "postgres://localhost/db",
				},
				Services: map[string]map[string]string{},
			},
			wantErr: false,
		},
		{
			name:  "value with spaces",
			input: "api:MESSAGE=hello world",
			want: &EnvOverrides{
				Global: map[string]string{},
				Services: map[string]map[string]string{
					"api": {
						"MESSAGE": "hello world",
					},
				},
			},
			wantErr: false,
		},
		{
			name:  "value with equals sign",
			input: "api:CONNECTION_STRING=postgres://user:pass=word@localhost/db",
			want: &EnvOverrides{
				Global: map[string]string{},
				Services: map[string]map[string]string{
					"api": {
						"CONNECTION_STRING": "postgres://user:pass=word@localhost/db",
					},
				},
			},
			wantErr: false,
		},
		{
			name:  "value with colons",
			input: "api:URL=http://localhost:3000",
			want: &EnvOverrides{
				Global: map[string]string{},
				Services: map[string]map[string]string{
					"api": {
						"URL": "http://localhost:3000",
					},
				},
			},
			wantErr: false,
		},
		{
			name:  "empty value",
			input: "api:PORT=",
			want: &EnvOverrides{
				Global: map[string]string{},
				Services: map[string]map[string]string{
					"api": {
						"PORT": "",
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "empty key (error)",
			input:   "api:=value",
			want:    nil,
			wantErr: true,
		},
		{
			name: "whitespace handling",
			input: `  GLOBAL:DATABASE_URL=postgres://localhost/db
  api:PORT=4201  `,
			want: &EnvOverrides{
				Global: map[string]string{
					"DATABASE_URL": "postgres://localhost/db",
				},
				Services: map[string]map[string]string{
					"api": {
						"PORT": "4201",
					},
				},
			},
			wantErr: false,
		},
		{
			name:  "no equals sign (ignored)",
			input: "api:PORT",
			want: &EnvOverrides{
				Global:   map[string]string{},
				Services: map[string]map[string]string{},
			},
			wantErr: false,
		},
		{
			name:  "no colon (ignored)",
			input: "PORT=4201",
			want: &EnvOverrides{
				Global:   map[string]string{},
				Services: map[string]map[string]string{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseEnvOverrides(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseEnvOverrides() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			// Compare global overrides
			if len(got.Global) != len(tt.want.Global) {
				t.Errorf("Global overrides count mismatch: got %d, want %d", len(got.Global), len(tt.want.Global))
			}
			for k, v := range tt.want.Global {
				if got.Global[k] != v {
					t.Errorf("Global[%q] = %q, want %q", k, got.Global[k], v)
				}
			}

			// Compare service overrides
			if len(got.Services) != len(tt.want.Services) {
				t.Errorf("Services count mismatch: got %d, want %d", len(got.Services), len(tt.want.Services))
			}
			for serviceName, wantVars := range tt.want.Services {
				gotVars, ok := got.Services[serviceName]
				if !ok {
					t.Errorf("Service %q not found in result", serviceName)
					continue
				}
				if len(gotVars) != len(wantVars) {
					t.Errorf("Service %q vars count mismatch: got %d, want %d", serviceName, len(gotVars), len(wantVars))
				}
				for k, v := range wantVars {
					if gotVars[k] != v {
						t.Errorf("Service %q var %q = %q, want %q", serviceName, k, gotVars[k], v)
					}
				}
			}
		})
	}
}

func TestEnvOverrides_Merge(t *testing.T) {
	tests := []struct {
		name  string
		base  *EnvOverrides
		other *EnvOverrides
		want  *EnvOverrides
	}{
		{
			name: "merge global overrides",
			base: &EnvOverrides{
				Global: map[string]string{
					"KEY1": "value1",
				},
				Services: map[string]map[string]string{},
			},
			other: &EnvOverrides{
				Global: map[string]string{
					"KEY2": "value2",
				},
				Services: map[string]map[string]string{},
			},
			want: &EnvOverrides{
				Global: map[string]string{
					"KEY1": "value1",
					"KEY2": "value2",
				},
				Services: map[string]map[string]string{},
			},
		},
		{
			name: "override global value",
			base: &EnvOverrides{
				Global: map[string]string{
					"KEY": "old_value",
				},
				Services: map[string]map[string]string{},
			},
			other: &EnvOverrides{
				Global: map[string]string{
					"KEY": "new_value",
				},
				Services: map[string]map[string]string{},
			},
			want: &EnvOverrides{
				Global: map[string]string{
					"KEY": "new_value",
				},
				Services: map[string]map[string]string{},
			},
		},
		{
			name: "merge service overrides",
			base: &EnvOverrides{
				Global: map[string]string{},
				Services: map[string]map[string]string{
					"api": {
						"PORT": "4201",
					},
				},
			},
			other: &EnvOverrides{
				Global: map[string]string{},
				Services: map[string]map[string]string{
					"web": {
						"PORT": "4202",
					},
				},
			},
			want: &EnvOverrides{
				Global: map[string]string{},
				Services: map[string]map[string]string{
					"api": {
						"PORT": "4201",
					},
					"web": {
						"PORT": "4202",
					},
				},
			},
		},
		{
			name: "override service value",
			base: &EnvOverrides{
				Global: map[string]string{},
				Services: map[string]map[string]string{
					"api": {
						"PORT": "4201",
						"KEY1": "value1",
					},
				},
			},
			other: &EnvOverrides{
				Global: map[string]string{},
				Services: map[string]map[string]string{
					"api": {
						"PORT": "9999",
					},
				},
			},
			want: &EnvOverrides{
				Global: map[string]string{},
				Services: map[string]map[string]string{
					"api": {
						"PORT": "9999",
						"KEY1": "value1",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.base.Merge(tt.other)

			// Compare global
			if len(tt.base.Global) != len(tt.want.Global) {
				t.Errorf("Global count mismatch: got %d, want %d", len(tt.base.Global), len(tt.want.Global))
			}
			for k, v := range tt.want.Global {
				if tt.base.Global[k] != v {
					t.Errorf("Global[%q] = %q, want %q", k, tt.base.Global[k], v)
				}
			}

			// Compare services
			if len(tt.base.Services) != len(tt.want.Services) {
				t.Errorf("Services count mismatch: got %d, want %d", len(tt.base.Services), len(tt.want.Services))
			}
			for serviceName, wantVars := range tt.want.Services {
				gotVars := tt.base.Services[serviceName]
				if len(gotVars) != len(wantVars) {
					t.Errorf("Service %q vars count mismatch: got %d, want %d", serviceName, len(gotVars), len(wantVars))
				}
				for k, v := range wantVars {
					if gotVars[k] != v {
						t.Errorf("Service %q var %q = %q, want %q", serviceName, k, gotVars[k], v)
					}
				}
			}
		})
	}
}

func TestEnvOverrides_IsEmpty(t *testing.T) {
	tests := []struct {
		name string
		env  *EnvOverrides
		want bool
	}{
		{
			name: "empty",
			env: &EnvOverrides{
				Global:   map[string]string{},
				Services: map[string]map[string]string{},
			},
			want: true,
		},
		{
			name: "has global",
			env: &EnvOverrides{
				Global: map[string]string{
					"KEY": "value",
				},
				Services: map[string]map[string]string{},
			},
			want: false,
		},
		{
			name: "has service",
			env: &EnvOverrides{
				Global: map[string]string{},
				Services: map[string]map[string]string{
					"api": {
						"KEY": "value",
					},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.env.IsEmpty(); got != tt.want {
				t.Errorf("IsEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseEnvOverrides_RealWorldExample(t *testing.T) {
	// Simulate output from a real hook script
	input := `
[dual] Setting up ports for worktree feat/new-feature
Calculating port assignments...
Base ports - api=4101, web=4102, worker=4103
Adding offset +100

api:PORT=4201
web:PORT=4202
worker:PORT=4203

Setting up database connection...
GLOBAL:DATABASE_URL=postgres://localhost/myapp_feat_new_feature
GLOBAL:DEBUG=true

Done! Environment configured.
`

	got, err := ParseEnvOverrides(input)
	if err != nil {
		t.Fatalf("ParseEnvOverrides() error = %v", err)
	}

	// Check global overrides
	wantGlobal := map[string]string{
		"DATABASE_URL": "postgres://localhost/myapp_feat_new_feature",
		"DEBUG":        "true",
	}
	if len(got.Global) != len(wantGlobal) {
		t.Errorf("Global count = %d, want %d", len(got.Global), len(wantGlobal))
	}
	for k, v := range wantGlobal {
		if got.Global[k] != v {
			t.Errorf("Global[%q] = %q, want %q", k, got.Global[k], v)
		}
	}

	// Check service overrides
	wantServices := map[string]map[string]string{
		"api": {
			"PORT": "4201",
		},
		"web": {
			"PORT": "4202",
		},
		"worker": {
			"PORT": "4203",
		},
	}
	if len(got.Services) != len(wantServices) {
		t.Errorf("Services count = %d, want %d", len(got.Services), len(wantServices))
		t.Errorf("Got services: %v", got.Services)
	}
	for serviceName, wantVars := range wantServices {
		gotVars := got.Services[serviceName]
		if len(gotVars) != len(wantVars) {
			t.Errorf("Service %q vars count = %d, want %d", serviceName, len(gotVars), len(wantVars))
		}
		for k, v := range wantVars {
			if gotVars[k] != v {
				t.Errorf("Service %q var %q = %q, want %q", serviceName, k, gotVars[k], v)
			}
		}
	}
}

func TestNewEnvOverrides(t *testing.T) {
	env := NewEnvOverrides()
	if env == nil {
		t.Fatal("NewEnvOverrides() returned nil")
	}
	if env.Global == nil {
		t.Error("Global map is nil")
	}
	if env.Services == nil {
		t.Error("Services map is nil")
	}
	if !env.IsEmpty() {
		t.Error("New EnvOverrides should be empty")
	}
}

// Benchmark parsing performance
func BenchmarkParseEnvOverrides(b *testing.B) {
	input := strings.Repeat("GLOBAL:KEY=value\napi:PORT=4201\nweb:PORT=4202\n", 10)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseEnvOverrides(input)
	}
}
