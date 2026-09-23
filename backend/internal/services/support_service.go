package services

import (
	"errors"
	"net/mail"
	"strings"
	"time"

	"mediguide/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrSupportInvalid    = errors.New("invalid support payload")
	ErrSupportForbidden  = errors.New("support operation forbidden")
	ErrSupportTransition = errors.New("invalid support status transition")
)

type SupportService struct{ DB *gorm.DB }

type SupportTicketQuery struct {
	Page                                            PageInput
	Search, Status, Priority, Category, Sort, Order string
	OwnerID, AssignedTo                             *uuid.UUID
}

type SupportTicketCreate struct {
	Subject     string  `json:"subject"`
	Description string  `json:"description"`
	Priority    string  `json:"priority"`
	Category    *string `json:"category"`
	// RequesterName and RequesterEmail identify unauthenticated submitters.
	// They are ignored for tickets created by a signed-in user.
	RequesterName  *string `json:"requester_name"`
	RequesterEmail *string `json:"requester_email"`
}

type SupportTicketUpdate struct {
	Subject     *string `json:"subject"`
	Description *string `json:"description"`
	Priority    *string `json:"priority"`
	Category    *string `json:"category"`
	Status      *string `json:"status"`
	AssignedTo  *string `json:"assigned_to"`
}

type SupportReplyCreate struct {
	Message    string `json:"message"`
	IsInternal bool   `json:"is_internal"`
}

func (s SupportService) ListTickets(actor uuid.UUID, staff bool, in SupportTicketQuery) (*PageResult[models.SupportTicket], error) {
	page := in.Page.Normalize(20, 100)
	query := s.ticketQuery()
	if !staff {
		query = query.Where("st.user_id = ?", actor)
	} else {
		if in.OwnerID != nil {
			query = query.Where("st.user_id = ?", *in.OwnerID)
		}
		if in.AssignedTo != nil {
			query = query.Where("st.assigned_to = ?", *in.AssignedTo)
		}
	}
	if search := strings.TrimSpace(in.Search); search != "" {
		like := "%" + search + "%"
		query = query.Where("LOWER(st.subject) LIKE LOWER(?) OR LOWER(st.description) LIKE LOWER(?)", like, like)
	}
	if in.Status != "" {
		if !oneOf(in.Status, "open", "in_progress", "resolved", "closed") {
			return nil, ErrSupportInvalid
		}
		query = query.Where("st.status = ?", in.Status)
	}
	if in.Priority != "" {
		if !oneOf(in.Priority, "low", "normal", "high", "urgent") {
			return nil, ErrSupportInvalid
		}
		query = query.Where("st.priority = ?", in.Priority)
	}
	if category := strings.TrimSpace(in.Category); category != "" {
		query = query.Where("st.category = ?", category)
	}
	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, err
	}
	orders := map[string]string{"created_at": "st.created_at", "updated_at": "st.updated_at", "subject": "st.subject", "priority": "st.priority", "status": "st.status"}
	column, ok := orders[in.Sort]
	if !ok {
		column = "st.updated_at"
	}
	direction := "DESC"
	if strings.EqualFold(in.Order, "asc") {
		direction = "ASC"
	}
	items := []models.SupportTicket{}
	if err := query.Session(&gorm.Session{}).Order(column + " " + direction).Limit(page.PerPage).Offset(page.Offset()).Find(&items).Error; err != nil {
		return nil, err
	}
	return NewPageResult(items, page, total), nil
}

func (s SupportService) GetTicket(actor, id uuid.UUID, staff bool) (*models.SupportTicket, error) {
	query := s.ticketQuery().Where("st.id = ?", id)
	if !staff {
		query = query.Where("st.user_id = ?", actor)
	}
	var item models.SupportTicket
	if err := query.First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (s SupportService) CreateTicket(actor uuid.UUID, in SupportTicketCreate) (*models.SupportTicket, error) {
	item, err := newSupportTicket(in)
	if err != nil {
		return nil, err
	}
	item.UserID = &actor
	if err := s.DB.Create(item).Error; err != nil {
		return nil, err
	}
	return s.GetTicket(actor, item.ID, false)
}

// CreateGuestTicket records a ticket for a visitor without an account. The
// requester's name and email are mandatory so support staff can follow up.
func (s SupportService) CreateGuestTicket(in SupportTicketCreate) (*models.SupportTicket, error) {
	item, err := newSupportTicket(in)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(derefString(in.RequesterName))
	email := normalizeSupportEmail(derefString(in.RequesterEmail))
	if name == "" || len(name) > 120 || email == "" {
		return nil, ErrSupportInvalid
	}
	item.RequesterName = &name
	item.RequesterEmail = &email
	if err := s.DB.Create(item).Error; err != nil {
		return nil, err
	}
	return s.GetTicket(uuid.Nil, item.ID, true)
}

func newSupportTicket(in SupportTicketCreate) (*models.SupportTicket, error) {
	item := models.SupportTicket{Subject: strings.TrimSpace(in.Subject), Description: strings.TrimSpace(in.Description), Status: "open", Priority: in.Priority, Category: cleanOptional(in.Category)}
	if item.Priority == "" {
		item.Priority = "normal"
	}
	if item.Subject == "" || item.Description == "" || !oneOf(item.Priority, "low", "normal", "high", "urgent") {
		return nil, ErrSupportInvalid
	}
	return &item, nil
}

// normalizeSupportEmail returns the lower-cased bare address, or "" when the
// value is not a single plain email address.
func normalizeSupportEmail(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 254 {
		return ""
	}
	parsed, err := mail.ParseAddress(value)
	if err != nil || parsed.Name != "" || !strings.EqualFold(parsed.Address, value) {
		return ""
	}
	return strings.ToLower(parsed.Address)
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func (s SupportService) UpdateTicket(actor, id uuid.UUID, staff bool, in SupportTicketUpdate) (*models.SupportTicket, error) {
	item, err := s.GetTicket(actor, id, staff)
	if err != nil {
		return nil, err
	}
	updates := map[string]any{}
	if in.Subject != nil {
		value := strings.TrimSpace(*in.Subject)
		if value == "" {
			return nil, ErrSupportInvalid
		}
		updates["subject"] = value
	}
	if in.Description != nil {
		value := strings.TrimSpace(*in.Description)
		if value == "" {
			return nil, ErrSupportInvalid
		}
		updates["description"] = value
	}
	if in.Priority != nil {
		if !oneOf(*in.Priority, "low", "normal", "high", "urgent") {
			return nil, ErrSupportInvalid
		}
		updates["priority"] = *in.Priority
	}
	if in.Category != nil {
		updates["category"] = cleanOptional(in.Category)
	}
	if !staff && (in.Status != nil || in.AssignedTo != nil || item.Status != "open") {
		return nil, ErrSupportForbidden
	}
	if staff && in.Status != nil {
		if !validTicketTransition(item.Status, *in.Status) {
			return nil, ErrSupportTransition
		}
		updates["status"] = *in.Status
	}
	if staff && in.AssignedTo != nil {
		value := strings.TrimSpace(*in.AssignedTo)
		if value == "" {
			updates["assigned_to"] = nil
		} else {
			assigned, parseErr := uuid.Parse(value)
			if parseErr != nil {
				return nil, ErrSupportInvalid
			}
			updates["assigned_to"] = assigned
		}
	}
	if len(updates) == 0 {
		return nil, ErrSupportInvalid
	}
	updates["updated_at"] = time.Now().UTC()
	if err := s.DB.Model(&models.SupportTicket{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, err
	}
	return s.GetTicket(actor, id, staff)
}

func (s SupportService) DeleteTicket(actor, id uuid.UUID, staff bool) error {
	item, err := s.GetTicket(actor, id, staff)
	if err != nil {
		return err
	}
	if !staff && item.Status != "open" {
		return ErrSupportForbidden
	}
	return s.DB.Delete(&models.SupportTicket{}, "id = ?", id).Error
}

func (s SupportService) ListReplies(actor, ticketID uuid.UUID, staff bool, page PageInput) (*PageResult[models.SupportTicketReply], error) {
	if _, err := s.GetTicket(actor, ticketID, staff); err != nil {
		return nil, err
	}
	p := page.Normalize(50, 100)
	query := s.DB.Table("support_ticket_replies str").Select("str.*, u.name AS user_name, u.email AS user_email").Joins("LEFT JOIN users u ON u.id = str.user_id").Where("str.deleted_at IS NULL AND str.ticket_id = ?", ticketID)
	if !staff {
		query = query.Where("str.is_internal = false")
	}
	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, err
	}
	items := []models.SupportTicketReply{}
	if err := query.Order("str.created_at ASC, str.id ASC").Limit(p.PerPage).Offset(p.Offset()).Find(&items).Error; err != nil {
		return nil, err
	}
	return NewPageResult(items, p, total), nil
}

func (s SupportService) CreateReply(actor, ticketID uuid.UUID, staff bool, in SupportReplyCreate) (*models.SupportTicketReply, error) {
	if _, err := s.GetTicket(actor, ticketID, staff); err != nil {
		return nil, err
	}
	message := strings.TrimSpace(in.Message)
	if message == "" {
		return nil, ErrSupportInvalid
	}
	if in.IsInternal && !staff {
		return nil, ErrSupportForbidden
	}
	item := models.SupportTicketReply{TicketID: ticketID, UserID: actor, Message: message, IsInternal: in.IsInternal}
	if err := s.DB.Create(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (s SupportService) ticketQuery() *gorm.DB {
	return s.DB.Table("support_tickets st").Select("st.*, COALESCE(owner.name, st.requester_name) AS user_name, COALESCE(owner.email, st.requester_email) AS user_email, assignee.name AS assignee_name").Joins("LEFT JOIN users owner ON owner.id = st.user_id").Joins("LEFT JOIN users assignee ON assignee.id = st.assigned_to").Where("st.deleted_at IS NULL")
}

func validTicketTransition(from, to string) bool {
	if from == to {
		return true
	}
	allowed := map[string]map[string]bool{"open": {"in_progress": true, "closed": true}, "in_progress": {"open": true, "resolved": true, "closed": true}, "resolved": {"in_progress": true, "closed": true}, "closed": {"open": true}}
	return allowed[from][to]
}
