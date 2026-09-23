package models

import "github.com/google/uuid"

type SupportTicket struct {
	Base
	// UserID is nil for tickets submitted by unauthenticated visitors; those
	// carry the requester's contact details instead.
	UserID         *uuid.UUID `json:"user_id,omitempty"`
	AssignedTo     *uuid.UUID `json:"assigned_to,omitempty"`
	Subject        string     `json:"subject"`
	Description    string     `json:"description"`
	Status         string     `json:"status"`
	Priority       string     `json:"priority"`
	Category       *string    `json:"category,omitempty"`
	RequesterName  *string    `json:"requester_name,omitempty"`
	RequesterEmail *string    `json:"requester_email,omitempty"`
	// UserName and UserEmail resolve to the owner account when present and
	// otherwise to the guest requester details.
	UserName     string `gorm:"->" json:"user_name,omitempty"`
	UserEmail    string `gorm:"->" json:"user_email,omitempty"`
	AssigneeName string `gorm:"->" json:"assignee_name,omitempty"`
}

// IsGuest reports whether the ticket was submitted without an account.
func (t SupportTicket) IsGuest() bool { return t.UserID == nil }

type SupportTicketReply struct {
	Base
	TicketID   uuid.UUID `json:"ticket_id"`
	UserID     uuid.UUID `json:"user_id"`
	Message    string    `json:"message"`
	IsInternal bool      `json:"is_internal"`
	UserName   string    `gorm:"->" json:"user_name,omitempty"`
	UserEmail  string    `gorm:"->" json:"user_email,omitempty"`
}
