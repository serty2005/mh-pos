package receipt

// Routing scope constants are shared by Edge route validation and worker target matching.
const (
	ScopeRestaurant = "restaurant"
	ScopeSalesPoint = "sales_point"
	ScopeSection    = "section"

	SectionModeHallSection     = "hall_section"
	SectionModeKitchenWorkshop = "kitchen_workshop"
)

// RequiredScopeType returns the single allowed routing scope for a document type.
func RequiredScopeType(documentType DocumentType) (string, bool) {
	switch documentType {
	case DocumentCheckNonfiscal:
		return ScopeSalesPoint, true
	case DocumentPrecheck, DocumentTicket:
		return ScopeSection, true
	case DocumentKitchenService:
		return ScopeSection, true
	case DocumentReport:
		return ScopeRestaurant, true
	default:
		return "", false
	}
}

// RequiredSectionMode returns the section mode required by section-scoped documents.
func RequiredSectionMode(documentType DocumentType) (string, bool) {
	switch documentType {
	case DocumentPrecheck, DocumentTicket:
		return SectionModeHallSection, true
	case DocumentKitchenService:
		return SectionModeKitchenWorkshop, true
	default:
		return "", false
	}
}
