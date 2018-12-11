# CSV Asset Import Go - [GO](https://golang.org/) Import Script to Hornbill

## Installation

### Windows

* Download the archive containing the import executable
* Extract zip into a folder you would like the application to run from e.g. `C:\asset_import\`
* Populate a CSV file fith asset data, and store in a folder or network location accessible to the user account running this tool
* Open the configuration file of your choice ('''conf_computerSystem.json''') and add in the necessary configration
* Open Command Line Prompt as Administrator
* Change Directory to the folder with goCSVAssetImport_*.exe `C:\asset_import\`
* Run the command :
* For Windows 32bit Systems: goCSVAssetImport_x86.exe -dryrun=true -file=conf_computerSystem.json
* For Windows 64bit Systems: goCSVAssetImport_x64.exe -dryrun=true -file=conf_computerSystem.json

### Configuration

Example JSON File:

```json
{
  "APIKey": "your_api_key_here",
  "InstanceId": "your_instance_name",
  "AssetIdentifier":"h_name",
  "CSVAssetIdentifier":"assetTag",
  "CSVCommaCharacter": ",",
  "CSVLazyQuotes": false,
  "CSVFieldsPerRecord": 0,
  "CSVCarriageReturnRemoval": false,
  "LogSizeBytes":1000000,
  "AssetTypes": {
      "Desktop": "\\\\path\\to\\csv\\assetcomputer_desktops.csv"
  },
  "AssetGenericFieldMapping":{
      "h_name":"[assetTag]",
      "h_site":"[site]",
      "h_asset_tag":"[assetTag]",
      "h_acq_method":"",
      "h_actual_retired_date":"",
      "h_beneficiary":"",
      "h_building":"[building]",
      "h_cost":"",
      "h_cost_center":"",
      "h_country":"",
      "h_created_date":"",
      "h_deprec_method":"",
      "h_deprec_start":"",
      "h_description":"[description]",
      "h_disposal_price":"",
      "h_disposal_reason":"",
      "h_floor":"[floor]",
      "h_geo_location":"",
      "h_invoice_number":"",
      "h_location":"",
      "h_location_type":"",
      "h_maintenance_cost":"",
      "h_maintenance_ref":"",
      "h_notes":"[notes]",
      "h_operational_state":"",
      "h_order_date":"",
      "h_order_number":"",
      "h_owned_by":"[ownedById]",
      "h_product_id":"",
      "h_received_date":"",
      "h_residual_value":"",
      "h_room":"",
      "h_scheduled_retire_date":"",
      "h_supplier_id":"",
      "h_supported_by":"",
      "h_used_by":"[usedById]",
      "h_version":"",
      "h_warranty_expires":"",
      "h_warranty_start":""
  },
  "AssetTypeFieldMapping":{
      "h_name":"[assetTag]",
      "h_mac_address":"[macAddress]",
      "h_net_ip_address":"[netIpAddress]",
      "h_net_computer_name":"[netComputerName]",
      "h_net_win_domain":"[netWinDomRole]",
      "h_model":"[model]",
      "h_manufacturer":"[manufacturer]",
      "h_cpu_info":"[ProcessorName]",
      "h_description":"[assetTag] ([model])",
      "h_last_logged_on":"",
      "h_last_logged_on_user":"[lastLoggedOnUser]",
      "h_memory_info":"[memoryInfo]",
      "h_net_win_dom_role":"",
      "h_optical_drive":"",
      "h_os_description":"[osDescription]",
      "h_os_registered_to":"",
      "h_os_serial_number":"",
      "h_os_service_pack":"[osServicePack]",
      "h_os_type":"[osType]",
      "h_os_version":"[osVersion]",
      "h_physical_disk_size":"[physicalDiskSize]",
      "h_serial_number":"[serialNumber]",
      "h_cpu_clock_speed":"[cpuClockSpeed]",
      "h_physical_cpus":"[physicalCpus]",
      "h_logical_cpus":"[logicalCpus]",
      "h_bios_name":"[biosName]",
      "h_bios_manufacturer":"[biosManufacturer]",
      "h_bios_serial_number":"",
      "h_bios_release_date":"",
      "h_bios_version":"",
      "h_max_memory_capacity":"",
      "h_number_memory_slots":"",
      "h_net_name":"",
      "h_subnet_mask":""
  }
}
```

#### InstanceConfig

* "APIKey" - a Hornbill API key for a user account with the correct permissions to carry out all of the required API calls
* "InstanceId" - Instance Id
* "AssetIdentifier" - The Hornbill asset attribute that holds the unique asset identifier (so that the code can work out which asset records are to be inserted or updated)
* "CSVAssetIdentifier" - The CSV column name that holds the unique asset identifier (so that the code can work out which asset records are to be inserted or updated)
* "CSVCommaCharacter" - The field separator (single) character - if left out, the default character will be a comma.
* "CSVLazyQuotes" - The ability to give the CSV reader a hint that the csv file might be using lazy quotes. Defaults to false.
* "CSVFieldsPerRecord" - The ability to give the CSV reader a hint about the amount of fields in each record. Defaults to 0, leaving the CSV reader to do the heavy lifting.
* "CSVCarriageReturnRemoval" - Certain CSV exporting systems will add extra carriage returns as a record delimiter. This is expected not to be common, hence the setting is left out of the configuration files (it is added to conf_computerSystem.json only for completeness sake). IF not set, then the default value is false and no carriages returns will be stripped from the data. IF set to true, then all carriage returns (possibly even intended ones) will be stripped.
* "LogSizeBytes" - The maximum size that the generated Log Files should be, in bytes. Setting this value to 0 will cause the tool to create one log file only and not split the results between multiple logs.

#### AssetTypes

* The left element contains the Asset Type Name, and the right contains the path and filename of the CSV file that holds the asset row data. Note: the Asset Type Name needs to match a correct Asset Type Name in your Hornbill Instance.

#### AssetGenericFieldMapping

* Maps data in to the generic Asset record
* Any value wrapped with [] will be populated with the corresponding column value from the CSV row
* Any Other Value is treated literally as written example:
  * "h_name":"[assetTag]", - the value of the assetTag column is taken from the CSV output and populated within this field
  * "h_description":"This is a description", - the value of "h_description" would be populated with "This is a description" for ALL imported assets
  * "h_site":"[site]", - When a string is passed to the h_site field, the script attempts to resolve the given site name against the Site entity, and populates this (and h_site_id) with the correct site information. If the site cannot be resolved, the site details are not populated for the Asset record being imported.
  * "h_owned_by":"[ownedById]" - when a valid Hornbill User ID (for a Full or Basic User) is passed to this field, the user is verified on your Hornbill instance, and the tool will complete the h_owned_by and h_owned_by_name columns appropriately.
  * "h_used_by":"[usedById]" - when a valid Hornbill User ID (for a Full or Basic User) is passed to this field, the user is verified on your Hornbill instance, and the tool will complete the h_used_by and h_used_by_name columns appropriately.
  * "h_company_name":"[CompanyName]" - when a valid Hornbill Company group name is passed to this field, the company is verified on your Hornbill instance, and the tool will complete the h_company_id and h_company_name columns appropriately.

#### AssetTypeFieldMapping

* Maps data in to the type-specific Asset record, so the same rules as AssetGenericFieldMapping

## Execute

### Command Line Parameters

* file - Defaults to `conf.json` - Name of the Configuration file to load
* dryrun - Defaults to `false` - Set to True and the XMLMC for Create and Update assets will not be called and instead the XML will be dumped to the log file, this is to aid in debugging the initial connection information.
* concurrent - defaults to `1`. This is to specify the number of assets that should be imported concurrently, and can be an integer between 1 and 10 (inclusive). 1 is the slowest level of import, but does not affect performance of your Hornbill instance, and 10 will process the import much more quickly but could affect instance performance while the import is running.

## Testing

If you run the application with the argument dryrun=true then no assets will be created or updated, the XML used to create or update will be saved in the log file so you can ensure the data mappings are correct before running the import.

'goCSVAssetImport_x64.exe -dryrun=true'

## Scheduling

### Windows Scheduler

You can schedule the executable to run with any optional command line argument from Windows Task Scheduler.

* Ensure the user account running the task has rights to goCSVAssetImport_x64.exe and the containing folder.
* Make sure the Start In parameter contains the folder where goCSVAssetImport_x64.exe resides in otherwise it will not be able to pick up the correct path.

## Logging

All Logging output is saved in the log directory in the same directory as the executable the file name contains the date and time the import was run 'Asset_Import_2015-11-06T14-26-13Z.log'

## Error Codes

* `100` - Unable to create log File
* `101` - Unable to create log folder
* `102` - Unable to Load Configuration File