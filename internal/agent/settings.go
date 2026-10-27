package agent

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/pelletier/go-toml/v2"
)

type Provider struct {
	APIType string `toml:"api_type"`
	BaseURL string `toml:"base_url"`
	APIKey  string `toml:"api_key"`
}

type Settings struct {
	AttachmentDir    string              `toml:"attachment_dir"`
	MaxCodeLines     int                 `toml:"max_code_lines"`
	MaxResultLines   int                 `toml:"max_result_lines"`
	MaxThinkingLines int                 `toml:"max_thinking_lines"`
	Provider         string              `toml:"provider"`
	Model            string              `toml:"model"`
	ReasoningSummary string              `toml:"reasoning_summary"`
	Effort           string              `toml:"effort"`
	Providers        map[string]Provider `toml:"providers"`
}

const defaultSettings = `# Empty uses the platform temporary directory/lopy-images. Relative paths use project cwd.
attachment_dir = ""
# No provider or model is configured by default. In lopy, use /login to add a
# provider, then /model to choose a model.
provider = ""
model = ""
# Leave empty to use the provider's default reasoning effort.
effort = ""
# Set to off for providers/models that do not support reasoning summaries.
reasoning_summary = "auto"
# Visible body rows per timeline block; 0 hides the whole block.
max_code_lines = 999
max_result_lines = 4
max_thinking_lines = 4
`

func loadSettings(dir string) (Settings, error) {
	s := Settings{MaxCodeLines: 999, MaxResultLines: 4, MaxThinkingLines: 4}
	path := filepath.Join(dir, "settings.toml")
	text, err := loadOrCreateFile(path, defaultSettings)
	if err != nil {
		return s, err
	}
	if err = toml.Unmarshal([]byte(text), &s); err != nil {
		return s, fmt.Errorf("%s: %w", path, err)
	}
	if err := validateSettings(s); err != nil {
		return s, fmt.Errorf("%s: %w", path, err)
	}
	return s, nil
}

func validateSettings(s Settings) error {
	if s.MaxCodeLines < 0 || s.MaxResultLines < 0 || s.MaxThinkingLines < 0 {
		return fmt.Errorf("max_*_lines must be non-negative")
	}
	// An empty provider and model mean nothing has been selected yet; that is
	// valid and lopy asks the user to /login and choose a /model.
	if s.Provider != "" {
		if _, ok := s.Providers[s.Provider]; !ok {
			return fmt.Errorf("selected provider %q is not configured", s.Provider)
		}
	}
	for name, p := range s.Providers {
		if p.APIType != "" && p.APIType != "responses" && p.APIType != "chat_completions" {
			return fmt.Errorf("provider %q: unsupported API type %q", name, p.APIType)
		}
		u, err := url.Parse(p.BaseURL)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
			return fmt.Errorf("provider %q needs an HTTP(S) base_url", name)
		}
	}
	if s.ReasoningSummary != "" && s.ReasoningSummary != "auto" && s.ReasoningSummary != "off" {
		return fmt.Errorf("reasoning_summary must be auto or off")
	}
	switch s.Effort {
	case "", "none", "minimal", "low", "medium", "high", "xhigh", "max":
	default:
		return fmt.Errorf("unsupported effort %q", s.Effort)
	}
	return nil
}

func configDir(cwd string) (string, error) {
	dir := os.Getenv("LOPY_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, ".lopy")
	} else if !filepath.IsAbs(dir) {
		dir = filepath.Join(cwd, dir)
	}
	return filepath.Abs(dir)
}

// Settings returns the selected configuration without exposing mutable provider state.
func (a *Agent) Settings() Settings {
	s := a.settings
	s.Providers = make(map[string]Provider, len(a.settings.Providers))
	for name, provider := range a.settings.Providers {
		s.Providers[name] = provider
	}
	return s
}

func (a *Agent) applySettings(s Settings) {
	a.settings = s
	a.client = newClient(s.Providers[s.Provider])
}

func newClient(p Provider) openai.Client {
	return openai.NewClient(
		option.WithBaseURL(strings.TrimRight(p.BaseURL, "/")+"/"),
		option.WithAPIKey(p.APIKey),
		option.WithOrganization(""),
		option.WithProject(""),
		option.WithMaxRetries(0),
		option.WithRequestTimeout(10*time.Minute),
	)
}
