package handlers

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"mediguide/internal/httpx"
)

// PublicList godoc
// @Summary List drugs
// @Tags drugs
// @Produce json
// @Param page query int false "Page number" minimum(1)
// @Param per_page query int false "Page size" minimum(1) maximum(100)
// @Param search query string false "Drug name, brand, indication, or keyword"
// @Param status query string false "Drug status"
// @Param review_status query string false "Review status"
// @Param drug_class_id query string false "Drug class UUID"
// @Param therapeutic_category_id query string false "Therapeutic category UUID"
// @Param route query string false "Route of administration"
// @Param pregnancy_category query string false "Pregnancy category"
// @Param who_eml query bool false "WHO essential medicines only"
// @Param antimicrobial query bool false "Antimicrobials only"
// @Param sort query string false "Allowed values: name, created_at, updated_at, usage_count"
// @Param order query string false "Allowed values: asc, desc"
// @Success 200 {object} handlers.PaginatedDrugsEnvelope
// @Failure 400 {object} handlers.ErrorResponse
// @Router /api/public/drugs [get]
func (h DrugHandler) PublicList(c *gin.Context) {
	query := c.Request.URL.Query()
	query.Set("status", "active")
	query.Set("review_status", "approved")
	c.Request.URL.RawQuery = query.Encode()
	h.List(c)
}

// PublicGet godoc
// @Summary Get a drug
// @Tags drugs
// @Produce json
// @Param id path string true "Drug ID" format(uuid)
// @Success 200 {object} handlers.DrugEnvelope
// @Failure 404 {object} handlers.ErrorResponse
// @Router /api/public/drugs/{id} [get]
func (h DrugHandler) PublicGet(c *gin.Context) {
	id, ok := drugID(c)
	if !ok {
		return
	}
	item, err := h.Service.Get(id)
	if err != nil {
		h.writeError(c, err)
		return
	}
	if item.Status != "active" || item.ReviewStatus != "approved" {
		h.writeError(c, gorm.ErrRecordNotFound)
		return
	}
	httpx.OK(c, item)
}

// PublicListCategories godoc
// @Summary List drug categories
// @Tags drugs
// @Produce json
// @Success 200 {object} handlers.PaginatedDrugCategoriesEnvelope
// @Router /api/public/drug-categories [get]
func (h DrugReferenceHandler) PublicListCategories(c *gin.Context) {
	query := c.Request.URL.Query()
	query.Set("status", "active")
	c.Request.URL.RawQuery = query.Encode()
	h.ListCategories(c)
}

// PublicListTags godoc
// @Summary List drug tags
// @Tags drugs
// @Produce json
// @Success 200 {object} handlers.PaginatedDrugTagsEnvelope
// @Router /api/public/drug-tags [get]
func (h DrugReferenceHandler) PublicListTags(c *gin.Context) {
	query := c.Request.URL.Query()
	query.Set("status", "active")
	c.Request.URL.RawQuery = query.Encode()
	h.ListTags(c)
}

// PublicListClasses godoc
// @Summary List drug classes
// @Tags drugs
// @Produce json
// @Success 200 {object} handlers.PaginatedDrugClassesEnvelope
// @Router /api/public/drug-classes [get]
func (h DrugReferenceHandler) PublicListClasses(c *gin.Context) {
	query := c.Request.URL.Query()
	query.Set("status", "active")
	c.Request.URL.RawQuery = query.Encode()
	h.ListClasses(c)
}

// PublicListTherapeuticCategories godoc
// @Summary List therapeutic categories
// @Tags drugs
// @Produce json
// @Success 200 {object} handlers.PaginatedTherapeuticCategoriesEnvelope
// @Router /api/public/therapeutic-categories [get]
func (h DrugReferenceHandler) PublicListTherapeuticCategories(c *gin.Context) {
	query := c.Request.URL.Query()
	query.Set("status", "active")
	c.Request.URL.RawQuery = query.Encode()
	h.ListTherapeuticCategories(c)
}

// PublicListAbbreviations godoc
// @Summary List abbreviations
// @Tags guideline-content
// @Success 200 {object} handlers.PaginatedAbbreviationsEnvelope
// @Router /api/public/abbreviations [get]
func (h GuidelineContentHandler) PublicListAbbreviations(c *gin.Context) { h.ListAbbreviations(c) }

// PublicGetAbbreviation godoc
// @Summary Get an abbreviation
// @Tags guideline-content
// @Param id path string true "Abbreviation UUID"
// @Success 200 {object} handlers.AbbreviationEnvelope
// @Router /api/public/abbreviations/{id} [get]
func (h GuidelineContentHandler) PublicGetAbbreviation(c *gin.Context) { h.GetAbbreviation(c) }

// PublicListCategories godoc
// @Summary List guideline categories
// @Tags guideline-taxonomy
// @Param page query int false "Page number"
// @Param per_page query int false "Items per page"
// @Param search query string false "Name, slug, or description"
// @Param status query string false "Status (editors only)"
// @Param parent_id query string false "Parent UUID"
// @Param sort query string false "Allowlisted sort field"
// @Param order query string false "asc or desc"
// @Success 200 {object} handlers.PaginatedGuidelineCategoriesEnvelope
// @Router /api/public/guideline-categories [get]
func (h GuidelineContentHandler) PublicListCategories(c *gin.Context) {
	query, ok := guidelineContentQuery(c)
	if !ok {
		return
	}
	items, err := h.Service.ListCategories(false, query)
	h.page(c, items, err)
}

// PublicGetCategory godoc
// @Summary Get a guideline category
// @Tags guideline-taxonomy
// @Param id path string true "Category UUID"
// @Success 200 {object} handlers.GuidelineCategoryEnvelope
// @Router /api/public/guideline-categories/{id} [get]
func (h GuidelineContentHandler) PublicGetCategory(c *gin.Context) {
	id, ok := guidelineContentID(c)
	if !ok {
		return
	}
	item, err := h.Service.GetCategory(id, false)
	h.result(c, item, err, 200)
}

// PublicListTags godoc
// @Summary List guideline tags
// @Tags guideline-taxonomy
// @Success 200 {object} handlers.PaginatedGuidelineTagsEnvelope
// @Router /api/public/guideline-tags [get]
func (h GuidelineContentHandler) PublicListTags(c *gin.Context) { h.ListTags(c) }

// PublicListFacilities godoc
// @Summary List health facilities
// @Tags facilities
// @Param page query int false "Page"
// @Param per_page query int false "Items per page"
// @Param search query string false "Name, code, or district search"
// @Param region_id query string false "Region UUID"
// @Param district_id query string false "District UUID"
// @Param facility_level_id query string false "Facility level UUID"
// @Param ownership_type_id query string false "Ownership type UUID"
// @Param sort query string false "name, created_at, updated_at, or usage_count"
// @Param order query string false "asc or desc"
// @Success 200 {object} services.FacilityPage
// @Failure 400 {object} handlers.ErrorResponse
// @Router /api/public/facilities [get]
func (h FacilityHandler) PublicListFacilities(c *gin.Context) { h.ListFacilities(c) }

// PublicGetFacility godoc
// @Summary Get a health facility
// @Tags facilities
// @Param id path string true "Facility UUID"
// @Success 200 {object} services.FacilityItem
// @Failure 400,404 {object} handlers.ErrorResponse
// @Router /api/public/facilities/{id} [get]
func (h FacilityHandler) PublicGetFacility(c *gin.Context) { h.GetFacility(c) }

// PublicListHealthSubRegions godoc
// @Summary Read public reference
// @Success 200 {object} map[string]interface{}
// @Router /api/public/health-sub-regions [get]
func (h FacilityHandler) PublicListHealthSubRegions(c *gin.Context) { h.ListHealthSubRegions(c) }

// PublicGetHealthSubRegion godoc
// @Summary Read public reference
// @Success 200 {object} map[string]interface{}
// @Router /api/public/health-sub-regions/{id} [get]
func (h FacilityHandler) PublicGetHealthSubRegion(c *gin.Context) { h.GetHealthSubRegion(c) }

// PublicListRegions godoc
// @Summary Read public reference
// @Success 200 {object} map[string]interface{}
// @Router /api/public/regions [get]
func (h FacilityHandler) PublicListRegions(c *gin.Context) { h.ListRegions(c) }

// PublicGetRegion godoc
// @Summary Read public reference
// @Success 200 {object} map[string]interface{}
// @Router /api/public/regions/{id} [get]
func (h FacilityHandler) PublicGetRegion(c *gin.Context) { h.GetRegion(c) }

// PublicGetRegionChildren godoc
// @Summary Get typed children for a region
// @Tags facilities
// @Param id path string true "Region UUID"
// @Success 200 {object} services.RegionChildren
// @Failure 400,404 {object} handlers.ErrorResponse
// @Router /api/public/regions/{id}/children [get]
func (h FacilityHandler) PublicGetRegionChildren(c *gin.Context) { h.GetRegionChildren(c) }

// PublicListDistricts godoc
// @Summary Read public reference
// @Success 200 {object} map[string]interface{}
// @Router /api/public/districts [get]
func (h FacilityHandler) PublicListDistricts(c *gin.Context) { h.ListDistricts(c) }

// PublicGetDistrict godoc
// @Summary Read public reference
// @Success 200 {object} map[string]interface{}
// @Router /api/public/districts/{id} [get]
func (h FacilityHandler) PublicGetDistrict(c *gin.Context) { h.GetDistrict(c) }

// PublicListHealthSubDistricts godoc
// @Summary Read public reference
// @Success 200 {object} map[string]interface{}
// @Router /api/public/health-sub-districts [get]
func (h FacilityHandler) PublicListHealthSubDistricts(c *gin.Context) { h.ListHealthSubDistricts(c) }

// PublicGetHealthSubDistrict godoc
// @Summary Read public reference
// @Success 200 {object} map[string]interface{}
// @Router /api/public/health-sub-districts/{id} [get]
func (h FacilityHandler) PublicGetHealthSubDistrict(c *gin.Context) { h.GetHealthSubDistrict(c) }

// PublicListCounties godoc
// @Summary Read public reference
// @Success 200 {object} map[string]interface{}
// @Router /api/public/counties [get]
func (h FacilityHandler) PublicListCounties(c *gin.Context) { h.ListCounties(c) }

// PublicGetCounty godoc
// @Summary Read public reference
// @Success 200 {object} map[string]interface{}
// @Router /api/public/counties/{id} [get]
func (h FacilityHandler) PublicGetCounty(c *gin.Context) { h.GetCounty(c) }

// PublicListSubcounties godoc
// @Summary Read public reference
// @Success 200 {object} map[string]interface{}
// @Router /api/public/subcounties [get]
func (h FacilityHandler) PublicListSubcounties(c *gin.Context) { h.ListSubcounties(c) }

// PublicGetSubcounty godoc
// @Summary Read public reference
// @Success 200 {object} map[string]interface{}
// @Router /api/public/subcounties/{id} [get]
func (h FacilityHandler) PublicGetSubcounty(c *gin.Context) { h.GetSubcounty(c) }

// PublicListParishes godoc
// @Summary Read public reference
// @Success 200 {object} map[string]interface{}
// @Router /api/public/parishes [get]
func (h FacilityHandler) PublicListParishes(c *gin.Context) { h.ListParishes(c) }

// PublicGetParish godoc
// @Summary Read public reference
// @Success 200 {object} map[string]interface{}
// @Router /api/public/parishes/{id} [get]
func (h FacilityHandler) PublicGetParish(c *gin.Context) { h.GetParish(c) }

// PublicListFacilityLevels godoc
// @Summary Read public reference
// @Success 200 {object} map[string]interface{}
// @Router /api/public/facility-levels [get]
func (h FacilityHandler) PublicListFacilityLevels(c *gin.Context) { h.ListFacilityLevels(c) }

// PublicGetFacilityLevel godoc
// @Summary Read public reference
// @Success 200 {object} map[string]interface{}
// @Router /api/public/facility-levels/{id} [get]
func (h FacilityHandler) PublicGetFacilityLevel(c *gin.Context) { h.GetFacilityLevel(c) }

// PublicListOwnershipTypes godoc
// @Summary Read public reference
// @Success 200 {object} map[string]interface{}
// @Router /api/public/ownership-types [get]
func (h FacilityHandler) PublicListOwnershipTypes(c *gin.Context) { h.ListOwnershipTypes(c) }

// PublicGetOwnershipType godoc
// @Summary Read public reference
// @Success 200 {object} map[string]interface{}
// @Router /api/public/ownership-types/{id} [get]
func (h FacilityHandler) PublicGetOwnershipType(c *gin.Context) { h.GetOwnershipType(c) }

// PublicListAuthorities godoc
// @Summary Read public reference
// @Success 200 {object} map[string]interface{}
// @Router /api/public/authorities [get]
func (h FacilityHandler) PublicListAuthorities(c *gin.Context) { h.ListAuthorities(c) }

// PublicGetAuthority godoc
// @Summary Read public reference
// @Success 200 {object} map[string]interface{}
// @Router /api/public/authorities/{id} [get]
func (h FacilityHandler) PublicGetAuthority(c *gin.Context) { h.GetAuthority(c) }
