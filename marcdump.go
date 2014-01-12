// Copyright 2013-14 Thomas Emerson
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/TreeRex/marc21"
	"math"
	"os"
	"regexp"
	"text/tabwriter"
)

type selectionSpec struct {
	field string
	subfield string
	criterion *regexp.Regexp
}

// An actionFunc is called to display a record
type actionFunc func(record *marc21.MarcRecord, w *tabwriter.Writer) error

var (
	errInvalidSelectorSpec = errors.New("marcdump: invalid selector specification")
)

var (
	// Group 1: field
	// Group 2: subfield, or ""
	// Group 3: specification, or ""
	//                                    field           subfield        spec
	selectionSpecRegexp = regexp.MustCompile("^([0-9A-Za-z]{3})(?:_([0-9a-z]))?(?:=(.+))?$")
)

// Command-line options
var (
	maxRecords uint

	makeIndex string
	useIndex string

	selectorOpt string
	fieldsOpt string
)

// Select the record whose 020$a == 9780743264747, output field 650 value(s) only
//    marcdump -selector 020_a=9780743264747 -fields 650 <marcfile>
//  
// Select any record that has a value in 020_a and generate an index for the marcfile
//    marcdump -selector 020_a -mkindex <indxfile> <marcfile>
// marcdump -selector 020_a=9780743264747 -fields 650 -index <indexfile> <marcfile>

func init() {
	flag.UintVar(&maxRecords, "m", math.MaxUint32, "Maximum number of records to dump")
	flag.StringVar(&fieldsOpt, "f", "", "Colon separated field tags to output")
	flag.StringVar(&selectorOpt, "s", "", "Field selector(s)")
	flag.StringVar(&makeIndex, "mkindex", "", "Name of index file to generate")
	flag.StringVar(&useIndex, "index", "", "Name of index file")
}

func getSelectionSpec() (*selectionSpec, error) {
	selectionSpec := new(selectionSpec)

	if selectorOpt != "" {
		spec := selectionSpecRegexp.FindStringSubmatch(selectorOpt)
		if spec != nil {
			if spec[3] != "" {
				re,err := regexp.Compile(spec[3])
				if err != nil {
					return nil, err
				}
				selectionSpec.criterion = re
			}
			selectionSpec.field = spec[1]
			selectionSpec.subfield = spec[2]
		} else {
			return nil, errInvalidSelectorSpec
		}
	}
	return selectionSpec, nil
}


func (s *selectionSpec) match(r *marc21.MarcRecord) bool {
	if s.field == "" {
		return true
	}

	if marc21.IsControlFieldTag(s.field) {
		field, err := r.GetControlField(s.field)
		if err != nil {
			return false
		}
		if s.criterion != nil {
			return s.criterion.MatchString(field)
		}
		return true
	} else { // Data Field
		subfields := make([]string, 1)

		field, _ := r.GetDataField(s.field)

		for instance := 0; instance < field.ValueCount(); instance++ {
			// if no subfield is specified in the spec then
			// we want to search all of them. since these can
			// vary per field instance we need to get the list
			// each time.
			if s.subfield != "" {
				subfields[0] = s.subfield
			} else {
				subfields = field.GetSubfields(instance)
			}

			for _, subfield := range subfields {
				sfv := field.GetNthSubfield(subfield, instance)
				if sfv != "" {
					// the subfield exists: need to check because the
					// user supplied subfield may not exist in this
					// instance
					if s.criterion != nil {
						// and there is a search criterion
						if s.criterion.MatchString(sfv) {
							// and it matches
							return true;
						}
					} else {
						// no search criterion, but the field exists
						return true;
					}
				}
			}
		}
		return false;
	}
}


func getActionFunction() actionFunc {
	if makeIndex != "" {
		return nil
	} else {
		return printRecord
	}
}


func main() {
	flag.Parse()

	if flag.NArg() != 1 {
		usage()
	}

	file, err := os.Open(flag.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 3, ' ', 0)

	selector, err := getSelectionSpec()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	action := getActionFunction()
	if action == nil {
		fmt.Fprintln(os.Stderr, "Internal Error: could not get action function")
	}

	recordCount := uint(0)
	
	reader := marc21.NewReader(file, false)
	for {
		rec,err := reader.Next()

		if rec == nil && err == nil {
			break
		} else if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			break
		}

		if selector.match(rec) {
			action(rec, w)
			recordCount += 1
			if recordCount == maxRecords {
				break
			}
		}
	}
}

//
// Record Printing Functions
//

func printRecord(record *marc21.MarcRecord, w *tabwriter.Writer) error {
	fmt.Fprintf(w, "Leader\t%s\n", record.GetLeader())
	fields := record.GetFieldList()
	for _,f := range fields {
		if marc21.IsControlFieldTag(f) {
			v,_ := record.GetControlField(f)
			fmt.Fprintf(w, "%s\t%s\n", f, v)
		} else {
			v,_ := record.GetDataField(f)
			printDataField(w, v)
		}
	}
	w.Flush()
	return nil
}

func printDataField(w *tabwriter.Writer, field marc21.VariableField) {
	for i := 0; i < field.ValueCount(); i++ {
		value := field.GetIndicators(i)
		for _,sf := range field.GetSubfields(i) {
			value += fmt.Sprintf("$%s%s", sf, field.GetNthSubfield(sf, i))
		}
		fmt.Fprintf(w, "%s\t%s\n", field.Tag, value)
	}
}

//
// Record Selection Functions
//

func selectAll(record *marc21.MarcRecord) bool {
	return true
}


func usage() {
	fmt.Fprintf(os.Stderr, "usage: marcdump [-m max] marcfile\n")
	os.Exit(1)
}

// ~/shrc/hlom/data/hlom/ab.bib.00.20131101.full.mrc
