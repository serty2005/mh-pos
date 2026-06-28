package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5/middleware"

	"mh-pos-platform/receipt/engine"
	receiptsvg "mh-pos-platform/receipt/svg"
)

type receiptPreviewRequest struct {
	TemplateContent string          `json:"template_content"`
	DocumentType    string          `json:"document_type"`
	CPL             int             `json:"cpl"`
	PrintContext    json.RawMessage `json:"print_context"`
}

func (h *Handler) receiptPreview(w http.ResponseWriter, r *http.Request) {
	var req receiptPreviewRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		writeReceiptPreviewError(w, r, http.StatusBadRequest, "CONTEXT_SCHEMA_ERROR", "errors.receipts.contextSchema")
		return
	}
	if err := validateReceiptPreviewRequest(req); err != nil {
		writeReceiptPreviewError(w, r, http.StatusBadRequest, "CONTEXT_SCHEMA_ERROR", "errors.receipts.contextSchema")
		return
	}
	printContext, err := decodeReceiptPreviewContext(req.PrintContext)
	if err != nil {
		writeReceiptPreviewError(w, r, http.StatusBadRequest, "CONTEXT_SCHEMA_ERROR", "errors.receipts.contextSchema")
		return
	}
	blocks, err := engine.Render(req.TemplateContent, printContext)
	if err != nil {
		writeReceiptPreviewError(w, r, http.StatusBadRequest, "TEMPLATE_PARSE_ERROR", "errors.receipts.templateParse")
		return
	}
	out, err := receiptsvg.Render(blocks, receiptsvg.RenderOptions{CPL: req.CPL})
	if err != nil {
		writeReceiptPreviewError(w, r, http.StatusBadRequest, "TEMPLATE_PARSE_ERROR", "errors.receipts.templateParse")
		return
	}
	w.Header().Set("Content-Type", "image/svg+xml")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(out))
}

func validateReceiptPreviewRequest(req receiptPreviewRequest) error {
	if strings.TrimSpace(req.TemplateContent) == "" {
		return fmt.Errorf("template_content is required")
	}
	switch strings.TrimSpace(req.DocumentType) {
	case "precheck", "check_nonfiscal", "ticket", "kitchen_service", "cash_in_out", "acceptance":
	default:
		return fmt.Errorf("document_type is invalid")
	}
	if req.CPL != 32 && req.CPL != 48 {
		return fmt.Errorf("cpl must be 32 or 48")
	}
	if len(req.PrintContext) == 0 || string(req.PrintContext) == "null" {
		return fmt.Errorf("print_context is required")
	}
	if _, err := decodeReceiptPreviewContext(req.PrintContext); err != nil {
		return fmt.Errorf("print_context must be an object")
	}
	return nil
}

func decodeReceiptPreviewContext(raw json.RawMessage) (map[string]any, error) {
	var context map[string]any
	if err := json.Unmarshal(raw, &context); err != nil || context == nil {
		return nil, fmt.Errorf("print_context must be an object")
	}
	return context, nil
}

func writeReceiptPreviewError(w http.ResponseWriter, r *http.Request, status int, code, messageKey string) {
	correlationID := middleware.GetReqID(r.Context())
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Error-Code", code)
	if correlationID != "" {
		w.Header().Set("X-Request-ID", correlationID)
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{
		"code":           code,
		"message_key":    messageKey,
		"correlation_id": correlationID,
	}})
}
