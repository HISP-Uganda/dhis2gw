package main

type Project struct {
	ID                   int                 `json:"id"`
	Program              Program             `json:"program"`
	Classification       *string             `json:"classification"`
	BudgetCode           string              `json:"budget_code"`
	Name                 string              `json:"name"`
	StartDate            string              `json:"start_date"`
	EndDate              string              `json:"end_date"`
	ProjectStatus        string              `json:"project_status"`
	CurrentProjectDetail ProjectDetail       `json:"current_project_detail"`
	ProjectOrganization  ProjectOrganization `json:"project_organization"`
}

type Program struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type ProjectOrganization struct {
	Code   string     `json:"code"`
	Name   string     `json:"name"`
	Parent *ParentOrg `json:"parent"`
}

type ParentOrg struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type ProjectDetail struct {
	Outputs         []Output          `json:"outputs"`
	InvestmentStats InvestmentStats   `json:"investment_stats"`
	Activities      []Activity        `json:"activities"`
	Locations       []ProjectLocation `json:"locations"`
	Outcomes        []Outcome         `json:"outcomes"`
}

type Output struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type InvestmentStats map[string]interface{}

type Activity struct {
	Name      string `json:"name"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

type ProjectLocation struct {
	Location struct {
		Name string `json:"name"`
	} `json:"location"`
}

type Outcome struct {
	Name       string      `json:"name"`
	Indicators []Indicator `json:"indicators"`
}

type Indicator struct {
	Name              string            `json:"name"`
	Baseline          string            `json:"baseline"`
	Assumptions       string            `json:"assumptions"`
	RiskFactors       string            `json:"risk_factors"`
	VerificationMeans string            `json:"verification_means"`
	Targets           map[string]string `json:"targets"`
}
