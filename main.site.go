package main

import (
	"encoding/xml"
	"fmt"
	"strings"

	apiLib "github.com/hornbill/goApiLib"
)

// siteInCache -- Function to check if passed-thorugh site name has been cached
// if so, pass back the Site ID
func siteInCache(siteName string) (bool, int) {
	boolReturn := false
	intReturn := 0
	mutexSite.Lock()
	//-- Check if in Cache
	for _, site := range Sites {
		if site.SiteName == siteName {
			boolReturn = true
			intReturn = site.SiteID
			break
		}
	}
	mutexSite.Unlock()
	return boolReturn, intReturn
}

// seachSite -- Function to check if passed-through  site  name is on the instance
func searchSite(siteName string, espXmlmc *apiLib.XmlmcInstStruct) (bool, int) {
	boolReturn := false
	intReturn := 0
	//-- ESP Query for site
	espXmlmc.SetParam("application", "com.hornbill.core")
	espXmlmc.SetParam("entity", "Site")
	espXmlmc.SetParam("matchScope", "all")
	espXmlmc.OpenElement("searchFilter")
	espXmlmc.SetParam("column", "h_site_name")
	espXmlmc.SetParam("value", siteName)
	//espXmlmc.SetParam("h_site_name", siteName)
	espXmlmc.CloseElement("searchFilter")
	espXmlmc.SetParam("maxResults", "1")

	var XMLSTRING = espXmlmc.GetParam()
	XMLSiteSearch, xmlmcErr := espXmlmc.Invoke("data", "entityBrowseRecords2")
	if xmlmcErr != nil {
		logger(4, "API Call failed when Searching Site:"+fmt.Sprintf("%v", xmlmcErr), false)
		logger(1, "API XML: "+XMLSTRING, false)
	}
	var xmlRespon xmlmcSiteListResponse

	err := xml.Unmarshal([]byte(XMLSiteSearch), &xmlRespon)
	if err != nil {
		logger(3, "Unable to Search for Site: "+fmt.Sprintf("%v", err), true)
		logger(1, "API XML: "+XMLSTRING, false)
	} else {
		if xmlRespon.MethodResult != "ok" {
			logger(3, "Unable to Search for Site: "+xmlRespon.State.ErrorRet, true)
			logger(1, "API XML: "+XMLSTRING, false)
		} else {
			//-- Check Response
			if xmlRespon.Params.RowData.Row.SiteName != "" {
				if strings.EqualFold(xmlRespon.Params.RowData.Row.SiteName, siteName) {
					intReturn = xmlRespon.Params.RowData.Row.SiteID
					boolReturn = true
					//-- Add Site to Cache
					var newSiteForCache siteListStruct
					newSiteForCache.SiteID = intReturn
					newSiteForCache.SiteName = siteName
					name := []siteListStruct{newSiteForCache}
					mutexSite.Lock()
					Sites = append(Sites, name...)
					mutexSite.Unlock()
				}
			}
		}
	}
	return boolReturn, intReturn
}

/*loadSites
func loadSites()
	pageSize := 20
	rowStart := 0
	//-- Load Results in pages of pageSize
	bar := pb.StartNew(int(count))
	for (pageCount * pageSize) < count {
		pageCount++
		logger(1, "Loading Site List Offset: "+fmt.Sprintf("%d", pageCount)+"\n", false)

		hornbillImport.SetParam("rowstart", strconv.Itoa(pageCount))
		hornbillImport.SetParam("limit", strconv.Itoa(pageSize))
		hornbillImport.SetParam("orderByField", "h_site_name")
		hornbillImport.SetParam("orderByWay", "ascending")

		RespBody, xmlmcErr := hornbillImport.Invoke("apps/com.hornbill.core", "getSitesList")

		var JSONResp xmlmcGroupResponse
		if xmlmcErr != nil {
			logger(4, "Unable to Query Group List "+fmt.Sprintf("%s", xmlmcErr), false)
			break
		}
		err := json.Unmarshal([]byte(RespBody), &JSONResp)
		if err != nil {
			logger(4, "Unable to Query Groups List "+fmt.Sprintf("%s", err), false)
			break
		}
		if JSONResp.State.Error != "" {
			logger(4, "Unable to Query Groups List "+JSONResp.State.Error, false)
			break
		}
		//-- Push into Map

		for index := range JSONResp.Params.Group {
			var newSiteForCache siteListStruct
			newSiteForCache.SiteID = intReturn
			newSiteForCache.SiteName = siteName
			name := []siteListStruct{newSiteForCache}
			mutexSite.Lock()
			Sites = append(Sites, name...)
			mutexSite.Unlock()
		}

		// Add 100
		bar.Add(len(JSONResp.Params.Sites.Row))
		//-- Check for empty result set
		if len(JSONResp.Params.Group) == 0 {
			break
		}
	}
	bar.FinishPrint("Sites Loaded  \n")
}
<methodCall service="apps/com.hornbill.core" method="getSitesList">
	<params>
		<rowstart>0</rowstart>
		<limit>25</limit>
		<orderByField>h_site_name</orderByField>
		<orderByWay>ascending</orderByWay>
	</params>
</methodCall>
*/
