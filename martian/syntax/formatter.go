//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// MRO canonical formatting. Inspired by gofmt.
//

package syntax

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"
)

const (
	INDENT  string = "    "
	NEWLINE string = "\n"
)

func max(a, b int) int {
	if a > b {
		return a
	} else {
		return b
	}
}

type stringWriter interface {
	io.ByteWriter
	io.Writer
	WriteRune(rune) (int, error)
	WriteString(string) (int, error)
}

type printer struct {
	buf         strings.Builder
	comments    map[string][]*commentBlock
	lastComment SourceLoc
}

func (self *printer) printComments(node *AstNode, prefix string) {
	if self.lastComment.File != nil &&
		self.lastComment.File.FullPath != node.Loc.File.FullPath {
		for _, comment := range self.comments[self.lastComment.File.FullPath] {
			self.buf.WriteString(comment.Value)
			self.buf.WriteString(NEWLINE)
		}
		delete(self.comments, self.lastComment.File.FullPath)
		self.buf.WriteString("#\n# @include \"")
		self.buf.WriteString(node.Loc.File.FileName)
		self.buf.WriteString("\"\n#\n\n")
	}
	for _, c := range node.scopeComments {
		if self.lastComment.Line != 0 && self.lastComment.Line == c.Loc.Line-2 {
			self.buf.WriteString(NEWLINE)
		}

		self.lastComment = c.Loc
		self.buf.WriteString(prefix)
		self.buf.WriteString(c.Value)
		self.buf.WriteString(NEWLINE)
	}
	if len(node.scopeComments) > 0 {
		self.buf.WriteString(NEWLINE)
	}
	for _, c := range node.Comments {
		self.buf.WriteString(prefix)
		self.buf.WriteString(c)
		self.buf.WriteString(NEWLINE)
	}
	self.lastComment = node.Loc
}

func (self *printer) WriteString(s string) (int, error) {
	return self.buf.WriteString(s)
}

func (self *printer) Write(b []byte) (int, error) {
	return self.buf.Write(b)
}

func (self *printer) WriteByte(b byte) error {
	return self.buf.WriteByte(b)
}

func (self *printer) WriteRune(r rune) (int, error) {
	return self.buf.WriteRune(r)
}

func (self *printer) Printf(format string, args ...interface{}) {
	fmt.Fprintf(&self.buf, format, args...)
}

func (self *printer) DumpComments() {
	for _, fcomments := range self.comments {
		for _, comment := range fcomments {
			self.buf.WriteString(comment.Value)
			self.buf.WriteString(NEWLINE)
		}
	}
	self.comments = nil
}

func (self *printer) String() string {
	return self.buf.String()
}

//
// Expression
//
func (self *ValExp) format(w stringWriter, prefix string) {
	if self.Value == nil {
		w.WriteString("null")
	} else if self.Kind == KindInt {
		fmt.Fprintf(w, "%d", self.Value)
	} else if self.Kind == KindFloat {
		fmt.Fprintf(w, "%g", self.Value)
	} else if self.Kind == KindString {
		fmt.Fprintf(w, "\"%s\"", self.Value)
	} else if self.Kind == KindMap {
		self.formatMap(w, prefix)
	} else if self.Kind == KindArray {
		self.formatArray(w, prefix)
	} else {
		fmt.Fprint(w, self.Value)
	}
}

func (self *ValExp) formatSweep(w stringWriter, prefix string) {
	values := self.Value.([]Exp)
	w.WriteString("sweep(\n")
	vindent := prefix + INDENT
	for _, val := range values {
		w.WriteString(vindent)
		val.format(w, vindent)
		w.WriteString(",\n")
	}
	w.WriteString(prefix)
	w.WriteRune(')')
}

func (self *ValExp) formatArray(w stringWriter, prefix string) {
	values := self.Value.([]Exp)
	if len(values) == 0 {
		w.WriteString("[]")
	} else if len(values) == 1 {
		// Place single-element arrays on a single line.
		w.WriteRune('[')
		values[0].format(w, prefix)
		w.WriteRune(']')
	} else {
		w.WriteString("[\n")
		vindent := prefix + INDENT
		for _, val := range values {
			w.WriteString(vindent)
			val.format(w, vindent)
			w.WriteString(",\n")
		}
		w.WriteString(prefix)
		w.WriteRune(']')
	}
}

func (self *ValExp) formatMap(w stringWriter, prefix string) {
	if valExpMap, ok := self.Value.(map[string]Exp); ok && len(valExpMap) > 0 {
		w.WriteString("{\n")
		vindent := prefix + INDENT
		keys := make([]string, 0, len(valExpMap))
		for key := range valExpMap {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			w.WriteString(vindent)
			w.WriteRune('"')
			w.WriteString(key)
			w.WriteString(`": `)
			valExpMap[key].format(w, vindent)
			w.WriteString(",\n")
		}
		w.WriteString(prefix)
		w.WriteRune('}')
	} else {
		w.WriteString("{}")
	}
}

func (self *RefExp) format(w stringWriter, prefix string) {
	if self.Kind == KindCall {
		w.WriteString(self.Id)
		if self.OutputId != "default" {
			w.WriteRune('.')
			w.WriteString(self.OutputId)
		}
	} else {
		w.WriteString("self.")
		w.WriteString(self.Id)
	}
}

//
// Binding
//
func (self *BindStm) format(printer *printer, prefix string, idWidth int) {
	printer.printComments(self.getNode(), prefix+INDENT)
	printer.printComments(self.Exp.getNode(), prefix+INDENT)
	idPad := ""
	if len(self.Id) < idWidth {
		idPad = strings.Repeat(" ", idWidth-len(self.Id))
	}
	printer.Printf("%s%s%s%s = ", prefix, INDENT,
		self.Id, idPad)
	if ve, ok := self.Exp.(*ValExp); ok {
		if arr, ok := ve.Value.([]Exp); ok && self.Sweep && len(arr) > 1 {
			ve.formatSweep(printer, prefix+INDENT)
			printer.WriteRune(',')
			printer.WriteString(NEWLINE)
			return
		}
	}
	self.Exp.format(printer, prefix+INDENT)
	printer.WriteRune(',')
	printer.WriteString(NEWLINE)
}

func (self *BindStms) format(printer *printer, prefix string) {
	printer.printComments(self.getNode(), prefix)
	idWidth := 0
	for _, bindstm := range self.List {
		if len(bindstm.Id) < 30 {
			idWidth = max(idWidth, len(bindstm.Id))
		}
	}
	for _, bindstm := range self.List {
		bindstm.format(printer, prefix, idWidth)
	}
}

//
// Parameter
//
func paramFormat(printer *printer, param Param, modeWidth int, typeWidth int, idWidth int, helpWidth int) {
	printer.printComments(param.getNode(), INDENT)
	id := param.GetId()
	if id == "default" {
		id = ""
	}

	// Generate column alignment paddings.
	modePad := strings.Repeat(" ", modeWidth-len(param.getMode()))
	typePad := strings.Repeat(" ", typeWidth-len(param.GetTname())-2*param.GetArrayDim())
	idPad := ""
	if idWidth > len(id) {
		idPad = strings.Repeat(" ", idWidth-len(id))
	}
	helpPad := ""
	if helpWidth > len(param.GetHelp()) {
		helpPad = strings.Repeat(" ", helpWidth-len(param.GetHelp()))
	}

	// Common columns up to type name.
	printer.Printf("%s%s%s %s", INDENT,
		param.getMode(), modePad, param.GetTname())

	// If type is annotated as array, add brackets and shrink padding.
	for i := 0; i < param.GetArrayDim(); i++ {
		printer.WriteString("[]")
	}

	// Add id if not default.
	if id != "" {
		printer.Printf("%s %s", typePad, id)
	}

	// Add help string if it exists.
	if len(param.GetHelp()) > 0 {
		if id == "" {
			printer.Printf("%s ", typePad)
		}
		printer.Printf("%s  \"%s\"", idPad, param.GetHelp())
	}

	// Add outname string if it exists.
	if len(param.GetOutName()) > 0 {
		if param.GetHelp() == "" {
			printer.Printf("%s  ", idPad)
		}
		printer.Printf("%s  \"%s\"", helpPad, param.GetOutName())
	}
	printer.WriteString(",\n")
}

type Params interface {
	getWidths() (int, int, int, int)
}

func (self *InParams) getWidths() (int, int, int, int) {
	modeWidth := 0
	typeWidth := 0
	idWidth := 0
	helpWidth := 0
	for _, param := range self.List {
		modeWidth = max(modeWidth, len(param.getMode()))
		typeWidth = max(typeWidth, len(param.GetTname())+2*param.GetArrayDim())
		if len(param.GetId()) < 35 {
			idWidth = max(idWidth, len(param.GetId()))
		}
		if len(param.GetHelp()) < 25 {
			helpWidth = max(helpWidth, len(param.GetHelp()))
		}
	}
	return modeWidth, typeWidth, idWidth, helpWidth
}

func (self *OutParams) getWidths() (int, int, int, int) {
	modeWidth := 0
	typeWidth := 0
	idWidth := 0
	helpWidth := 0
	for _, param := range self.List {
		modeWidth = max(modeWidth, len(param.getMode()))
		typeWidth = max(typeWidth, len(param.GetTname())+2*param.GetArrayDim())
		if len(param.GetId()) < 35 {
			idWidth = max(idWidth, len(param.GetId()))
		}
		if len(param.GetHelp()) < 25 {
			helpWidth = max(helpWidth, len(param.GetHelp()))
		}
	}
	return modeWidth, typeWidth, idWidth, helpWidth
}

func measureParamsWidths(paramsList ...Params) (int, int, int, int) {
	modeWidth := 0
	typeWidth := 0
	idWidth := 0
	helpWidth := 0
	for _, params := range paramsList {
		mw, tw, iw, hw := params.getWidths()
		modeWidth = max(modeWidth, mw)
		typeWidth = max(typeWidth, tw)
		idWidth = max(idWidth, iw)
		helpWidth = max(helpWidth, hw)
	}
	return modeWidth, typeWidth, idWidth, helpWidth
}

func (self *InParams) format(printer *printer, modeWidth int, typeWidth int, idWidth int, helpWidth int) {
	for _, param := range self.List {
		paramFormat(printer, param, modeWidth, typeWidth, idWidth, helpWidth)
	}
}

func (self *OutParams) format(printer *printer, modeWidth int, typeWidth int, idWidth int, helpWidth int) {
	for _, param := range self.List {
		paramFormat(printer, param, modeWidth, typeWidth, idWidth, helpWidth)
	}
}

//
// Pipeline, Call, Return
//
func (self *Pipeline) format(printer *printer) {
	printer.printComments(&self.Node, "")

	modeWidth, typeWidth, idWidth, helpWidth := measureParamsWidths(
		self.InParams, self.OutParams,
	)

	printer.Printf("pipeline %s(\n", self.Id)
	self.InParams.format(printer, modeWidth, typeWidth, idWidth, helpWidth)
	self.OutParams.format(printer, modeWidth, typeWidth, idWidth, helpWidth)
	printer.WriteString(")\n{")
	self.topoSort()
	for _, callstm := range self.Calls {
		printer.WriteString(NEWLINE)
		callstm.format(printer, INDENT)
	}
	printer.WriteString(NEWLINE)
	self.Ret.format(printer)
	if self.Retain != nil {
		printer.WriteString(NEWLINE)
		self.Retain.format(printer)
	}
	printer.WriteString("}\n")
}

func (self *CallStm) format(printer *printer, prefix string) {
	printer.printComments(&self.Node, prefix)
	printer.WriteString(prefix)
	printer.WriteString("call ")
	printer.WriteString(self.DecId)
	if self.Id != self.DecId {
		printer.WriteString(" as ")
		printer.WriteString(self.Id)
	}
	printer.WriteString("(\n")
	self.Bindings.format(printer, prefix)
	printer.WriteString(prefix)

	if self.Modifiers.Bindings != nil && len(self.Modifiers.Bindings.List) > 0 ||
		self.Modifiers.Local || self.Modifiers.Preflight || self.Modifiers.Volatile {
		if self.Modifiers.Bindings == nil {
			self.Modifiers.Bindings = &BindStms{
				Node: self.Node,
			}
		}
		printer.WriteString(") using (\n")
		// Convert unbound-form mods to bound form.
		// Because we remove elements from the binding table if they're
		// static, we can't just use the table to see if they're needed.
		var foundMods Modifiers
		for _, binding := range self.Modifiers.Bindings.List {
			switch binding.Id {
			case local:
				foundMods.Local = true
			case preflight:
				foundMods.Preflight = true
			case volatile:
				foundMods.Volatile = true
			}
		}
		if self.Modifiers.Local && !foundMods.Local {
			self.Modifiers.Bindings.List = append(self.Modifiers.Bindings.List,
				&BindStm{
					Node: self.Modifiers.Bindings.Node,
					Id:   "local",
					Exp:  &ValExp{self.Modifiers.Bindings.Node, KindBool, true},
				})
		}
		if self.Modifiers.Preflight && !foundMods.Preflight {
			self.Modifiers.Bindings.List = append(self.Modifiers.Bindings.List,
				&BindStm{
					Node: self.Modifiers.Bindings.Node,
					Id:   "preflight",
					Exp:  &ValExp{self.Modifiers.Bindings.Node, KindBool, true},
				})
		}
		if self.Modifiers.Volatile && !foundMods.Volatile {
			self.Modifiers.Bindings.List = append(self.Modifiers.Bindings.List,
				&BindStm{
					Node: self.Modifiers.Bindings.Node,
					Id:   "volatile",
					Exp:  &ValExp{self.Modifiers.Bindings.Node, KindBool, true},
				})
		}
		sort.Slice(self.Modifiers.Bindings.List, func(i, j int) bool {
			return self.Modifiers.Bindings.List[i].Id < self.Modifiers.Bindings.List[j].Id
		})
		self.Modifiers.Bindings.format(printer, prefix)
		printer.WriteString(prefix)
	}
	printer.WriteString(")\n")
}

func (self *ReturnStm) format(printer *printer) {
	printer.printComments(&self.Node, INDENT)
	printer.WriteString(INDENT)
	printer.WriteString("return (\n")
	self.Bindings.format(printer, INDENT)
	printer.WriteString(INDENT)
	printer.WriteString(")\n")
}

func (self *PipelineRetains) format(printer *printer) {
	printer.printComments(&self.Node, INDENT)
	printer.WriteString(INDENT)
	printer.WriteString("retain (\n")
	for _, ref := range self.Refs {
		printer.WriteString(INDENT)
		printer.WriteString(INDENT)
		ref.format(printer, INDENT+INDENT)
		printer.WriteString(",\n")
	}
	printer.WriteString(INDENT)
	printer.WriteString(")\n")
}

//
// Stage
//
func (self *Stage) format(printer *printer) {
	printer.printComments(&self.Node, "")

	modeWidth, typeWidth, idWidth, helpWidth := measureParamsWidths(
		self.InParams, self.OutParams, self.ChunkIns, self.ChunkOuts,
	)
	modeWidth = max(modeWidth, len("src"))

	printer.Printf("stage %s(\n", self.Id)
	self.InParams.format(printer, modeWidth, typeWidth, idWidth, helpWidth)
	self.OutParams.format(printer, modeWidth, typeWidth, idWidth, helpWidth)
	self.Src.format(printer, modeWidth, typeWidth, idWidth)
	if idWidth > 30 || helpWidth > 20 {
		_, _, idWidth, helpWidth = measureParamsWidths(
			self.ChunkIns, self.ChunkOuts)
	}
	if self.Split {
		printer.WriteString(") split (\n")
		self.ChunkIns.format(printer, modeWidth, typeWidth, idWidth, helpWidth)
		self.ChunkOuts.format(printer, modeWidth, typeWidth, idWidth, helpWidth)
	}
	if self.Resources != nil {
		self.Resources.format(printer)
	}
	if self.Retain != nil {
		self.Retain.format(printer)
	}
	printer.WriteString(")\n")
}

func (self *Resources) format(printer *printer) {
	printer.printComments(&self.Node, INDENT)
	printer.WriteString(") using (\n")
	// Pad depending on which arguments are present.
	// mem_gb   = x,
	// special  = y
	// threads  = y,
	// volatile = z,
	var memPad, threadPad string
	if self.VolatileNode != nil {
		memPad = "  "
		threadPad = " "
	} else if self.SpecialNode != nil || self.ThreadNode != nil {
		memPad = " "
	}
	if self.MemNode != nil {
		printer.printComments(self.MemNode, INDENT)
		printer.WriteString(INDENT)
		printer.Printf("mem_gb%s = %d,\n", memPad, self.MemGB)
	}
	if self.SpecialNode != nil {
		printer.printComments(self.SpecialNode, INDENT)
		printer.WriteString(INDENT)
		printer.Printf("special%s = \"%s\",\n", threadPad, self.Special)
	}
	if self.ThreadNode != nil {
		printer.printComments(self.ThreadNode, INDENT)
		printer.WriteString(INDENT)
		printer.Printf("threads%s = %d,\n", threadPad, self.Threads)
	}
	if self.VolatileNode != nil {
		printer.printComments(self.VolatileNode, INDENT)
		printer.WriteString(INDENT)
		printer.WriteString("volatile = strict,\n")
	}
}

func (self *RetainParams) format(printer *printer) {
	printer.printComments(&self.Node, INDENT)
	printer.WriteString(") retain (\n")
	for _, param := range self.Params {
		printer.printComments(&param.Node, INDENT)
		printer.WriteString(INDENT)
		printer.WriteString(param.Id)
		printer.WriteString(",\n")
	}
}

func (self *SrcParam) format(printer *printer, modeWidth int, typeWidth int, idWidth int) {
	printer.printComments(&self.Node, INDENT)
	langPad := strings.Repeat(" ", typeWidth-len(string(self.Lang)))
	modePad := strings.Repeat(" ", modeWidth-len("src"))
	printer.Printf("%ssrc%s %v%s \"%s\",\n", INDENT,
		modePad, self.Lang, langPad,
		strings.Join(append([]string{self.Path}, self.Args...), " "))
}

//
// Callable
//
func (self *Callables) format(printer *printer) {
	for i, callable := range self.List {
		if i != 0 {
			printer.WriteString(NEWLINE)
		}
		callable.format(printer)
	}
}

//
// Filetype
//
func (self *UserType) format(printer *printer) {
	printer.printComments(&self.Node, "")
	printer.Printf("filetype %s;\n", self.Id)
}

//
// AST
//
func (self *Ast) format(writeIncludes bool) string {
	needSpacer := false
	printer := printer{
		comments: make(map[string][]*commentBlock, len(self.Files)),
	}
	if len(self.Files) > 0 {
		// Set the printer's last comment location to the top of the
		// top-level file, so that the top-level include is reported
		// correctly.
		var topFile *SourceFile
		for _, f := range self.Files {
			topFile = f
			break
		}
		for topFile != nil && len(topFile.IncludedFrom) > 0 {
			topFile = topFile.IncludedFrom[0].File
		}
		printer.lastComment = SourceLoc{
			Line: 0,
			File: topFile,
		}
	}

	for _, comment := range self.comments {
		printer.comments[comment.Loc.File.FullPath] = append(
			printer.comments[comment.Loc.File.FullPath],
			comment)
	}
	if writeIncludes {
		for _, directive := range self.Includes {
			printer.printComments(&directive.Node, "")
			printer.WriteString("@include \"")
			printer.WriteString(directive.Value)
			printer.WriteRune('"')
			printer.WriteString(NEWLINE)
			needSpacer = true
		}
	}

	// filetype declarations.
	if needSpacer && len(self.UserTypes) > 0 {
		printer.WriteString(NEWLINE)
	}
	for _, filetype := range self.UserTypes {
		filetype.format(&printer)
		needSpacer = true
	}

	// callables.
	if needSpacer && len(self.Callables.List) > 0 {
		printer.WriteString(NEWLINE)
	}
	self.Callables.format(&printer)

	// call.
	if self.Call != nil {
		if len(self.Callables.List) > 0 || needSpacer {
			printer.WriteString(NEWLINE)
		}
		self.Call.format(&printer, "")
	}

	// Any comments which went at the ends of a file, after any nodes.
	printer.DumpComments()
	return printer.String()
}

//
// Exported API
//

func FormatFile(filename string, fixIncludes bool, mropath []string) (string, error) {
	var parser Parser
	return parser.FormatFile(filename, fixIncludes, mropath)
}

func (parser *Parser) FormatFile(filename string, fixIncludes bool, mropath []string) (string, error) {
	// Read MRO source file.
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return FormatSrcBytes(data, filename, fixIncludes, mropath)
}

func Format(src string, filename string, fixIncludes bool, mropath []string) (string, error) {
	return FormatSrcBytes([]byte(src), filename, fixIncludes, mropath)
}

func FormatSrcBytes(src []byte, filename string, fixIncludes bool, mropath []string) (string, error) {
	var parser Parser
	return parser.FormatSrcBytes(src, filename, fixIncludes, mropath)
}

func (parser *Parser) FormatSrcBytes(src []byte, filename string, fixIncludes bool, mropath []string) (string, error) {
	absPath, _ := filepath.Abs(filename)
	// Parse and generate the AST.
	srcFile := SourceFile{
		FileName: filename,
		FullPath: absPath,
	}
	global, mmli := yaccParse(src, &srcFile, parser.getIntern())
	if mmli != nil { // mmli is an mmLexInfo struct
		return "", mmli
	}
	var err error
	if fixIncludes {
		err = fixIncludesTop(global, mropath, parser.getIntern())
	}

	// Format the source.
	return global.format(true), err
}

func JsonDumpAsts(asts []*Ast) string {
	type JsonDump struct {
		UserTypes map[string]*UserType
		Stages    map[string]*Stage
		Pipelines map[string]*Pipeline
	}

	jd := JsonDump{
		UserTypes: map[string]*UserType{},
		Stages:    map[string]*Stage{},
		Pipelines: map[string]*Pipeline{},
	}

	for _, ast := range asts {
		for _, t := range ast.UserTypes {
			jd.UserTypes[t.Id] = t
		}
		for _, stage := range ast.Stages {
			jd.Stages[stage.Id] = stage
		}
		for _, pipeline := range ast.Pipelines {
			jd.Pipelines[pipeline.Id] = pipeline
		}
	}
	if jsonBytes, err := json.MarshalIndent(jd, "", "    "); err == nil {
		return string(jsonBytes)
	} else {
		return fmt.Sprintf("{ error: \"%s\" }", err.Error())
	}
}
