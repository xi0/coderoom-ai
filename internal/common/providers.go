package common

type Provider struct {
	Name       string
	ProviderID string
	BaseURL    string
	Models     []Model
}

type Model struct {
	Name          string
	ModelID       string
	ContextWindow int
}

type ProviderModel struct {
	ProviderName  string
	ModelName     string
	BaseURL       string
	ContextWindow int
}

var (
	Providers = []*Provider{
		&Provider{
			Name:       "DeepSeek",
			ProviderID: "deepseek",
			BaseURL:    "https://api.deepseek.com",
			Models: []Model{
				Model{
					Name:          "DeepSeek Flash",
					ModelID:       "deepseek-flash",
					ContextWindow: 1048576,
				},
				Model{
					Name:          "DeepSeek V4 Pro",
					ModelID:       "deepseek-v4-pro",
					ContextWindow: 1048576,
				},
			},
		},
		&Provider{
			Name:       "Depaza",
			ProviderID: "depaza",
			BaseURL:    "https://depaza.com/v1",
			Models: []Model{
				Model{
					Name:          "Max",
					ModelID:       "max",
					ContextWindow: 128000,
				},
				Model{
					Name:          "Advanced",
					ModelID:       "advanced",
					ContextWindow: 128000,
				},
			},
		},
		&Provider{
			Name:       "GDPRchat",
			ProviderID: "gdprchat",
			BaseURL:    "https://gdprchat.eu/api/v1",
			Models: []Model{
				Model{
					Name:          "Code",
					ModelID:       "code",
					ContextWindow: 128000,
				},
			},
		},
		&Provider{
			Name:       "Kimi (Moonshot AI)",
			ProviderID: "kimi",
			BaseURL:    "https://api.moonshot.ai/v1",
			Models: []Model{
				Model{
					Name:          "Kimi K3",
					ModelID:       "kimi-k3",
					ContextWindow: 256000,
				},
			},
		},
		&Provider{
			Name:       "OpenAI",
			ProviderID: "openai",
			BaseURL:    "https://api.openai.com/v1",
			Models: []Model{
				Model{
					Name:          "GPT-6 Luna",
					ModelID:       "gpt-6-luna",
					ContextWindow: 128000,
				},
			},
		},
	}
	ProvidersMap map[string]map[string]*ProviderModel
)

func init() {
	ProvidersMap = make(map[string]map[string]*ProviderModel)

	for _, p := range Providers {
		for _, m := range p.Models {
			if _, ok := ProvidersMap[p.ProviderID]; !ok {
				ProvidersMap[p.ProviderID] = make(map[string]*ProviderModel)
			}

			ProvidersMap[p.ProviderID][m.ModelID] = &ProviderModel{
				ProviderName:  p.Name,
				ModelName:     m.Name,
				BaseURL:       p.BaseURL,
				ContextWindow: m.ContextWindow,
			}
		}
	}
}

// ContextWindow returns the context window size in tokens for the given
// provider and model, or 0 if the model is not known.
func ContextWindow(providerID, modelID string) int {
	models, ok := ProvidersMap[providerID]
	if !ok {
		return 0
	}

	model, ok := models[modelID]
	if !ok {
		return 0
	}

	return model.ContextWindow
}
