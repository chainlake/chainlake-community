// Package providers registers OpenCode (console gateway) as a single vendor,
// mirroring the OpenRouter pattern: one provider entry, endpoint chosen per
// model row. Two plans share the OpenAI-compatible protocol and the same
// Console personal API key (oc_sk_...):
//   - Go subscription (default below): https://opencode.ai/inference/go/openai/v1
//     (/zen/go is an alias serving the same models). Requires the
//     x-opencode-session header per conversation (400 without it); supply it
//     via the model's Custom Headers.
//   - Pay-as-you-go: https://opencode.ai/inference/openai/v1
//     (same key, no session header required).
// Switch plans by editing the model row's Base URL only; protocol, thinking
// switch (chat-template-kwargs, off/auto) and max_tokens field stay the same.
// Verified against the live gateway (2026-11): deepseek-v4.1-flash, kimi-k3,
// glm-5.3 answer OpenAI chat completions.
// Model IDs are the short IDs from https://opencode.ai/inference/go/v1/models
// (e.g. deepseek-v4.1-flash, kimi-k3, glm-5.3); seed data lives in
// internal/models/catalog/data/seed.json ("opencode").
package providers

import (
	_ "embed"

	"github.com/Tencent/WeKnora/internal/models/api"
	"github.com/Tencent/WeKnora/internal/types"
)

//go:embed assets/opencode.svg
var opencodeIcon []byte

// OpencodeID is the provider identifier stored on model rows.
const OpencodeID = "opencode"

// OpencodeGoBaseURL is the documented OpenAI-compatible Go endpoint.
const OpencodeGoBaseURL = "https://opencode.ai/inference/go/openai/v1"

func newOpencodeProvider() *Definition {
	return &Definition{
		ID:   OpencodeID,
		Name: "OpenCode",
		Names: map[string]string{
			"zh-CN": "OpenCode",
		},
		Description: "OpenCode gateway: Go subscription (x-opencode-session header required) and pay-as-you-go share this entry; switch plans via the model row's Base URL.",
		Descriptions: map[string]string{
			"zh-CN": "OpenCode 网关：Go 订阅（需 x-opencode-session 请求头）与按量付费共用此入口，切换套餐只需改模型行的 Base URL。",
		},
		Website: "https://opencode.ai",
		Icon:    opencodeIcon,
		API:     api.APIOpenAICompletions,
		Order:   43,
		RequiresAuth: true,
		Auth:         AuthBearer,
		URLPatterns:  []string{"opencode.ai"},
		DefaultBaseURLs: map[types.ModelType]string{
			types.ModelTypeKnowledgeQA: OpencodeGoBaseURL,
		},
		ModelTypes: []types.ModelType{
			types.ModelTypeKnowledgeQA,
		},
		Compat: VendorCompat{
			OpenAICompletions: api.OpenAICompletionsCompat{
				MaxTokensField: api.Ptr("max_tokens"),
				ThinkingFormat: api.Ptr(api.ThinkingFormatChatTemplateKwargs),
			},
		},
		// off/auto are usable by default (Supports reports them true without
		// a map entry); no wider ladder is claimed until verified per model.
	}
}
