// Package immutable provides a go/analysis analyzer that enforces the
// "read freely, mutate through methods" rule stated in docs/code-style.md: the
// state of a domain type changes only through its own methods, which return a
// modified copy, so no state transition can be bypassed with a field assignment
// from the outside.
//
// The analyzer allows two kinds of field assignment and rejects everything else:
//
//   - Inside a method of the type itself: that is where state transitions and
//     their invariants live.
//   - On a value the enclosing function owns — a local variable or a
//     by-value parameter, reached without going through a pointer, an index or a
//     package-level variable. Those assignments mutate a copy nobody else can
//     observe, which is how an adapter or a use case builds ("hydrates") a domain
//     object from its source data before returning it.
//
// What is left is precisely the observable mutation of shared state: assigning
// through a pointer or a pointer receiver, into a slice or map element, or into a
// package-level variable. Those are the state transitions that must be expressed
// as a method returning a modified copy.
package immutable

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// ProtectedTypes lists the domain types guarded by the analyzer, by fully
// qualified name (import path + "." + type name). Add a type here as soon as it
// becomes a domain value type with state-transition methods.
var ProtectedTypes = []string{
	"gitbot/internal/app.Application",
	"gitbot/internal/event.Event",
	"gitbot/internal/event.PullRequest",
	"gitbot/internal/event.EventResponse",
}

// Analyzer is the analyzer configured with ProtectedTypes. It is the one run by
// the repository-wide test; use NewAnalyzer to protect a different set of types.
var Analyzer = NewAnalyzer(ProtectedTypes...)

// NewAnalyzer builds an analyzer that reports assignments to fields of the given
// types (fully qualified names, e.g. "gitbot/internal/app.Application") when they
// mutate state the enclosing function does not own. See the package comment for
// the exact rule.
func NewAnalyzer(protected ...string) *analysis.Analyzer {
	set := make(map[string]bool, len(protected))
	for _, name := range protected {
		set[name] = true
	}
	return &analysis.Analyzer{
		Name: "immutable",
		Doc:  "reports assignments to fields of domain types made outside their own methods",
		Run: func(pass *analysis.Pass) (any, error) {
			return run(pass, set)
		},
	}
}

func run(pass *analysis.Pass, protected map[string]bool) (any, error) {
	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			// The receiver type of the enclosing method is what makes an assignment
			// legal. Declarations that are not methods (plain functions, or a
			// package-level var initialised with a function literal) have no
			// receiver and are always checked.
			checkDecl(pass, decl, receiverTypeName(pass, decl), protected)
		}
	}
	return nil, nil
}

// checkDecl walks a declaration reporting every assignment to a field of a
// protected type that is neither made from a method of that type nor made on a
// value owned by the enclosing function.
func checkDecl(pass *analysis.Pass, decl ast.Decl, receiver string, protected map[string]bool) {
	ast.Inspect(decl, func(node ast.Node) bool {
		// Both assignments (x.F = v, x.F += v) and increments (x.F++) mutate the
		// selected field.
		var targets []ast.Expr
		switch stmt := node.(type) {
		case *ast.AssignStmt:
			targets = stmt.Lhs
		case *ast.IncDecStmt:
			targets = []ast.Expr{stmt.X}
		default:
			return true
		}
		for _, target := range targets {
			selector, ok := target.(*ast.SelectorExpr)
			if !ok {
				continue
			}
			owner, field, base, ok := protectedOwner(pass, selector, protected)
			if !ok || owner == receiver || isOwnedValue(pass, base) {
				continue
			}
			pass.Reportf(target.Pos(),
				"field %s of %s is mutated outside its own methods: %s changes only through its methods, which return a modified copy",
				field, owner, owner)
		}
		return true
	})
}

// protectedOwner walks the selector chain of an assignment target from the
// assigned field inwards, looking for the innermost protected type that owns it.
// It returns that type name, the name of its field on the path (which is the
// assigned field itself, or the intermediate field that contains it), and the
// expression denoting the protected value. ok is false when no protected type is
// involved.
func protectedOwner(pass *analysis.Pass, selector *ast.SelectorExpr, protected map[string]bool) (owner string, field string, base ast.Expr, ok bool) {
	field = selector.Sel.Name
	expr := selector.X
	for {
		if name := namedTypeName(pass.TypesInfo.TypeOf(expr)); protected[name] {
			return name, field, expr, true
		}
		// Keep walking outwards only while the path is made of field selections:
		// an assignment to app.Nested.Label is still a mutation of app.
		inner, isSelector := unparen(expr).(*ast.SelectorExpr)
		if !isSelector {
			return "", "", nil, false
		}
		field = inner.Sel.Name
		expr = inner.X
	}
}

// unparen strips the redundant parentheses around an expression.
func unparen(expr ast.Expr) ast.Expr {
	for {
		paren, ok := expr.(*ast.ParenExpr)
		if !ok {
			return expr
		}
		expr = paren.X
	}
}

// isOwnedValue reports whether expr denotes a value the enclosing function owns,
// so that assigning one of its fields cannot be observed by anyone else: a local
// variable or a by-value parameter (receiver included), reached through field
// selections only. Any pointer, index, dereference, function result or
// package-level variable on the path means the value is shared, and the answer is
// false.
func isOwnedValue(pass *analysis.Pass, expr ast.Expr) bool {
	// A pointer never denotes an owned copy: assigning through it reaches the
	// value the pointer refers to.
	if isPointer(pass.TypesInfo.TypeOf(expr)) {
		return false
	}
	for {
		switch node := expr.(type) {
		case *ast.ParenExpr:
			expr = node.X
		case *ast.SelectorExpr:
			// Selecting a field through a pointer reaches shared memory; so does a
			// qualified identifier, which is a package-level variable.
			if isPointer(pass.TypesInfo.TypeOf(node.X)) || isPackageName(pass, node.X) {
				return false
			}
			expr = node.X
		case *ast.Ident:
			variable, ok := pass.TypesInfo.Uses[node].(*types.Var)
			if !ok || variable.Pkg() == nil {
				return false
			}
			// A package-level variable is shared, and so is whatever a local pointer
			// variable points to.
			if variable.Parent() == variable.Pkg().Scope() || isPointer(variable.Type()) {
				return false
			}
			return true
		default:
			// Index expressions, dereferences, type assertions, call results...
			return false
		}
	}
}

// isPointer reports whether typ is a pointer type.
func isPointer(typ types.Type) bool {
	if typ == nil {
		return false
	}
	_, ok := types.Unalias(typ).(*types.Pointer)
	return ok
}

// isPackageName reports whether expr is the package qualifier of a qualified
// identifier (the "pkg" in pkg.Var).
func isPackageName(pass *analysis.Pass, expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	if !ok {
		return false
	}
	_, ok = pass.TypesInfo.Uses[ident].(*types.PkgName)
	return ok
}

// receiverTypeName returns the fully qualified name of the receiver type of decl
// when it is a method, or "" when decl is not a method.
func receiverTypeName(pass *analysis.Pass, decl ast.Decl) string {
	fn, ok := decl.(*ast.FuncDecl)
	if !ok || fn.Recv == nil || len(fn.Recv.List) == 0 {
		return ""
	}
	return namedTypeName(pass.TypesInfo.TypeOf(fn.Recv.List[0].Type))
}

// namedTypeName returns the fully qualified name (import path + "." + type name)
// of a named type, dereferencing pointers and resolving aliases. It returns ""
// for anything else (unnamed structs, maps, builtin types...).
func namedTypeName(typ types.Type) string {
	if typ == nil {
		return ""
	}
	if pointer, ok := types.Unalias(typ).(*types.Pointer); ok {
		typ = pointer.Elem()
	}
	named, ok := types.Unalias(typ).(*types.Named)
	if !ok || named.Obj().Pkg() == nil {
		return ""
	}
	return named.Obj().Pkg().Path() + "." + named.Obj().Name()
}
