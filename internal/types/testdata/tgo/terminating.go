package test

import "github.com/mateusz834/tgo"

func _(tgo.Ctx) error {
	for {
		<div>
			break // ERROR "break prevents reaching the end tag"
		</div>
	}
} // ERROR "missing return"

func _(tgo.Ctx) error {
	for {
		<div>
			continue // ERROR "continue prevents reaching the end tag"
		</div>
	}
}

func _(tgo.Ctx) error {
	<div>
		for {
			continue
		}
	</div>
} // ERROR "missing return"

func _(tgo.Ctx) error {
	<div>
		for {
			break
		}
	</div>
} // ERROR "missing return"

func _(tgo.Ctx) error {
	<div
		for {
			continue
		}
	>
	</div>
} // ERROR "missing return"

func _(tgo.Ctx) error {
	<div
		for {
			break
		}
	>
	</div>
} // ERROR "missing return"

func _(tgo.Ctx) error {
	"\{"test"}"
} // ERROR "missing return"

func _(tgo.Ctx) error {
	@ /* ERROR "attribute is not allowed outside a tag" */ attr
} // ERROR "missing return"

func _(tgo.Ctx) error {
	@ /* ERROR "attribute is not allowed outside a tag" */ attr="value"
} // ERROR "missing return"

func _(tgo.Ctx) error {
	@ /* ERROR "attribute is not allowed outside a tag" */ attr="\{1}"
} // ERROR "missing return"
