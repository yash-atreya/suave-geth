package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"unicode"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/vm"
)

var (
	formatFlag bool
	writeFlag  bool
)

var structs []structObj

type structObj struct {
	Name  string
	Type  *abi.Type
	Types []abi.Argument
}

func tryAddStruct(typ abi.Type) {
	// de-reference any slice first
	for {
		if typ.T == abi.SliceTy {
			typ = *typ.Elem
		} else {
			break
		}
	}

	name := typ.InternalType
	if name == "" {
		// not a complex type
		return
	}

	// check if we already have this struct
	for _, s := range structs {
		if s.Name == name {
			return
		}
	}

	if typ.T != abi.TupleTy {
		// Basic type (i.e. type Bid is uint256). Since we use `InternalType`
		// to represent the type on the template, we remove it here so that
		// when the type declaration is generated, it will use the basic type.
		typ.InternalType = ""

		structs = append(structs, structObj{
			Name: name,
			Type: &typ,
		})
		return
	}

	// figure out if any internal element is a struct itself
	for _, arg := range typ.TupleElems {
		tryAddStruct(*arg)
	}

	args := []abi.Argument{}
	for indx, arg := range typ.TupleElems {
		args = append(args, abi.Argument{
			Name: typ.TupleRawNames[indx],
			Type: *arg,
		})
	}

	structs = append(structs, structObj{
		Name:  name,
		Types: args,
	})
}

func main() {
	flag.BoolVar(&formatFlag, "format", false, "format the output")
	flag.BoolVar(&writeFlag, "write", false, "write the output to the file")
	flag.Parse()

	methods := vm.GetRuntime().GetMethods()
	for _, method := range methods {
		for _, input := range method.Inputs {
			tryAddStruct(input.Type)
		}
		for _, output := range method.Outputs {
			tryAddStruct(output.Type)
		}
	}

	// sort the structs by name
	sort.Slice(structs, func(i, j int) bool {
		return structs[i].Name < structs[j].Name
	})

	// sort the methods by name
	sort.Slice(methods, func(i, j int) bool {
		return methods[i].Name < methods[j].Name
	})

	input := map[string]interface{}{
		"Methods": methods,
		"Structs": structs,
	}
	if err := applyTemplate(suaveLibTemplate, input, "./suave/sol/libraries/Suave2.sol"); err != nil {
		panic(err)
	}
}

func renderType(param interface{}, inFunc bool) string {
	typ, ok := param.(abi.Type)
	if !ok {
		typP, ok := param.(*abi.Type)
		if !ok {
			panic(errors.New("typ: invalid type"))
		}
		typ = *typP
	}

	isMemory := false

	suffix := ""
	if typ.T == abi.SliceTy {
		typ = *typ.Elem
		suffix += "[]"
		isMemory = true
	}
	if typ.T == abi.StringTy || typ.T == abi.BytesTy || typ.T == abi.TupleTy {
		isMemory = true
	}

	if isMemory && inFunc {
		suffix += " memory"
	}

	if typ.InternalType != "" {
		return typ.InternalType + suffix
	}

	return typ.String() + suffix
}

func toAddressName(input string) string {
	var result strings.Builder
	upperPrev := true

	for _, r := range input {
		if unicode.IsUpper(r) && !upperPrev {
			result.WriteString("_")
		}
		result.WriteRune(unicode.ToUpper(r))
		upperPrev = unicode.IsUpper(r)
	}

	return result.String()
}

func applyTemplate(templateText string, input interface{}, out string) error {
	funcMap := template.FuncMap{
		"typS": func(param interface{}) string {
			return renderType(param, false)
		},
		"typ": func(param interface{}) string {
			return renderType(param, true)
		},
		"toLower": func(param interface{}) string {
			str := param.(string)
			if str == "Address" {
				return str
			}
			return firstLetterToLower(param.(string))
		},
		"encodeAddrName": func(param interface{}) string {
			return toAddressName(param.(string))
		},
	}

	t, err := template.New("template").Funcs(funcMap).Parse(templateText)
	if err != nil {
		return err
	}

	var outputRaw bytes.Buffer
	if err = t.Execute(&outputRaw, input); err != nil {
		return err
	}

	// escape any quotes
	str := outputRaw.String()
	str = strings.Replace(str, "&#34;", "\"", -1)
	str = strings.Replace(str, "&amp;", "&", -1)
	str = strings.Replace(str, ", )", ")", -1)

	if formatFlag || writeFlag {
		if str, err = formatSolidity(str); err != nil {
			return err
		}
	}

	if err := outputFile(out, str); err != nil {
		return err
	}
	return nil
}

var suaveLibTemplate = `// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.8;

library Suave {
	error PeekerReverted(address, bytes);

	{{range .Structs}}
	{{ if .Type }}
	type {{ .Name }} is {{ typS .Type }};
	{{ else }}
	struct {{.Name}} {
	{{ range .Types }}
	{{ typS .Type }} {{ toLower .Name }};
	{{ end }}
	}
	{{ end }}
	{{end}}

	address public constant IS_CONFIDENTIAL_ADDR =
	0x0000000000000000000000000000000042010000;
	{{range .Methods}}
	address public constant {{encodeAddrName .Name}} =
	{{.Addr}};
	{{end}}

	// Returns whether execution is off- or on-chain
	function isConfidential() internal view returns (bool b) {
		(bool success, bytes memory isConfidentialBytes) = IS_CONFIDENTIAL_ADDR.staticcall("");
		if (!success) {
			revert PeekerReverted(IS_CONFIDENTIAL_ADDR, isConfidentialBytes);
		}
		assembly {
			// Load the length of data (first 32 bytes)
			let len := mload(isConfidentialBytes)
			// Load the data after 32 bytes, so add 0x20
			b := mload(add(isConfidentialBytes, 0x20))
		}
	}

	{{ range .Methods }}
	function {{.Name}} ( {{range .Inputs }} {{typ .Type}} {{toLower .Name}}, {{ end }}) internal view returns ( {{range .Outputs }} {{typ .Type}}, {{ end }}) {
		(bool success, bytes memory data) = {{encodeAddrName .Name}}.staticcall(abi.encode({{range .Inputs}}{{toLower .Name}}, {{end}}));
		if (!success) {
			revert PeekerReverted({{encodeAddrName .Name}}, data);
		}
		return abi.decode(data, ({{range .Outputs}}{{typS .Type}}, {{end}}));
	}
	{{ end }}
}
`

func formatSolidity(code string) (string, error) {
	// Check if "forge" command is available in PATH
	_, err := exec.LookPath("forge")
	if err != nil {
		return "", fmt.Errorf("forge command not found in PATH: %v", err)
	}

	// Command and arguments for forge fmt
	command := "forge"
	args := []string{"fmt", "--raw", "-"}

	// Create a command to run the forge fmt command
	cmd := exec.Command(command, args...)

	// Set up input from stdin
	cmd.Stdin = bytes.NewBufferString(code)

	// Set up output buffer
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	// Run the command
	if err = cmd.Run(); err != nil {
		return "", fmt.Errorf("error running command: %v", err)
	}

	return outBuf.String(), nil
}

func outputFile(out string, str string) error {
	if !writeFlag {
		fmt.Println("=> " + out)
		fmt.Println(str)
	} else {
		fmt.Println("Write: " + out)
		// write file to output and create any parent directories if necessary
		if err := os.MkdirAll(filepath.Dir(out), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(out, []byte(str), 0644); err != nil {
			return err
		}
	}
	return nil
}

func firstLetterToLower(s string) string {
	if len(s) == 0 {
		return s
	}

	r := []rune(s)
	r[0] = unicode.ToLower(r[0])

	return string(r)
}
