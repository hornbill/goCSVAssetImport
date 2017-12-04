package main

//----- Packages -----
import (
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hornbill/color"
	"github.com/hornbill/goApiLib"
	"github.com/hornbill/pb"
)

//----- Main Function -----
func main() {
	//-- Start Time for Durration
	startTime = time.Now()
	//-- Start Time for Log File
	TimeNow = time.Now().Format(time.RFC3339)
	APITimeNow = strings.Replace(TimeNow, "T", " ", 1)
	APITimeNow = strings.Replace(APITimeNow, "Z", "", 1)
	//-- Remove :
	TimeNow = strings.Replace(TimeNow, ":", "-", -1)
	//-- Grab Flags
	flag.StringVar(&configFileName, "file", "conf.json", "Name of Configuration File To Load")
	flag.StringVar(&configZone, "zone", "eur", "Override the default Zone the instance sits in")
	flag.BoolVar(&configDryRun, "dryrun", false, "Allow the Import to run without Creating or Updating Assets")
	flag.StringVar(&configMaxRoutines, "concurrent", "1", "Maximum number of Assets to import concurrently.")
	//-- Parse Flags
	flag.Parse()

	//-- Output
	logger(1, "---- Hornbill CSV Asset Import Utility V"+version+" ----", true)
	logger(1, "Flag - Config File "+configFileName, true)
	logger(1, "Flag - Zone "+configZone, true)
	logger(1, "Flag - Dry Run "+fmt.Sprintf("%v", configDryRun), true)

	//Check maxGoroutines for valid value
	maxRoutines, err := strconv.Atoi(configMaxRoutines)
	if err != nil {
		color.Red("Unable to convert maximum concurrency of [" + configMaxRoutines + "] to type INT for processing")
		return
	}
	maxGoroutines = maxRoutines

	if maxGoroutines < 1 || maxGoroutines > 10 {
		color.Red("The maximum concurrent assets allowed is between 1 and 10 (inclusive).\n\n")
		color.Red("You have selected " + configMaxRoutines + ". Please try again, with a valid value against ")
		color.Red("the -concurrent switch.")
		return
	}

	//--
	//-- Load Configuration File Into Struct
	CSVImportConf = loadConfig()
	if CSVImportConf.LogSizeBytes > 0 {
		maxLogFileSize = CSVImportConf.LogSizeBytes
	}

	//-- Set Instance ID
	SetInstance(configZone, CSVImportConf.InstanceID)
	//-- Generate Instance XMLMC Endpoint
	CSVImportConf.URL = getInstanceURL()

	//Get asset types, process accordingly
	for k, v := range CSVImportConf.AssetTypes {
		StrAssetType = fmt.Sprintf("%v", k)
		StrCSVFile = fmt.Sprintf("%v", v)
		//Set Asset Class & Type vars from instance
		AssetClass, AssetTypeID = getAssetClass(StrAssetType)
		//-- Query Database
		var boolAssets, arrAssets = getAssetsFromCSV(StrCSVFile, StrAssetType)
		if boolAssets {
			//Process records returned by query
			processAssets(arrAssets)
		}
	}

	//-- End output
	logger(1, "Updated: "+fmt.Sprintf("%d", counters.updated), true)
	logger(1, "Updated Skipped: "+fmt.Sprintf("%d", counters.updatedSkipped), true)
	logger(1, "Created: "+fmt.Sprintf("%d", counters.created), true)
	logger(1, "Created Skipped: "+fmt.Sprintf("%d", counters.createskipped), true)
	//-- Show Time Takens
	endTime = time.Since(startTime)
	logger(1, "Time Taken: "+fmt.Sprintf("%v", endTime), true)
	logger(1, "---- Hornbill CSV Asset Import Complete ---- ", true)
}

//loadConfig -- Function to Load Configruation File
func loadConfig() csvImportConfStruct {
	//-- Check Config File File Exists
	cwd, _ := os.Getwd()
	configurationFilePath := cwd + "/" + configFileName
	logger(1, "Loading Config File: "+configurationFilePath, false)
	if _, fileCheckErr := os.Stat(configurationFilePath); os.IsNotExist(fileCheckErr) {
		logger(4, "No Configuration File", true)
		os.Exit(102)
	}
	//-- Load Config File
	file, fileError := os.Open(configurationFilePath)
	//-- Check For Error Reading File
	if fileError != nil {
		logger(4, "Error Opening Configuration File: "+fmt.Sprintf("%v", fileError), true)
	}

	//-- New Decoder
	decoder := json.NewDecoder(file)
	//-- New Var based on CSVImportConf
	ecsvConf := csvImportConfStruct{}
	//-- Decode JSON
	err := decoder.Decode(&ecsvConf)
	//-- Error Checking
	if err != nil {
		logger(4, "Error Decoding Configuration File: "+fmt.Sprintf("%v", err), true)
	}
	//-- Return New Congfig
	return ecsvConf
}

//getAssetClass -- Get Asset Class & Type ID from Asset Type Name
func getAssetClass(confAssetType string) (assetClass string, assetType int) {
	espXmlmc := apiLib.NewXmlmcInstance(CSVImportConf.URL)
	espXmlmc.SetAPIKey(CSVImportConf.APIKey)
	espXmlmc.SetParam("application", appServiceManager)
	espXmlmc.SetParam("entity", "AssetsTypes")
	espXmlmc.OpenElement("searchFilter")
	espXmlmc.SetParam("h_name", confAssetType)
	espXmlmc.CloseElement("searchFilter")
	espXmlmc.SetParam("maxResults", "1")
	var XMLSTRING = espXmlmc.GetParam()
	XMLGetMeta, xmlmcErr := espXmlmc.Invoke("data", "entityBrowseRecords")
	if xmlmcErr != nil {
		logger(4, "API Call failed when retrieving Asset Class:"+fmt.Sprintf("%v", xmlmcErr), false)
		logger(1, "API XML: "+XMLSTRING, false)
	}

	var xmlRespon xmlmcTypeListResponse
	err := xml.Unmarshal([]byte(XMLGetMeta), &xmlRespon)
	if err != nil {
		logger(4, "Could not get Asset Class and Type. Please check AssetType within your configuration file:"+fmt.Sprintf("%v", err), true)
		logger(1, "API XML: "+XMLSTRING, false)
	} else {
		assetClass = xmlRespon.Params.RowData.Row.TypeClass
		assetType = xmlRespon.Params.RowData.Row.TypeID
	}
	return
}

//processAssets -- Processes Assets from Asset Map
//--If asset already exists on the instance, update
//--If asset doesn't exist, create
func processAssets(arrAssets []map[string]string) {
	bar := pb.StartNew(len(arrAssets))
	logger(1, "Processing Assets", false)

	//Create 2 channels for sending and recieving from workers
	jobs := make(chan map[string]string, len(arrAssets))
	results := make(chan bool, len(arrAssets))

	for w := 1; w <= maxGoroutines; w++ {
		go workers(jobs, results)
	}

	for _, assetRecord := range arrAssets {
		jobs <- assetRecord
		//Update our progressbar
		mutexBar.Lock()
		bar.Increment()
		mutexBar.Unlock()
	}
	//closing the channel kills the go routine workers we have spun up, once the channel returns nil when reading
	close(jobs)

	//Loop over our results channel and throw away the results, this is to stop terminating early with work still going on.
	for a := 0; a < len(arrAssets); a++ {
		//We are not really worried about the results just want to make sure all workers complete before me move on
		<-results
	}

	bar.FinishPrint("Processing Complete!")
}

//Workers that do the updating and reuse the same apilib pool for all work.
func workers(jobs chan map[string]string, result chan bool) {

	espXmlmc := apiLib.NewXmlmcInstance(CSVImportConf.URL)
	espXmlmc.SetAPIKey(CSVImportConf.APIKey)
	assetIDIdent := fmt.Sprintf("%v", CSVImportConf.CSVAssetIdentifier)

	for asset := range jobs {

		//Get the asset ID for the current record
		assetID := asset[assetIDIdent]
		time.Sleep(1 * time.Millisecond)

		var boolUpdate = false
		boolUpdate, assetIDInstance := getAssetID(assetID, espXmlmc)
		//-- Update or Create Asset
		if boolUpdate {
			logger(1, "Update Asset: "+assetID, false)
			updateAsset(asset, assetIDInstance, espXmlmc)
		} else {
			logger(1, "Create Asset: "+assetID, false)
			createAsset(asset, espXmlmc)
		}

		result <- true
	}
}

//getAssetID -- Check if asset is on the instance
//-- Returns true, assetid if so
//-- Returns false, "" if not
func getAssetID(assetName string, espXmlmc *apiLib.XmlmcInstStruct) (bool, string) {
	boolReturn := false
	returnAssetID := ""
	espXmlmc.SetParam("application", appServiceManager)
	espXmlmc.SetParam("entity", "Asset")
	espXmlmc.OpenElement("searchFilter")
	espXmlmc.SetParam("h_name", assetName)
	espXmlmc.CloseElement("searchFilter")
	espXmlmc.SetParam("maxResults", "1")
	var XMLSTRING = espXmlmc.GetParam()
	XMLAssetSearch, xmlmcErr := espXmlmc.Invoke("data", "entityBrowseRecords")
	if xmlmcErr != nil {
		logger(4, "API Call failed when searching instance for existing Asset:"+fmt.Sprintf("%v", xmlmcErr), false)
		logger(1, "API Call XML: "+XMLSTRING, false)
	} else {
		var xmlRespon xmlmcAssetResponse
		err := xml.Unmarshal([]byte(XMLAssetSearch), &xmlRespon)
		if err != nil {
			logger(3, "Unable to Search for Asset: "+fmt.Sprintf("%v", err), true)
			logger(1, "API Call XML: "+XMLSTRING, false)
		} else {
			if xmlRespon.MethodResult != "ok" {
				logger(3, "Unable to Search for Asset: "+xmlRespon.State.ErrorRet, true)
				logger(1, "API Call XML: "+XMLSTRING, false)
			} else {
				returnAssetID = xmlRespon.Params.RowData.Row.AssetID
				//-- Check Response
				if returnAssetID != "" {
					boolReturn = true
				}
			}
		}
	}
	return boolReturn, returnAssetID
}

// createAsset -- Creates Asset record from the passed through map data
func createAsset(u map[string]string, espXmlmc *apiLib.XmlmcInstStruct) {
	//Get site ID
	siteID := ""
	siteNameMapping := fmt.Sprintf("%v", CSVImportConf.AssetGenericFieldMapping["h_site"])
	siteName := getFieldValue("h_site", siteNameMapping, u)
	if siteName != "" {
		siteIsInCache, SiteIDCache := siteInCache(siteName)
		//-- Check if we have cached the site already
		if siteIsInCache {
			siteID = strconv.Itoa(SiteIDCache)
		} else {
			siteIsOnInstance, SiteIDInstance := searchSite(siteName, espXmlmc)
			//-- If Returned set output
			if siteIsOnInstance {
				siteID = strconv.Itoa(SiteIDInstance)
			}
		}
	}

	//Get Owned By name
	ownedByName := ""
	ownedByURN := ""
	ownedByMapping := fmt.Sprintf("%v", CSVImportConf.AssetGenericFieldMapping["h_owned_by"])
	ownedByID := getFieldValue("h_owned_by", ownedByMapping, u)
	if ownedByID != "" {
		ownedByIsInCache, ownedByNameCache := customerInCache(ownedByID)
		//-- Check if we have cached the customer already
		if ownedByIsInCache {
			ownedByName = ownedByNameCache
		} else {
			ownedByIsOnInstance, ownedByNameInstance := searchCustomer(ownedByID, espXmlmc)
			//-- If Returned set output
			if ownedByIsOnInstance {
				ownedByName = ownedByNameInstance
			}
		}
	}
	if ownedByName != "" {
		ownedByURN = "urn:sys:0:" + ownedByName + ":" + ownedByID
	}

	//Get Used By name
	usedByName := ""
	usedByURN := ""
	usedByMapping := fmt.Sprintf("%v", CSVImportConf.AssetGenericFieldMapping["h_used_by"])
	usedByID := getFieldValue("h_owned_by", usedByMapping, u)
	if usedByID != "" {
		usedByIsInCache, usedByNameCache := customerInCache(usedByID)
		//-- Check if we have cached the customer already
		if usedByIsInCache {
			usedByName = usedByNameCache
		} else {
			usedByIsOnInstance, usedByNameInstance := searchCustomer(usedByID, espXmlmc)
			//-- If Returned set output
			if usedByIsOnInstance {
				usedByName = usedByNameInstance
			}
		}
	}
	if usedByName != "" {
		usedByURN = "urn:sys:0:" + usedByName + ":" + usedByID
	}

	//Get/Set params from map stored against FieldMapping
	strAttribute := ""
	strMapping := ""
	espXmlmc.SetParam("application", appServiceManager)
	espXmlmc.SetParam("entity", "Asset")
	espXmlmc.OpenElement("primaryEntityData")
	espXmlmc.OpenElement("record")
	//Set Class & TypeID
	espXmlmc.SetParam("h_class", AssetClass)
	espXmlmc.SetParam("h_type", strconv.Itoa(AssetTypeID))

	espXmlmc.SetParam("h_last_updated", APITimeNow)
	espXmlmc.SetParam("h_last_updated_by", "Import - Add")

	//Get asset field mapping
	for k, v := range CSVImportConf.AssetGenericFieldMapping {
		strAttribute = fmt.Sprintf("%v", k)
		strMapping = fmt.Sprintf("%v", v)
		if strAttribute == "h_used_by" && usedByName != "" && usedByURN != "" {
			espXmlmc.SetParam("h_used_by", usedByURN)
			espXmlmc.SetParam("h_used_by_name", usedByName)
		}
		if strAttribute == "h_owned_by" && ownedByName != "" && ownedByURN != "" {
			espXmlmc.SetParam("h_owned_by", ownedByURN)
			espXmlmc.SetParam("h_owned_by_name", ownedByName)
		}
		if strAttribute == "h_site" && siteID != "" && siteName != "" {
			espXmlmc.SetParam("h_site", siteName)
			espXmlmc.SetParam("h_site_id", siteID)
		}
		if strAttribute != "h_site" &&
			strAttribute != "h_used_by" &&
			strAttribute != "h_owned_by" &&
			strMapping != "" && getFieldValue(strAttribute, strMapping, u) != "" {
			espXmlmc.SetParam(strAttribute, getFieldValue(strAttribute, strMapping, u))
		}
	}
	espXmlmc.CloseElement("record")
	espXmlmc.CloseElement("primaryEntityData")

	//Add extended asset type field mapping
	espXmlmc.OpenElement("relatedEntityData")
	//Set Class & TypeID
	espXmlmc.SetParam("relationshipName", "AssetClass")
	espXmlmc.SetParam("entityAction", "insert")
	espXmlmc.OpenElement("record")
	espXmlmc.SetParam("h_type", strconv.Itoa(AssetTypeID))
	//Get asset field mapping
	for k, v := range CSVImportConf.AssetTypeFieldMapping {
		strAttribute = fmt.Sprintf("%v", k)
		strMapping = fmt.Sprintf("%v", v)
		if strMapping != "" && getFieldValue(strAttribute, strMapping, u) != "" {
			espXmlmc.SetParam(strAttribute, getFieldValue(strAttribute, strMapping, u))
		}
	}
	espXmlmc.CloseElement("record")
	espXmlmc.CloseElement("relatedEntityData")

	//-- Check for Dry Run
	if !configDryRun {
		var XMLSTRING = espXmlmc.GetParam()
		XMLCreate, xmlmcErr := espXmlmc.Invoke("data", "entityAddRecord")
		if xmlmcErr != nil {
			logger(4, "Error running entityAddRecord API for createAsset:"+fmt.Sprintf("%v", xmlmcErr), false)
			logger(1, "API Call XML: "+XMLSTRING, false)
			return
		}
		var xmlRespon xmlmcUpdateResponse

		err := xml.Unmarshal([]byte(XMLCreate), &xmlRespon)
		if err != nil {
			mutexCounters.Lock()
			counters.createskipped++
			mutexCounters.Unlock()
			logger(4, "Unable to read response from Hornbill instance from entityAddRecord API for createAsset:"+fmt.Sprintf("%v", err), false)
			logger(1, "API Call XML: "+XMLSTRING, false)
			return
		}
		if xmlRespon.MethodResult != "ok" {
			logger(3, "Unable to add asset: "+xmlRespon.State.ErrorRet, false)
			logger(1, "API Call XML: "+XMLSTRING, false)
			mutexCounters.Lock()
			counters.createskipped++
			mutexCounters.Unlock()
		} else {
			mutexCounters.Lock()
			counters.created++
			mutexCounters.Unlock()
			assetID := xmlRespon.UpdatedColsPrimary.PrimaryKey
			//Now add asset URN
			espXmlmc.SetParam("application", "com.hornbill.servicemanager")
			espXmlmc.SetParam("entity", "Asset")
			espXmlmc.OpenElement("primaryEntityData")
			espXmlmc.OpenElement("record")
			espXmlmc.SetParam("h_pk_asset_id", assetID)
			espXmlmc.SetParam("h_asset_urn", "urn:sys:entity:com.hornbill.servicemanager:Asset:"+assetID)
			espXmlmc.CloseElement("record")
			espXmlmc.CloseElement("primaryEntityData")
			XMLUpdate, xmlmcErr := espXmlmc.Invoke("data", "entityUpdateRecord")
			if xmlmcErr != nil {
				logger(4, "API Call failed when Updating Asset URN:"+fmt.Sprintf("%v", xmlmcErr), false)
				return
			}
			var xmlRespon xmlmcUpdateResponse

			err := xml.Unmarshal([]byte(XMLUpdate), &xmlRespon)
			if err != nil {
				logger(4, "Unable to read response from Hornbill instance when Updating Asset URN:"+fmt.Sprintf("%v", err), false)
				return
			}
			if xmlRespon.MethodResult != "ok" && xmlRespon.State.ErrorRet != "There are no values to update" {
				logger(3, "Unable to update Asset URN: "+xmlRespon.State.ErrorRet, false)
				return
			}
		}
	} else {
		//-- DEBUG XML TO LOG FILE
		var XMLSTRING = espXmlmc.GetParam()
		logger(1, "Asset Create XML "+XMLSTRING, false)
		mutexCounters.Lock()
		counters.createskipped++
		mutexCounters.Unlock()
		espXmlmc.ClearParam()
	}
	return
}

// updateAsset -- Updates Asset record from the passed through map data and asset ID
func updateAsset(u map[string]string, strAssetID string, espXmlmc *apiLib.XmlmcInstStruct) bool {

	boolRecordUpdated := false
	//Get site ID
	siteID := ""
	siteNameMapping := fmt.Sprintf("%v", CSVImportConf.AssetGenericFieldMapping["h_site"])
	siteName := getFieldValue("h_site", siteNameMapping, u)
	if siteName != "" {
		siteIsInCache, SiteIDCache := siteInCache(siteName)
		//-- Check if we have cached the site already
		if siteIsInCache {
			siteID = strconv.Itoa(SiteIDCache)
		} else {
			siteIsOnInstance, SiteIDInstance := searchSite(siteName, espXmlmc)
			//-- If Returned set output
			if siteIsOnInstance {
				siteID = strconv.Itoa(SiteIDInstance)
			}
		}
	}

	//Get Owned By name
	ownedByName := ""
	ownedByURN := ""
	ownedByMapping := fmt.Sprintf("%v", CSVImportConf.AssetGenericFieldMapping["h_owned_by"])
	ownedByID := getFieldValue("h_owned_by", ownedByMapping, u)
	if ownedByID != "" {
		ownedByIsInCache, ownedByNameCache := customerInCache(ownedByID)
		//-- Check if we have cached the customer already
		if ownedByIsInCache {
			ownedByName = ownedByNameCache
		} else {
			ownedByIsOnInstance, ownedByNameInstance := searchCustomer(ownedByID, espXmlmc)
			//-- If Returned set output
			if ownedByIsOnInstance {
				ownedByName = ownedByNameInstance
			}
		}
	}
	if ownedByName != "" {
		ownedByURN = "urn:sys:0:" + ownedByName + ":" + ownedByID
	}

	//Get Used By name
	usedByName := ""
	usedByURN := ""
	usedByMapping := fmt.Sprintf("%v", CSVImportConf.AssetGenericFieldMapping["h_used_by"])
	usedByID := getFieldValue("h_owned_by", usedByMapping, u)
	if usedByID != "" {
		usedByIsInCache, usedByNameCache := customerInCache(usedByID)
		//-- Check if we have cached the customer already
		if usedByIsInCache {
			usedByName = usedByNameCache
		} else {
			usedByIsOnInstance, usedByNameInstance := searchCustomer(usedByID, espXmlmc)
			//-- If Returned set output
			if usedByIsOnInstance {
				usedByName = usedByNameInstance
			}
		}
	}
	if usedByName != "" {
		usedByURN = "urn:sys:0:" + usedByName + ":" + usedByID
	}

	//Get/Set params from map stored against FieldMapping
	strAttribute := ""
	strMapping := ""
	espXmlmc.SetParam("application", appServiceManager)
	espXmlmc.SetParam("entity", "Asset")
	espXmlmc.SetParam("returnModifiedData", "true")
	espXmlmc.OpenElement("primaryEntityData")
	espXmlmc.OpenElement("record")
	espXmlmc.SetParam("h_pk_asset_id", strAssetID)
	espXmlmc.SetParam("h_asset_urn", "urn:sys:entity:com.hornbill.servicemanager:Asset:"+strAssetID)

	//Get asset field mapping
	for k, v := range CSVImportConf.AssetGenericFieldMapping {
		strAttribute = fmt.Sprintf("%v", k)
		strMapping = fmt.Sprintf("%v", v)
		if strAttribute == "h_used_by" && usedByName != "" && usedByURN != "" {
			espXmlmc.SetParam("h_used_by", usedByURN)
			espXmlmc.SetParam("h_used_by_name", usedByName)
		}
		if strAttribute == "h_owned_by" && ownedByName != "" && ownedByURN != "" {
			espXmlmc.SetParam("h_owned_by", ownedByURN)
			espXmlmc.SetParam("h_owned_by_name", ownedByName)
		}
		if strAttribute == "h_site" && siteID != "" && siteName != "" {
			espXmlmc.SetParam("h_site", siteName)
			espXmlmc.SetParam("h_site_id", siteID)
		}
		if strAttribute != "h_site" &&
			strAttribute != "h_used_by" &&
			strAttribute != "h_owned_by" &&
			strMapping != "" && getFieldValue(strAttribute, strMapping, u) != "" {
			espXmlmc.SetParam(strAttribute, getFieldValue(strAttribute, strMapping, u))
		}
	}

	espXmlmc.CloseElement("record")
	espXmlmc.CloseElement("primaryEntityData")

	var XMLSTRING = espXmlmc.GetParam()
	//-- Check for Dry Run
	if !configDryRun {
		XMLUpdate, xmlmcErr := espXmlmc.Invoke("data", "entityUpdateRecord")
		if xmlmcErr != nil {
			logger(4, "API Call failed when Updating Asset:"+fmt.Sprintf("%v", xmlmcErr), false)
			logger(1, "API Call XML: "+XMLSTRING, false)
			mutexCounters.Lock()
			counters.updatedSkipped++
			mutexCounters.Unlock()
			return false
		}

		var xmlRespon xmlmcUpdateResponse

		err := xml.Unmarshal([]byte(XMLUpdate), &xmlRespon)
		if err != nil {
			logger(4, "Unable to read response from Hornbill instance when Updating Asset:"+fmt.Sprintf("%v", err), false)
			logger(1, "API Call XML: "+XMLSTRING, false)
			mutexCounters.Lock()
			counters.updatedSkipped++
			mutexCounters.Unlock()
			return false
		}
		if xmlRespon.MethodResult != "ok" && xmlRespon.State.ErrorRet != "There are no values to update" {
			logger(3, "Unable to Update Asset: "+xmlRespon.State.ErrorRet, false)
			logger(1, "API Call XML: "+XMLSTRING, false)
			mutexCounters.Lock()
			counters.updatedSkipped++
			mutexCounters.Unlock()
			return false
		}

		if len(xmlRespon.UpdatedColsPrimary.ColList) > 0 {
			boolRecordUpdated = true
		}

		//-- now process extended record data
		espXmlmc.SetParam("application", appServiceManager)
		espXmlmc.SetParam("entity", "Asset")
		espXmlmc.SetParam("returnModifiedData", "true")
		espXmlmc.OpenElement("primaryEntityData")
		espXmlmc.OpenElement("record")
		espXmlmc.SetParam("h_pk_asset_id", strAssetID)
		espXmlmc.CloseElement("record")
		espXmlmc.CloseElement("primaryEntityData")
		//Add extended asset type field mapping
		espXmlmc.OpenElement("relatedEntityData")
		//Set Class & TypeID
		espXmlmc.SetParam("relationshipName", "AssetClass")
		espXmlmc.SetParam("entityAction", "update")
		espXmlmc.OpenElement("record")
		espXmlmc.SetParam("h_pk_asset_id", strAssetID)
		//Get asset field mapping
		for k, v := range CSVImportConf.AssetTypeFieldMapping {
			strAttribute = fmt.Sprintf("%v", k)
			strMapping = fmt.Sprintf("%v", v)
			if strMapping != "" && getFieldValue(strAttribute, strMapping, u) != "" {
				espXmlmc.SetParam(strAttribute, getFieldValue(strAttribute, strMapping, u))
			}
		}
		espXmlmc.CloseElement("record")
		espXmlmc.CloseElement("relatedEntityData")
		var XMLSTRINGEXT = espXmlmc.GetParam()
		XMLUpdateExt, xmlmcErrExt := espXmlmc.Invoke("data", "entityUpdateRecord")
		if xmlmcErrExt != nil {
			logger(4, "API Call failed when Updating Asset Extended Details:"+fmt.Sprintf("%v", xmlmcErrExt), false)
			logger(1, "API Call XML: "+XMLSTRINGEXT, false)
			mutexCounters.Lock()
			counters.updatedSkipped++
			mutexCounters.Unlock()
			return false
		}
		var xmlResponExt xmlmcUpdateResponse

		err = xml.Unmarshal([]byte(XMLUpdateExt), &xmlResponExt)
		if err != nil {
			logger(4, "Unable to read response from Hornbill instance when Updating Asset Extended Details:"+fmt.Sprintf("%v", err), false)
			logger(1, "API Call XML: "+XMLSTRINGEXT, false)
			mutexCounters.Lock()
			counters.updatedSkipped++
			mutexCounters.Unlock()
			return false
		}
		if xmlResponExt.MethodResult != "ok" && xmlResponExt.State.ErrorRet != "There are no values to update" {
			logger(3, "Unable to Update Asset Extended Details: "+xmlResponExt.State.ErrorRet, false)
			logger(1, "API Call XML: "+XMLSTRINGEXT, false)
			mutexCounters.Lock()
			counters.updatedSkipped++
			mutexCounters.Unlock()
			return false
		}

		if len(xmlResponExt.UpdatedColsRelated.ColList) > 0 {
			boolRecordUpdated = true
		}

		if !boolRecordUpdated {
			mutexCounters.Lock()
			counters.updatedSkipped++
			mutexCounters.Unlock()
		} else {
			//-- Asset Updated!
			//-- Need to run another update against the Asset for LAST UPDATED and LAST UPDATE BY!
			espXmlmc.SetParam("application", appServiceManager)
			espXmlmc.SetParam("entity", "Asset")
			espXmlmc.OpenElement("primaryEntityData")
			espXmlmc.OpenElement("record")
			espXmlmc.SetParam("h_pk_asset_id", strAssetID)
			espXmlmc.SetParam("h_last_updated", APITimeNow)
			espXmlmc.SetParam("h_last_updated_by", "Import - Update")
			espXmlmc.CloseElement("record")
			espXmlmc.CloseElement("primaryEntityData")
			var XMLSTRING = espXmlmc.GetParam()
			XMLUpdate, xmlmcErr := espXmlmc.Invoke("data", "entityUpdateRecord")
			if xmlmcErr != nil {
				logger(4, "API Call failed when setting Last Updated values:"+fmt.Sprintf("%v", xmlmcErr), false)
				logger(1, "Asset Last Updated XML: "+XMLSTRING, false)
			}
			var xmlRespon xmlmcResponse
			err := xml.Unmarshal([]byte(XMLUpdate), &xmlRespon)
			if err != nil {
				logger(4, "Unable to read response from Hornbill instance when setting Last Updated values:"+fmt.Sprintf("%v", err), false)
				logger(1, "Asset Last Updated XML: "+XMLSTRING, false)
			}
			if xmlRespon.MethodResult != "ok" && xmlRespon.State.ErrorRet != "There are no values to update" {
				logger(3, "Unable to set Last Updated details for asset: "+xmlRespon.State.ErrorRet, false)
				logger(1, "Asset Last Updated XML: "+XMLSTRING, false)
			}
			mutexCounters.Lock()
			counters.updated++
			mutexCounters.Unlock()
		}

	} else {
		//-- Inc Counter
		mutexCounters.Lock()
		counters.updatedSkipped++
		mutexCounters.Unlock()
		logger(1, "Asset Update XML "+XMLSTRING, false)
		espXmlmc.ClearParam()
	}
	return true
}

// getFieldValue --Retrieve field value from mapping via CSV record map
func getFieldValue(k string, v string, u map[string]string) string {
	fieldMap := v
	//-- Match $variable from String
	re1, err := regexp.Compile(`\[(.*?)\]`)
	if err != nil {
		fmt.Printf("[ERROR] %v", err)
	}

	result := re1.FindAllString(fieldMap, 100)
	valFieldMap := ""
	//-- Loop Matches
	for _, val := range result {
		valFieldMap = ""
		valFieldMap = strings.Replace(val, "[", "", 1)
		valFieldMap = strings.Replace(valFieldMap, "]", "", 1)
		if valFieldMap == "HBAssetType" {
			valFieldMap = StrAssetType
		} else {
			valFieldMap = fmt.Sprintf("%v", u[valFieldMap])
		}
		if valFieldMap != "" {
			if strings.Contains(strings.ToLower(k), "date") {
				valFieldMap = checkDateString(valFieldMap)
			}
			if strings.Contains(valFieldMap, "[") {
				valFieldMap = ""
			} else {
				//20160215 Check for NULL (<nil>) field value
				//Cannot do this when Scanning CSV data, as we don't know the returned cols
				if valFieldMap == "<nil>" {
					valFieldMap = ""
				}
			}
		}
		fieldMap = strings.Replace(fieldMap, val, valFieldMap, 1)
	}
	return fieldMap
}

// logger -- function to append to the current log file
func logger(t int, s string, outputtoCLI bool) {
	//-- Current working dir
	cwd, _ := os.Getwd()

	//-- Log Folder
	logPath := cwd + "/log"

	//-- If Folder Does Not Exist then create it
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		err := os.Mkdir(logPath, 0777)
		if err != nil {
			fmt.Printf("Error Creating Log Folder %q: %s \r", logPath, err)
			os.Exit(101)
		}
	}

	//-- Log File
	logFileName := logPath + "/Asset_Import_" + TimeNow + "_" + strconv.Itoa(logFilePart) + ".log"
	if maxLogFileSize > 0 {
		//Check log file size
		fileLoad, e := os.Stat(logFileName)
		if e != nil {
			//File does not exist - do nothing!
		} else {
			fileSize := fileLoad.Size()
			if fileSize > maxLogFileSize {
				logFilePart++
				logFileName = logPath + "/Asset_Import_" + TimeNow + "_" + strconv.Itoa(logFilePart) + ".log"
			}
		}
	}

	//-- Open Log File
	f, err := os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0777)
	if err != nil {
		fmt.Printf("Error Creating Log File %q: %s \n", logFileName, err)
		os.Exit(100)
	}
	// don't forget to close it
	defer f.Close()
	var errorLogPrefix string
	//-- Create Log Entry
	switch t {
	case 1:
		errorLogPrefix = "[DEBUG] "
		if outputtoCLI {
			color.Set(color.FgGreen)
			defer color.Unset()
		}
	case 2:
		errorLogPrefix = "[MESSAGE] "
		if outputtoCLI {
			color.Set(color.FgGreen)
			defer color.Unset()
		}
	case 3:
		if outputtoCLI {
			color.Set(color.FgGreen)
			defer color.Unset()
		}
	case 4:
		errorLogPrefix = "[ERROR] "
		if outputtoCLI {
			color.Set(color.FgRed)
			defer color.Unset()
		}
	}
	if outputtoCLI {
		fmt.Printf("%v \n", errorLogPrefix+s)
	}
	mutex.Lock()
	// assign it to the standard logger
	log.SetOutput(f)
	log.Println(errorLogPrefix + s)
	mutex.Unlock()
}

// checkDateString - returns date from supplied string
func checkDateString(strDate string) string {
	re, _ := regexp.Compile("\\d{4}-\\d{2}-\\d{2} \\d{2}:\\d{2}:\\d{2}")
	strNewDate := re.FindString(strDate)
	return strNewDate
}
