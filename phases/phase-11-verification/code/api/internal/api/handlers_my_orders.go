package api

import (
	"net/http"
	"strconv"

	"github.com/biletflow/api/internal/httpx"
	"github.com/biletflow/api/internal/store"
)

type myOrdersResponse struct {
	Orders []store.BuyerOrder `json:"orders"`
}

// handleListMyOrders lists the signed-in attendee's own orders (SRS 4.9).
//
// The per-order page is reachable by its unguessable id alone, which is what
// lets a guest keep their tickets. This endpoint is the other half: once
// somebody has an account, their orders have to be findable without digging out
// the original email (SRS 4.7, "through the attendee's account").
func (s *Server) handleListMyOrders(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil || parsed <= 0 {
			httpx.WriteValidationError(w, fieldErrors{"limit": "Limit must be a positive whole number."})
			return
		}
		limit = parsed
	}

	orders, err := s.myOrders.ListForUser(r.Context(), mustUserID(r.Context()), limit)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, myOrdersResponse{Orders: orders})
}
