package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/rs/zerolog/log"

	"finance_tracker/internal/config"
	llmclient "finance_tracker/internal/llm"
	"finance_tracker/internal/store"
)

type CategoryHandler struct {
	catStore *store.CategoryStore
	txnStore *store.TransactionStore
	events   *EventHub
	cfg      *config.Config
}

func NewCategoryHandler(cs *store.CategoryStore, ts *store.TransactionStore, events *EventHub, cfg *config.Config) *CategoryHandler {
	return &CategoryHandler{catStore: cs, txnStore: ts, events: events, cfg: cfg}
}

func (h *CategoryHandler) List(w http.ResponseWriter, r *http.Request) {
	categories, err := h.catStore.ListAll(r.Context())
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	WriteData(w, categories)
}

// ListUnique returns distinct category names with exclusion status and merchant count.
func (h *CategoryHandler) ListUnique(w http.ResponseWriter, r *http.Request) {
	cats, err := h.catStore.ListUniqueCategories(r.Context())
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	if cats == nil {
		cats = []store.CategoryInfo{}
	}
	WriteData(w, cats)
}

// SetExcluded toggles whether a category is excluded from analysis.
func (h *CategoryHandler) SetExcluded(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Category string `json:"category"`
		Excluded bool   `json:"excluded"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid JSON")
		return
	}
	if body.Category == "" {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "Category name required")
		return
	}

	if err := h.catStore.SetCategoryExcluded(r.Context(), body.Category, body.Excluded); err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	h.events.Broadcast("categories_updated", `{"status":"ok"}`)
	WriteData(w, map[string]string{"status": "ok"})
}

func (h *CategoryHandler) Summary(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	start, _ := strconv.ParseInt(q.Get("start"), 10, 64)
	end, _ := strconv.ParseInt(q.Get("end"), 10, 64)

	if start == 0 || end == 0 {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "start and end query params required (unix timestamps)")
		return
	}

	totals, err := h.txnStore.CountByCategory(r.Context(), start, end)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	WriteData(w, totals)
}

// FindSimilar uses the LLM to find merchants similar to a recently corrected one.
func (h *CategoryHandler) FindSimilar(w http.ResponseWriter, r *http.Request) {
	var body struct {
		MerchantDescription string `json:"merchant_description"`
		Category            string `json:"category"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid JSON")
		return
	}
	if body.MerchantDescription == "" || body.Category == "" {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "merchant_description and category required")
		return
	}

	if h.cfg.OpenRouterURL == "" || h.cfg.OpenRouterAPIKey == "" || h.cfg.OpenRouterModel == "" {
		WriteData(w, []store.SimilarMerchantInfo{})
		return
	}

	ctx := r.Context()

	// Get candidates: LLM-assigned merchants in a different category.
	candidates, err := h.catStore.ListCandidatesForSimilarity(ctx, body.Category)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	if len(candidates) == 0 {
		WriteData(w, []store.SimilarMerchantInfo{})
		return
	}

	// Ask LLM to find similar merchants.
	client := llmclient.NewClient(h.cfg.OpenRouterURL, h.cfg.OpenRouterAPIKey, h.cfg.OpenRouterModel)
	similar, err := llmclient.FindSimilarMerchants(client, body.MerchantDescription, body.Category, candidates)
	if err != nil {
		log.Error().Err(err).Str("merchant", body.MerchantDescription).Msg("Failed to find similar merchants")
		WriteData(w, []store.SimilarMerchantInfo{})
		return
	}

	if len(similar) == 0 {
		WriteData(w, []store.SimilarMerchantInfo{})
		return
	}

	// Get transaction counts for the similar merchants.
	counts, err := h.catStore.GetMerchantTransactionCounts(ctx, similar)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get transaction counts for similar merchants")
		// Continue without counts rather than failing.
		counts = map[string]int{}
	}

	// Build category lookup from candidates.
	catLookup := make(map[string]string)
	for _, c := range candidates {
		catLookup[c.MerchantDescription] = c.Category
	}

	// Build response.
	var results []store.SimilarMerchantInfo
	for _, m := range similar {
		results = append(results, store.SimilarMerchantInfo{
			MerchantDescription: m,
			CurrentCategory:     catLookup[m],
			TransactionCount:    counts[m],
		})
	}

	WriteData(w, results)
}

// BulkApply applies a category to multiple merchants at once.
func (h *CategoryHandler) BulkApply(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Merchants []string `json:"merchants"`
		Category  string   `json:"category"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid JSON")
		return
	}
	if len(body.Merchants) == 0 || body.Category == "" {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "merchants and category required")
		return
	}

	if err := h.catStore.BulkSetCategory(r.Context(), body.Merchants, body.Category); err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	h.events.Broadcast("categories_updated", `{"status":"ok"}`)
	WriteData(w, map[string]string{"status": "ok"})
}
