package test

import "github.com/mateusz834/tgo"

func _() {
	< /* ERROR "open tag is not allowed inside a non-tgo function" */ div>
	</ /* ERROR "end tag is not allowed inside a non-tgo function" */ div>
	"test" // ERROR `"test" (untyped string constant) is not used`
	"\{ /* ERROR "template literal is not allowed inside a non-tgo function" */ "test"}"
	< /* ERROR "open tag is not allowed inside a non-tgo function" */ div
		@ /* ERROR "attribute is not allowed inside a non-tgo function" */ attr="value"
		@ /* ERROR "attribute is not allowed inside a non-tgo function" */ attr="\{"value"}"
		@ /* ERROR "attribute is not allowed inside a non-tgo function" */ attr
	>
	</ /* ERROR "end tag is not allowed inside a non-tgo function" */ div>
}

var _ = func() {
	< /* ERROR "open tag is not allowed inside a non-tgo function" */ div>
	</ /* ERROR "end tag is not allowed inside a non-tgo function" */ div>
	"test" // ERROR `"test" (untyped string constant) is not used`
	"\{ /* ERROR "template literal is not allowed inside a non-tgo function" */ "test"}"
	< /* ERROR "open tag is not allowed inside a non-tgo function" */ div
		@ /* ERROR "attribute is not allowed inside a non-tgo function" */ attr="value"
		@ /* ERROR "attribute is not allowed inside a non-tgo function" */ attr="\{"value"}"
		@ /* ERROR "attribute is not allowed inside a non-tgo function" */ attr
	>
	</ /* ERROR "end tag is not allowed inside a non-tgo function" */ div>
}

func _(tgo.Ctx) error {
	<div>
	</div>
	"test"
	"\{"test"}"
	<br>
	<div
		@attr="value"
		@attr="\{1}"
		@attr
	>
	</div>

	<article>
		<div>
			"test"
		</div>

		<div>
			"\{"test"}"
		</div>
	</article>

	return nil
}

var _ = func(tgo.Ctx) error {
	<div>
	</div>
	"test"
	"\{"test"}"
	<br>
	<div
		@attr="value"
		@attr="\{1}"
		@attr
	>
	</div>
	return nil
}

func _(tgo.Ctx) error {
	<div
		"test" // ERROR `"test" (untyped string constant) is not used`
		"\{ /* ERROR "template literal inside of an tag" */ "test"}"
	>
	</div>
	return nil
}

func _(tgo.Ctx) error {
	<div>
		<div
			"test" // ERROR `"test" (untyped string constant) is not used`
			"\{ /* ERROR "template literal inside of an tag" */ "test"}"
		>
		</div>
	</div>
	return nil
}

func _(tgo.Ctx) error {
	@ /* ERROR "attribute is not allowed outside a tag" */ attr="value"
	@ /* ERROR "attribute is not allowed outside a tag" */ attr="{1}"
	@ /* ERROR "attribute is not allowed outside a tag" */ attr
	<div>
		@ /* ERROR "attribute is not allowed outside a tag" */ attr="value"
		@ /* ERROR "attribute is not allowed outside a tag" */ attr="{1}"
		@ /* ERROR "attribute is not allowed outside a tag" */ attr
	</div>
	<div>
		<div>
			@ /* ERROR "attribute is not allowed outside a tag" */ attr="value"
			@ /* ERROR "attribute is not allowed outside a tag" */ attr="{1}"
			@ /* ERROR "attribute is not allowed outside a tag" */ attr
		</div>
	</div>
	return nil
}

func _(tgo.Ctx) error {
	var t func(tgo.Ctx) error
	<div
		t = func(tgo.Ctx) error {
			@ /* ERROR "attribute is not allowed outside a tag" */ attr="value"
			<div
				@attr="value"
			>
				"test"
				"\{"test"}"
				@ /* ERROR "attribute is not allowed outside a tag" */ attr="value"
			</div>
			return nil
		}
	>
	</div>
	_ = t
	return nil
}

func _(tgo.Ctx) error {
	<div
		{
			< /* ERROR "tag is not allowed inside a tag" */ div>
			</ /* ERROR "end tag is not allowed inside a tag" */ div>
		}
	>
	</div>
	return nil
}
