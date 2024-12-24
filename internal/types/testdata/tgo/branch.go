package test

import "github.com/mateusz834/tgo"

func _(tgo.Ctx) error {
	for {
		<div>
			continue // ERROR "continue prevents reaching the end tag"
			break // ERROR "break prevents reaching the end tag"
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
			for {
				continue
				break
			}
			continue // ERROR "continue prevents reaching the end tag"
			break // ERROR "break prevents reaching the end tag"
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
				continue // ERROR "continue prevents reaching the end tag"
			}
			switch any(nil).(type) {
			case nil:
				break
			default:
				continue // ERROR "continue prevents reaching the end tag"
			}
		</div>
	}
	return nil
}

func _(tgo.Ctx) error {
	for range 5 {
		<div>
			break // ERROR "break prevents reaching the end tag"
		</div>
	}

	for range 5 {
		<div>
			continue // ERROR "continue prevents reaching the end tag"
		</div>
	}

	switch "a" {
	case "a":
		<div>
			break // ERROR "break prevents reaching the end tag"
		</div>
	default:
		<div>
			if true {
				break // ERROR "break prevents reaching the end tag"
			}
		</div>
	}

	for {
		<div>
			switch "a" {
			case "a":
				<div>
					continue // ERROR "continue prevents reaching the end tag"
					break // ERROR "break prevents reaching the end tag"
				</div>
				continue // ERROR "continue prevents reaching the end tag"
			}
			continue // ERROR "continue prevents reaching the end tag"
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
				continue outer // ERROR "continue outer prevents reaching the end tag"
				break outer // ERROR "break outer prevents reaching the end tag"
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
					continue outer // ERROR "continue outer prevents reaching the end tag"
					break outer // ERROR "break outer prevents reaching the end tag"
				}
				<div>
					break outer // ERROR "break outer prevents reaching the end tag"
				</div>
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
			continue outer // ERROR "continue outer prevents reaching the end tag"
			break outer // ERROR "break outer prevents reaching the end tag"
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
				continue outer // ERROR "continue outer prevents reaching the end tag"
				break outer // ERROR "break outer prevents reaching the end tag"
			}
		</div>
	}
	return nil
}

func _(tgo.Ctx) error {
goto a
	<div>
	</div>
a:
	return nil
}

func _(tgo.Ctx) error {
a:
	<div>
	</div>
	goto a
	return nil
}

func _(tgo.Ctx) error {
	<div>
	goto a // ERROR "goto a prevents reaching the end tag"
	</div>
a:
	return nil
}

func _(tgo.Ctx) error {
a:
	<div>
	goto a // ERROR "goto a prevents reaching the end tag"
	</div>
	return nil
}

func _(tgo.Ctx) error {
	goto a // ERROR "goto a jumps into block"
	<div>
	a:
	</div>
	goto a // ERROR "goto a jumps into block"
	return nil
}

func _(tgo.Ctx) error {
	goto a // ERROR "goto a jumps into block"
	<div
	a:
		@attr="value"
	>
		goto a /* ERROR "goto a jumps into block" */ /* ERROR "goto a prevents reaching the end tag" */
	</div>
	goto a // ERROR "goto a jumps into block"
	return nil
}

func _(tgo.Ctx) error {
	<div>
		// TODO this  error is not quite right
		goto a /* ERROR "goto a jumps into block" */ /* ERROR "goto a prevents reaching the end tag" */
		<div>
		a:
		</div>
	</div>
	return nil
}

func _(tgo.Ctx) error {
	<div>
		<div>
		goto a // ERROR "goto a prevents reaching the end tag"
		</div>
	a:
	</div>
	return nil
}

func _(tgo.Ctx) error {
	<div>
	a:
		<div>
		goto a // ERROR "goto a prevents reaching the end tag"
		</div>
	</div>
	return nil
}
