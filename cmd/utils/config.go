package utils

import (
	"gitlab.com/aquachain/aquachain/common/config"
)

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
// 		err := fmt.Errorf("field '%s' is not defined in %s%s", field, rt.String(), link)
// 		if sense.Getenv("TOML_MISSING_FIELD") == "OK" {
// 			log.Warn(err.Error())
// 			return nil
// 		}
// 		// wrong config file, or outdated config file
// 		return err
// 	},
// }

type AquachainConfig = config.AquachainConfigFull
