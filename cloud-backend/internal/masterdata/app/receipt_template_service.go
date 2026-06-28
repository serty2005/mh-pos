package app

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"cloud-backend/internal/masterdata/domain"
)

// ReceiptTemplateFilter описывает фильтры списка Cloud receipt templates.
type ReceiptTemplateFilter struct {
	OrgID        string
	RestaurantID string
	DocumentType string
	IsDefault    *bool
	IsActive     *bool
}

// CreateReceiptTemplateCommand описывает создание Cloud-owned шаблона печати.
// RestaurantID пустой = tenant-level default (restaurant_id IS NULL).
type CreateReceiptTemplateCommand struct {
	OrgID        string `json:"org_id,omitempty"`
	RestaurantID string `json:"restaurant_id,omitempty"`
	DocumentType string `json:"document_type"`
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	Content      string `json:"content"`
	Level        *int   `json:"level,omitempty"`
	CPL          int    `json:"cpl"`
	PrinterClass string `json:"printer_class,omitempty"`
	IsDefault    bool   `json:"is_default,omitempty"`
}

// UpdateReceiptTemplateCommand описывает изменение шаблона; version инкрементируется.
type UpdateReceiptTemplateCommand struct {
	DocumentType string  `json:"document_type,omitempty"`
	Name         string  `json:"name,omitempty"`
	Description  *string `json:"description,omitempty"`
	Content      string  `json:"content,omitempty"`
	Level        *int    `json:"level,omitempty"`
	CPL          *int    `json:"cpl,omitempty"`
	PrinterClass *string `json:"printer_class,omitempty"`
	IsDefault    *bool   `json:"is_default,omitempty"`
}

// CreateReceiptTemplate создает Cloud-authored шаблон печати и публикует delivery refresh.
func (s *Service) CreateReceiptTemplate(ctx context.Context, cmd CreateReceiptTemplateCommand) (domain.ReceiptTemplate, error) {
	documentType := domain.ReceiptTemplateDocumentType(strings.TrimSpace(cmd.DocumentType))
	if err := domain.ValidateReceiptTemplateDocumentType(documentType); err != nil {
		return domain.ReceiptTemplate{}, err
	}
	name := strings.TrimSpace(cmd.Name)
	content := cmd.Content
	if name == "" || strings.TrimSpace(content) == "" {
		return domain.ReceiptTemplate{}, fmt.Errorf("%w: name and content are required", domain.ErrInvalid)
	}
	if err := domain.ValidateReceiptTemplateCPL(cmd.CPL); err != nil {
		return domain.ReceiptTemplate{}, err
	}
	level := 1
	if cmd.Level != nil {
		level = *cmd.Level
	}
	if err := domain.ValidateReceiptTemplateLevel(level); err != nil {
		return domain.ReceiptTemplate{}, err
	}
	now := s.clock.Now().UTC()
	template := domain.ReceiptTemplate{
		ID:           s.ids.NewID(),
		OrgID:        strings.TrimSpace(cmd.OrgID),
		RestaurantID: strings.TrimSpace(cmd.RestaurantID),
		DocumentType: documentType,
		Name:         name,
		Description:  strings.TrimSpace(cmd.Description),
		Content:      content,
		Level:        level,
		CPL:          cmd.CPL,
		PrinterClass: domain.NormalizePrinterClass(cmd.PrinterClass),
		IsDefault:    cmd.IsDefault,
		Version:      1,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	stored, err := s.repo.CreateReceiptTemplate(ctx, template)
	return s.afterReceiptTemplateCommit(ctx, stored, err)
}

// UpdateReceiptTemplate изменяет шаблон, инкрементирует version и публикует delivery refresh.
func (s *Service) UpdateReceiptTemplate(ctx context.Context, id string, cmd UpdateReceiptTemplateCommand) (domain.ReceiptTemplate, error) {
	template, err := s.repo.GetReceiptTemplate(ctx, strings.TrimSpace(id))
	if err != nil {
		return domain.ReceiptTemplate{}, err
	}
	if strings.TrimSpace(cmd.DocumentType) != "" {
		documentType := domain.ReceiptTemplateDocumentType(strings.TrimSpace(cmd.DocumentType))
		if err := domain.ValidateReceiptTemplateDocumentType(documentType); err != nil {
			return domain.ReceiptTemplate{}, err
		}
		template.DocumentType = documentType
	}
	if strings.TrimSpace(cmd.Name) != "" {
		template.Name = strings.TrimSpace(cmd.Name)
	}
	if cmd.Description != nil {
		template.Description = strings.TrimSpace(*cmd.Description)
	}
	if strings.TrimSpace(cmd.Content) != "" {
		template.Content = cmd.Content
	}
	if cmd.Level != nil {
		if err := domain.ValidateReceiptTemplateLevel(*cmd.Level); err != nil {
			return domain.ReceiptTemplate{}, err
		}
		template.Level = *cmd.Level
	}
	if cmd.CPL != nil {
		if err := domain.ValidateReceiptTemplateCPL(*cmd.CPL); err != nil {
			return domain.ReceiptTemplate{}, err
		}
		template.CPL = *cmd.CPL
	}
	if cmd.PrinterClass != nil {
		template.PrinterClass = domain.NormalizePrinterClass(*cmd.PrinterClass)
	}
	if cmd.IsDefault != nil {
		template.IsDefault = *cmd.IsDefault
	}
	template.Version++
	template.UpdatedAt = s.clock.Now().UTC()
	stored, err := s.repo.UpdateReceiptTemplate(ctx, template)
	return s.afterReceiptTemplateCommit(ctx, stored, err)
}

// GetReceiptTemplate возвращает один Cloud-owned шаблон печати.
func (s *Service) GetReceiptTemplate(ctx context.Context, id string) (domain.ReceiptTemplate, error) {
	return s.repo.GetReceiptTemplate(ctx, strings.TrimSpace(id))
}

// ListReceiptTemplates возвращает шаблоны с фильтрами document_type/is_default/is_active.
func (s *Service) ListReceiptTemplates(ctx context.Context, filter ReceiptTemplateFilter) ([]domain.ReceiptTemplate, error) {
	filter.OrgID = strings.TrimSpace(filter.OrgID)
	filter.RestaurantID = strings.TrimSpace(filter.RestaurantID)
	filter.DocumentType = strings.TrimSpace(filter.DocumentType)
	return s.repo.ListReceiptTemplates(ctx, filter)
}

// DeactivateReceiptTemplate выполняет soft-delete (is_active = FALSE) и публикует delivery refresh.
func (s *Service) DeactivateReceiptTemplate(ctx context.Context, id string) (domain.ReceiptTemplate, error) {
	template, err := s.repo.GetReceiptTemplate(ctx, strings.TrimSpace(id))
	if err != nil {
		return domain.ReceiptTemplate{}, err
	}
	if !template.IsActive {
		return template, nil
	}
	template.IsActive = false
	template.IsDefault = false
	template.Version++
	template.UpdatedAt = s.clock.Now().UTC()
	stored, err := s.repo.UpdateReceiptTemplate(ctx, template)
	return s.afterReceiptTemplateCommit(ctx, stored, err)
}

// afterReceiptTemplateCommit публикует delivery packages: restaurant-scoped шаблон обновляет
// один ресторан, tenant-level default — всех назначенных.
func (s *Service) afterReceiptTemplateCommit(ctx context.Context, template domain.ReceiptTemplate, err error) (domain.ReceiptTemplate, error) {
	if err != nil {
		return template, err
	}
	if strings.TrimSpace(template.RestaurantID) == "" {
		return afterTenantCommit(s, ctx, template, nil)
	}
	return afterRestaurantCommit(s, ctx, template.RestaurantID, template, nil)
}

// edgeReceiptTemplate описывает строку шаблона в Cloud -> Edge stream receipt_templates.
type edgeReceiptTemplate struct {
	ID           string `json:"id"`
	RestaurantID string `json:"restaurant_id,omitempty"`
	DocumentType string `json:"document_type"`
	Name         string `json:"name"`
	Content      string `json:"content"`
	Level        int    `json:"level"`
	CPL          int    `json:"cpl"`
	PrinterClass string `json:"printer_class"`
	IsDefault    bool   `json:"is_default"`
	Version      int    `json:"version"`
}

type receiptTemplatesPayload struct {
	NodeDeviceID     string                `json:"node_device_id,omitempty"`
	RestaurantID     string                `json:"restaurant_id"`
	SyncMode         string                `json:"sync_mode"`
	CheckpointToken  string                `json:"checkpoint_token,omitempty"`
	CloudVersion     int64                 `json:"cloud_version"`
	CloudUpdatedAt   time.Time             `json:"cloud_updated_at"`
	ReceiptTemplates []edgeReceiptTemplate `json:"receipt_templates"`
}

// receiptTemplatesStream собирает Cloud -> Edge package шаблонов печати для ресторана.
// Эффективный набор: restaurant-specific активные шаблоны плюс tenant-level (restaurant_id IS NULL)
// активные шаблоны для тех document_type, по которым у ресторана нет собственного шаблона.
// Версия пакета фиксируется по MAX(updated_at) активных строк в checkpoint token.
func (s *Service) receiptTemplatesStream(ctx context.Context, restaurantID, nodeDeviceID, syncMode string, cloudVersion int64, updatedAt time.Time) (StreamPackage, int, error) {
	rows, err := s.repo.ListActiveReceiptTemplatesForRestaurant(ctx, restaurantID)
	if err != nil {
		return StreamPackage{}, 0, err
	}
	effective := effectiveReceiptTemplates(rows, restaurantID)
	maxUpdated := time.Unix(0, 0).UTC()
	templates := make([]edgeReceiptTemplate, 0, len(effective))
	for _, row := range effective {
		if row.UpdatedAt.After(maxUpdated) {
			maxUpdated = row.UpdatedAt.UTC()
		}
		templates = append(templates, edgeReceiptTemplate{
			ID:           row.ID,
			RestaurantID: strings.TrimSpace(row.RestaurantID),
			DocumentType: string(row.DocumentType),
			Name:         row.Name,
			Content:      row.Content,
			Level:        row.Level,
			CPL:          row.CPL,
			PrinterClass: row.PrinterClass,
			IsDefault:    row.IsDefault,
			Version:      row.Version,
		})
	}
	// Stream-specific checkpoint фиксирует MAX(updated_at) и число активных строк:
	// Edge применяет пакет только если этот checkpoint отличается от локального.
	checkpoint := fmt.Sprintf("receipt-templates:%s:%d:%d", restaurantID, maxUpdated.UnixMilli(), len(templates))
	body, err := json.Marshal(receiptTemplatesPayload{
		NodeDeviceID:     nodeDeviceID,
		RestaurantID:     restaurantID,
		SyncMode:         syncMode,
		CheckpointToken:  checkpoint,
		CloudVersion:     cloudVersion,
		CloudUpdatedAt:   updatedAt,
		ReceiptTemplates: templates,
	})
	if err != nil {
		return StreamPackage{}, 0, err
	}
	return StreamPackage{
		StreamName:      "receipt_templates",
		NodeDeviceID:    nodeDeviceID,
		RestaurantID:    restaurantID,
		SyncMode:        syncMode,
		CloudVersion:    cloudVersion,
		CheckpointToken: checkpoint,
		CloudUpdatedAt:  updatedAt,
		PayloadJSON:     body,
	}, len(templates), nil
}

// effectiveReceiptTemplates применяет override: restaurant-specific шаблон перекрывает
// tenant-level default для того же document_type. Возвращает детерминированный порядок.
func effectiveReceiptTemplates(rows []domain.ReceiptTemplate, restaurantID string) []domain.ReceiptTemplate {
	restaurantID = strings.TrimSpace(restaurantID)
	coveredByRestaurant := map[string]struct{}{}
	for _, row := range rows {
		if !row.IsActive {
			continue
		}
		if strings.TrimSpace(row.RestaurantID) == restaurantID && restaurantID != "" {
			coveredByRestaurant[string(row.DocumentType)] = struct{}{}
		}
	}
	out := make([]domain.ReceiptTemplate, 0, len(rows))
	for _, row := range rows {
		if !row.IsActive {
			continue
		}
		if strings.TrimSpace(row.RestaurantID) == "" {
			// tenant-level: включаем, только если ресторан не перекрывает этот document_type.
			if _, covered := coveredByRestaurant[string(row.DocumentType)]; covered {
				continue
			}
		}
		out = append(out, row)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
