package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"

	"charm.land/fantasy"
	"charm.land/x/vcr"
	"github.com/charmbracelet/catwalk/pkg/catwalk"
	"github.com/charmbracelet/crush/internal/agent/tools"
	"github.com/charmbracelet/crush/internal/message"
	"github.com/charmbracelet/crush/internal/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/joho/godotenv/autoload"
)

var modelPairs = []modelPair{
	{"anthropic-sonnet", anthropicBuilder("claude-sonnet-4-5-20250929"), anthropicBuilder("claude-3-5-haiku-20241022")},
	{"openai-gpt-5", openaiBuilder("gpt-5"), openaiBuilder("gpt-4o")},
	{"openrouter-kimi-k2", openRouterBuilder("moonshotai/kimi-k2-0905"), openRouterBuilder("qwen/qwen3-next-80b-a3b-instruct")},
	{"zai-glm4.6", zAIBuilder("glm-4.6"), zAIBuilder("glm-4.5-air")},
}

func getModels(t *testing.T, r *vcr.Recorder, pair modelPair) (fantasy.LanguageModel, fantasy.LanguageModel) {
	large, err := pair.largeModel(t, r)
	require.NoError(t, err)
	small, err := pair.smallModel(t, r)
	require.NoError(t, err)
	return large, small
}

func setupAgent(t *testing.T, pair modelPair) (SessionAgent, fakeEnv) {
	// Skip zai tests if API key is not set.
	if strings.HasPrefix(pair.name, "zai") && os.Getenv("CRUSH_ZAI_API_KEY") == "" {
		t.Skip("skipping zai tests: CRUSH_ZAI_API_KEY not set")
	}

	r := vcr.NewRecorder(t)
	large, small := getModels(t, r, pair)
	env := testEnv(t)

	createSimpleGoProject(t, env.workingDir)
	agent, err := coderAgent(r, env, large, small)
	require.NoError(t, err)
	return agent, env
}

func TestCoderAgent(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows for now")
	}

	for _, pair := range modelPairs {
		t.Run(pair.name, func(t *testing.T) {
			t.Run("simple test", func(t *testing.T) {
				agent, env := setupAgent(t, pair)

				session, err := env.sessions.Create(t.Context(), "New Session")
				require.NoError(t, err)

				res, err := agent.Run(t.Context(), SessionAgentCall{
					Prompt:          "Hello",
					SessionID:       session.ID,
					MaxOutputTokens: 10000,
				})
				require.NoError(t, err)
				assert.NotNil(t, res)

				msgs, err := env.messages.List(t.Context(), session.ID)
				require.NoError(t, err)
				// Should have the agent and user message
				assert.Equal(t, len(msgs), 2)
			})
			t.Run("read a file", func(t *testing.T) {
				agent, env := setupAgent(t, pair)

				session, err := env.sessions.Create(t.Context(), "New Session")
				require.NoError(t, err)
				res, err := agent.Run(t.Context(), SessionAgentCall{
					Prompt:          "Read the go mod",
					SessionID:       session.ID,
					MaxOutputTokens: 10000,
				})

				require.NoError(t, err)
				assert.NotNil(t, res)

				msgs, err := env.messages.List(t.Context(), session.ID)
				require.NoError(t, err)
				foundFile := false
				var tcID string
			out:
				for _, msg := range msgs {
					if msg.Role == message.Assistant {
						for _, tc := range msg.ToolCalls() {
							if tc.Name == tools.ViewToolName {
								tcID = tc.ID
							}
						}
					}
					if msg.Role == message.Tool {
						for _, tr := range msg.ToolResults() {
							if tr.ToolCallID == tcID {
								if strings.Contains(tr.Content, "module example.com/testproject") {
									foundFile = true
									break out
								}
							}
						}
					}
				}
				require.True(t, foundFile)
			})
			t.Run("update a file", func(t *testing.T) {
				agent, env := setupAgent(t, pair)

				session, err := env.sessions.Create(t.Context(), "New Session")
				require.NoError(t, err)

				res, err := agent.Run(t.Context(), SessionAgentCall{
					Prompt:          "update the main.go file by changing the print to say hello from crush",
					SessionID:       session.ID,
					MaxOutputTokens: 10000,
				})
				require.NoError(t, err)
				assert.NotNil(t, res)

				msgs, err := env.messages.List(t.Context(), session.ID)
				require.NoError(t, err)

				foundRead := false
				foundWrite := false
				var readTCID, writeTCID string

				for _, msg := range msgs {
					if msg.Role == message.Assistant {
						for _, tc := range msg.ToolCalls() {
							if tc.Name == tools.ViewToolName {
								readTCID = tc.ID
							}
							if tc.Name == tools.EditToolName || tc.Name == tools.WriteToolName {
								writeTCID = tc.ID
							}
						}
					}
					if msg.Role == message.Tool {
						for _, tr := range msg.ToolResults() {
							if tr.ToolCallID == readTCID {
								foundRead = true
							}
							if tr.ToolCallID == writeTCID {
								foundWrite = true
							}
						}
					}
				}

				require.True(t, foundRead, "Expected to find a read operation")
				require.True(t, foundWrite, "Expected to find a write operation")

				mainGoPath := filepath.Join(env.workingDir, "main.go")
				content, err := os.ReadFile(mainGoPath)
				require.NoError(t, err)
				require.Contains(t, strings.ToLower(string(content)), "hello from crush")
			})
			t.Run("bash tool", func(t *testing.T) {
				agent, env := setupAgent(t, pair)

				session, err := env.sessions.Create(t.Context(), "New Session")
				require.NoError(t, err)

				res, err := agent.Run(t.Context(), SessionAgentCall{
					Prompt:          "use bash to create a file named test.txt with content 'hello bash'. do not print its timestamp",
					SessionID:       session.ID,
					MaxOutputTokens: 10000,
				})
				require.NoError(t, err)
				assert.NotNil(t, res)

				msgs, err := env.messages.List(t.Context(), session.ID)
				require.NoError(t, err)

				foundBash := false
				var bashTCID string

				for _, msg := range msgs {
					if msg.Role == message.Assistant {
						for _, tc := range msg.ToolCalls() {
							if tc.Name == tools.BashToolName {
								bashTCID = tc.ID
							}
						}
					}
					if msg.Role == message.Tool {
						for _, tr := range msg.ToolResults() {
							if tr.ToolCallID == bashTCID {
								foundBash = true
							}
						}
					}
				}

				require.True(t, foundBash, "Expected to find a bash operation")

				testFilePath := filepath.Join(env.workingDir, "test.txt")
				content, err := os.ReadFile(testFilePath)
				require.NoError(t, err)
				require.Contains(t, string(content), "hello bash")
			})
			t.Run("download tool", func(t *testing.T) {
				agent, env := setupAgent(t, pair)

				session, err := env.sessions.Create(t.Context(), "New Session")
				require.NoError(t, err)

				res, err := agent.Run(t.Context(), SessionAgentCall{
					Prompt:          "download the file from https://example-files.online-convert.com/document/txt/example.txt and save it as example.txt",
					SessionID:       session.ID,
					MaxOutputTokens: 10000,
				})
				require.NoError(t, err)
				assert.NotNil(t, res)

				msgs, err := env.messages.List(t.Context(), session.ID)
				require.NoError(t, err)

				foundDownload := false
				var downloadTCID string

				for _, msg := range msgs {
					if msg.Role == message.Assistant {
						for _, tc := range msg.ToolCalls() {
							if tc.Name == tools.DownloadToolName {
								downloadTCID = tc.ID
							}
						}
					}
					if msg.Role == message.Tool {
						for _, tr := range msg.ToolResults() {
							if tr.ToolCallID == downloadTCID {
								foundDownload = true
							}
						}
					}
				}

				require.True(t, foundDownload, "Expected to find a download operation")

				examplePath := filepath.Join(env.workingDir, "example.txt")
				_, err = os.Stat(examplePath)
				require.NoError(t, err, "Expected example.txt file to exist")
			})
			t.Run("fetch tool", func(t *testing.T) {
				agent, env := setupAgent(t, pair)

				session, err := env.sessions.Create(t.Context(), "New Session")
				require.NoError(t, err)

				res, err := agent.Run(t.Context(), SessionAgentCall{
					Prompt:          "fetch the content from https://example-files.online-convert.com/website/html/example.html and tell me if it contains the word 'John Doe'",
					SessionID:       session.ID,
					MaxOutputTokens: 10000,
				})
				require.NoError(t, err)
				assert.NotNil(t, res)

				msgs, err := env.messages.List(t.Context(), session.ID)
				require.NoError(t, err)

				foundFetch := false
				var fetchTCID string

				for _, msg := range msgs {
					if msg.Role == message.Assistant {
						for _, tc := range msg.ToolCalls() {
							if tc.Name == tools.FetchToolName {
								fetchTCID = tc.ID
							}
						}
					}
					if msg.Role == message.Tool {
						for _, tr := range msg.ToolResults() {
							if tr.ToolCallID == fetchTCID {
								foundFetch = true
							}
						}
					}
				}

				require.True(t, foundFetch, "Expected to find a fetch operation")
			})
			t.Run("glob tool", func(t *testing.T) {
				agent, env := setupAgent(t, pair)

				session, err := env.sessions.Create(t.Context(), "New Session")
				require.NoError(t, err)

				res, err := agent.Run(t.Context(), SessionAgentCall{
					Prompt:          "use glob to find all .go files in the current directory",
					SessionID:       session.ID,
					MaxOutputTokens: 10000,
				})
				require.NoError(t, err)
				assert.NotNil(t, res)

				msgs, err := env.messages.List(t.Context(), session.ID)
				require.NoError(t, err)

				foundGlob := false
				var globTCID string

				for _, msg := range msgs {
					if msg.Role == message.Assistant {
						for _, tc := range msg.ToolCalls() {
							if tc.Name == tools.GlobToolName {
								globTCID = tc.ID
							}
						}
					}
					if msg.Role == message.Tool {
						for _, tr := range msg.ToolResults() {
							if tr.ToolCallID == globTCID {
								foundGlob = true
								require.Contains(t, tr.Content, "main.go", "Expected glob to find main.go")
							}
						}
					}
				}

				require.True(t, foundGlob, "Expected to find a glob operation")
			})
			t.Run("grep tool", func(t *testing.T) {
				agent, env := setupAgent(t, pair)

				session, err := env.sessions.Create(t.Context(), "New Session")
				require.NoError(t, err)

				res, err := agent.Run(t.Context(), SessionAgentCall{
					Prompt:          "use grep to search for the word 'package' in go files",
					SessionID:       session.ID,
					MaxOutputTokens: 10000,
				})
				require.NoError(t, err)
				assert.NotNil(t, res)

				msgs, err := env.messages.List(t.Context(), session.ID)
				require.NoError(t, err)

				foundGrep := false
				var grepTCID string

				for _, msg := range msgs {
					if msg.Role == message.Assistant {
						for _, tc := range msg.ToolCalls() {
							if tc.Name == tools.GrepToolName {
								grepTCID = tc.ID
							}
						}
					}
					if msg.Role == message.Tool {
						for _, tr := range msg.ToolResults() {
							if tr.ToolCallID == grepTCID {
								foundGrep = true
								require.Contains(t, tr.Content, "main.go", "Expected grep to find main.go")
							}
						}
					}
				}

				require.True(t, foundGrep, "Expected to find a grep operation")
			})
			t.Run("ls tool", func(t *testing.T) {
				agent, env := setupAgent(t, pair)

				session, err := env.sessions.Create(t.Context(), "New Session")
				require.NoError(t, err)

				res, err := agent.Run(t.Context(), SessionAgentCall{
					Prompt:          "use ls to list the files in the current directory",
					SessionID:       session.ID,
					MaxOutputTokens: 10000,
				})
				require.NoError(t, err)
				assert.NotNil(t, res)

				msgs, err := env.messages.List(t.Context(), session.ID)
				require.NoError(t, err)

				foundLS := false
				var lsTCID string

				for _, msg := range msgs {
					if msg.Role == message.Assistant {
						for _, tc := range msg.ToolCalls() {
							if tc.Name == tools.LSToolName {
								lsTCID = tc.ID
							}
						}
					}
					if msg.Role == message.Tool {
						for _, tr := range msg.ToolResults() {
							if tr.ToolCallID == lsTCID {
								foundLS = true
								require.Contains(t, tr.Content, "main.go", "Expected ls to list main.go")
								require.Contains(t, tr.Content, "go.mod", "Expected ls to list go.mod")
							}
						}
					}
				}

				require.True(t, foundLS, "Expected to find an ls operation")
			})
			t.Run("multiedit tool", func(t *testing.T) {
				agent, env := setupAgent(t, pair)

				session, err := env.sessions.Create(t.Context(), "New Session")
				require.NoError(t, err)

				res, err := agent.Run(t.Context(), SessionAgentCall{
					Prompt:          "use multiedit to change 'Hello, World!' to 'Hello, Crush!' and add a comment '// Greeting' above the fmt.Println line in main.go",
					SessionID:       session.ID,
					MaxOutputTokens: 10000,
				})
				require.NoError(t, err)
				assert.NotNil(t, res)

				msgs, err := env.messages.List(t.Context(), session.ID)
				require.NoError(t, err)

				foundMultiEdit := false
				var multiEditTCID string

				for _, msg := range msgs {
					if msg.Role == message.Assistant {
						for _, tc := range msg.ToolCalls() {
							if tc.Name == tools.MultiEditToolName {
								multiEditTCID = tc.ID
							}
						}
					}
					if msg.Role == message.Tool {
						for _, tr := range msg.ToolResults() {
							if tr.ToolCallID == multiEditTCID {
								foundMultiEdit = true
							}
						}
					}
				}

				require.True(t, foundMultiEdit, "Expected to find a multiedit operation")

				mainGoPath := filepath.Join(env.workingDir, "main.go")
				content, err := os.ReadFile(mainGoPath)
				require.NoError(t, err)
				require.Contains(t, string(content), "Hello, Crush!", "Expected file to contain 'Hello, Crush!'")
			})
			t.Run("sourcegraph tool", func(t *testing.T) {
				if runtime.GOOS == "darwin" {
					t.Skip("skipping flakey test on macos for now")
				}

				agent, env := setupAgent(t, pair)

				session, err := env.sessions.Create(t.Context(), "New Session")
				require.NoError(t, err)

				res, err := agent.Run(t.Context(), SessionAgentCall{
					Prompt:          "use sourcegraph to search for 'func main' in Go repositories",
					SessionID:       session.ID,
					MaxOutputTokens: 10000,
				})
				require.NoError(t, err)
				assert.NotNil(t, res)

				msgs, err := env.messages.List(t.Context(), session.ID)
				require.NoError(t, err)

				foundSourcegraph := false
				var sourcegraphTCID string

				for _, msg := range msgs {
					if msg.Role == message.Assistant {
						for _, tc := range msg.ToolCalls() {
							if tc.Name == tools.SourcegraphToolName {
								sourcegraphTCID = tc.ID
							}
						}
					}
					if msg.Role == message.Tool {
						for _, tr := range msg.ToolResults() {
							if tr.ToolCallID == sourcegraphTCID {
								foundSourcegraph = true
							}
						}
					}
				}

				require.True(t, foundSourcegraph, "Expected to find a sourcegraph operation")
			})
			t.Run("write tool", func(t *testing.T) {
				agent, env := setupAgent(t, pair)

				session, err := env.sessions.Create(t.Context(), "New Session")
				require.NoError(t, err)

				res, err := agent.Run(t.Context(), SessionAgentCall{
					Prompt:          "use write to create a new file called config.json with content '{\"name\": \"test\", \"version\": \"1.0.0\"}'",
					SessionID:       session.ID,
					MaxOutputTokens: 10000,
				})
				require.NoError(t, err)
				assert.NotNil(t, res)

				msgs, err := env.messages.List(t.Context(), session.ID)
				require.NoError(t, err)

				foundWrite := false
				var writeTCID string

				for _, msg := range msgs {
					if msg.Role == message.Assistant {
						for _, tc := range msg.ToolCalls() {
							if tc.Name == tools.WriteToolName {
								writeTCID = tc.ID
							}
						}
					}
					if msg.Role == message.Tool {
						for _, tr := range msg.ToolResults() {
							if tr.ToolCallID == writeTCID {
								foundWrite = true
							}
						}
					}
				}

				require.True(t, foundWrite, "Expected to find a write operation")

				configPath := filepath.Join(env.workingDir, "config.json")
				content, err := os.ReadFile(configPath)
				require.NoError(t, err)
				require.Contains(t, string(content), "test", "Expected config.json to contain 'test'")
				require.Contains(t, string(content), "1.0.0", "Expected config.json to contain '1.0.0'")
			})
			t.Run("parallel tool calls", func(t *testing.T) {
				agent, env := setupAgent(t, pair)

				session, err := env.sessions.Create(t.Context(), "New Session")
				require.NoError(t, err)

				res, err := agent.Run(t.Context(), SessionAgentCall{
					Prompt:          "use glob to find all .go files and use ls to list the current directory, it is very important that you run both tool calls in parallel",
					SessionID:       session.ID,
					MaxOutputTokens: 10000,
				})
				require.NoError(t, err)
				assert.NotNil(t, res)

				msgs, err := env.messages.List(t.Context(), session.ID)
				require.NoError(t, err)

				var assistantMsg *message.Message
				var toolMsgs []message.Message

				for _, msg := range msgs {
					if msg.Role == message.Assistant && len(msg.ToolCalls()) > 0 {
						assistantMsg = &msg
					}
					if msg.Role == message.Tool {
						toolMsgs = append(toolMsgs, msg)
					}
				}

				require.NotNil(t, assistantMsg, "Expected to find an assistant message with tool calls")
				require.NotNil(t, toolMsgs, "Expected to find a tool message")

				toolCalls := assistantMsg.ToolCalls()
				require.GreaterOrEqual(t, len(toolCalls), 2, "Expected at least 2 tool calls in parallel")

				foundGlob := false
				foundLS := false
				var globTCID, lsTCID string

				for _, tc := range toolCalls {
					if tc.Name == tools.GlobToolName {
						foundGlob = true
						globTCID = tc.ID
					}
					if tc.Name == tools.LSToolName {
						foundLS = true
						lsTCID = tc.ID
					}
				}

				require.True(t, foundGlob, "Expected to find a glob tool call")
				require.True(t, foundLS, "Expected to find an ls tool call")

				require.GreaterOrEqual(t, len(toolMsgs), 2, "Expected at least 2 tool results in the same message")

				foundGlobResult := false
				foundLSResult := false

				for _, msg := range toolMsgs {
					for _, tr := range msg.ToolResults() {
						if tr.ToolCallID == globTCID {
							foundGlobResult = true
							require.Contains(t, tr.Content, "main.go", "Expected glob result to contain main.go")
							require.False(t, tr.IsError, "Expected glob result to not be an error")
						}
						if tr.ToolCallID == lsTCID {
							foundLSResult = true
							require.Contains(t, tr.Content, "main.go", "Expected ls result to contain main.go")
							require.False(t, tr.IsError, "Expected ls result to not be an error")
						}
					}
				}

				require.True(t, foundGlobResult, "Expected to find glob tool result")
				require.True(t, foundLSResult, "Expected to find ls tool result")
			})
		})
	}
}

func makeTestTodos(n int) []session.Todo {
	todos := make([]session.Todo, n)
	for i := range n {
		todos[i] = session.Todo{
			Status:  session.TodoStatusPending,
			Content: fmt.Sprintf("Task %d: Implement feature with some description that makes it realistic", i),
		}
	}
	return todos
}

func BenchmarkBuildSummaryPrompt(b *testing.B) {
	cases := []struct {
		name     string
		numTodos int
	}{
		{"0todos", 0},
		{"5todos", 5},
		{"10todos", 10},
		{"50todos", 50},
	}

	for _, tc := range cases {
		todos := makeTestTodos(tc.numTodos)

		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				_ = buildSummaryPrompt(todos)
			}
		})
	}
}

// mockLanguageModel is a mock language model for testing error handling.
type mockLanguageModel struct {
	streamCallCount atomic.Int32
	errorOnCall     int32 // 0-indexed call number to error on, -1 means never error
	errorToReturn   error
	responses       []string
}

func (m *mockLanguageModel) Generate(_ context.Context, _ fantasy.Call) (*fantasy.Response, error) {
	return &fantasy.Response{
		Content:      fantasy.ResponseContent{fantasy.TextContent{Text: "Generated response"}},
		FinishReason: fantasy.FinishReasonStop,
	}, nil
}

func (m *mockLanguageModel) Stream(_ context.Context, _ fantasy.Call) (fantasy.StreamResponse, error) {
	callNum := m.streamCallCount.Add(1) - 1

	if m.errorOnCall >= 0 && callNum == m.errorOnCall {
		return nil, m.errorToReturn
	}

	responseIdx := int(callNum)
	if m.errorOnCall >= 0 && callNum > m.errorOnCall {
		responseIdx-- // Adjust index since we skipped the error call
	}
	if responseIdx >= len(m.responses) {
		responseIdx = len(m.responses) - 1
	}

	responseText := "Hello! How can I help you?"
	if responseIdx >= 0 && responseIdx < len(m.responses) {
		responseText = m.responses[responseIdx]
	}

	return func(yield func(fantasy.StreamPart) bool) {
		if !yield(fantasy.StreamPart{Type: fantasy.StreamPartTypeTextDelta, Delta: responseText}) {
			return
		}
		yield(fantasy.StreamPart{
			Type:         fantasy.StreamPartTypeFinish,
			FinishReason: fantasy.FinishReasonStop,
			Usage:        fantasy.Usage{InputTokens: 100, OutputTokens: 50, TotalTokens: 150},
		})
	}, nil
}

func (m *mockLanguageModel) GenerateObject(_ context.Context, _ fantasy.ObjectCall) (*fantasy.ObjectResponse, error) {
	return nil, nil
}

func (m *mockLanguageModel) StreamObject(_ context.Context, _ fantasy.ObjectCall) (fantasy.ObjectStreamResponse, error) {
	return nil, nil
}

func (m *mockLanguageModel) Provider() string {
	return "mock"
}

func (m *mockLanguageModel) Model() string {
	return "mock-model"
}

func TestInputTooLongAutoSummarize(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows for now")
	}

	t.Run("auto summarizes and resumes on input too long error", func(t *testing.T) {
		env := testEnv(t)
		createSimpleGoProject(t, env.workingDir)

		// Create a mock model that returns "input too long" error on first call.
		largeModel := &mockLanguageModel{
			errorOnCall:   0, // Error on first call
			errorToReturn: &fantasy.ProviderError{Message: "Input is too long for requested model"},
			responses: []string{
				// Response for summarization call
				"This is a summary of the conversation.",
				// Response after resumption
				"I can help you with that now that we have summarized.",
			},
		}
		smallModel := &mockLanguageModel{
			errorOnCall: -1, // Never error
			responses:   []string{"Summary title"},
		}

		largeModelWrapper := Model{
			Model: largeModel,
			CatwalkCfg: catwalk.Model{
				ContextWindow:    200000,
				DefaultMaxTokens: 10000,
			},
		}
		smallModelWrapper := Model{
			Model: smallModel,
			CatwalkCfg: catwalk.Model{
				ContextWindow:    200000,
				DefaultMaxTokens: 10000,
			},
		}

		agent := NewSessionAgent(SessionAgentOptions{
			LargeModel:           largeModelWrapper,
			SmallModel:           smallModelWrapper,
			SystemPrompt:         "You are a helpful assistant.",
			DisableAutoSummarize: false,
			Sessions:             env.sessions,
			Messages:             env.messages,
			Tools:                []fantasy.AgentTool{},
		})

		sess, err := env.sessions.Create(t.Context(), "Test Session")
		require.NoError(t, err)

		// Run the agent - it should handle the "input too long" error gracefully.
		res, err := agent.Run(t.Context(), SessionAgentCall{
			Prompt:          "Hello, help me with something",
			SessionID:       sess.ID,
			MaxOutputTokens: 10000,
		})

		// The agent should recover from the error by summarizing and resuming.
		require.NoError(t, err)
		require.NotNil(t, res)

		// Verify the large model was called multiple times:
		// 1. First call (errored with input too long)
		// 2. Summarization call
		// 3. Resumed call after summarization
		callCount := largeModel.streamCallCount.Load()
		require.GreaterOrEqual(t, callCount, int32(2), "Expected at least 2 calls to large model (error + resumption)")

		// Verify messages were created in the session.
		msgs, err := env.messages.List(t.Context(), sess.ID)
		require.NoError(t, err)
		require.Greater(t, len(msgs), 0, "Expected messages to be created")
	})

	t.Run("does not summarize when disabled", func(t *testing.T) {
		env := testEnv(t)
		createSimpleGoProject(t, env.workingDir)

		// Create a mock model that returns "input too long" error.
		largeModel := &mockLanguageModel{
			errorOnCall:   0, // Error on first call
			errorToReturn: &fantasy.ProviderError{Message: "Input is too long for requested model"},
			responses:     []string{"Response"},
		}
		smallModel := &mockLanguageModel{
			errorOnCall: -1,
			responses:   []string{"Title"},
		}

		largeModelWrapper := Model{
			Model: largeModel,
			CatwalkCfg: catwalk.Model{
				ContextWindow:    200000,
				DefaultMaxTokens: 10000,
			},
		}
		smallModelWrapper := Model{
			Model: smallModel,
			CatwalkCfg: catwalk.Model{
				ContextWindow:    200000,
				DefaultMaxTokens: 10000,
			},
		}

		agent := NewSessionAgent(SessionAgentOptions{
			LargeModel:           largeModelWrapper,
			SmallModel:           smallModelWrapper,
			SystemPrompt:         "You are a helpful assistant.",
			DisableAutoSummarize: true, // Disabled
			Sessions:             env.sessions,
			Messages:             env.messages,
			Tools:                []fantasy.AgentTool{},
		})

		sess, err := env.sessions.Create(t.Context(), "Test Session")
		require.NoError(t, err)

		// Run the agent - with auto-summarize disabled, it should return the error.
		_, err = agent.Run(t.Context(), SessionAgentCall{
			Prompt:          "Hello",
			SessionID:       sess.ID,
			MaxOutputTokens: 10000,
		})

		// Should return the error since auto-summarize is disabled.
		require.Error(t, err)
		require.True(t, isInputTooLongError(err), "Expected input too long error")

		// Verify only one call was made (no summarization attempt).
		callCount := largeModel.streamCallCount.Load()
		require.Equal(t, int32(1), callCount, "Expected exactly 1 call to large model")
	})

	t.Run("handles various input too long error messages", func(t *testing.T) {
		errorMessages := []string{
			"Input is too long for requested model",
			"context_length_exceeded: max 128000 tokens",
			"This request exceeds the maximum context length",
			"The token limit has been exceeded",
			"prompt is too long for this model",
			"request too large for context window",
		}

		for _, errMsg := range errorMessages {
			t.Run(errMsg, func(t *testing.T) {
				env := testEnv(t)
				createSimpleGoProject(t, env.workingDir)

				largeModel := &mockLanguageModel{
					errorOnCall:   0,
					errorToReturn: &fantasy.ProviderError{Message: errMsg},
					responses: []string{
						"Summary",
						"Resumed response",
					},
				}
				smallModel := &mockLanguageModel{
					errorOnCall: -1,
					responses:   []string{"Title"},
				}

				largeModelWrapper := Model{
					Model: largeModel,
					CatwalkCfg: catwalk.Model{
						ContextWindow:    200000,
						DefaultMaxTokens: 10000,
					},
				}
				smallModelWrapper := Model{
					Model: smallModel,
					CatwalkCfg: catwalk.Model{
						ContextWindow:    200000,
						DefaultMaxTokens: 10000,
					},
				}

				agent := NewSessionAgent(SessionAgentOptions{
					LargeModel:           largeModelWrapper,
					SmallModel:           smallModelWrapper,
					SystemPrompt:         "You are a helpful assistant.",
					DisableAutoSummarize: false,
					Sessions:             env.sessions,
					Messages:             env.messages,
					Tools:                []fantasy.AgentTool{},
				})

				sess, err := env.sessions.Create(t.Context(), "Test Session")
				require.NoError(t, err)

				res, err := agent.Run(t.Context(), SessionAgentCall{
					Prompt:          "Test prompt",
					SessionID:       sess.ID,
					MaxOutputTokens: 10000,
				})

				require.NoError(t, err, "Expected agent to handle '%s' error gracefully", errMsg)
				require.NotNil(t, res)
			})
		}
	})

	t.Run("verifies work continues after input too long recovery", func(t *testing.T) {
		env := testEnv(t)
		createSimpleGoProject(t, env.workingDir)

		// Create a mock model that errors on first call, then succeeds with
		// meaningful content on subsequent calls.
		summaryResponse := "Conversation summary: User asked for help with code."
		resumedResponse := "Here is the help you requested after summarization."
		largeModel := &mockLanguageModel{
			errorOnCall:   0, // Error on first call
			errorToReturn: &fantasy.ProviderError{Message: "Input is too long for requested model"},
			responses: []string{
				// Response for summarization call
				summaryResponse,
				// Response after resumption
				resumedResponse,
			},
		}
		smallModel := &mockLanguageModel{
			errorOnCall: -1, // Never error
			responses:   []string{"Session Summary"},
		}

		largeModelWrapper := Model{
			Model: largeModel,
			CatwalkCfg: catwalk.Model{
				ContextWindow:    200000,
				DefaultMaxTokens: 10000,
			},
		}
		smallModelWrapper := Model{
			Model: smallModel,
			CatwalkCfg: catwalk.Model{
				ContextWindow:    200000,
				DefaultMaxTokens: 10000,
			},
		}

		agent := NewSessionAgent(SessionAgentOptions{
			LargeModel:           largeModelWrapper,
			SmallModel:           smallModelWrapper,
			SystemPrompt:         "You are a helpful assistant.",
			DisableAutoSummarize: false,
			Sessions:             env.sessions,
			Messages:             env.messages,
			Tools:                []fantasy.AgentTool{},
		})

		sess, err := env.sessions.Create(t.Context(), "Test Session")
		require.NoError(t, err)

		res, err := agent.Run(t.Context(), SessionAgentCall{
			Prompt:          "Help me with my code",
			SessionID:       sess.ID,
			MaxOutputTokens: 10000,
		})

		require.NoError(t, err)
		require.NotNil(t, res)

		// Verify messages were created - check for summary message and resumed response.
		msgs, err := env.messages.List(t.Context(), sess.ID)
		require.NoError(t, err)
		require.Greater(t, len(msgs), 1, "Expected multiple messages after recovery")

		// Verify we have at least one assistant message with the resumed content.
		foundSummary := false
		foundResumed := false
		for _, msg := range msgs {
			if msg.Role == message.Assistant {
				content := msg.Content().Text
				if strings.Contains(content, "summary") || strings.Contains(content, "Summary") {
					foundSummary = true
				}
				if strings.Contains(content, resumedResponse) {
					foundResumed = true
				}
			}
		}

		// We should find the summary message and the resumed response.
		require.True(t, foundSummary || foundResumed,
			"Expected to find assistant messages after input too long recovery")

		// Verify the model was called multiple times (error + summary + resumed).
		callCount := largeModel.streamCallCount.Load()
		require.GreaterOrEqual(t, callCount, int32(2),
			"Expected at least 2 calls to large model (error + resumption)")
	})
}

// TestRateLimitErrorDetection verifies that rate-limit errors are correctly
// identified by the error detection functions. The actual retry behavior
// is handled by fantasy's built-in retry mechanism with our OnRetry callback.
func TestRateLimitErrorDetection(t *testing.T) {
	t.Parallel()

	t.Run("detects 429 status code", func(t *testing.T) {
		t.Parallel()
		err := &fantasy.ProviderError{
			Message:    "Too Many Requests",
			StatusCode: 429,
		}
		require.True(t, isRateLimitError(err))
		require.True(t, isRetryableError(err))
	})

	t.Run("detects rate limit message without 429", func(t *testing.T) {
		t.Parallel()
		err := &fantasy.ProviderError{
			Message:    "Rate limit exceeded. Please try again later.",
			StatusCode: 200,
		}
		require.True(t, isRateLimitError(err))
		require.True(t, isRetryableError(err))
	})

	t.Run("detects RetryError wrapper", func(t *testing.T) {
		t.Parallel()
		err := &fantasy.RetryError{
			Errors: []error{
				&fantasy.ProviderError{Message: "first attempt", StatusCode: 429},
				&fantasy.ProviderError{Message: "second attempt", StatusCode: 429},
			},
		}
		require.True(t, isRetryableError(err))
	})

	t.Run("distinguishes rate limit from input too long", func(t *testing.T) {
		t.Parallel()
		rateLimitErr := &fantasy.ProviderError{
			Message:    "Rate limit exceeded",
			StatusCode: 429,
		}
		inputTooLongErr := &fantasy.ProviderError{
			Message: "Input is too long for requested model",
		}

		require.True(t, isRateLimitError(rateLimitErr))
		require.False(t, isInputTooLongError(rateLimitErr))

		require.False(t, isRateLimitError(inputTooLongErr))
		require.True(t, isInputTooLongError(inputTooLongErr))
	})
}
