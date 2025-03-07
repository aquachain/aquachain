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

package aquaflags

import (
	"encoding"
	"errors"
	"flag"
	"fmt"
	"math/big"
	"os"
	"os/user"
	"path"
	"strings"

	"github.com/urfave/cli/v3"
	"gitlab.com/aquachain/aquachain/common/log"
	"gitlab.com/aquachain/aquachain/common/math"
	"gitlab.com/aquachain/aquachain/common/sense"
)

// Custom type which is registered in the flags library which cli uses for
// argument parsing. This allows us to expand Value to an absolute path when
// the argument is parsed
type DirectoryString struct {
	Value string
	isSet bool
}

func NewDirectoryString(value string) DirectoryString {
	return DirectoryString{Value: expandPath(value)}
}

var _ flag.Getter = (*DirectoryString)(nil)

func (self *DirectoryString) String() string {
	return self.Value
}

func (ds *DirectoryString) Set(value string) error {
	ds.Value = expandPath(value)
	return nil
}

func (self *DirectoryString) Get() any {
	return self.Value
}

func (df *DirectoryFlag) Set(value string) {
	df.Value.Set(value)
	df.Value.isSet = true
}

// Custom cli.Flag type which expand the received string to an absolute path.
// e.g. ~/.aquachain -> /home/username/.aquachain
type DirectoryFlag struct {
	Name  string
	Value DirectoryString
	Usage string
}

func (df DirectoryFlag) Names() []string {
	return strings.Split(df.Name, ",")
}

func (df DirectoryFlag) GetName() string {
	return df.Name
}

func (df DirectoryFlag) String() string {
	fmtString := "%s %v\t%v"
	if len(df.Value.Value) > 0 {
		fmtString = "%s \"%v\"\t%v"
	}
	return fmt.Sprintf(fmtString, prefixedNames(df.Name), df.Value.Value, df.Usage)
}

func (df DirectoryFlag) IsSet() bool {
	return df.Value.isSet
}

func (self DirectoryFlag) Get() string {
	return self.Value.Value
}

// in case a flag has multiple names, we need to split them and add them to the flag set
func eachName(parts []string, fn func(string)) {
	for _, name := range parts {
		name = strings.Trim(name, " ")
		fn(name)
	}

}

// called by cli library, grabs variable from environment (if in env)
// and adds variable to flag set for parsing.
func (df *DirectoryFlag) Apply(set *flag.FlagSet) error {
	eachName(df.Names(), func(name string) {
		set.Var(&df.Value, name, df.Usage)
	})
	return nil
}

type TextMarshaler interface {
	encoding.TextMarshaler
	encoding.TextUnmarshaler
}

// textMarshalerVal turns a TextMarshaler into a flag.Value
type textMarshalerVal struct {
	v TextMarshaler
}

func (v textMarshalerVal) String() string {
	if v.v == nil {
		return ""
	}
	text, err := v.v.MarshalText()
	if err != nil {
		log.Warn("failed to marshal text", "err", err.Error())
	}
	return string(text)
}

func (v textMarshalerVal) Set(s string) error {
	return v.v.UnmarshalText([]byte(s))
}

func (v textMarshalerVal) Get() any {
	return v.v
}

// TextMarshalerFlag wraps a TextMarshaler value.
type TextMarshalerFlag struct {
	Name  string
	Value TextMarshaler
	Usage string
	isSet bool
}

func (f TextMarshalerFlag) Names() []string {
	return strings.Split(f.Name, ",")
}

func (f TextMarshalerFlag) IsSet() bool {
	return f.isSet
}

func (f TextMarshalerFlag) GetName() string {
	return f.Name
}

func (f TextMarshalerFlag) String() string {
	return fmt.Sprintf("%s \"%v\"\t%v", prefixedNames(f.Name), f.Value, f.Usage)
}

func (f *TextMarshalerFlag) Apply(set *flag.FlagSet) error {
	eachName(f.Names(), func(name string) {
		set.Var(textMarshalerVal{f.Value}, f.Name, f.Usage)
	})
	return nil
}

// // GlobalTextMarshaler returns the value of a TextMarshalerFlag from the global flag set.
// func GlobalTextMarshaler(cmd *cli.Command, name string) TextMarshaler {
// 	val := cmd.Generic(name)
// 	if val == nil {
// 		return nil
// 	}
// 	return val.Get().(textMarshalerVal).v
// }

// BigFlag is a command line flag that accepts 256 bit big integers in decimal or
// hexadecimal syntax.
type BigFlag struct {
	Name  string
	Value *big.Int
	Usage string
	isSet bool
}

// bigValue turns *big.Int into a flag.Value
type bigValue big.Int

func (b *bigValue) String() string {
	if b == nil {
		return ""
	}
	return (*big.Int)(b).String()
}

func (b *bigValue) Set(s string) error {
	n, ok := math.ParseBig256(s)
	if !ok {
		return errors.New("invalid integer syntax")
	}
	*b = (bigValue)(*n)
	return nil
}

func (b *bigValue) Get() any {
	return (*big.Int)(b)
}

func (f BigFlag) Names() []string {
	return strings.Split(f.Name, ",")
}

func (f BigFlag) IsSet() bool {
	return f.isSet
}

func (f BigFlag) GetName() string {
	return f.Name
}

func (f BigFlag) String() string {
	fmtString := "%s %v\t%v"
	if f.Value != nil {
		fmtString = "%s \"%v\"\t%v"
	}
	return fmt.Sprintf(fmtString, prefixedNames(f.Name), f.Value, f.Usage)
}

func (f *BigFlag) Apply(set *flag.FlagSet) error {
	eachName(f.Names(), func(name string) {
		set.Var((*bigValue)(f.Value), f.Name, f.Usage)
	})
	return nil
}

// GlobalBig returns the value of a BigFlag from the global flag set.
func GlobalBig(cmd *cli.Command, name string) *big.Int {
	val := cmd.Generic(name)
	if val == nil {
		panic("bad flag is named " + name)
	}

	return (*big.Int)(val.Get().(*bigValue))
}

func prefixFor(name string) (prefix string) {
	if len(name) == 1 {
		prefix = "-"
	} else {
		prefix = "--"
	}

	return
}

func prefixedNames(fullName string) (prefixed string) {
	parts := strings.Split(fullName, ",")
	for i, name := range parts {
		name = strings.Trim(name, " ")
		prefixed += prefixFor(name) + name
		if i < len(parts)-1 {
			prefixed += ", "
		}
	}
	return
}

// Expands a file path
// 1. replace tilde with users home dir
// 2. expands embedded environment variables
// 3. cleans the path, e.g. /a/b/../c -> /a/c
// Note, it has limitations, e.g. ~someuser/tmp will not be expanded
func expandPath(p string) string {
	if strings.HasPrefix(p, "~/") || strings.HasPrefix(p, "~\\") {
		if home := homeDir(); home != "" {
			p = home + p[1:]
		}
	}
	return path.Clean(os.ExpandEnv(p))
}

func homeDir() string {
	if home := sense.Getenv("HOME"); home != "" {
		return home
	}
	if usr, err := user.Current(); err == nil {
		return usr.HomeDir
	}
	return ""
}

func workingDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	return dir
}
