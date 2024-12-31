package test

import (
	"github.com/mateusz834/tgo"
	"math"
)

var (
	strVar        string         = "str"
	inteagerVar   int            = -100
	uInteagerVar  uint           = 100
	charVar       rune           = 'r'
	unsafeHTMLVar tgo.UnsafeHTML = "<div></div>"
)

const (
	strTyped        string         = "str"
	inteagerTyped   int            = -100
	uInteagerTyped  uint           = 100
	charTyped       rune           = 'r'
	unsafeHTMLTyped tgo.UnsafeHTML = "<div></div>"
)

const (
	str       = "str"
	inteager  = -100
	uInteager = 100
	char      = 'r'
)

func _(tgo.Ctx) error {
	"\{"str"} \{100} \{-100} \{'r'} \{tgo.UnsafeHTML("<div></div>")}"
	"\{strTyped} \{inteagerTyped} \{uInteagerTyped} \{charTyped} \{unsafeHTMLTyped}"
	"\{strVar} \{inteagerVar} \{uInteagerVar} \{charVar} \{unsafeHTMLVar}"
	"\{str} \{inteager} \{uInteager} \{char}"
	return nil
}

func _(tgo.Ctx) error {
	<div
		@attr="\{"str"} \{100} \{-100} \{'r'} \{tgo.UnsafeHTML("<div></div>")}"
		@attr="\{strTyped} \{inteagerTyped} \{uInteagerTyped} \{charTyped} \{unsafeHTMLTyped}"
		@attr="\{strVar} \{inteagerVar} \{uInteagerVar} \{charVar} \{unsafeHTMLVar}"
		@attr="\{str} \{inteager} \{uInteager} \{char}"
	>
	</div>
	return nil
}

func _(tgo.Ctx) error {
	"\{math.MaxUint} \{math.MaxInt}"
	"\{uint(math.MaxUint)} \{int(math.MaxInt)}"
	return nil
}

func _[T tgo.DynamicWriteAllowed](_ tgo.Ctx, t T) error {
	var zero T
	"\{zero} \{*new(T)} \{t}"
	<div
		@attr="\{zero} \{*new(T)} \{t}"
	>
	</div>
	return nil
}

func _[T int|string](_ tgo.Ctx, t T) error {
	var zero T
	"\{zero} \{*new(T)} \{t}"
	<div
		@attr="\{zero} \{*new(T)} \{t}"
	>
	</div>
	return nil
}

func _(tgo.Ctx) error {
	type strWrapperType string
	"\{strWrapperType /* ERROR "strWrapperType does not satisfy tgo.DynamicWriteAllowed" */ ("test")}"

	var a strWrapperType
	"\{a /* ERROR "strWrapperType does not satisfy tgo.DynamicWriteAllowed" */}"

	"\{3.3 /* ERROR "float64 does not satisfy tgo.DynamicWriteAllowed" */}"
	return nil
}

func _(tgo.Ctx) error {
	"\{a /* ERROR "undefined: a" */}"
	"\{tgo.unsafeHTML /* ERROR "undefined: tgo.unsafeHTML" */ ("test")}"
	return nil
}

func f1[T tgo.DynamicWriteAllowed]() T {
	return *new(T)
}

func f2[T int|string]() T {
	return *new(T)
}

type strWrapperType string

func f3[T strWrapperType|string]() T {
	return *new(T)
}

func _(tgo.Ctx) error {
	"\{f1 /* ERROR "in call to f1, cannot infer T" */ ()}"
	"\{f2 /* ERROR "in call to f2, cannot infer T" */ ()}"
	"\{f3 /* ERROR "in call to f3, cannot infer T" */ ()}"
	return nil
}
