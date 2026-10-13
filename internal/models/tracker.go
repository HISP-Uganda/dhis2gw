package models

import (
	"fmt"
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
)

// "programStages[id,name,programStageDataElements[compulsary,dataElement[id,name]]],programTrackedEntityAttributes[mandatory,valueType,trackedEntityAttribute[id,name]]"

type Identifiable struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	Code string `json:"code,omitempty"`
}
type OptionSet struct {
	ID      string         `json:"id,omitempty"`
	Options []Identifiable `json:"options,omitempty"`
}
type DataElement struct {
	ID             string    `json:"id"`
	Name           string    `json:"name,omitempty"`
	ValueType      string    `json:"valueType,omitempty"`
	OptionSetValue bool      `json:"optionSetValue,omitempty"`
	OptionSet      OptionSet `json:"optionSet,omitempty"`
}
type TrackedEntityAttribute struct {
	ID             string    `json:"id"`
	Name           string    `json:"name,omitempty"`
	OptionSetValue bool      `json:"optionSetValue,omitempty"`
	OptionSet      OptionSet `json:"optionSet,omitempty"`
}

type TrackedEntityTypeAttribute struct {
	ID                     string                 `json:"id"`
	Name                   string                 `json:"name,omitempty"`
	Mandatory              bool                   `json:"mandatory,omitempty"`
	TrackedEntityAttribute TrackedEntityAttribute `json:"trackedEntityAttribute,omitempty"`
}

type TrackedEntityType struct {
	ID                          string                       `json:"id"`
	Name                        string                       `json:"name,omitempty"`
	TrackedEntityTypeAttributes []TrackedEntityTypeAttribute `json:"trackedEntityTypeAttributes,omitempty"`
}

type ProgramTrackedEntityAttribute struct {
	Mandatory              bool                   `json:"mandatory"`
	ValueType              string                 `json:"valueType,omitempty"`
	TrackedEntityAttribute TrackedEntityAttribute `json:"trackedEntityAttribute,omitempty"`
}
type ProgramStageDataElement struct {
	Compulsory  bool        `json:"compulsory,omitempty"`
	DataElement DataElement `json:"dataElement"`
}

type ProgramStage struct {
	ID                       string                    `json:"id"`
	Name                     string                    `json:"name,omitempty"`
	ProgramStageDataElements []ProgramStageDataElement `json:"programStageDataElements,omitempty"`
}

type Program struct {
	ID                             string                          `json:"id"`
	Name                           string                          `json:"name,omitempty"`
	ProgramStages                  []ProgramStage                  `json:"programStages,omitempty"`
	ProgramTrackedEntityAttributes []ProgramTrackedEntityAttribute `json:"programTrackedEntityAttributes,omitempty"`
}

// GetMandatoryTrackedEntityAttributes returns all mandatory tracked entity attributes in a Program.
func (p Program) GetMandatoryTrackedEntityAttributes() []TrackedEntityAttribute {
	var attrs []TrackedEntityAttribute
	for _, ptea := range p.ProgramTrackedEntityAttributes {
		if ptea.Mandatory {
			attrs = append(attrs, ptea.TrackedEntityAttribute)
		}
	}
	return attrs
}

// GetMandatoryDataElements returns all compulsory data elements in a given ProgramStage.
func (ps ProgramStage) GetMandatoryDataElements() []DataElement {
	var elements []DataElement
	for _, pde := range ps.ProgramStageDataElements {
		if pde.Compulsory {
			elements = append(elements, pde.DataElement)
		}
	}
	return elements
}

func (t TrackedEntityType) GetMandatoryTrackedEntityAttributes() []TrackedEntityAttribute {
	var attrs []TrackedEntityAttribute
	for _, teta := range t.TrackedEntityTypeAttributes {
		if teta.Mandatory {
			attrs = append(attrs, teta.TrackedEntityAttribute)
		}
	}
	return attrs
}

// PrintMandatoryDetails prints tracked entity attributes and data elements.
// - includeAll: if true, includes all items; if false, only mandatory ones.
// - grouped: if true, prints each ProgramStage as its own subtable.
func (p Program) PrintMandatoryDetails(includeAll, grouped bool) {
	fmt.Printf("PROGRAM: %s (%s)\n\n", p.Name, p.ID)

	// --- Section 1: Tracked Entity Attributes ---
	attrTable := table.NewWriter()
	attrTable.SetOutputMirror(os.Stdout)
	attrTable.SetStyle(table.StyleRounded)

	attrTable.AppendHeader(table.Row{"Type", "ID", "Name", "Mandatory", "Has OptionSet", "OptionSet ID"})

	for _, ptea := range p.ProgramTrackedEntityAttributes {
		if includeAll || ptea.Mandatory {
			attr := ptea.TrackedEntityAttribute
			hasOptionSet := len(attr.OptionSet.Options) > 0 || attr.OptionSet.ID != ""
			attrTable.AppendRow(table.Row{
				"Tracked Entity Attribute",
				attr.ID,
				attr.Name,
				ptea.Mandatory,
				hasOptionSet,
				attr.OptionSet.ID,
			})
		}
	}

	attrTable.Render()
	fmt.Println()

	// --- Section 2: Program Stages & Data Elements ---
	if grouped {
		for _, stage := range p.ProgramStages {
			stageTable := table.NewWriter()
			stageTable.SetOutputMirror(os.Stdout)
			stageTable.SetStyle(table.StyleRounded)

			fmt.Printf("PROGRAM STAGE: %s (%s)\n", stage.Name, stage.ID)

			stageTable.AppendHeader(table.Row{
				"Type", "ID", "Name", "Value Type", "Mandatory", "Has OptionSet", "OptionSet ID",
			})

			for _, psde := range stage.ProgramStageDataElements {
				if includeAll || psde.Compulsory {
					de := psde.DataElement
					hasOptionSet := len(de.OptionSet.Options) > 0 || de.OptionSet.ID != ""
					stageTable.AppendRow(table.Row{
						"Data Element",
						de.ID,
						de.Name,
						de.ValueType,
						psde.Compulsory,
						hasOptionSet,
						de.OptionSet.ID,
					})
				}
			}

			stageTable.Render()
			fmt.Println()
		}
	} else {
		// Combined table for all ProgramStages
		combined := table.NewWriter()
		combined.SetOutputMirror(os.Stdout)
		combined.SetStyle(table.StyleRounded)
		combined.AppendHeader(table.Row{
			"Stage", "Type", "ID", "Name", "Value Type", "Mandatory", "Has OptionSet", "OptionSet ID",
		})

		for _, stage := range p.ProgramStages {
			for _, psde := range stage.ProgramStageDataElements {
				if includeAll || psde.Compulsory {
					de := psde.DataElement
					hasOptionSet := len(de.OptionSet.Options) > 0 || de.OptionSet.ID != ""
					combined.AppendRow(table.Row{
						stage.Name,
						"Data Element",
						de.ID,
						de.Name,
						de.ValueType,
						psde.Compulsory,
						hasOptionSet,
						de.OptionSet.ID,
					})
				}
			}
		}

		combined.Render()
		fmt.Println()
	}
}
