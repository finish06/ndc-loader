package api

// OpenFDAResponse is the top-level response matching the openFDA /drug/ndc.json format.
type OpenFDAResponse struct {
	Meta    OpenFDAMeta      `json:"meta"`
	Results []OpenFDAProduct `json:"results"`
}

// OpenFDAMeta matches the openFDA meta object.
type OpenFDAMeta struct {
	Disclaimer  string            `json:"disclaimer"`
	Terms       string            `json:"terms"`
	License     string            `json:"license"`
	LastUpdated string            `json:"last_updated"`
	Results     OpenFDAPagination `json:"results"`
}

// OpenFDAPagination matches the openFDA meta.results pagination object.
type OpenFDAPagination struct {
	Skip  int `json:"skip"`
	Limit int `json:"limit"`
	Total int `json:"total"`
}

// OpenFDAProduct matches a single result in the openFDA /drug/ndc.json response.
type OpenFDAProduct struct {
	ProductNDC         string                    `json:"product_ndc"`
	GenericName        string                    `json:"generic_name"`
	LabelerName        string                    `json:"labeler_name"`
	BrandName          string                    `json:"brand_name"`
	ActiveIngredients  []OpenFDAActiveIngredient `json:"active_ingredients"`
	Finished           bool                      `json:"finished"`
	Packaging          []OpenFDAPackaging        `json:"packaging"`
	OpenFDA            OpenFDANested             `json:"openfda"`
	MarketingCategory  string                    `json:"marketing_category"`
	DosageForm         string                    `json:"dosage_form"`
	SPLID              string                    `json:"spl_id"`
	ProductType        string                    `json:"product_type"`
	Route              []string                  `json:"route"`
	MarketingStartDate string                    `json:"marketing_start_date"`
	ProductID          string                    `json:"product_id"`
	ApplicationNumber  string                    `json:"application_number"`
	BrandNameBase      string                    `json:"brand_name_base"`
	PharmClass         []string                  `json:"pharm_class"`
}

// OpenFDAActiveIngredient matches the active_ingredients array element.
type OpenFDAActiveIngredient struct {
	Name     string `json:"name"`
	Strength string `json:"strength"`
}

// OpenFDAPackaging matches the packaging array element.
type OpenFDAPackaging struct {
	PackageNDC         string `json:"package_ndc"`
	Description        string `json:"description"`
	MarketingStartDate string `json:"marketing_start_date"`
	Sample             bool   `json:"sample"`
}

// OpenFDANested matches the openfda nested object.
type OpenFDANested struct {
	ManufacturerName   []string `json:"manufacturer_name"`
	RXCUI              []string `json:"rxcui"`
	SPLSetID           []string `json:"spl_set_id"`
	IsOriginalPackager []bool   `json:"is_original_packager"`
	UPC                []string `json:"upc"`
	UNII               []string `json:"unii"`
}

// OpenFDAError matches the openFDA error response format.
type OpenFDAError struct {
	Error OpenFDAErrorDetail `json:"error"`
}

// OpenFDAErrorDetail is the error detail object.
type OpenFDAErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

const (
	openFDADisclaimer = "This data is sourced from the FDA NDC Directory bulk download, not the openFDA API."
	openFDATerms      = "https://open.fda.gov/terms/"
	openFDALicense    = "https://open.fda.gov/license/"
)
