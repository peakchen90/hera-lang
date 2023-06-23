package compiler

import (
	"github.com/peakchen90/noah-lang/internal/ast"
	"github.com/peakchen90/noah-lang/internal/helper"
	"path/filepath"
	"strings"
)

func (m *Module) preCompile() {
	stack := m.scopes
	stack.push()

	for _, stmt := range m.Ast.Body {
		switch stmt.Node.(type) {
		case *ast.ImportDecl:
			m.compileImportDecl(stmt.Node.(*ast.ImportDecl))
		case *ast.FuncDecl:
			m.compileFuncDecl(stmt.Node.(*ast.FuncDecl), nil, true)
		case *ast.ImplDecl:
			m.compileImplDecl(stmt.Node.(*ast.ImplDecl), true)
		case *ast.VarDecl:
			m.compileVarDecl(stmt.Node.(*ast.VarDecl), true)
		case *ast.TAliasDecl:
			m.compileTAliasDecl(stmt.Node.(*ast.TAliasDecl))
		case *ast.TInterfaceDecl:
			m.compileTInterfaceDecl(stmt.Node.(*ast.TInterfaceDecl))
		case *ast.TStructDecl:
			m.compileTStructDecl(stmt.Node.(*ast.TStructDecl))
		case *ast.TEnumDecl:
			m.compileTEnumDecl(stmt.Node.(*ast.TEnumDecl))
		}
	}
}

func (m *Module) compileFile() {
	for _, stmt := range m.Ast.Body {
		m.compileStmt(stmt)
	}
}

func (m *Module) compileStmt(stmt *ast.Stmt) {
	switch (stmt.Node).(type) {
	case *ast.ImportDecl:
	case *ast.FuncDecl:
	case *ast.ImplDecl:
	case *ast.VarDecl:
	case *ast.BlockStmt:
	case *ast.ReturnStmt:
	case *ast.ExprStmt:
	case *ast.IfStmt:
	case *ast.ForStmt:
	case *ast.BreakStmt:
	case *ast.ContinueStmt:
	}
}

func (m *Module) compileImportDecl(node *ast.ImportDecl) {
	paths := node.Paths
	local := node.Local
	packageName := m.packageName

	if node.Package != nil {
		packageName = node.Package.Name
	}

	if local == nil {
		local = paths[len(paths)-1]
	}

	pathIdBuilder := strings.Builder{}
	for i, item := range paths {
		pathIdBuilder.WriteString(item.Name)
		if i < len(paths)-1 {
			pathIdBuilder.WriteByte('/')
		}
	}

	pathId := pathIdBuilder.String()
	moduleId := packageName + ":" + pathId
	module := m.compiler.Modules.get(moduleId)

	if module == nil {
		modulePath := ""

		if len(packageName) == 0 {
			modulePath = filepath.Join(m.compiler.VirtualFS.Root, pathId+".noah")
		} else {
			// TODO resolve module
		}

		code, err := m.compiler.VirtualFS.ReadFile(modulePath)
		if err != nil {
			panic(err)
		}

		module = NewModule(m.compiler, string(code), packageName, moduleId)
		m.compiler.Modules.add(module)
	}

	value := &ModuleValue{
		Name:   local.Name,
		Module: module,
	}
	m.putValue(local, value, true)

	module.compile()
}

func (m *Module) compileFuncDecl(node *ast.FuncDecl, target Kind, isPrecompile bool) *FuncValue {
	name := node.Name.Name
	var value *FuncValue

	if isPrecompile {
		value = &FuncValue{
			Name: name,
			Kind: m.compileKindExpr(node.Kind),
		}

		if target != nil {
			if target.getImpl().hasFunc(name) {
				m.unexpectedPos(node.Name.Start, "Duplicate key: "+name)
			}
			target.getImpl().addFunc(value)
		} else {
			m.putValue(node.Name, value, true)
			if node.Pub {
				m.exports.setValue(name, value)
			}
		}
	} else {
		if target != nil {
			value = target.getImpl().getFunc(name)
		} else {
			value = m.findValue(node.Name, true).(*FuncValue)
		}

		// TODO ptr
	}

	return value
}

func (m *Module) compileImplDecl(node *ast.ImplDecl, isPrecompile bool) {
	target := m.compileKindExpr(node.Target)
	m.scopes.push()
	m.putSelfKind(target)

	switch target.(type) {
	case *TInterface:
		m.unexpectedPos(node.Target.Start, "Cannot implements for interface type")
	case *TAny:
		m.unexpectedPos(node.Target.Start, "Cannot implements for any type")
	case *TSelf:
		m.unexpectedPos(node.Target.Start, "Cannot implements for self type")
	}

	implValues := make(map[string]*FuncValue)
	implDecls := make(map[string]*ast.Stmt)
	for _, stmt := range node.Body {
		val := m.compileFuncDecl(stmt.Node.(*ast.FuncDecl), target, isPrecompile)
		if isPrecompile {
			implValues[val.Name] = val
			implDecls[val.Name] = stmt
		}
	}

	if isPrecompile {
		if node.Interface != nil {
			t, ok := m.compileKindExpr(node.Interface).(*TInterface)
			if ok {
				t.Refers = append(t.Refers, target)
				for key, kind := range t.Properties {
					if implValues[key] == nil {
						m.unexpectedPos(node.Target.Start, "No implement method: "+key)
					}
					if !compareKind(kind, implValues[key].Kind, true) {
						// TODO panic
						m.unexpectedPos(implDecls[key].Start, "Unable to match interface method signature: "+key)
					}
				}
			} else {
				if t == nil {
					m.unexpectedPos(node.Interface.Start, "Cannot found: "+getKindExprId(node.Interface))
				}
				m.unexpectedPos(node.Interface.Start, "Expect be an interface type")
			}
		}
	}

	m.scopes.pop()
}

func (m *Module) compileVarDecl(node *ast.VarDecl, isPrecompile bool) {
	name := node.Id.Name

	if isPrecompile {
		scope := &VarValue{
			Name:  name,
			Kind:  m.compileKindExpr(node.Kind),
			Const: node.Const,
		}
		m.putValue(node.Id, scope, true)
		if node.Pub {
			m.exports.setValue(name, scope)
		}
	} else {
		// TODO assignment
		// TODO infer kind
	}
}

func (m *Module) compileTAliasDecl(node *ast.TAliasDecl) {
	kind := &TCustom{
		Kind: m.compileKindExpr(node.Kind),
		Impl: newImpl(),
	}

	m.putKind(node.Name, kind, true)
	if node.Pub {
		m.exports.setKind(node.Name.Name, kind)
	}
}

func (m *Module) compileTInterfaceDecl(node *ast.TInterfaceDecl) {
	kind := &TInterface{
		Properties: make(map[string]Kind),
		Refers:     make([]Kind, 0, helper.DefaultCap),
	}

	m.scopes.push()
	m.putSelfKind(kind)
	for _, pair := range node.Properties {
		key := pair.Key.Name
		_, has := kind.Properties[key]
		if has {
			m.unexpectedPos(pair.Key.Start, "Duplicate key: "+key)
		} else if key[0] == '_' {
			m.unexpectedPos(pair.Key.Start, "Should not be private method: "+key)
		}
		kind.Properties[key] = m.compileKindExpr(pair.Kind)
	}
	m.scopes.pop()

	m.putKind(node.Name, kind, true)
	if node.Pub {
		m.exports.setKind(node.Name.Name, kind)
	}
}

func (m *Module) compileTStructDecl(node *ast.TStructDecl) {
	kind := m.compileStructKind(node.Kind.Node.(*ast.TStructKind))

	m.putKind(node.Name, kind, true)
	if node.Pub {
		m.exports.setKind(node.Name.Name, kind)
	}
}

func (m *Module) compileTEnumDecl(node *ast.TEnumDecl) {
	choices := make(map[string]int)

	for i, item := range node.Choices {
		name := item.Name
		_, has := choices[name]
		if has {
			// TODO
			m.unexpectedPos(item.Start, "Duplicate item: "+name)
		}
		choices[name] = i
	}

	kind := &TEnum{
		Choices: choices,
	}

	m.putKind(node.Name, kind, true)
	if node.Pub {
		m.exports.setKind(node.Name.Name, kind)
	}
}
