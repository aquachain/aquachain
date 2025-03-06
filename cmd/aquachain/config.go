// Copyright 2018 The aquachain Authors
// This file is part of aquachain.
//
// aquachain is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// aquachain is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with aquachain. If not, see <http://www.gnu.org/licenses/>.

package main

// var mainctx, maincancel = mainctxs.Main(), mainctxs.MainCancelCause()

// // These settings ensure that TOML keys use the same names as Go struct fields.
// var tomlSettings = toml.Config{
// 	NormFieldName: func(rt reflect.Type, key string) string {
// 		return key
// 	},
// 	FieldToKey: func(rt reflect.Type, field string) string {
// 		return field
// 	},
// 	MissingField: func(rt reflect.Type, field string) error {
// 		link := ""
// 		if unicode.IsUpper(rune(rt.Name()[0])) && rt.PkgPath() != "main" {
// 			link = fmt.Sprintf(", see https://pkg.go.dev/%s#%s for available fields", rt.PkgPath(), rt.Name())
// 		}
// 		return fmt.Errorf("field '%s' is not defined in %s%s", field, rt.String(), link)
// 	},
// }

//	func closeMain(err error) {
//		log.Warn("got closemain signal", "err", err)
//		maincancel(err)
//	}
