package test

import "github.com/mateusz834/tgo"

func _(tgo.Ctx) error {
	<div>
		return /* ERROR "invalid return in element body" */ nil
	</div>
	return nil
}

func _(tgo.Ctx) error {
	<div
		return /* ERROR "invalid return in open tag" */ nil
	>
	</div>
	return nil
}

func _(tgo.Ctx) error {
	<br
		return /* ERROR "invalid return in open tag" */ nil
	>
	return nil
}

func _(tgo.Ctx) error {
	<div>
		<div
			return /* ERROR "invalid return in open tag" */ /* ERROR "invalid return in element body" */ nil
		>
			return /* ERROR "invalid return in element body" */ nil
		</div>
	</div>
	return nil
}
