package blueprint

import (
	"fmt"
	"path/filepath"
	"text/scanner"
)

type Module interface {
	GenerateBuildActions(ModuleContext)
}

type ModuleContext interface {
	ModuleName() string
	OtherModuleName(m Module) string
	ModuleDir() string
	Config() interface{}

	Errorf(pos scanner.Position, fmt string, args ...interface{})
	ModuleErrorf(fmt string, args ...interface{})
	PropertyErrorf(property, fmt string, args ...interface{})
	OtherModuleErrorf(m Module, fmt string, args ...interface{})
	Failed() bool

	Variable(name, value string)
	Rule(name string, params RuleParams, argNames ...string) Rule
	Build(params BuildParams)

	VisitDepsDepthFirst(visit func(Module))
	VisitDepsDepthFirstIf(pred func(Module) bool, visit func(Module))

	AddNinjaFileDeps(deps ...string)
}

var _ ModuleContext = (*moduleContext)(nil)

type moduleContext struct {
	context *Context
	config  interface{}
	module  Module
	scope   *localScope
	info    *moduleInfo

	ninjaFileDeps []string
	errs          []error

	actionDefs localBuildActions
}

func (m *moduleContext) ModuleName() string {
	return m.info.properties.Name
}

func (m *moduleContext) OtherModuleName(module Module) string {
	info := m.context.moduleInfo[module]
	return info.properties.Name
}

func (m *moduleContext) ModuleDir() string {
	return filepath.Dir(m.info.relBlueprintsFile)
}

func (m *moduleContext) Config() interface{} {
	return m.config
}

func (m *moduleContext) Errorf(pos scanner.Position, format string,
	args ...interface{}) {

	m.errs = append(m.errs, &Error{
		Err: fmt.Errorf(format, args...),
		Pos: pos,
	})
}

func (m *moduleContext) ModuleErrorf(format string, args ...interface{}) {
	m.errs = append(m.errs, &Error{
		Err: fmt.Errorf(format, args...),
		Pos: m.info.pos,
	})
}

func (m *moduleContext) PropertyErrorf(property, format string,
	args ...interface{}) {

	pos, ok := m.info.propertyPos[property]
	if !ok {
		panic(fmt.Errorf("property %q was not set for this module", property))
	}

	m.errs = append(m.errs, &Error{
		Err: fmt.Errorf(format, args...),
		Pos: pos,
	})
}

func (m *moduleContext) OtherModuleErrorf(module Module, format string,
	args ...interface{}) {

	info := m.context.moduleInfo[module]
	m.errs = append(m.errs, &Error{
		Err: fmt.Errorf(format, args...),
		Pos: info.pos,
	})
}

func (m *moduleContext) Failed() bool {
	return len(m.errs) > 0
}

func (m *moduleContext) Variable(name, value string) {
	const skip = 2
	m.scope.ReparentToCallerPackage(skip)

	v, err := m.scope.AddLocalVariable(name, value)
	if err != nil {
		panic(err)
	}

	m.actionDefs.variables = append(m.actionDefs.variables, v)
}

func (m *moduleContext) Rule(name string, params RuleParams,
	argNames ...string) Rule {

	const skip = 2
	m.scope.ReparentToCallerPackage(skip)

	r, err := m.scope.AddLocalRule(name, &params, argNames...)
	if err != nil {
		panic(err)
	}

	m.actionDefs.rules = append(m.actionDefs.rules, r)

	return r
}

func (m *moduleContext) Build(params BuildParams) {
	const skip = 2
	m.scope.ReparentToCallerPackage(skip)

	def, err := parseBuildParams(m.scope, &params)
	if err != nil {
		panic(err)
	}

	m.actionDefs.buildDefs = append(m.actionDefs.buildDefs, def)
}

func (m *moduleContext) VisitDepsDepthFirst(visit func(Module)) {
	m.context.visitDepsDepthFirst(m.module, visit)
}

func (m *moduleContext) VisitDepsDepthFirstIf(pred func(Module) bool,
	visit func(Module)) {

	m.context.visitDepsDepthFirstIf(m.module, pred, visit)
}

func (m *moduleContext) AddNinjaFileDeps(deps ...string) {
	m.ninjaFileDeps = append(m.ninjaFileDeps, deps...)
}