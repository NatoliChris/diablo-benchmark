package core


import (
	"fmt"
)


type scope interface {
	get(string) (Variable, string, bool)
}


type basicScope struct {
	parent   scope
	entries  map[string]*basicScopeEntry
}

type basicScopeEntry struct {
	variable  Variable
	domain    string
}

func (this *basicScope) init(parent scope) {
	this.parent = parent
	this.entries = make(map[string]*basicScopeEntry, 0)
}

func (this *basicScope) parse(expr BenchmarkExpression) error {
	var positions map[string]string = make(map[string]string, 0)
	var child BenchmarkExpression
	var name, domain string
	var variable Variable
	var position string
	var old scope
	var err error
	var ok bool

	old = expr.local()
	expr.specialize(this)
	defer expr.specialize(old)

	for _, child = range expr.Slice() {
		Tracef("parse scope variable: %s", child.Position())

		variable, name, domain, err = parseVariable(child)
		if err != nil {
			return err
		}

		position, ok = positions[name]
		if ok {
			return fmt.Errorf("%s: variable '%s' redefined " +
				"(previously at %s)", child.FullPosition(),
				name, position)
		}

		this.add(name, domain, variable)

		positions[name] = child.Position()
	}

	return nil
}

func (this *basicScope) add(name, domain string, variable Variable) {
	this.entries[name] = &basicScopeEntry{
		variable,
		domain,
	}
}

func (this *basicScope) get(name string) (Variable, string, bool) {
	var entry *basicScopeEntry
	var ok bool

	entry, ok = this.entries[name]
	if !ok {
		if this.parent == nil {
			return nil, "", false
		} else {
			return this.parent.get(name)
		}
	}

	return entry.variable, entry.domain, true
}
