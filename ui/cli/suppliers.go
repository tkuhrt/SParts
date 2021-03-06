package main

/*
	The functions for the supplier routines can be found in this file.
*/

/*
 * NOTICE:
 * =======
 *  Copyright (c) 2018 Wind River Systems, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at:
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software  distributed
 * under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES
 * OR CONDITIONS OF ANY KIND, either express or implied.
 */

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	//"log"
	//"path"
	//"path/filepath"
	"net/http"
	"os"
	//"os/user"
	"sort"
	"strings"
	"text/tabwriter"
)

type SupplierRecord struct {
	UUID    string `json:"uuid"`             // UUID provide w/previous registration
	Name    string `json:"name"`             // Fullname
	ShortId string `json:"short_id"`         // 1-5 alphanumeric characters (unique)
	Passwd  string `json:"passwd,omitempty"` // RUNNING, DOWN, NOT RESPONDING
	Url     string `json:"url,omitempty"`    // 2-3 sentence description
}

type Part struct {
	PartId string `json:"part_id"`
}

type SupplierWithParts struct {
	UUID    string `json:"uuid"`               // UUID provide w/previous registration
	Name    string `json:"name"`               // Fullname
	ShortId string `json:"short_id,omitempty"` // 1-5 alphanumeric characters (unique)
	Passwd  string `json:"passwd,omitempty"`   // RUNNING, DOWN, NOT RESPONDING
	Url     string `json:"url,omitempty"`      // 2-3 sentence description
	Parts   []Part
}

func getSupplierList() (supplierList []SupplierRecord, err error) {

	replyAsBytes, err := httpGetAPIRequest(getLocalConfigValue(_LEDGER_ADDRESS_KEY),
		"/api/sparts/ledger/suppliers")
	if err != nil {
		if _DEBUG_DISPLAY_ON {
			displayErrorMsg(err.Error())
		}
		return nil, errors.New(fmt.Sprintf("Ledger may not be accessible."))
	}

	err = json.Unmarshal(replyAsBytes, &supplierList)
	if err != nil {
		if _DEBUG_DISPLAY_ON {
			displayErrorMsg(err.Error())
		}
		return nil, errors.New(fmt.Sprintf("Ledger response may not be properly formatted"))
	}

	return supplierList, nil
}

func displaySupplierList() {

	var supplierList []SupplierRecord
	supplierList, err := getSupplierList()
	if checkAndReportError(err) {
		return
	}

	// Let's check if the list of suppliers is empty
	if len(supplierList) == 0 {
		fmt.Println("  No suppliers are registered with the ledger.")
		return
	}

	//Sort the list
	supplierList = sortSupplierList(supplierList)

	const padding = 1
	w := tabwriter.NewWriter(os.Stdout, 0, 0, padding, ' ',
		tabwriter.Debug)
	fmt.Fprintf(w, "\n")
	fmt.Fprintf(w, "\t%s\t %s\t %s\n", " ------------------", "-------", "--------")
	fmt.Fprintf(w, "\t%s\t %s\t %s\n", "   Name  ", " Alias", "  UUID  ")
	fmt.Fprintf(w, "\t%s\t %s\t %s\n", " ------------------", "-------", "--------")

	for k := range supplierList {
		url := supplierList[k].Url
		if url == "" {
			url = "      "
		}
		alias, _ := getAliasUsingValue(supplierList[k].UUID)
		// format alias field for nil values for short length ones
		if alias == "" {
			alias = "   - "
		} else if len(alias) < 4 {
			alias = "  " + alias
		}

		fmt.Fprintf(w, "\t %s\t %s\t %s\n", supplierList[k].Name, alias, supplierList[k].UUID)
	}
	//fmt.Println()
	fmt.Fprintf(w, "\n")
	w.Flush()
}

// supplier.UUID == "" if error.
func getSupplierInfo(uuid string) (supplier SupplierWithParts, err error) {

	supplier.UUID = "" // set in case there is an error later

	if !isValidUUID(uuid) {
		return supplier, errors.New(fmt.Sprintf("Supplier UUID is not in a valid format: %s", uuid))
	}

	replyAsBytes, err := httpGetAPIRequest(getLocalConfigValue(_LEDGER_ADDRESS_KEY),
		"/api/sparts/ledger/suppliers/"+uuid)
	if err != nil {
		if _DEBUG_DISPLAY_ON {
			displayErrorMsg(err.Error())
		}
		return supplier, errors.New(fmt.Sprintf("Ledger may not be accessible."))
	}

	// Let's unpack the response as json structure
	err = json.Unmarshal(replyAsBytes, &supplier)
	if err != nil {
		if _DEBUG_DISPLAY_ON {
			displayErrorMsg(err.Error())
		}
		return supplier, errors.New(fmt.Sprintf("Ledger response may not be properly formatted"))
	}

	// Check if supplier exists. UUID should of been set in the json.Unmarshal call
	if supplier.UUID != uuid {
		return supplier, errors.New(fmt.Sprintf("Supplier not found with uuid = '%s'", uuid))
	}
	return supplier, nil
}

func displaySupplier(uuid string) {
	supplier, err := getSupplierInfo(uuid)
	if err != nil || supplier.UUID == "" {
		displayErrorMsg(err.Error())
		return
	}

	alias, _ := getAliasUsingValue(uuid)
	// format alias field for nil or for short length ones
	if alias == "" {
		alias = "<not defined>"
	} else {
		alias = _ALIAS_TOKEN + _CYAN_FG + alias + _COLOR_END
	}
	fmt.Println("  -----------------------------------------------")
	fmt.Printf("  Name   : %s%s%s\n", _CYAN_FG, supplier.Name, _COLOR_END)
	fmt.Println("  -----------------------------------------------")
	fmt.Println("  Label  :", supplier.ShortId)
	fmt.Println("  UUID   :", supplier.UUID)
	fmt.Println("  Alias  :", alias)
	fmt.Println("  URL    :", supplier.Url)

	if len(supplier.Parts) == 0 {
		// Supplier has no parts
		fmt.Println("  Parts  : <none> ")
	} else {
		displayParts(supplier.Parts)
	}
}

// if uuid is "" then it will automatically be generated.
func createSupplier(name string, shortID string, uuid string, passwd string, url string) string {
	var supplier SupplierRecord

	if uuid != "" && isValidUUID(uuid) {
		supplier.UUID = uuid
	} else {
		supplier.UUID = getUUID()
	}
	supplier.Name = name
	supplier.ShortId = shortID
	supplier.Passwd = passwd
	supplier.Url = url

	supplierAsBytes, err := json.Marshal(supplier)
	if err != nil {
		fmt.Printf("Error: %s", err)
		return ""
	}

	//fmt.Println (string(supplierAsBytes))
	requestURL := "http://" + getLocalConfigValue(_LEDGER_ADDRESS_KEY) + "/api/sparts/ledger/suppliers"
	req, err := http.NewRequest("POST", requestURL, bytes.NewBuffer(supplierAsBytes))
	if err != nil {
		fmt.Println(err)
		return ""
	}
	req.Header.Set("X-Custom-Header", "myvalue")
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
		return ""
	}
	defer resp.Body.Close()

	//fmt.Println("response Status:", resp.Status)
	//fmt.Println("response Headers:", resp.Header)
	body, _ := ioutil.ReadAll(resp.Body)
	//  {"status":"success"}
	if strings.Contains(string(body), "success") {
		//fmt.Println("response Body:", string(body))
		return supplier.UUID
	} else {
		return ""
	}
}

//----------------------------------------
// Supplier Slice List Sorting
//-----------------------------------------

// We use Go's sort slice functionn.
// https://golang.org/pkg/sort/#Slice

type By func(p1, p2 *SupplierRecord) bool

// Sort is a method on the function type, By, that sorts the argument slice according to the function.
func (by By) Sort(theList []SupplierRecord) {
	ps := &listSorter{
		theList: theList,
		by:      by, // The Sort method's receiver is the function (closure) that defines the sort order.
	}
	sort.Sort(ps)
}

// listSorter joins a By function and a slice of SupplierRecord to be sorted.
type listSorter struct {
	theList []SupplierRecord
	by      func(p1, p2 *SupplierRecord) bool // Closure used in the Less method.
}

// Len is part of sort.Interface.
func (s *listSorter) Len() int {
	return len(s.theList)
}

// Swap is part of sort.Interface.
func (s *listSorter) Swap(i, j int) {
	s.theList[i], s.theList[j] = s.theList[j], s.theList[i]
}

// Less is part of sort.Interface. It is implemented by calling the "by" closure in the sorter.
func (s *listSorter) Less(i, j int) bool {
	return s.by(&s.theList[i], &s.theList[j])
}

func sortSupplierList(theList []SupplierRecord) []SupplierRecord {

	name := func(p1, p2 *SupplierRecord) bool {
		return p1.Name < p2.Name
	}
	By(name).Sort(theList)
	return theList
}
