package services

import (
	"errors"
	"fmt"
	"strings"

	"mediguide/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const maxGuidelineLinkLength = 2048

var (
	// ErrGuidelinePublishedAsLink rejects uploads and content editing for
	// documents whose kind publishes an external link.
	ErrGuidelinePublishedAsLink = errors.New("this document is published as an external link and has no uploaded file or editable content")
	// ErrGuidelineLinkNotSupported rejects a link for documents whose kind does
	// not publish as a link.
	ErrGuidelineLinkNotSupported = errors.New("only documents whose kind publishes as a link can have an external URL")
	// ErrGuidelineLinkInvalid rejects anything but a full https URL.
	ErrGuidelineLinkInvalid = errors.New("the link must be a full URL starting with https://, for example https://www.who.int/publications")
)

// VersionPublishesAsLink reports whether the version's document kind publishes
// an external URL.
func (s GuidelineService) VersionPublishesAsLink(versionID uuid.UUID) (bool, error) {
	return versionPublishesAsLink(s.DB, versionID)
}

func versionPublishesAsLink(tx *gorm.DB, versionID uuid.UUID) (bool, error) {
	var version models.GuidelineVersion
	if err := tx.Select("id", "document_id").First(&version, "id = ?", versionID).Error; err != nil {
		return false, err
	}
	var document models.GuidelineDocument
	if err := tx.Select("id", "document_kind_id").First(&document, "id = ?", version.DocumentID).Error; err != nil {
		return false, err
	}
	return documentKindPublishesAsLink(tx, document.DocumentKindID)
}

func documentKindPublishesAsLink(tx *gorm.DB, kindID *uuid.UUID) (bool, error) {
	if kindID == nil {
		return false, nil
	}
	var kind models.DocumentKind
	// Archived kinds still describe the documents that use them.
	if err := tx.Unscoped().Select("id", "publish_as_link").First(&kind, "id = ?", *kindID).Error; err != nil {
		return false, err
	}
	return kind.PublishAsLink, nil
}

// rejectLinkVersion stops file uploads for versions published as a link.
func rejectLinkVersion(tx *gorm.DB, versionID uuid.UUID) error {
	asLink, err := versionPublishesAsLink(tx, versionID)
	if err != nil {
		return err
	}
	if asLink {
		return ErrGuidelinePublishedAsLink
	}
	return nil
}

// validGuidelineLink accepts the same https URLs as outbreak resource links.
func validGuidelineLink(value string) bool {
	return len(value) <= maxGuidelineLinkLength && validApprovedHTTPSURL(value, nil, true)
}

// SetVersionLink stores the external URL a link version publishes. Readers
// open the URL itself, so nothing is extracted or indexed.
func (s GuidelineService) SetVersionLink(versionID, userID uuid.UUID, link, ipAddress string) (*models.GuidelineVersion, error) {
	link = strings.TrimSpace(link)
	var version models.GuidelineVersion
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&version, "id = ?", versionID).Error; err != nil {
			return err
		}
		if err := validateVersionAllowsIngestion(&version); err != nil {
			return err
		}
		asLink, err := versionPublishesAsLink(tx, versionID)
		if err != nil {
			return err
		}
		if !asLink {
			return ErrGuidelineLinkNotSupported
		}
		if !validGuidelineLink(link) {
			return ErrGuidelineLinkInvalid
		}
		if err := tx.Model(&version).Update("external_url", link).Error; err != nil {
			return err
		}
		return writeGuidelineAudit(tx, userID, "guideline.version.link_set", "guideline_version", versionID, ipAddress, map[string]any{"document_id": version.DocumentID, "external_url": link})
	})
	if err != nil {
		return nil, err
	}
	return &version, nil
}

// ensureLinkVersionReadyForPublish requires a valid external URL.
func ensureLinkVersionReadyForPublish(version *models.GuidelineVersion) error {
	if !validGuidelineLink(strings.TrimSpace(version.ExternalURL)) {
		return fmt.Errorf("%w: add the https link before publishing", ErrGuidelineIngestionIncomplete)
	}
	return nil
}
