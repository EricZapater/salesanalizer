package prospector_test

import (
	"salesanalizer/backend/internal/prospector"
	"testing"
)

func TestService_DailyLimit(t *testing.T) {
	service := prospector.NewService(nil)
	limit := service.GetDailyLimit()
	if limit != 50 {
		t.Errorf("esperava límit per defecte de 50, obtingut %d", limit)
	}
}
