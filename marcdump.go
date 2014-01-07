// Copyright 2013 Thomas Emerson. All rights reserved.

package main

import (
	"flag"
	"fmt"
	"github.com/treerex/marc21"
	"math"
	"os"
	"text/tabwriter"
)

var maxRecords uint

func init() {
	flag.UintVar(&maxRecords, "m", math.MaxUint32, "Maximum number of records to dump")
}

func main() {
	flag.Parse()

	if flag.NArg() != 1 {
		usage()
	}

	file, err := os.Open(flag.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
	
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 3, ' ', 0)

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

		printRecord(rec, w)

		recordCount += 1
		if recordCount == maxRecords {
			break
		}
	}
}

func printRecord(record *marc21.MarcRecord, w *tabwriter.Writer) {
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

func usage() {
	fmt.Fprintf(os.Stderr, "usage: marcdump [-m max] marcfile\n")
	os.Exit(1)
}

// ~/shrc/hlom/data/hlom/ab.bib.00.20131101.full.mrc
