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
