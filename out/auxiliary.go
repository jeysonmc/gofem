// Copyright 2016 The Gofem Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package out

import "github.com/cpmech/gosl/utl"

// CombineEidIps combines eids and ips(ids)
func CombineEidIps(eids, ips []int) (eids_ips [][]int) {
	eids_ips = utl.IntsAlloc(len(eids)*len(ips), 2)
	k := 0
	for _, eid := range eids {
		for _, ip := range ips {
			eids_ips[k][0] = eid
			eids_ips[k][1] = ip
			k++
		}
	}
	return
}

// ParseKey parses key like "duxdt" returning "ux" and time derivative number
//  Output: {key, number-of-time-derivatives}
//  Examples:  "ux"      => "ux", 0
//             "duxdt"   => "ux", 1
//             "d2uxdt2" => "ux", 2
func ParseKey(key string) (string, int) {
	if len(key) > 3 {
		n := len(key)
		if key[:1] == "d" && key[n-2:] == "dt" {
			return key[1 : n-2], 1
		}
		if len(key) > 5 {
			if key[:2] == "d2" && key[n-3:] == "dt2" {
				return key[2 : n-3], 2
			}
		}
	}
	return key, 0
}
