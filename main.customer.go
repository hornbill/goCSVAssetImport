package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"

	apiLib "github.com/hornbill/goApiLib"
	"github.com/hornbill/pb"
)

var (
	hornbillImport *apiLib.XmlmcInstStruct
	pageSize       int
)

type xmlmcCountResponse struct {
	Params struct {
		RowData struct {
			Row []struct {
				Count string `json:"count"`
			} `json:"row"`
		} `json:"rowData"`
	} `json:"params"`
	State stateJSONStruct `json:"state"`
}
type xmlmcUserListResponse struct {
	Params struct {
		RowData struct {
			Row []userAccountStruct `json:"row"`
		} `json:"rowData"`
	} `json:"params"`
	State stateJSONStruct `json:"state"`
}
type userAccountStruct struct {
	HUserID     string `json:"h_user_id"`
	HLoginID    string `json:"h_login_id"`
	HEmployeeID string `json:"h_employee_id"`
	HName       string `json:"h_name"`
	HFirstName  string `json:"h_first_name"`
	HMiddleName string `json:"h_middle_name"`
	HLastName   string `json:"h_last_name"`
	HPhone      string `json:"h_phone"`
	HEmail      string `json:"h_email"`
	HMobile     string `json:"h_mobile"`
	HSnA        string `json:"h_sn_a"`
	HSnB        string `json:"h_sn_b"`
	HAttrib1    string `json:"h_attrib_1"`
	HAttrib2    string `json:"h_attrib_2"`
	HAttrib3    string `json:"h_attrib_3"`
	/*	HJobTitle            string `json:"h_job_title"`
		HLoginCreds          string `json:"h_login_creds"`
		HClass               string `json:"h_class"`
		HAvailStatus         string `json:"h_avail_status"`
		HAvailStatusMsg      string `json:"h_avail_status_msg"`
		HTimezone            string `json:"h_timezone"`
		HCountry             string `json:"h_country"`
		HLanguage            string `json:"h_language"`
		HDateTimeFormat      string `json:"h_date_time_format"`
		HDateFormat          string `json:"h_date_format"`
		HTimeFormat          string `json:"h_time_format"`
		HCurrencySymbol      string `json:"h_currency_symbol"`
		HLastLogon           string `json:"h_last_logon"`
		HSnC                 string `json:"h_sn_c"`
		HSnD                 string `json:"h_sn_d"`
		HSnE                 string `json:"h_sn_e"`
		HSnF                 string `json:"h_sn_f"`
		HSnG                 string `json:"h_sn_g"`
		HSnH                 string `json:"h_sn_h"`
		HIconRef             string `json:"h_icon_ref"`
		HIconChecksum        string `json:"h_icon_checksum"`
		HDob                 string `json:"h_dob"`
		HAccountStatus       string `json:"h_account_status"`
		HFailedAttempts      string `json:"h_failed_attempts"`
		HIdxRef              string `json:"h_idx_ref"`
		HSite                string `json:"h_site"`
		HManager             string `json:"h_manager"`
		HSummary             string `json:"h_summary"`
		HInterests           string `json:"h_interests"`
		HQualifications      string `json:"h_qualifications"`
		HPersonalInterests   string `json:"h_personal_interests"`
		HSkills              string `json:"h_skills"`
		HGender              string `json:"h_gender"`
		HNationality         string `json:"h_nationality"`
		HReligion            string `json:"h_religion"`
		HHomeTelephoneNumber string `json:"h_home_telephone_number"`
		HHomeAddress         string `json:"h_home_address"`
		HBlog                string `json:"h_blog"`
		HAttrib4             string `json:"h_attrib_4"`
		HAttrib5             string `json:"h_attrib_5"`
		HAttrib6             string `json:"h_attrib_6"`
		HAttrib7             string `json:"h_attrib_7"`
		HAttrib8             string `json:"h_attrib_8"`
		HHomeOrg             string `json:"h_home_organization"`
	*/
}
type stateJSONStruct struct {
	Code      string `json:"code"`
	Service   string `json:"service"`
	Operation string `json:"operation"`
	Error     string `json:"error"`
}

func initXMLMC() {

	hornbillImport = apiLib.NewXmlmcInstance(CSVImportConf.InstanceID)
	hornbillImport.SetAPIKey(CSVImportConf.APIKey)
	hornbillImport.SetTimeout(60)
	hornbillImport.SetJSONResponse(true)

	pageSize = 0

	if pageSize == 0 {
		pageSize = 100
	}
}

// customerInCache -- Function to check if passed-thorugh Customer ID has been cached
// if so, pass back the Customer Name
func customerInCache(customerID string) (bool, string, string) {
	boolReturn := false
	customerName := ""
	customerIDReturn := ""
	mutexCustomers.Lock()
	//-- Check if in Cache
	for _, customer := range Customers {
		if strings.EqualFold(customer.CustomerID, customerID) {
			boolReturn = true
			customerName = customer.CustomerName
			//customerIDReturn = customer.CustomerID
			customerIDReturn = customer.CustomerHandle
			break
		}
	}
	mutexCustomers.Unlock()
	return boolReturn, customerName, customerIDReturn
}

// seachCustomer -- Function to check if passed-through customer name is on the instance
func searchCustomer(custID string, espXmlmc *apiLib.XmlmcInstStruct) (bool, string, string) {
	boolReturn := false
	custNameReturn := ""
	custIDReturn := ""
	//Get Analyst Info
	espXmlmc.SetParam("customerId", custID)
	espXmlmc.SetParam("customerType", "0")
	var XMLSTRING = espXmlmc.GetParam()
	XMLCustomerSearch, xmlmcErr := espXmlmc.Invoke("apps/"+appServiceManager, "shrGetCustomerDetails")
	if xmlmcErr != nil {
		logger(4, "Unable to Search for Customer ["+custID+"]: "+fmt.Sprintf("%v", xmlmcErr), true)
		logger(1, "API XML: "+XMLSTRING, false)
	}

	var xmlRespon xmlmcCustomerListResponse
	err := xml.Unmarshal([]byte(XMLCustomerSearch), &xmlRespon)
	if err != nil {
		logger(4, "Unable to Search for Customer ["+custID+"]: "+fmt.Sprintf("%v", err), false)
		logger(1, "API XML: "+XMLSTRING, false)
	} else {
		if xmlRespon.MethodResult != "ok" {
			//Customer most likely does not exist
			logger(4, "Unable to Search for Customer ["+custID+"]: "+xmlRespon.State.ErrorRet, false)
			logger(1, "API XML: "+XMLSTRING, false)
		} else {
			//-- Check Response
			if xmlRespon.CustomerFirstName != "" {
				boolReturn = true
				//-- Add Customer to Cache
				var newCustomerForCache customerListStruct
				newCustomerForCache.CustomerID = xmlRespon.CustomerID
				//				newCustomerForCache.Handle = xmlRespon.CustomerID
				newCustomerForCache.CustomerName = xmlRespon.CustomerFirstName + " " + xmlRespon.CustomerLastName
				custNameReturn = newCustomerForCache.CustomerName
				custIDReturn = newCustomerForCache.CustomerID
				customerNamedMap := []customerListStruct{newCustomerForCache}
				mutexCustomers.Lock()
				Customers = append(Customers, customerNamedMap...)
				mutexCustomers.Unlock()
			}
		}
	}
	return boolReturn, custNameReturn, custIDReturn
}

func loadUsers() {
	//-- Init One connection to Hornbill to load all data
	initXMLMC()
	logger(1, "Loading Users from Hornbill", false)

	count := getCount("getUserAccountsList")
	logger(1, "getUserAccountsList Count: "+fmt.Sprintf("%d", count), false)
	getUserAccountList(count)

	logger(1, "Users Loaded: "+fmt.Sprintf("%d", len(Customers)), false)
}

func getUserAccountList(count uint64) {
	var loopCount uint64
	//-- Init Map
	//-- Load Results in pages of pageSize
	bar := pb.StartNew(int(count))
	for loopCount < count {
		logger(1, "Loading User Accounts List Offset: "+fmt.Sprintf("%d", loopCount)+"\n", false)

		hornbillImport.SetParam("application", "com.hornbill.core")
		hornbillImport.SetParam("queryName", "getUserAccountsList")
		hornbillImport.OpenElement("queryParams")
		hornbillImport.SetParam("rowstart", strconv.FormatUint(loopCount, 10))
		hornbillImport.SetParam("limit", strconv.Itoa(pageSize))
		hornbillImport.CloseElement("queryParams")
		RespBody, xmlmcErr := hornbillImport.Invoke("data", "queryExec")

		var JSONResp xmlmcUserListResponse
		if xmlmcErr != nil {
			logger(4, "Unable to Query Accounts List "+fmt.Sprintf("%s", xmlmcErr), false)
			break
		}
		err := json.Unmarshal([]byte(RespBody), &JSONResp)
		if err != nil {
			logger(4, "Unable to Query Accounts List "+fmt.Sprintf("%s", err), false)
			break
		}
		if JSONResp.State.Error != "" {
			logger(4, "Unable to Query Accounts List "+JSONResp.State.Error, false)
			break
		}
		//-- Push into Map

		switch CSVImportConf.HornbillUserIDColumn {
		case "h_employee_id":
			{
				for index := range JSONResp.Params.RowData.Row {
					var newCustomerForCache customerListStruct
					newCustomerForCache.CustomerID = JSONResp.Params.RowData.Row[index].HEmployeeID
					newCustomerForCache.CustomerHandle = JSONResp.Params.RowData.Row[index].HUserID
					newCustomerForCache.CustomerName = JSONResp.Params.RowData.Row[index].HFirstName + " " + JSONResp.Params.RowData.Row[index].HLastName
					customerNamedMap := []customerListStruct{newCustomerForCache}
					mutexCustomers.Lock()
					Customers = append(Customers, customerNamedMap...)
					mutexCustomers.Unlock()
				}
			}
		case "h_login_id":
			{
				for index := range JSONResp.Params.RowData.Row {
					var newCustomerForCache customerListStruct
					newCustomerForCache.CustomerID = JSONResp.Params.RowData.Row[index].HLoginID
					newCustomerForCache.CustomerHandle = JSONResp.Params.RowData.Row[index].HUserID
					newCustomerForCache.CustomerName = JSONResp.Params.RowData.Row[index].HFirstName + " " + JSONResp.Params.RowData.Row[index].HLastName
					customerNamedMap := []customerListStruct{newCustomerForCache}
					mutexCustomers.Lock()
					Customers = append(Customers, customerNamedMap...)
					mutexCustomers.Unlock()
				}
			}
		case "h_email":
			{
				for index := range JSONResp.Params.RowData.Row {
					var newCustomerForCache customerListStruct
					newCustomerForCache.CustomerID = JSONResp.Params.RowData.Row[index].HEmail
					newCustomerForCache.CustomerHandle = JSONResp.Params.RowData.Row[index].HUserID
					newCustomerForCache.CustomerName = JSONResp.Params.RowData.Row[index].HFirstName + " " + JSONResp.Params.RowData.Row[index].HLastName
					customerNamedMap := []customerListStruct{newCustomerForCache}
					mutexCustomers.Lock()
					Customers = append(Customers, customerNamedMap...)
					mutexCustomers.Unlock()
				}
			}
		case "h_name":
			{
				for index := range JSONResp.Params.RowData.Row {
					var newCustomerForCache customerListStruct
					newCustomerForCache.CustomerID = JSONResp.Params.RowData.Row[index].HName
					newCustomerForCache.CustomerHandle = JSONResp.Params.RowData.Row[index].HUserID
					newCustomerForCache.CustomerName = JSONResp.Params.RowData.Row[index].HFirstName + " " + JSONResp.Params.RowData.Row[index].HLastName
					customerNamedMap := []customerListStruct{newCustomerForCache}
					mutexCustomers.Lock()
					Customers = append(Customers, customerNamedMap...)
					mutexCustomers.Unlock()
				}
			}
		case "h_attrib_1":
			{
				for index := range JSONResp.Params.RowData.Row {
					var newCustomerForCache customerListStruct
					newCustomerForCache.CustomerID = JSONResp.Params.RowData.Row[index].HAttrib1
					newCustomerForCache.CustomerHandle = JSONResp.Params.RowData.Row[index].HUserID
					newCustomerForCache.CustomerName = JSONResp.Params.RowData.Row[index].HFirstName + " " + JSONResp.Params.RowData.Row[index].HLastName
					customerNamedMap := []customerListStruct{newCustomerForCache}
					mutexCustomers.Lock()
					Customers = append(Customers, customerNamedMap...)
					mutexCustomers.Unlock()
				}
			}
		case "h_user_id":
			{ // as Go Switch doesn't fall through
				for index := range JSONResp.Params.RowData.Row {
					var newCustomerForCache customerListStruct
					newCustomerForCache.CustomerID = JSONResp.Params.RowData.Row[index].HUserID
					newCustomerForCache.CustomerHandle = JSONResp.Params.RowData.Row[index].HUserID
					newCustomerForCache.CustomerName = JSONResp.Params.RowData.Row[index].HFirstName + " " + JSONResp.Params.RowData.Row[index].HLastName
					customerNamedMap := []customerListStruct{newCustomerForCache}
					mutexCustomers.Lock()
					Customers = append(Customers, customerNamedMap...)
					mutexCustomers.Unlock()
				}
			}
		default:
			{
				for index := range JSONResp.Params.RowData.Row {
					var newCustomerForCache customerListStruct
					newCustomerForCache.CustomerID = JSONResp.Params.RowData.Row[index].HUserID
					newCustomerForCache.CustomerHandle = JSONResp.Params.RowData.Row[index].HUserID
					newCustomerForCache.CustomerName = JSONResp.Params.RowData.Row[index].HFirstName + " " + JSONResp.Params.RowData.Row[index].HLastName
					customerNamedMap := []customerListStruct{newCustomerForCache}
					mutexCustomers.Lock()
					Customers = append(Customers, customerNamedMap...)
					mutexCustomers.Unlock()
				}
			}
		}

		// Add 100
		loopCount += uint64(pageSize)
		bar.Add(len(JSONResp.Params.RowData.Row))
		//-- Check for empty result set
		if len(JSONResp.Params.RowData.Row) == 0 {
			break
		}
	}
	bar.FinishPrint("Accounts Loaded  \n")
}

func getCount(query string) uint64 {

	hornbillImport.SetParam("application", "com.hornbill.core")
	hornbillImport.SetParam("queryName", query)
	hornbillImport.OpenElement("queryParams")
	hornbillImport.SetParam("getCount", "true")
	hornbillImport.CloseElement("queryParams")

	RespBody, xmlmcErr := hornbillImport.Invoke("data", "queryExec")

	var JSONResp xmlmcCountResponse
	if xmlmcErr != nil {
		logger(4, "Unable to run Query ["+query+"] "+fmt.Sprintf("%s", xmlmcErr), false)
		return 0
	}
	err := json.Unmarshal([]byte(RespBody), &JSONResp)
	if err != nil {
		logger(4, "Unable to run Query ["+query+"] "+fmt.Sprintf("%s", err), false)
		return 0
	}
	if JSONResp.State.Error != "" {
		logger(4, "Unable to run Query ["+query+"] "+JSONResp.State.Error, false)
		return 0
	}

	//-- return Count
	count, errC := strconv.ParseUint(JSONResp.Params.RowData.Row[0].Count, 10, 16)
	//-- Check for Error
	if errC != nil {
		logger(4, "Unable to get Count for Query ["+query+"] "+fmt.Sprintf("%s", err), false)
		return 0
	}
	return count
}
