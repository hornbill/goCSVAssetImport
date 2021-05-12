# CHANGELOG

## 1.2.5 (May 12th, 2021)

Features:
- Departments are now mapped (if h_department_name is set - h_department_id will be set behind the scenes) - this now matches the behaviour of organisations (the h_company_name field)
- Ability to specific the Hornbill User ID column for matching users (HornbillUserIDColumn; options: h_user_id (default), h_employee_id, h_email, h_name, h_attrib_1 & h_login_id) - please note that last logged on, owned by and used by will use the same field - i.e. one can NOT specify which column to match to individually.

Changes:
- Front-loading of groups (organisations & departments) - to prevent search for each asset
- If no Asset ID is specified the record will be skipped - instead of activating a search for the asset (to check whether the asset has already been created)

Fixes:
- Possible fix whereby h_asset_urn was not populated correctly

## 1.2.4 (March 10th, 2021)

Fixes:
- Exact Matching of asset name/identifier and type

## 1.2.3 (January 18th, 2021)

Fixes:
- fixed usage of AssetIdentifier

## 1.2.2 (April 16th, 2020)

Changes:

- Updated code to support Core application and platform changes
- Added version flag to enable auto-build

## 1.2.1 (December 28th, 2018)

Fixes:

- last_logged_on_user to URN conversion

## 1.2.0 (December 11th, 2018)

Features:

- Added support for populating the company fields against an asset. The tool will perform a Company look-up if a company name (in the h_company_name mapping) has been provided, before populating the company name and ID fields against the new or updated asset
- Removed need to provide zone CLI parameter

## 1.1.2 (September 26th, 2018)

Fixes:

- Recoding to use entityBrowseRecords2 instead of entityBrowseRecords.

## 1.1.1 (January 26th, 2018)

Defect fix:

- Fixed issue with Used By not being populated with a valid URN

## 1.1.0 (December 4th, 2017)

Features:

- Adds Asset URN to record once asset record has been created
- Updates Asset URN during asset update

## 1.0.3 (November 1st, 2017)

Features:

- In certain circumstances (csv exporters) extra carriage returns are added as a record delimiter. An additional optional boolean CSVCarriageReturnRemoval will strip out all non-record delimiting carriage returns when set to true (default is false).

## 1.0.2 (September 20th, 2017)

Features:

- Added code to deal with UTF-8 BOM at the beginning of some CSV files
- Allow for setting a different field separator (Comma), LazyQuotes and FieldsPerRecord within the import configuration

## 1.0.1 (August 3rd, 2017)

Features:

- Removed unnecessary type casting;
- Only one HTTP connection per worker rather than per asset being imported.

## 1.0.0 (August 3rd, 2017)

Features:

- Initial Release