package app

import (
	"github.com/gin-gonic/gin"
	"mediguide/internal/handlers"
)

func registerPublicReferenceRoutes(public *gin.RouterGroup, drugH handlers.DrugHandler, drugReferenceH handlers.DrugReferenceHandler, guidelineContentH handlers.GuidelineContentHandler, facilityH handlers.FacilityHandler) {
	public.GET("/drugs", drugH.PublicList)
	public.GET("/drugs/:id", drugH.PublicGet)
	public.GET("/drug-categories", drugReferenceH.PublicListCategories)
	public.GET("/drug-tags", drugReferenceH.PublicListTags)
	public.GET("/drug-classes", drugReferenceH.PublicListClasses)
	public.GET("/therapeutic-categories", drugReferenceH.PublicListTherapeuticCategories)
	public.GET("/abbreviations", guidelineContentH.PublicListAbbreviations)
	public.GET("/abbreviations/:id", guidelineContentH.PublicGetAbbreviation)
	public.GET("/guideline-categories", guidelineContentH.PublicListCategories)
	public.GET("/guideline-categories/:id", guidelineContentH.PublicGetCategory)
	public.GET("/guideline-tags", guidelineContentH.PublicListTags)
	public.GET("/facilities", facilityH.PublicListFacilities)
	public.GET("/facilities/:id", facilityH.PublicGetFacility)
	public.GET("/health-sub-regions", facilityH.PublicListHealthSubRegions)
	public.GET("/health-sub-regions/:id", facilityH.PublicGetHealthSubRegion)
	public.GET("/regions", facilityH.PublicListRegions)
	public.GET("/regions/:id", facilityH.PublicGetRegion)
	public.GET("/regions/:id/children", facilityH.PublicGetRegionChildren)
	public.GET("/districts", facilityH.PublicListDistricts)
	public.GET("/districts/:id", facilityH.PublicGetDistrict)
	public.GET("/health-sub-districts", facilityH.PublicListHealthSubDistricts)
	public.GET("/health-sub-districts/:id", facilityH.PublicGetHealthSubDistrict)
	public.GET("/counties", facilityH.PublicListCounties)
	public.GET("/counties/:id", facilityH.PublicGetCounty)
	public.GET("/subcounties", facilityH.PublicListSubcounties)
	public.GET("/subcounties/:id", facilityH.PublicGetSubcounty)
	public.GET("/parishes", facilityH.PublicListParishes)
	public.GET("/parishes/:id", facilityH.PublicGetParish)
	public.GET("/facility-levels", facilityH.PublicListFacilityLevels)
	public.GET("/facility-levels/:id", facilityH.PublicGetFacilityLevel)
	public.GET("/ownership-types", facilityH.PublicListOwnershipTypes)
	public.GET("/ownership-types/:id", facilityH.PublicGetOwnershipType)
	public.GET("/authorities", facilityH.PublicListAuthorities)
	public.GET("/authorities/:id", facilityH.PublicGetAuthority)
}
