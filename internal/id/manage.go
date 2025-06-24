// Copyright 2020 Thomas.Hoehenleitner [at] seerose.net
// Use of this source code is governed by a license that can be found in the LICENSE file.

// Package id List is responsible for id List managing
package id

// List management

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"strconv"
	"strings"

	"github.com/rokath/trice/pkg/msg"
	"github.com/spf13/afero"
)

// NewLut returns a look-up map generated from JSON map file named fn.
func NewLut(w io.Writer, fSys *afero.Afero, fn string) TriceIDLookUp {
	lu := make(TriceIDLookUp)
	if fn == "emptyFile" { // reserved name for tests only
		return lu
	}
	msg.FatalOnErr(lu.fromFile(fSys, fn))
	if Verbose {
		fmt.Fprintln(w, "Read ID List file", fn, "with", len(lu), "items.")
	}
	return lu
}

// NewLutLI returns a look-up map generated from JSON map file named fn.
func NewLutLI(w io.Writer, fSys *afero.Afero, fn string) TriceIDLookUpLI {
	li := make(TriceIDLookUpLI)
	if fn == "emptyFile" { // reserved name for tests only
		return li
	}
	msg.FatalOnErr(li.fromFile(fSys, fn))
	if Verbose {
		fmt.Fprintln(w, "Read ID location information file", fn, "with", len(li), "items.")
	}
	return li
}

// newID() gets a new ID not used so far.
// The delivered id is usable as key for lu, but not added. So calling fn twice without adding to ilu could give the same value back.
// It is important that ilu was refreshed before with all sources to avoid finding as a new ID an ID which is already used in the source tree.
func (ilu TriceIDLookUp) newID(w io.Writer, min, max TriceID, searchMethod string) TriceID {
	if Verbose {
		fmt.Fprintln(w, "IDMin=", min, "IDMax=", max, "IDMethod=", searchMethod)
	}
	switch searchMethod {
	case "random":
		return ilu.newRandomID(w, min, max)
	case "upward":
		return ilu.newUpwardID(min, max)
	case "downward":
		return ilu.newDownwardID(min, max)
	}
	msg.Info(fmt.Sprint("ERROR:", searchMethod, "is unknown ID search method."))
	return 0
}

// newRandomID provides a random free ID inside interval [min,max].
// The delivered id is usable as key for lu, but not added. So calling fn twice without adding to ilu could give the same value back.
func (ilu TriceIDLookUp) newRandomID(w io.Writer, min, max TriceID) (id TriceID) {
	interval := int(max - min + 1)
	freeIDs := interval - len(ilu)
	msg.FatalInfoOnFalse(freeIDs > 0, "no new ID possible, "+fmt.Sprint(min, max, len(ilu)))
	wrnLimit := interval >> 3 // 12.5%
	msg.InfoOnTrue(freeIDs < wrnLimit, "WARNING: Less than 12.5% IDs free!")
	if interval <= 0 {
		log.Fatal(w, "No ID space left:", min, max)
	}
	id = min + TriceID(rand.Intn(interval))
	if len(ilu) == 0 {
		return
	}
	for {
	nextTry:
		for k := range ilu {
			if id == k { // id used
				fmt.Fprintln(w, "ID", id, "used, next try...")
				id = min + TriceID(rand.Intn(interval))
				goto nextTry
			}
		}
		return
	}
}

// newUpwardID provides the smallest free ID inside interval [min,max].
// The delivered id is usable as key for lut, but not added. So calling fn twice without adding to ilu gives the same value back.
func (ilu TriceIDLookUp) newUpwardID(min, max TriceID) (id TriceID) {
	interval := int(max - min + 1)
	freeIDs := interval - len(ilu)
	msg.FatalInfoOnFalse(freeIDs > 0, "no new ID possible: "+fmt.Sprint("min=", min, ", max=", max, ", used=", len(ilu)))
	id = min
	if len(ilu) == 0 {
		return
	}
	for {
	nextTry:
		for k := range ilu {
			if id == k { // id used
				id++
				goto nextTry
			}
		}
		return
	}
}

// newDownwardID provides the biggest free ID inside interval [min,max].
// The delivered id is usable as key for lut, but not added. So calling fn twice without adding to ilu gives the same value back.
func (ilu TriceIDLookUp) newDownwardID(min, max TriceID) (id TriceID) {
	interval := int(max - min + 1)
	freeIDs := interval - len(ilu)
	msg.FatalInfoOnFalse(freeIDs > 0, "no new ID possible: "+fmt.Sprint("min=", min, ", max=", max, ", used=", len(ilu)))
	id = max
	if len(ilu) == 0 {
		return
	}
	for {
	nextTry:
		for k := range ilu {
			if id == k { // id used
				id--
				goto nextTry
			}
		}
		return
	}
}

// FromJSON converts JSON byte slice to ilu.
func (ilu TriceIDLookUp) FromJSON(b []byte) (err error) {
	if 0 < len(b) {
		if Verbose {
			fmt.Println("Updating ilu.")
		}
		err = json.Unmarshal(b, &ilu)
	}
	return
}

// FromJSON converts JSON byte slice to li.
func (li TriceIDLookUpLI) FromJSON(b []byte) (err error) {
	if 0 < len(b) {
		if Verbose {
			fmt.Println("Updating li.")
		}
		err = json.Unmarshal(b, &li)
	}
	return
}

// fromFile reads file fn into lut. Existing keys are overwritten, lut is extended with new keys.
func (ilu TriceIDLookUp) fromFile(fSys *afero.Afero, fn string) error {
	b, e := fSys.ReadFile(fn)
	s := fmt.Sprintf("fn=%s, maybe need to create an empty file first? (Safety feature)", fn)
	msg.FatalInfoOnErr(e, s)
	if Verbose {
		fmt.Println("ilu.fromFile", fn, "- file size is", len(b))
	}
	return ilu.FromJSON(b)
}

var Logging bool // Logging is true, when sub command log is active.

// fromFile reads fSys file fn into lut.
func (li TriceIDLookUpLI) fromFile(fSys *afero.Afero, fn string) error {
	b, err := fSys.ReadFile(fn)
	if err == nil { // file found
		if Verbose {
			fmt.Println("li.fromFile", fn, "- file size is", len(b))
		}
		return li.FromJSON(b)
	}
	// no li.json
	if Logging {
		if Verbose {
			fmt.Println("File ", fn, "not found, not showing location information")
		}
		return nil // silently ignore non existing file
	}
	s := fmt.Sprintf("%s not found, maybe need to create an empty file first? (Safety feature)", fn)
	msg.FatalInfoOnErr(err, s)
	return err // not reached
}

// AddFmtCount adds inside ilu to all trice type names without format specifier count the appropriate count.
// Special Trices are ignored, but checked, because they have a fixed format specifier count:
// TRICE_S triceS TriceS TRiceS: 1 (1 value)
// TRICE_N triceN TriceN TRiceN: 1 (2 values)
// TRICE_B triceB TriceB TRiceB: 1 (2 values)
// TRICE_F triceF TriceF TRiceF: 0 (2 values)
// example change:
// `map[10000:{Trice8_2 hi %03u, %5x} 10001:{TRICE16 hi %03u, %5x}]
// `map[10000:{Trice8_2 hi %03u, %5x} 10001:{TRICE16_2 hi %03u, %5x}]
// ice" -> ice_n", ice8" -> ice8_n"
// B" -> B", F" -> F", S" -> S", N" -> N"
// _n" -> _n",
// see also ConstructFullTriceInfo
func (ilu TriceIDLookUp) AddFmtCount(w io.Writer) {
	for i, x := range ilu {
		n := formatSpecifierCount(x.Strg)
		if !(0 <= n && n <= 12) {
			fmt.Fprintln(w, "Invalid format specifier count", n, "- please check", x)
			continue
		}
		if strings.ContainsAny(x.Type, "S") || strings.ContainsAny(x.Type, "N") || strings.ContainsAny(x.Type, "B") {
			if n != 1 {
				if strings.HasPrefix(x.Strg, SAliasStrgPrefix) && strings.HasSuffix(x.Strg, SAliasStrgSuffix) {
					continue // We do not check parameter count here.
				}
				fmt.Fprintf(w, "%+v <- Expected format specifier count is 1 but got %d", x, n)
			}
			continue
		}
		if strings.ContainsAny(x.Type, "F") {
			if n != 0 {
				fmt.Fprintf(w, "%+v <- Expected format specifier count is 0 but got %d", x, n)
			}
			continue
		}
		s := strings.Split(x.Type, "_")
		if len(s) > 2 { // TRICE_B_1 -> "B" already excluded here
			fmt.Fprintln(w, "Unexpected Trice type - please check", x)
			continue
		}
		if len(s) == 2 { // example: trice8_3
			i, err := strconv.Atoi(s[1])
			if err != nil || i < 0 || i > 12 {
				fmt.Fprintln(w, "Unexpected Trice type - please check", x)
			}
			continue
		}
		if n == 0 {
			continue //
		}
		x.Type = fmt.Sprintf(x.Type+"_%d", n) // trice* -> trice*_n
		ilu[i] = x
	}
}

// toJSON converts lut into JSON byte slice in human-readable form.
func (lu TriceIDLookUp) toJSON() ([]byte, error) {
	return json.MarshalIndent(lu, "", "\t")
}

// toFile writes lut into file fn as indented JSON and in verbose mode helpers for third party.
func (ilu TriceIDLookUp) toFile(fSys afero.Fs, fn string) (err error) {
	var fJSON afero.File
	fJSON, err = fSys.Create(fn)
	msg.FatalOnErr(err)
	defer func() {
		err = fJSON.Close()
		msg.FatalOnErr(err)
	}()
	var b []byte
	b, err = ilu.toJSON()
	msg.FatalOnErr(err)
	_, err = fJSON.Write(b)
	msg.FatalOnErr(err)
	/////////
	return
}

// reverseS returns a reversed map. If different triceID's assigned to several equal TriceFmt all of the TriceID gets it into flu.
func (ilu TriceIDLookUp) reverseS() (flu triceFmtLookUp) {
	flu = make(triceFmtLookUp)
	for id, tF := range ilu {
		addID(tF, id, flu)
	}
	return
}

// addID adds tF and id to flu. If tF already exists inside flu, its id slice is extended with id.
func addID(tF TriceFmt, id TriceID, flu triceFmtLookUp) {
	// tF.Type = strings.ToUpper(tF.Type) // no distinction for lower and upper case Type
	idSlice := flu[tF] // If the key doesn't exist, the first value will be the default zero value.
	idSlice = append(idSlice, id)
	flu[tF] = idSlice
}

// toFile writes lut into file fn as indented JSON.
func (lim TriceIDLookUpLI) toFile(fSys afero.Fs, fn string) (err error) {
	f0, err := fSys.Create(fn)
	msg.FatalOnErr(err)
	defer func() {
		err = f0.Close()
		msg.FatalOnErr(err)
	}()

	b, err := lim.toJSON()
	msg.FatalOnErr(err)

	_, err = f0.Write(b)
	msg.FatalOnErr(err)

	return
}

// toJSON converts lim into JSON byte slice in human-readable form.
func (lim TriceIDLookUpLI) toJSON() ([]byte, error) {
	return json.MarshalIndent(lim, "", "\t")
}

/*
// distance returns 80 - len(s) spaces as string
func distance(s string) string {
	switch 80 - len(s) {
	default:
		return ""
	case 0:
		return ""
	case 1:
		return " "
	case 2:
		return "  "
	case 3:
		return "   "
	case 4:
		return "    "
	case 5:
		return "     "
	case 6:
		return "      "
	case 7:
		return "       "
	case 8:
		return "        "
	case 9:
		return "         "
	case 10:
		return "          "
	case 11:
		return "           "
	case 12:
		return "            "
	case 13:
		return "             "
	case 14:
		return "              "
	case 15:
		return "               "
	case 16:
		return "                "
	case 17:
		return "                 "
	case 18:
		return "                  "
	case 19:
		return "                   "
	case 20:
		return "                    "
	case 21:
		return "                     "
	case 22:
		return "                      "
	case 23:
		return "                       "
	case 24:
		return "                        "
	case 25:
		return "                         "
	case 26:
		return "                          "
	case 27:
		return "                           "
	case 28:
		return "                            "
	case 29:
		return "                             "
	case 30:
		return "                              "
	case 31:
		return "                               "
	case 32:
		return "                                "
	case 33:
		return "                                 "
	case 34:
		return "                                  "
	case 35:
		return "                                   "
	case 36:
		return "                                    "
	case 37:
		return "                                     "
	case 38:
		return "                                      "
	case 39:
		return "                                       "
	case 40:
		return "                                        "
	case 41:
		return "                                         "
	case 42:
		return "                                          "
	case 43:
		return "                                           "
	case 44:
		return "                                            "
	case 45:
		return "                                             "
	case 46:
		return "                                              "
	case 47:
		return "                                               "
	case 48:
		return "                                                "
	case 49:
		return "                                                 "
	case 50:
		return "                                                  "
	case 51:
		return "                                                   "
	case 52:
		return "                                                    "
	case 53:
		return "                                                     "
	case 54:
		return "                                                      "
	case 55:
		return "                                                       "
	case 56:
		return "                                                        "
	case 57:
		return "                                                         "
	case 58:
		return "                                                          "
	case 59:
		return "                                                           "
	case 60:
		return "                                                            "
	case 61:
		return "                                                             "
	case 62:
		return "                                                              "
	case 63:
		return "                                                               "
	case 64:
		return "                                                                "
	case 65:
		return "                                                                 "
	case 66:
		return "                                                                  "
	case 67:
		return "                                                                   "
	case 68:
		return "                                                                    "
	case 69:
		return "                                                                     "
	case 70:
		return "                                                                      "
	case 71:
		return "                                                                       "
	case 72:
		return "                                                                        "
	case 73:
		return "                                                                         "
	case 74:
		return "                                                                          "
	case 75:
		return "                                                                           "
	case 76:
		return "                                                                            "
	case 77:
		return "                                                                             "
	case 78:
		return "                                                                              "
	case 79:
		return "                                                                               "
	}
}
*/
