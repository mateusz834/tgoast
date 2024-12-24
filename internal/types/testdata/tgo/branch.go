package test

import "github.com/mateusz834/tgo"

func _(tgo.Ctx) error {
	for range 5 {
		<div>
			break // ERROR "break escapes end tag"
		</div>
	}

	for range 5 {
		<div>
			continue // ERROR "continue escapes end tag"
		</div>
	}

	switch "a" {
	case "a":
		<div>
			break // ERROR "break escapes end tag"
		</div>
	default:
		<div>
			if true {
				break // ERROR "break escapes end tag"
			}
		</div>
	}

	for {
		<div>
			switch "a" {
			case "a":
				<div>
					continue // ERROR "continue escapes end tag"
					break // ERROR "break escapes end tag"
				</div>
				continue // ERROR "continue escapes end tag"
			}
			continue // ERROR "continue escapes end tag"
		</div>
	}

	return nil
}

func _(tgo.Ctx) error {
	<div>
	outer:
		for {
			for {
				continue outer
				break outer
			}
		}
	</div>
	return nil
}

func _(tgo.Ctx) error {
outer:
	for {
		<div>
			for {
				continue outer // ERROR "invalid continue label outer exits body tag"
				break outer // ERROR "invalid break label outer exits body tag"
			}
		</div>
	}
	return nil // TODO: panics when removed
}

func _(tgo.Ctx) error {
outer:
	for {
		<div>
			for {
				if true {
					continue outer // ERROR "invalid continue label outer exits body tag"
					break outer // ERROR "invalid break label outer exits body tag"
				}
				<div>
					break outer // ERROR "invalid break label outer exits body tag"
				</div>
			}
		</div>
	}
	return nil
}

func _(tgo.Ctx) error {
	<div>
		for {
			continue
			break
		}
	</div>
	return nil
}

func _(tgo.Ctx) error {
	for {
		<div>
			continue // ERROR "continue escapes end tag"
			break // ERROR "break escapes end tag"
		</div>
	}
	return nil
}

func _(tgo.Ctx) error {
	<div>
		continue // ERROR "continue not in for statement"
		break // ERROR "break not in for, switch, or select statement"
	</div>
	return nil
}

func _(tgo.Ctx) error {
	<div
		for {
			continue
			break
		}
	>
	</div>
	return nil
}

func _(tgo.Ctx) error {
	for {
		<div>
			for {
				continue
				break
			}
			continue // ERROR "continue escapes end tag"
			break // ERROR "break escapes end tag"
		</div>
		continue
		break
	}
	return nil
}

func _(tgo.Ctx) error {
	for {
		<div>
			for {
				continue
				break
			}
			switch "a" {
			case "a":
				break
			default:
				continue // ERROR "continue escapes end tag"
			}
			switch any(nil).(type) {
			case nil:
				break
			default:
				continue // ERROR "continue escapes end tag"
			}
		</div>
	}
	return nil
}

func _(tgo.Ctx) error {
	<div>
	outer:
		for {
			continue outer
			break outer
		}
	</div>
	return nil
}

func _(tgo.Ctx) error {
outer:
	for {
		<div>
			continue outer // ERROR "invalid continue label outer exits body tag"
			break outer // ERROR "invalid break label outer exits body tag"
		</div>
	}
	return nil
}

func _(tgo.Ctx) error {
outer:
	for {
		<div>
			for {
				continue
				break
			}
			for {
				continue outer // ERROR "invalid continue label outer exits body tag"
				break outer // ERROR "invalid break label outer exits body tag"
			}
		</div>
	}
	return nil
}
