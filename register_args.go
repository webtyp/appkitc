package appkitc

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

const (
	// registerFuncName is the entry point every config/routes.go exports.
	registerFuncName = "Register"
	// routerArgExpr is the first argument passed to config.Register: the
	// running server's router.
	routerArgExpr = "s.Router()"
	// dependencyArg is what the generated call passes for every parameter
	// after the router.
	dependencyArg = "nil"
)

// detectRegisterArgs renders the argument list for the routes.Register call in
// the generated main. routescan reports the routes a project declares but not
// the signature of Register itself, so its arity is read here from the AST:
// the router is always the first argument, and every dependency parameter after
// it becomes a nil placeholder the developer fills in. Any parse failure falls
// back to the single-argument form rather than aborting generation.
func detectRegisterArgs(routesFile string) string {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, routesFile, nil, 0)
	if err != nil {
		return routerArgExpr
	}

	var targetFunc *ast.FuncDecl
	for _, decl := range node.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Type == nil || fn.Type.Params == nil {
			continue
		}
		if fn.Name.Name == registerFuncName {
			targetFunc = fn
			break
		}
		if targetFunc == nil && len(fn.Type.Params.List) > 0 {
			if isRouterParam(fn.Type.Params.List[0].Type) {
				targetFunc = fn
			}
		}
	}

	if targetFunc == nil {
		return routerArgExpr
	}

	paramCount := 0
	for _, field := range targetFunc.Type.Params.List {
		if len(field.Names) == 0 {
			paramCount++
		} else {
			paramCount += len(field.Names)
		}
	}

	if paramCount <= 1 {
		return routerArgExpr
	}

	args := make([]string, paramCount)
	args[0] = routerArgExpr
	for i := 1; i < paramCount; i++ {
		args[i] = dependencyArg
	}
	return strings.Join(args, ", ")
}

func isRouterParam(expr ast.Expr) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	return (ident.Name == "router" || ident.Name == "Router") && sel.Sel.Name == "Router"
}
