package rag

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ORG028658/TheSecondBrain/tui/internal/config"
	"github.com/ORG028658/TheSecondBrain/tui/internal/embeddings"
	"github.com/ORG028658/TheSecondBrain/tui/internal/store"
	openai "github.com/sashabaranov/go-openai"
)

// ConvMsg is one turn in the conversation history.
type ConvMsg struct {
	Role    string // "user" or "assistant"
	Content string
}

// StreamMsg is one event from a streaming query.
type StreamMsg struct {
	Chunk string
	Done  bool
	Refs  []string
	Err   error
}

type RAG struct {
	store *store.Store
	embed *embeddings.Client
	llm   *openai.Client
	cfg   *config.Config
}

func New(s *store.Store, e *embeddings.Client, cfg *config.Config) *RAG {
	c := openai.DefaultConfig(os.Getenv("LLM_COMPATIBLE_API_KEY"))
	c.BaseURL = cfg.LLM.BaseURL
	return &RAG{store: s, embed: e, llm: openai.NewClientWithConfig(c), cfg: cfg}
}

// IndexPage chunks a wiki page and upserts embeddings only if content changed.
func (r *RAG) IndexPage(ctx context.Context, relPath, content, contentHash string) error {
	if r.store.PageHash(relPath) == contentHash {
		return nil
	}
	chunks := chunkText(content, r.cfg.RAG.ChunkSize)
	if len(chunks) == 0 {
		return nil
	}
	vectors, err := r.embed.Embed(ctx, chunks)
	if err != nil {
		return fmt.Errorf("embedding %s: %w", relPath, err)
	}
	r.store.Upsert(relPath, contentHash, chunks, vectors)
	return nil
}

// QueryStream retrieves relevant wiki context, injects conversation history,
// and streams the LLM answer. Answers come strictly from wiki content only.
func (r *RAG) QueryStream(ctx context.Context, question string, history []ConvMsg) <-chan StreamMsg {
	ch := make(chan StreamMsg, 50)

	go func() {
		defer close(ch)

		qVec, err := r.embed.EmbedOne(ctx, question)
		if err != nil {
			ch <- StreamMsg{Err: fmt.Errorf("embedding question: %w", err)}
			return
		}

		rawResults := r.store.Search(qVec, r.cfg.RAG.TopK)

		// Filter by minimum similarity threshold — drop noise before sending to LLM
		minSim := r.cfg.RAG.MinSimilarity
		if minSim == 0 {
			minSim = 0.20 // safe default if not configured
		}
		var results []store.SearchResult
		for _, res := range rawResults {
			if float64(res.Score) >= minSim {
				results = append(results, res)
			}
		}

		if len(results) == 0 {
			ch <- StreamMsg{Chunk: randomNoResultPhrase(question), Done: true}
			return
		}

		// Build context with similarity scores so LLM can make better confidence judgements
		var contextParts []string
		seen := map[string]bool{}
		var refs []string
		for _, res := range results {
			contextParts = append(contextParts, fmt.Sprintf(
				"[wiki: %s | similarity: %.0f%%]\n%s",
				res.WikiPath, res.Score*100, res.Text))
			if !seen[res.WikiPath] {
				seen[res.WikiPath] = true
				refs = append(refs, res.WikiPath)
			}
		}

		const system = `You are a thoughtful assistant for a personal wiki knowledge base. Tone: conversational, like a knowledgeable colleague.

STRICT RULES:
1. Answer ONLY from the provided wiki context. Never use outside knowledge.
2. If the context doesn't contain the answer, say so conversationally and vary your phrasing. Suggest /gap <topic> to flag it.
3. Use [[WikiLink]] notation for concepts from the wiki.
4. Use conversation history to understand follow-ups, but still answer only from the wiki.
5. Keep answers focused — every sentence should add value.

REFERENCES (critical):
- After your answer, list ONLY the sources you actually drew information from.
- For each, include an AI-calculated confidence score (0–100%) showing how directly it contributed to your answer — not just similarity, but actual usage.
- Include a brief reason (5–10 words) why this source was relevant.
- OMIT any source you didn't use or that was only tangentially related.
- Do NOT list a source just because it appeared in the context.

Reference format (exactly):
References:
→ wiki/path/page.md  [score%] — reason`

		// Build message list: system → history → current question with wiki context
		messages := []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: system},
		}

		// Inject last N conversation turns for follow-up awareness
		for _, h := range history {
			role := openai.ChatMessageRoleUser
			if h.Role == "assistant" {
				role = openai.ChatMessageRoleAssistant
			}
			messages = append(messages, openai.ChatCompletionMessage{
				Role:    role,
				Content: h.Content,
			})
		}

		// Current question with retrieved wiki context
		userMsg := fmt.Sprintf("Wiki context:\n%s\n\nQuestion: %s",
			strings.Join(contextParts, "\n\n---\n\n"), question)
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: userMsg,
		})

		stream, err := r.llm.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
			Model:     r.cfg.LLM.Model,
			MaxTokens: r.cfg.LLM.MaxTokens,
			Messages:  messages,
		})
		if err != nil {
			ch <- StreamMsg{Err: fmt.Errorf("starting stream: %w", err)}
			return
		}
		defer stream.Close()

		for {
			resp, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				ch <- StreamMsg{Done: true, Refs: refs}
				return
			}
			if err != nil {
				ch <- StreamMsg{Err: fmt.Errorf("stream error: %w", err)}
				return
			}
			if len(resp.Choices) > 0 {
				ch <- StreamMsg{Chunk: resp.Choices[0].Delta.Content}
			}
		}
	}()

	return ch
}

// TopResult returns the single highest-scoring wiki page for a query.
// Used for correction targeting.
func (r *RAG) TopResult(ctx context.Context, query string) (wikiPath, text string, err error) {
	qVec, err := r.embed.EmbedOne(ctx, query)
	if err != nil {
		return "", "", err
	}
	results := r.store.Search(qVec, 1)
	if len(results) == 0 {
		return "", "", fmt.Errorf("no matching wiki page found")
	}
	return results[0].WikiPath, results[0].Text, nil
}

// randomNoResultPhrase returns a varied, conversational "not in wiki" message.
func randomNoResultPhrase(question string) string {
	// Shorten the question for embedding in the response
	topic := question
	if len(topic) > 60 {
		topic = topic[:57] + "..."
	}
	phrases := []string{
		fmt.Sprintf("Hmm, I don't see anything about \"%s\" in your wiki yet.\n\nMaybe this is a gap worth filling? Drop a relevant source into raw/ and run /pull — or type /gap to flag it for future research.", topic),
		fmt.Sprintf("That doesn't seem to be covered in your wiki yet.\n\nShould I note \"%s\" as a missing topic? Type /gap %s to flag it, or drop a source into raw/ and run /pull.", topic, slugifySimple(topic)),
		fmt.Sprintf("Your wiki doesn't have anything on that yet. It might be worth adding.\n\nDrop a relevant file into raw/ and run /pull, or type /gap to mark this as a research gap."),
		fmt.Sprintf("I couldn't find \"%s\" in your knowledge base.\n\nThis could be a gap in your wiki. Use /gap <topic> to flag it, or add a source to raw/ and run /pull to fill it.", topic),
		fmt.Sprintf("Nothing on that topic in your wiki yet — could be missing information.\n\nWant to add it? Drop a source into raw/ and run /pull, or type /gap to note it for later."),
	}
	return phrases[len(question)%len(phrases)]
}

func slugifySimple(s string) string {
	words := strings.Fields(strings.ToLower(s))
	if len(words) > 4 {
		words = words[:4]
	}
	return strings.Join(words, "-")
}

func chunkText(text string, maxChars int) []string {
	paragraphs := strings.Split(text, "\n\n")
	var chunks []string
	var buf strings.Builder
	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}
		if buf.Len()+len(para)+2 > maxChars && buf.Len() > 0 {
			chunks = append(chunks, buf.String())
			buf.Reset()
		}
		if buf.Len() > 0 {
			buf.WriteString("\n\n")
		}
		buf.WriteString(para)
	}
	if buf.Len() > 0 {
		chunks = append(chunks, buf.String())
	}
	return chunks
}
